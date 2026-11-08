// open.go —— live 开仓装配缝：lifecycle.Opener→风控闸→腿构→原子执行。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/findom"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/dcdexec"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
	"o/internal/trading/okxexec"
)

// hedgeOpener funding-hedge 开仓适配（cmd 组合根——投资域不反向依赖交易/风控域）。
type hedgeOpener struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	gate  *risk.Gate
	r     *execution.Runner
	cli   *okxexec.Client
	dcd   dcdVenue             // T-0259 桩测注伪缝——nil=cli 产线形
	xcli  xvenueSess           // T-0145：mt5 懒连会话件（xvenue-carry 执行/方言/现读同件——窄面桩测注假）
	xdial mtexec.Dialect       // T-0145：方言译件（官名→venue 符号唯一天）
	xds   *mt5.DialectStore    // T-0145：方言行仓（ct_val/min_sz/lot_step 基线读）
	xmeta execution.MetaLoader // T-0145 返修①桩测缝——nil=h.loadMeta 产线形（xvenue e2e 桩绕 venue 活读）
}

// Open live 策略开仓：双帽取小→闸→腿构→原子执行。任一环败=err 上抛
// （lifecycle 计数续试——ev_strategy 事件驱非热环）。
func (h hedgeOpener) Open(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	switch spec.Template {
	case "dcd_call", "dcd_put", "vol_sell_single_call":
		// T-0160 DCD 产品单腿执行；T-0215 vol_sell_single_call 同型。
		// dcd_put=T-0353 复役同径（quote/trade 方向无关，notionalCcy
		// 从产品 raw 锁仓币回填 fail-closed；USDT 计价腿走基金钱常径）。
		return h.dcdOpen(ctx, spec)
	case "quarter_basis_long", "perp_quarter_spread":
		// T-0216 basis executor（quarter_basis_long_executor/perp_quarter_spread_
		// executor）——衍生腿=到期日期货，多头腿 spot/perp；atomic 双腿同
		// funding-hedge 径（Execute token/rollback 内建）。
		return h.basisOpen(ctx, spec)
	case "vol_sell_strangle":
		// T-0215 strangle 双腿 executor（假想卡 id=6 钦定名
		// vol_sell_strangle_executor——Cprod+Pprod 复合键拆腿序贯买）。
		return h.volSellStrangle(ctx, spec)
	case "xvenue-carry":
		// T-0145 跨所 carry 真执行——mt5 cfd 腿先行+okx perp 腿双腿原子序
		// （DM 裁定钦定面——xvenue.go）。
		return h.xvenueOpen(ctx, spec)
	}
	notional := spec.NotionalUSD
	// 双帽取小（openfirst 同径：probe_notional/max_notional——缺键 fail-closed）。
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2)
	}
	notional = min(notional, cap1, cap2)
	if notional <= 0 {
		return 0, fmt.Errorf("notional≤0 拒（strategy=%.2f caps=%.2f/%.2f）",
			spec.NotionalUSD, cap1, cap2)
	}
	// T-0107 再开冷却：guard 触发平仓后同策略同标的间隔窗（exec.reopen_cooldown_sec
	// 热启；缺键/探测败 fail-closed 拒开）——瞬态触发禁 45s 级连绞再装填。
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("冷却窗探测 fail-closed: %w", err)
	}
	if blocked {
		return 0, fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID)
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{"perp_short", "spot_long"},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	legs, _, lerr := execution.BuildLegs(ctx, h.pool, h.loadMeta, spec.Venue, spec.InstID, notional)
	if lerr != nil {
		// T-0147 终局分类收口（错误冒泡点统一——腿构 meta/校验径与下单提交
		// 径同闸）：51001 在腿构 spot 元数据拉取段返回（实案 286078），
		// TerminalInstErr 分类不得只钉 order-submit 出口。
		if okxexec.TerminalInstErr(lerr) {
			if herr := h.untradeable(ctx, spec, lerr); herr != nil {
				return 0, fmt.Errorf("终局处置败（须人查）: %w", herr)
			}
			return 0, fmt.Errorf("venue_untradeable: %w", lerr)
		}
		return 0, fmt.Errorf("腿构: %w", lerr)
	}
	eid, xerr := h.r.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional, Legs: legs})
	// T-0142 终局分类：instrument 终局拒（51001/51003 族）=venue 拒语终局事实
	// ——黑名单入册+策略退（retired,venue_untradeable）+同 inst armed 提案退。
	// 策略退后 openLive 查询（state='live'）不再选中=续试风暴根除（禁事件续试）。
	if okxexec.TerminalInstErr(xerr) {
		if herr := h.untradeable(ctx, spec, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}

// untradeable 终局处置三件（T-0142）：黑名单入册（学费一次）→策略退
// （Transition 留痕同决策席手动处置写径）→同 inst armed 提案退 fin_retired。
func (h hedgeOpener) untradeable(ctx context.Context, spec lifecycle.OpenSpec, xerr error) error {
	var ae *okxexec.APIError
	_ = errors.As(xerr, &ae)
	code, msg := "", ""
	if ae != nil {
		code, msg = ae.Code, ae.Msg
	}
	// T-0147 返修消解（显式迁移先于入册——后者更优形）：显式 Transition 的
	// reason 精确（venue_untradeable:<code>），随后入册——0061 trigger 续扫
	// 同 inst 余量（其余策略/armed 提案=真回扫覆盖）。顺序反置则 trigger 先
	// 退役本策略，显式迁移撞「retired→retired 非法迁移」闭集。
	// 容退守卫（T-0151 谓词集对齐 0062——retired 外五态全可显式迁移，序位
	// 不动）：state IN 五态才显式迁移（已 retired=幂等重入/trigger 先至=跳过，
	// 溯源性已由先至方 transition 行承载）。
	var live int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE id=$1 AND state IN ('live','paper','proposed','probe','degraded')`,
		spec.StrategyRef).Scan(&live); err != nil {
		return err
	}
	if live > 0 {
		if err := lifecycle.Transition(ctx, h.pool, spec.StrategyRef, "retired",
			"venue_untradeable:"+code+" "+msg, false); err != nil {
			return fmt.Errorf("策略退: %w", err)
		}
	}
	if _, err := h.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired',retire_reason='venue_untradeable',retired_at=now()
		WHERE inst_id=$1 AND state='fin_armed'`, spec.InstID); err != nil {
		return err
	}
	if _, err := h.pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES($1,$2,$3)
		ON CONFLICT (inst_id) DO UPDATE SET code=EXCLUDED.code, msg=EXCLUDED.msg`,
		spec.InstID, code, msg); err != nil {
		return err
	}
	log.Printf("ALERT [open] 终局拒 %s（%s %s）——显式迁移先于入册+trigger 续扫余量",
		spec.InstID, code, msg)
	return nil
}

// loadMeta 乐器元数据现读（/public/instruments——SPOT/SWAP 按后缀分派）。
func (h hedgeOpener) loadMeta(ctx context.Context, iid string) (execution.InstMeta, error) {
	it := "SPOT"
	if strings.HasSuffix(iid, "-SWAP") {
		it = "SWAP"
	}
	if datedFutRe.MatchString(iid) { // T-0216：-YYMMDD 到期日期货=FUTURES 乐器域
		it = "FUTURES"
	}
	m, err := h.cli.InstrumentMeta(ctx, it, iid)
	return execution.InstMeta{CtVal: m.CtVal, CtValCcy: m.CtValCcy, LotSz: m.LotSz, MinSz: m.MinSz}, err
}

// dcdOpen DCD 产品开仓（T-0160 ②executor→T-0461 抽立 internal/trading/
// dcdexec 单源——产线与 simvenue 矩阵 DCD 演习格同径消费，venue 三步缝
// 注入式全产线同码；cmd 侧仅规格映射）。
func (h hedgeOpener) dcdOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	return dcdexec.Open(ctx, dcdexec.Deps{Pool: h.pool, Gate: h.gate, Dom: findom.Live}, h.dcdCli(),
		dcdexec.Spec{StrategyRef: spec.StrategyRef, Venue: spec.Venue, InstID: spec.InstID,
			Direction: spec.Direction, Template: spec.Template, NotionalUSD: spec.NotionalUSD})
}
