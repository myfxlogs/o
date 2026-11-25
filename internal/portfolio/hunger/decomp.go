package hunger

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// decomp_pos_fragment 升件类（O-39480 细胞A——已部署分解眼：
// 「哪个分解件持续正 ∧ 其负件吞噬比超阈」只述事实，定性归唤知）。
const decompEscClass = "decomp_pos_fragment"

// decompMinGain/decompDragRatio/decompCooldownH 阈件全 cfg 读
// （explore.decomp_*——fin_0263 种键；缺/坏=fail-closed 上抛）。

// ProbeDecompWatch 已部署分解眼一拍（hungerTick 同槽——ProbeDeskLifecycle
// 后 Detect 前钦定位；败情隔离位同款归 hungerTick 自报非静默）：
//
//	bot面：fin_bot_equity_snap 每 bot 两窗（24h/全期）——grid_profit
//	  窗内单调非降正增 ∧ float_profit 末值<0 ∧ |float|>grid×drag_ratio
//	  ∧ 窗增量≥min_gain → scope=bot:<id>；
//	paper面：fin_paper_fill 按 strategy 分解——租金件(funding/swap/
//	  swap_snapshot)正∧价差件(pnl/close)负，或价差件正∧租金件负，
//	  同吞噬比判据 → scope=strategy:<id>（24h 窗——spec 窗钉只在 bot 面）。
//
// 升件 note=实据（件名/两窗增量/吞噬比/窗龄），禁对策词；同 scope
// cooldown_h 内抑复。
func (s *Sensor) ProbeDecompWatch(ctx context.Context) error {
	minGain, err := s.cfg.GetFloat("explore.decomp_min_gain_usd")
	if err != nil {
		return fmt.Errorf("分解正件阈键不可读: %w", err)
	}
	ratio, err := s.cfg.GetFloat("explore.decomp_drag_ratio")
	if err != nil {
		return fmt.Errorf("分解吞噬比键不可读: %w", err)
	}
	coolH, err := s.cfg.GetInt("explore.decomp_cooldown_h")
	if err != nil {
		return fmt.Errorf("分解抑复窗键不可读: %w", err)
	}
	if minGain <= 0 || ratio <= 0 || coolH <= 0 {
		return fmt.Errorf("分解阈件非正（gain=%.3f ratio=%.3f coolH=%d——fail-closed）",
			minGain, ratio, coolH)
	}
	if err := s.decompBots(ctx, minGain, ratio, coolH); err != nil {
		return err
	}
	return s.decompPaper(ctx, minGain, ratio, coolH)
}

// decompWin 单窗分解判定结果（note 实据件）。
type decompWin struct {
	name     string
	dur      time.Duration
	posDelta float64 // 正件窗增量
	negEnd   float64 // 负件末值
	posEnd   float64 // 正件末值
	mono     bool    // 正件窗内单调非降
	qual     bool    // 本窗判据全过
}

// decompEvalBotWin bot 面单窗判定：pts=(grid,float) 对升序。
// 判据：grid 单调非降（eps 容差）∧ 窗增量≥minGain ∧ 末值>0
// ∧ float 末值<0 ∧ |float末值|>grid末值×ratio。
func decompEvalBotWin(name string, dur time.Duration, pts [][3]float64,
	minGain, ratio float64) decompWin {
	w := decompWin{name: name, dur: dur}
	if len(pts) < 2 {
		return w
	}
	w.posDelta = pts[len(pts)-1][0] - pts[0][0]
	w.posEnd = pts[len(pts)-1][0]
	w.negEnd = pts[len(pts)-1][1]
	mono := true
	for i := 1; i < len(pts); i++ {
		if pts[i][0] < pts[i-1][0]-1e-9 {
			mono = false
			break
		}
	}
	w.mono = mono
	w.qual = mono && w.posDelta >= minGain && w.posEnd > 0 &&
		w.negEnd < 0 && math.Abs(w.negEnd) > w.posEnd*ratio
	return w
}

