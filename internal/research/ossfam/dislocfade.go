// dislocfade.go —— mark-last-disloc-fade 评估臂（工单O-41394：卡#578382
// mark-last deviation MR——mark/last 价差反向 mean_reversion；
// criterion 真值=fin_0243 research.disloc_fade.* app_config 机读）。
//
// 信号：md_ticks（state 表最新行）last 价 vs md_mark_index
// channel='mark-price' px——disloc_bp=|mark−last|/last×1e4，
// ≥disloc_gate_bp(10)=候选错位事件；fade 方向=向 mark 收敛
// （last<mark→long_perp；last>mark→short_perp，evidence.fade_side）。
//
// 期望净 APR（卡载结构判词——「结构判词无数值键」钦定）：
// per-event MR edge 卡载区间 edge_bp_min~edge_bp_max 取中值为点估
// （evidence.edge_bp_basis='card_range_mid' 如实标）；单笔净边际=
// edge_mid − friction_rt×1e4（taker RT 7.2bp）。event_rate 不可测
// ——md_ticks/md_mark_index 为 state 表无历史，falsify 判式
// event_rate×net_edge 无分母可算 → gross_apr/net_apr=0 如实空，
// 判词全压 per_event_net_bp 符号：taker 成本>edge 卡载即结构性
// 非正（maker-only 或更大错位闸=另案），verdict=rejected/
// 'edge_sub_cost'；错位<闸='disloc_thin'；mark/last 缺=该 inst
// 不入卡（数据不可证不虚构）。影子径零提案零执行。
package ossfam

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
)

type dislocParams struct {
	ScanIv     time.Duration
	StaleMin   float64
	GateBP     float64
	EdgeMinBP  float64
	EdgeMaxBP  float64
	FrictionRT float64
}

var dislocKeys = []string{
	"research.disloc_fade.scan_interval_min", "research.disloc_fade.stale_min",
	"research.disloc_fade.disloc_gate_bp", "research.disloc_fade.edge_bp_min",
	"research.disloc_fade.edge_bp_max", "research.disloc_fade.friction_rt",
}

// DislocRunner mark-last 错位反向评估岗（md 池——md_ticks/
// md_mark_index state 面+fin_opp_eval 正本，研究域同池惯例）。
type DislocRunner struct{ *arm }

func NewDislocRunner(pool *pgxpool.Pool, cfg *config.Client) *DislocRunner {
	return &DislocRunner{newArm(pool, cfg, rejects.New("research", "disloc_fade_missing_params", "disloc_fade_write_err", "gate_kill"), "评估臂-disloc-fade",
		"mark-last 错位反向评估臂（mark-last-disloc-fade candidate 态——md_ticks×md_mark_index→fin_opp_eval 影子产出零执行零提案，O-41394）",
		"mark-last-disloc-fade", "perp_fade")}
}

// AttachAlerter 挂业主告警通道（臂静默——T-0117 返修同径；具型返回保链）。
func (r *DislocRunner) AttachAlerter(a *notify.Alerter) *DislocRunner {
	r.arm.AttachAlerter(a)
	return r
}

// ScanOnce 单拍壳。
func (r *DislocRunner) ScanOnce(ctx context.Context) error {
	_, err := r.tick(ctx)
	return err
}

// Run 在岗。
func (r *DislocRunner) Run(ctx context.Context, errc chan<- error) {
	r.arm.Run(ctx, errc, r.tick)
}

// params 判据全 app_config 机读（缺/坏=fail-closed）。结构下界：
// stale_min>0、gate_bp>0、0<edge_min≤edge_max、friction_rt≥0。
func (r *DislocRunner) params() (dislocParams, error) {
	var p dislocParams
	get := func(k string) (float64, error) { return r.cfg.GetFloat(k) }
	var err error
	var iv float64
	if iv, err = get(dislocKeys[0]); err != nil {
		return p, fmt.Errorf("%s: %w", dislocKeys[0], err)
	}
	p.ScanIv = time.Duration(iv) * time.Minute
	vals := make([]float64, len(dislocKeys)-1)
	for i, k := range dislocKeys[1:] {
		if vals[i], err = get(k); err != nil {
			return p, fmt.Errorf("%s: %w", k, err)
		}
	}
	p.StaleMin, p.GateBP, p.EdgeMinBP = vals[0], vals[1], vals[2]
	p.EdgeMaxBP, p.FrictionRT = vals[3], vals[4]
	if p.StaleMin <= 0 || p.GateBP <= 0 || p.EdgeMinBP <= 0 ||
		p.EdgeMaxBP < p.EdgeMinBP || p.FrictionRT < 0 {
		return p, fmt.Errorf("判据下界违例 stale_min=%.1f gate=%.2fbp edge=[%.1f,%.1f] friction=%.5f（结构下界）",
			p.StaleMin, p.GateBP, p.EdgeMinBP, p.EdgeMaxBP, p.FrictionRT)
	}
	return p, nil
}

