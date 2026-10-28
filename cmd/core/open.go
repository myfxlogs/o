// open.go —— live 开仓装配缝（T-0050 首仓序④）：lifecycle.Opener 适配——
// risk.Gate.Check（唯一交易前闸）→execution.BuildLegs（perp 短+现货长，乐器
// 元数据决策时点 instruments 现读）→Runner.Execute（token/对冲带/Earn 赎回/
// 双腿原子——与 openfirst 同码径，事件驱自动径）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/okxexec"
)

// hedgeOpener funding-hedge 开仓适配（cmd 组合根——投资域不反向依赖交易/风控域）。
type hedgeOpener struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	gate *risk.Gate
	r    *execution.Runner
	cli  *okxexec.Client
}

// Open live 策略开仓：双帽取小→闸→腿构→原子执行。任一环败=err 上抛
// （lifecycle 计数续试——ev_strategy 事件驱非热环）。
func (h hedgeOpener) Open(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	switch spec.Template {
	case "dcd_call", "vol_sell_single_call":
		// T-0160 DCD 执行面分派（产品买非双腿原子径）；T-0215
		// vol_sell_single_call=同型单腿 executor（假想卡 id=7 钦定名
		// vol_sell_single_call_executor）。
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
	if _, err := h.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired'
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

// reopenCooldownBlocked 再开冷却窗判定（T-0107）：同策略同标的存在冷却窗内
// guard 平仓记录（updated_at=任一 guard 态变时点）→true。判据=config 快照件
// GetFloat（缺键 fail-closed true——宁可冻结不再开也不裸奔连绞）；0=显式关闭。
func reopenCooldownBlocked(ctx context.Context, cfg *config.Client, pool *pgxpool.Pool,
	strategyRef int64, instID string) (bool, error) {
	sec, err := cfg.GetFloat("exec.reopen_cooldown_sec")
	if err != nil {
		return true, fmt.Errorf("exec.reopen_cooldown_sec: %w", err)
	}
	if sec <= 0 {
		return false, nil // 冷却显式关闭（0=运维豁免位）
	}
	var blocked bool
	err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_guard_close g
		JOIN fin_position p ON p.id=g.position_id
		WHERE p.strategy_ref=$1 AND p.inst_id=$2
		  AND g.updated_at > now()-make_interval(secs=>$3))`,
		strategyRef, instID, sec).Scan(&blocked)
	return blocked, err
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

// dcdOpen DCD 产品开仓（T-0160 ②executor：quote 重取→trade→order-status
// 对拍→fin_execution/fin_position 落位——陈旧提案 productId 重 quote 不盲执；
// clOrdId=幂等键，歧态势=禁补单走 order-status 定真态）。
func (h hedgeOpener) dcdOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	// T-0207：notionalCcy=产品锁定币（md_dcd_product.raw 回填勿猜——
	// quote 真径三参；缺件 fail-closed 不臆造）。
	var notionalCcy string
	if err := h.pool.QueryRow(ctx, `SELECT raw->>'notionalCcy' FROM md_dcd_product
		WHERE product_id=$1 ORDER BY ts DESC LIMIT 1`, spec.InstID).Scan(&notionalCcy); err != nil || notionalCcy == "" {
		return 0, fmt.Errorf("dcd notionalCcy 缺 fail-closed（产品 raw 无此字段）: %v", err)
	}
	// 陈旧提案闸：按 spec.InstID=productId 重取 quote——产品失效=弃权如实；
	// quoteId 报价绑单径（每开必新 quote=陈旧提案不盲执语义自然保）。
	quoteID, err := h.cli.DcdQuote(ctx, spec.InstID, dcdSz(spec.NotionalUSD), notionalCcy)
	if err != nil {
		return 0, fmt.Errorf("dcd quote 失效弃权（陈旧提案不盲执）: %w", err)
	}
	// 风控过闸同 carry 仓（notional/exposure 谓词现行；Legs 记 dcd 单腿形）。
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: []string{"dcd_call_buy"},
		NotionalUSD: spec.NotionalUSD, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	var execID int64
	if err := h.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,'open') RETURNING id`,
		spec.StrategyRef, dec.DecisionID, spec.InstID, spec.NotionalUSD).Scan(&execID); err != nil {
		return 0, err
	}
	// T-0207：venue trade 无 clOrdId 幂等键——fin_execution intent 行承幂等
	// 径照旧；歧态势=order-status?ordId/order-history 反查真态禁盲补。
	raw, err := h.cli.DcdTrade(ctx, quoteID)
	if err != nil {
		// T-0208②：429=未执行定态分流（venue 限速拒非歧态势——trade 未
		// 落即拒可退避重试；真歧态势=请求已入未定才 ambiguous）。
		if strings.Contains(err.Error(), "429") {
			_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='failed' WHERE id=$1", execID)
			return execID, fmt.Errorf("dcd trade 429 限速退避（exec_id=%d）: %w", execID, err)
		}
		// 传输/超时败=歧态势：禁补单——exec 行待人工/读件定真态（okx-exec 纪律）。
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='ambiguous' WHERE id=$1", execID)
		return execID, fmt.Errorf("dcd trade 歧态势禁补单（exec_id=%d）: %w", execID, err)
	}
	var rows []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='ambiguous' WHERE id=$1", execID)
		return execID, fmt.Errorf("dcd trade 响应解析败（歧态势禁补单）: %v", err)
	}
	if rows[0].SCode != "0" {
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='failed' WHERE id=$2",
			rows[0].SCode+":"+rows[0].SMsg, execID)
		return execID, fmt.Errorf("dcd 业务拒 %s: %s", rows[0].SCode, rows[0].SMsg)
	}
	// order-status 对拍定真态（T-0160 返修①：读败/空态→ambiguous 禁默认
	// LIVE 造仓——歧态势禁补单纪律同 clOrdId 律；exec 行待查证非吞）。
	oraw, serr := h.cli.DcdOrderStatus(ctx, rows[0].OrdID)
	var state string
	if serr == nil {
		var orows []struct {
			State string `json:"state"`
		}
		if json.Unmarshal(oraw, &orows) == nil && len(orows) > 0 {
			state = orows[0].State
		}
	}
	if serr != nil || state == "" {
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='ambiguous' WHERE id=$1", execID)
		return execID, fmt.Errorf("order-status 读败/空态（歧态势禁补单——exec_id=%d 待查证非默认 LIVE 造仓）: %w", execID, serr)
	}
	// T-0160 返修⑤：exec 状态更新+position 落位同事务（position 写败→
	// exec orphan 标非半成——歧态势=exec 行在仓行缺=查证面不吞）。
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return execID, err
	}
	var pid int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,$3,$4,now()) RETURNING id`, execID, spec.StrategyRef, spec.InstID, spec.NotionalUSD).Scan(&pid); err != nil {
		_ = tx.Rollback(ctx)
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='orphan' WHERE id=$1", execID)
		return execID, fmt.Errorf("position 落位败（exec orphan 标）: %w", err)
	}
	estate := dcdExecState(state)
	if estate == "" {
		_ = tx.Rollback(ctx)
		_, _ = h.pool.Exec(ctx, "UPDATE fin_execution SET state='ambiguous' WHERE id=$1", execID)
		return execID, fmt.Errorf("order-status 态 %q 未载映射表（禁默认 open 造仓——歧态势）", state)
	}
	if _, err := tx.Exec(ctx, "UPDATE fin_execution SET state=$1 WHERE id=$2", estate, execID); err != nil {
		_ = tx.Rollback(ctx)
		return execID, err
	}
	if err := tx.Commit(ctx); err != nil {
		return execID, err
	}
	log.Printf("[lifecycle] DCD 开仓 exec=%d pos=%d ordId=%s state=%s", execID, pid, rows[0].OrdID, state)
	return execID, nil
}

// dcdExecState order-status 态→fin_execution 态映射（T-0160 返修⑤桩锚件——
// 未载态=空串让位（禁默认 open）；歧态势由调用径置 ambiguous）。
func dcdExecState(vstate string) string {
	return map[string]string{"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"}[vstate]
}

// dcdSz 数量格式化（venue sz 方言拒 sci——FormatFloat f,-1 去指数形；
// T-0160 返修②实证锚）。
func dcdSz(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