// decompBots bot 面（fin_bot_equity_snap——快照 ~5min 拍在册）：
// 逐 bot 取 24h+全期两窗序列，同判据任一窗过→升件（note 双窗实据）。
func (s *Sensor) decompBots(ctx context.Context, minGain, ratio float64, coolH int64) error {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT bot_id FROM fin_bot_equity_snap`)
	if err != nil {
		return fmt.Errorf("分解 bot 集读败: %w", err)
	}
	var bots []int64
	for rows.Next() {
		var b int64
		if err := rows.Scan(&b); err != nil {
			rows.Close()
			return fmt.Errorf("分解 bot 扫败: %w", err)
		}
		bots = append(bots, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, b := range bots {
		if err := s.decompBotOne(ctx, b, minGain, ratio, coolH); err != nil {
			return err
		}
	}
	return nil
}

// decompBotOne 单 bot 双窗取数+判定+抑复升件。
func (s *Sensor) decompBotOne(ctx context.Context, botID int64, minGain, ratio float64, coolH int64) error {
	rows, err := s.pool.Query(ctx,
		`SELECT ts,grid_profit,float_profit FROM fin_bot_equity_snap
		 WHERE bot_id=$1 ORDER BY ts`, botID)
	if err != nil {
		return fmt.Errorf("分解快照读败 bot=%d: %w", botID, err)
	}
	type pt struct {
		ts        time.Time
		grid, flt float64
	}
	var all []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.ts, &p.grid, &p.flt); err != nil {
			rows.Close()
			return fmt.Errorf("分解快照扫败 bot=%d: %w", botID, err)
		}
		all = append(all, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(all) < 2 {
		return nil
	}
	cut24 := s.now().Add(-24 * time.Hour)
	var w24, wall [][3]float64
	for _, p := range all {
		v := [3]float64{p.grid, p.flt, 0}
		wall = append(wall, v)
		if !p.ts.Before(cut24) {
			w24 = append(w24, v)
		}
	}
	// 全期窗=全序列；24h 窗不足 2 点=该窗不参判（note 标缺）。
	d24 := decompEvalBotWin("24h", 24*time.Hour, w24, minGain, ratio)
	dall := decompEvalBotWin("all", 0, wall, minGain, ratio)
	if !d24.qual && !dall.qual {
		return nil
	}
	scope := fmt.Sprintf("bot:%d", botID)
	dup, err := s.escDup(ctx, decompEscClass, scope, coolH)
	if err != nil {
		return err
	}
	if dup {
		return nil
	}
	ageH := s.now().Sub(all[0].ts).Hours()
	qualWin := "all"
	if d24.qual && dall.qual {
		qualWin = "24h+all"
	} else if d24.qual {
		qualWin = "24h"
	}
	note := fmt.Sprintf("分解件 bot#%d：grid_profit 单调正增∧float_profit 负件吞噬——窗=%s "+
		"24h(增量=%.4f 末grid=%.4f 末float=%.4f 单调=%v) 全期(增量=%.4f 末grid=%.4f 末float=%.4f 单调=%v) "+
		"吞噬比阈=%.2f 窗龄=%.1fh 快照n=%d——正件持续增负件被吞（只述事实——定性归唤知）",
		botID, qualWin,
		d24.posDelta, d24.posEnd, d24.negEnd, d24.mono,
		dall.posDelta, dall.posEnd, dall.negEnd, dall.mono,
		ratio, ageH, len(all))
	return s.escInsert(ctx, decompEscClass, scope, note)
}

// decompPaper paper 面（fin_paper_fill——kind 分件聚合）：
// 租金件={funding,swap,swap_snapshot}，价差件={pnl,close}（open=入场锚
// amount=0、fee=费用非分量不入两侧——如实记 note）。24h 窗双向判：
// 租金正∧价差负∧|价差|>租金×ratio，或价差正∧租金负∧|租金|>价差×ratio，
// 正件窗和≥minGain。
func (s *Sensor) decompPaper(ctx context.Context, minGain, ratio float64, coolH int64) error {
	rows, err := s.pool.Query(ctx,
		`SELECT strategy_id,
		  COALESCE(sum(amount_usd) FILTER (WHERE kind IN ('funding','swap','swap_snapshot')),0),
		  COALESCE(sum(amount_usd) FILTER (WHERE kind IN ('pnl','close')),0),
		  COALESCE(sum(amount_usd) FILTER (WHERE kind='fee'),0),
		  count(*)
		 FROM fin_paper_fill WHERE ts>now()-interval '24 hours'
		 GROUP BY strategy_id`)
	if err != nil {
		return fmt.Errorf("分解 paper 面读败: %w", err)
	}
	type agg struct {
		rent, spread, fee float64
		n                 int64
	}
	var m = map[int64]agg{}
	var order []int64
	for rows.Next() {
		var sid int64
		var a agg
		if err := rows.Scan(&sid, &a.rent, &a.spread, &a.fee, &a.n); err != nil {
			rows.Close()
			return fmt.Errorf("分解 paper 扫败: %w", err)
		}
		m[sid] = a
		order = append(order, sid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	for _, sid := range order {
		a := m[sid]
		var posName, negName string
		var pos, neg float64
		switch {
		case a.rent > 0 && a.spread < 0 && math.Abs(a.spread) > a.rent*ratio && a.rent >= minGain:
			posName, negName, pos, neg = "租金件", "价差件", a.rent, a.spread
		case a.spread > 0 && a.rent < 0 && math.Abs(a.rent) > a.spread*ratio && a.spread >= minGain:
			posName, negName, pos, neg = "价差件", "租金件", a.spread, a.rent
		default:
			continue
		}
		scope := fmt.Sprintf("strategy:%d", sid)
		dup, err := s.escDup(ctx, decompEscClass, scope, coolH)
		if err != nil {
			return err
		}
		if dup {
			continue
		}
		note := fmt.Sprintf("分解件 strategy#%d：%s=%.4f 正∧%s=%.4f 负被吞（24h窗 fill n=%d fee=%.4f 不入两侧）——"+
			"吞噬比=%.2f（实据分件聚合——定性归唤知）",
			sid, posName, pos, negName, neg, a.n, a.fee,
			math.Abs(neg)/math.Max(pos, 1e-9))
		if err := s.escInsert(ctx, decompEscClass, scope, note); err != nil {
			return err
		}
	}
	return nil
}

// escDup 同 scope 冷却窗内已升件=抑复（ops_event append-only 读面——
// 判据=event='升件'+esc_class+esc_scope+窗内存在）。
func (s *Sensor) escDup(ctx context.Context, class, scope string, coolH int64) (bool, error) {
	var dup bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3))`, class, scope, coolH).Scan(&dup)
	return dup, err
}

// escInsert 升件行（seat=探测席/event=升件——breedstall 同构三件套；
// class 参数化=decomp/oppCensus 两细胞共用）。
func (s *Sensor) escInsert(ctx context.Context, class, scope, note string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket, note, class, scope)
	return err
}
