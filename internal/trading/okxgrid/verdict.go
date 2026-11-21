// verdict.go —— bot 判决环（业主令 2026-11-21 全域自判授权：
// 所有策略/订单的开平加减全归 O 自判，判据 O 自定禁给建议值；
// 唯一业主判据=全项目年总收益>8% 实证，证实前不得请款增资）。
//
// 每拍（snapSync 后）对在飞 bot 读 fin_bot_equity_snap 快照账下三判：
//
//	平:age≥warmup 且 total_pnl≤-stop_frac×investment 于 patience 窗内
//	   快照全亏穿 → venue Stop(钦定回收形 stopType)+state迁+判例行
//	加:age≥min_age 且网格 APR≥scale_apr_min 且 float≥-float_frac×investment
//	   且聚合投入+scale_sz≤agg_cap 且同 inst 平行数<max_parallel
//	   → 平行 bot 再拉（克隆参数换 quoteSz）
//	收:grid_profit≥harvest_min 且距上次收割≥cooldown → WithdrawIncome
//
// 判据全 dm 域（机器自调——业主令钦「判据 O 自行判断」）；agg_cap 是
// 线内机器护栏非业主键（红线=本金总额一条业主线，线内归 system 域自裁）。
// 幂等:state 迁移防重停、平行计数防重开、收割阈+cooldown 防重收；
// 手动单(source=manual-owner-order)同入判域——业主令授权不分出处。
package okxgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

// verdictCfg 判据集（dm 键机器自调——业主令「判据 O 自行判断」钦定域）。
type verdictCfg struct {
	stopFrac      float64 // total_pnl ≤ -stopFrac×investment = 亏穿线
	stopPatienceH float64 // 亏穿持续窗（快照全穿才算）
	stopWarmupH   float64 // 判官热身——新单不判死
	scaleMinAgeH  float64
	scaleAprMin   float64 // 网格 APR 判据（grid_profit/invest/龄×365d）
	scaleFloatMax float64 // 浮亏界（float ≥ -frac×investment）
	scaleSize     float64
	aggCap        float64 // 聚合投入帽（机器线内护栏）
	maxParallel   int
	harvestMin    float64
	harvestCoolS  float64
	enabled       bool
}

func (r *Runner) verdictCfg(ctx context.Context) verdictCfg {
	f := func(k string, d float64) float64 {
		if v, err := r.cfg.GetFloat(k); err == nil && v > 0 {
			return v
		}
		return d
	}
	en := true
	if v, err := r.cfg.GetBool("okxgrid.verdict_enabled"); err == nil && !v {
		en = false
	}
	return verdictCfg{
		stopFrac: f("okxgrid.stop_loss_frac", 0.15), stopPatienceH: f("okxgrid.stop_patience_h", 72),
		stopWarmupH: f("okxgrid.stop_warmup_h", 24), scaleMinAgeH: f("okxgrid.scale_min_age_h", 168),
		scaleAprMin: f("okxgrid.scale_apr_min", 0.20), scaleFloatMax: f("okxgrid.scale_float_max_frac", 0.05),
		scaleSize: f("okxgrid.scale_size_usd", 100), aggCap: f("okxgrid.agg_invest_max_usd", 500),
		maxParallel: 3, harvestMin: f("okxgrid.harvest_min_usd", 5),
		harvestCoolS: f("okxgrid.harvest_cooldown_s", 86400), enabled: en,
	}
}

// snapRow 快照行（判官读面）。
type snapRow struct {
	invest, gridP, floatP, totalP float64
}

// verdictKind 判词三态。
type verdictKind int

const (
	vHold verdictKind = iota
	vStop
	vScale
)