// tick 单拍：判据→gate→state 面评估→批落表。
func (r *DislocRunner) tick(ctx context.Context) (time.Duration, error) {
	to := 2 * time.Minute
	if v, err := r.cfg.GetFloat("funding.scan_timeout_s"); err == nil && v > 0 {
		to = time.Duration(v) * time.Second
	}
	sctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	ctx = sctx
	if k, err := r.cfg.GetBool("gate.kill.disloc_fade"); err == nil && k {
		r.rej.Inc("gate_kill")
		return 0, nil
	}
	r.watch.SetBeat(24 * time.Hour)
	r.watch.Check(ctx)
	p, err := r.params()
	if err != nil {
		r.rej.Inc("disloc_fade_missing_params")
		return 0, fmt.Errorf("判据缺位=不评估（fail-closed）: %w", err)
	}
	rows, err := r.evalUniverse(ctx, p)
	if err != nil {
		r.rej.Inc("disloc_fade_write_err")
		return 0, err
	}
	if err := r.commit(ctx, rows); err != nil {
		r.rej.Inc("disloc_fade_write_err")
		return 0, err
	}
	if len(rows) > 0 {
		r.watch.OK()
	} else {
		r.watch.Quiet()
	}
	return p.ScanIv, nil
}

// evalUniverse md_ticks.last × md_mark_index(mark-price) state join——
// 双面齐备∧tick 新鲜才入卡（mark 缺=无锚不评；tick 陈=快照禁令同
// funding 新鲜度语义）。state 表无历史→event_rate 不可测→结构判词。
func (r *DislocRunner) evalUniverse(ctx context.Context, p dislocParams) ([]evalRow, error) {
	cutoff := time.Now().Add(-time.Duration(p.StaleMin * float64(time.Minute)))
	rows, err := r.pool.Query(ctx, `SELECT t.inst_id, t.last, m.px, t.recv_ts
		FROM md_ticks t
		JOIN md_mark_index m
		  ON m.venue=t.venue AND m.inst_id=t.inst_id AND m.channel='mark-price'
		WHERE t.venue=$1 AND t.recv_ts > $2
		  AND t.last IS NOT NULL AND t.last > 0
		  AND m.px IS NOT NULL AND m.px > 0`, fundingVenue, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	edgeMidBP := (p.EdgeMinBP + p.EdgeMaxBP) / 2
	frictionBP := p.FrictionRT * 1e4
	perEventNetBP := edgeMidBP - frictionBP
	var out []evalRow
	for rows.Next() {
		var inst string
		var last, mark float64
		var recvTs time.Time
		if err := rows.Scan(&inst, &last, &mark, &recvTs); err != nil {
			return nil, err
		}
		dislocBP := math.Abs(mark-last) / last * 1e4
		extreme := dislocBP >= p.GateBP
		fadeSide := "long_perp"
		if last > mark {
			fadeSide = "short_perp" // last 高于 mark→向 mark 收敛=空
		}
		verdict, reason := "rejected", "disloc_thin"
		switch {
		case extreme && perEventNetBP > 0:
			verdict, reason = "proposed", ""
		case extreme:
			reason = "edge_sub_cost"
		}
		out = append(out, evalRow{
			inst: inst, verdict: verdict, reason: reason,
			grossAPR: 0, netAPR: 0, // event_rate 不可测——结构判词如实空（卡载钦定）
			legs: []string{"perp_fade_dyn"},
			evidence: map[string]any{
				"mark_px": mark, "last_px": last,
				"disloc_bp": dislocBP, "disloc_gate_bp": p.GateBP,
				"edge_bp_min": p.EdgeMinBP, "edge_bp_max": p.EdgeMaxBP,
				"edge_bp_mid": edgeMidBP, "edge_bp_basis": "card_range_mid",
				"friction_bp": frictionBP, "per_event_net_bp": perEventNetBP,
				"fade_side": fadeSide, "tick_recv_ts": recvTs, "stale_min": p.StaleMin,
				"event_rate_note": "state 表无历史——event_rate×net_edge falsify 判式无分母，结构判词压 per_event_net_bp 符号",
				"reason":          reason,
			},
		})
	}
	return out, rows.Err()
}
