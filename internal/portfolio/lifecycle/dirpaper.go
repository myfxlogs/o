// dirpaper.go —— 方向族 paper MTM 记账+出场面（棒1 paper-only 族消费径）：
// dir_perp_mom 单腿方向位=价变动量仓——funding 腿由通用 paperAccounting
// 照记（perp 方向行），本件补价差 PnL 摊提笔（kind='pnl'——收入流观测
// 粒度入 confGate 序列/晋升净额）+四路出场（止损/止盈/TTL/信号翻面）。
// 价源 md_ticks WS 行（venue<>'sim'，鲜度闸 exec.tick_max_age_s 同 dcd 径；
// 陈=该拍让位 fail-closed 不造 fill）。出场=Transition 留痕同 retire 径。
package lifecycle

import (
	"context"
	"log"
	"time"

	"o/internal/platform/execfam"
)

// dirPaperAccounting paper 态方向族 MTM 记账+出场判定（非致命径——记账败
// 不拖开户/晋升拍，同 basis/xvenue 位）。
func (r *Runner) dirPaperAccounting(ctx context.Context) error {
	markIv, stop, take, ttl := 240.0, 0.15, 0.30, 240.0 // 缺省判据（键缺=内置值非虚构——lifecycle.* 懒读惯例同 armed_ttl_h）
	if v, err := r.cfg.GetFloat("lifecycle.dir.mark_iv_min"); err == nil && v > 0 {
		markIv = v
	}
	if v, err := r.cfg.GetFloat("lifecycle.dir.stop_frac"); err == nil && v > 0 {
		stop = v
	}
	if v, err := r.cfg.GetFloat("lifecycle.dir.take_frac"); err == nil && v > 0 {
		take = v
	}
	if v, err := r.cfg.GetFloat("lifecycle.dir.ttl_h"); err == nil && v > 0 {
		ttl = v
	}
	maxAge := 120.0
	if v, err := r.cfg.GetFloat("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd,
		s.entered_at,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f WHERE f.strategy_id=s.id
			AND f.kind IN ('open','pnl')), s.entered_at),
		COALESCE((SELECT sum(amount_usd) FROM fin_paper_fill f WHERE f.strategy_id=s.id
			AND f.kind='pnl'),0),
		(SELECT (f.detail#>>'{px}')::float8 FROM fin_paper_fill f
			WHERE f.strategy_id=s.id AND f.kind IN ('open','pnl')
			ORDER BY ts DESC LIMIT 1)
		FROM fin_strategy s WHERE s.state='paper' AND s.template_ref=ANY($1)`,
		execfam.PaperOnlyList())
	if err != nil {
		return err
	}
	type st struct {
		id       int64
		inst     string
		dir      string
		notional float64
		entered  time.Time
		lastMark time.Time
		cumPnl   float64
		lastPx   *float64
	}
	var sts []st
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.id, &x.inst, &x.dir, &x.notional,
			&x.entered, &x.lastMark, &x.cumPnl, &x.lastPx); err != nil {
			rows.Close()
			return err
		}
		sts = append(sts, x)
	}
	rows.Close()
	for _, x := range sts {
		if x.lastPx != nil && time.Since(x.lastMark) < time.Duration(markIv*float64(time.Minute)) {
			continue // 摊提节距未满——粒度闸非事实缺（首见=立即锚入场价）
		}
		var px float64
		var tickTs time.Time
		if err := r.pool.QueryRow(ctx, `SELECT last,ts FROM md_ticks
			WHERE inst_id=$1 AND venue<>'sim' ORDER BY ts DESC LIMIT 1`, x.inst).Scan(&px, &tickTs); err != nil || px <= 0 ||
			time.Since(tickTs) > time.Duration(maxAge*float64(time.Second)) {
			r.rej.Inc("dir_mark_stale")
			continue // 价缺/陈=事实缺让位不造 fill（T-0171 同径）
		}
		if x.lastPx == nil {
			// 首见=开户锚（kind='open' amount=0 detail 载入场价——价差基准行）。
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
				VALUES($1,'open',$2,0,$3,now())`, x.id, x.inst,
				[]byte(`{"px":`+ftoa(px)+`,"src":"md_ticks"}`)); err != nil {
				return err
			}
			continue
		}
		sign := 1.0
		if x.dir == "short_perp" {
			sign = -1.0
		}
		delta := sign * x.notional * (px / *x.lastPx - 1)
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'pnl',$2,$3,$4,now())`, x.id, x.inst, delta,
			[]byte(`{"px":`+ftoa(px)+`,"prev":`+ftoa(*x.lastPx)+`}`)); err != nil {
			return err
		}
		cum := x.cumPnl + delta
		var reason string
		switch {
		case cum <= -stop*x.notional:
			reason = "dir_stop_loss"
		case cum >= take*x.notional:
			reason = "dir_take_profit"
		case time.Since(x.entered) > time.Duration(ttl*float64(time.Hour)):
			reason = "dir_ttl"
		default:
			// 信号翻面出场：自身方向最新评估已拒=持仓信念消亡（z 弱化/
			// 反信号——方向仓的信念续存判据，与 TTL 机械闸互补）。
			var v string
			if err := r.pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
				WHERE inst_id=$1 AND template='dir_perp_mom' AND direction=$2
				ORDER BY ts DESC LIMIT 1`, x.inst, x.dir).Scan(&v); err == nil && v == "rejected" {
				reason = "dir_signal_flip"
			}
		}
		if reason == "" {
			continue
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'close',$2,0,$3,now())`, x.id, x.inst,
			[]byte(`{"px":`+ftoa(px)+`,"reason":"`+reason+`"}`)); err != nil {
			return err
		}
		if err := Transition(ctx, r.pool, x.id, "retired", reason, false); err != nil {
			r.rej.Inc("illegal_transition")
			return err
		}
		log.Printf("[lifecycle] #%d %s %s 出场退役（%s cum_pnl=%.2f）", x.id, x.inst, x.dir, reason, cum)
	}
	return nil
}