// judge 纯判定（零 IO 可测）：ageH=单龄小时，bad=窗内亏穿快照数,all=窗内快照总数。
func (c verdictCfg) judge(ageH float64, s snapRow, bad, all int) (verdictKind, string) {
	if ageH >= c.stopWarmupH && all >= 3 && bad == all &&
		s.totalP <= -c.stopFrac*s.invest && s.invest > 0 {
		return vStop, fmt.Sprintf("总收益 %.2f 亏穿 -%.0f%%×投额 %.0f 持续窗全穿(%d/%d快照)", s.totalP, c.stopFrac*100, s.invest, bad, all)
	}
	if s.invest > 0 && ageH >= c.scaleMinAgeH {
		apr := s.gridP / s.invest / ageH * 24 * 365
		if apr >= c.scaleAprMin && s.floatP >= -c.scaleFloatMax*s.invest {
			return vScale, fmt.Sprintf("网格 APR %.1f%%≥%.0f%% 且浮亏 %.2f 在界——产能实证可加", apr*100, c.scaleAprMin*100, s.floatP)
		}
	}
	return vHold, ""
}

// verdictBots 判决拍——snapSync 后对在飞 bot 逐台下判。
func (r *Runner) verdictBots(ctx context.Context) error {
	vc := r.verdictCfg(ctx)
	if !vc.enabled {
		return nil
	}
	kill, _ := r.cfg.GetBool("gate.kill.okx_grid")
	sw, _ := r.cfg.GetBool("okxgrid.live_switch")
	rows, err := r.pool.Query(ctx,
		`SELECT b.id,b.algo_id,b.inst_id,b.params,COALESCE(b.strategy_id,0),
		        EXTRACT(EPOCH FROM (now()-b.launched_at))/3600.0,
		        s.investment,s.grid_profit,s.float_profit,s.total_pnl
		 FROM fin_bot_order b JOIN LATERAL (
		   SELECT investment,grid_profit,float_profit,total_pnl
		   FROM fin_bot_equity_snap e WHERE e.bot_id=b.id ORDER BY ts DESC LIMIT 1
		 ) s ON true WHERE b.state='launched'`)
	if err != nil {
		return err
	}
	type cand struct {
		id, sid    int64
		algo, inst string
		params     json.RawMessage
		ageH       float64
		s          snapRow
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.algo, &c.inst, &c.params, &c.sid, &c.ageH,
			&c.s.invest, &c.s.gridP, &c.s.floatP, &c.s.totalP); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	var agg float64
	for _, c := range cs {
		agg += c.s.invest
	}
	for _, c := range cs {
		var bad, all int
		if err := r.pool.QueryRow(ctx,
			`SELECT count(*) FILTER (WHERE total_pnl <= $2), count(*)
			 FROM fin_bot_equity_snap WHERE bot_id=$1 AND ts>now()-make_interval(hours=>$3)`,
			c.id, -vc.stopFrac*c.s.invest, vc.stopPatienceH).Scan(&bad, &all); err != nil {
			log.Printf("[okxgrid] bot#%d 窗快照读败（判官让位下拍）: %v", c.id, err)
			continue
		}
		switch k, why := vc.judge(c.ageH, c.s, bad, all); k {
		case vStop:
			if kill {
				continue // 拔线闸在案——killWatch 段接管,判官不重复处置
			}
			st := "1"
			if v, err := r.cfg.GetFloat("okxgrid.kill_stop_type"); err == nil && v == 2 {
				st = "2"
			}
			if err := r.ad.Stop(ctx, c.algo, c.inst, st); err != nil {
				r.esc(ctx, "okxgrid_verdict_stop_fail", "钱径",
					fmt.Sprintf("bot#%d %s 判平(stopType=%s)venue 停败（下拍重试）: %v", c.id, c.inst, st, err))
				continue
			}
			if _, err := r.pool.Exec(ctx,
				`UPDATE fin_bot_order SET state='stopped',stopped_at=now() WHERE id=$1`, c.id); err != nil {
				return err
			}
			r.note(ctx, fmt.Sprintf("判例:bot#%d %s 判平——%s（venue 已停回收形%s）", c.id, c.inst, why, st))
			log.Printf("[okxgrid] bot#%d 判平: %s", c.id, why)
		case vScale:
			par, aggOK := 0, agg+vc.scaleSize <= vc.aggCap
			_ = r.pool.QueryRow(ctx,
				`SELECT count(*) FROM fin_bot_order WHERE inst_id=$1 AND state='launched'`,
				c.inst).Scan(&par)
			if !sw || kill || !aggOK || par >= vc.maxParallel {
				continue
			}
			var p struct {
				MaxPx, MinPx float64
				GridNum      int
			}
			if err := json.Unmarshal(c.params, &p); err != nil || p.MaxPx <= p.MinPx || p.GridNum < 2 {
				continue
			}
			if err := r.ensureFunds(ctx, quoteCcy(c.inst), vc.scaleSize); err != nil {
				var br errBalanceRead
				if !errors.As(err, &br) {
					r.esc(ctx, "okxgrid_verdict_scale_hold", "钱径",
						fmt.Sprintf("bot#%d %s 判加 $%.0f 资金棒歧态——禁自动重试（歧态势铁律）: %v", c.id, c.inst, vc.scaleSize, err))
				}
				continue
			}
			algoID, err := r.ad.Launch(ctx, LaunchReq{
				InstID: c.inst, MaxPx: p.MaxPx, MinPx: p.MinPx,
				GridNum: p.GridNum, QuoteSz: vc.scaleSize,
				AlgoClID: fmt.Sprintf("ogv%dx%d", c.id, par+1), Tag: "okxgrid-verdict",
			})
			if err != nil {
				if launchDefinite(err) {
					r.esc(ctx, "okxgrid_verdict_scale_fail", "钱径",
						fmt.Sprintf("bot#%d %s 判加拉起败（$%.0f）: %v", c.id, c.inst, vc.scaleSize, err))
				}
				continue // 不确定态——不补单（歧态势铁律,下拍不判同因重试?快照变则重判）
			}
			params, _ := json.Marshal(map[string]any{
				"maxPx": p.MaxPx, "minPx": p.MinPx, "gridNum": p.GridNum,
				"quoteSz": vc.scaleSize, "runType": "1", "source": "verdict-scale", "parent_bot": c.id,
			})
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_bot_order
				(venue,inst_id,algo_id,algo_ord_type,params,state)
				VALUES('okx',$1,$2,'grid',$3,'launched')`, c.inst, algoID, params); err != nil {
				return err
			}
			agg += vc.scaleSize
			r.note(ctx, fmt.Sprintf("判例:bot#%d %s 判加 $%.0f——%s（新 bot algoId=%s 平行第%d台）", c.id, c.inst, vc.scaleSize, why, algoID, par+1))
			log.Printf("[okxgrid] bot#%d 判加: %s", c.id, why)
		}
		// 收割判（独立于平/加——产利落 quote 回笼）
		if c.s.gridP >= vc.harvestMin {
			var last time.Time
			_ = r.pool.QueryRow(ctx,
				`SELECT max(ts) FROM ops_event WHERE seat='okxgrid' AND note LIKE $1`,
				"%bot#"+fmt.Sprint(c.id)+" %收割%").Scan(&last)
			if time.Since(last).Seconds() < vc.harvestCoolS && !last.IsZero() {
				continue
			}
			got, err := r.ad.WithdrawIncome(ctx, c.algo)
			if err != nil {
				r.esc(ctx, "okxgrid_harvest_fail", "钱径",
					fmt.Sprintf("bot#%d %s 收割败（grid_profit=%.2f≥阈%.0f）: %v", c.id, c.inst, c.s.gridP, vc.harvestMin, err))
				continue
			}
			r.note(ctx, fmt.Sprintf("判例:bot#%d %s 收割 $%.2f→quote（grid_profit 达阈）", c.id, c.inst, got))
		}
	}
	return nil
}

func (r *Runner) note(ctx context.Context, note string) {
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,note) VALUES('okxgrid','判例',$1)`, note); err != nil {
		log.Printf("[okxgrid] 判例行写败: %v", err)
	}
}
