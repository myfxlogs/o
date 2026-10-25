// Package guard —— 看护平仓（G8 持仓哨戒）：扫 fin_position 在仓行，逐仓对实时
// 事实行（md_funding/md_ticks/fin_strategy.state）判触发——funding 翻面/delta 漂移/
// 持仓超龄/策略降级联动；触发即生成 kind=close 平仓 intent 过 G6 闸（close 径
// kill_switch/上限不拦）走 G7 执行缝双腿反向原子平。判据全 guard.*/exec.* 热启键
// 机读，缺键 fail-closed 跳过+计数——看护判定永不走 LLM（T-0015 范围外红线）。
package guard

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"o/internal/platform/events"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/risk"
)

// CloseSpec 平仓执行件规格（双腿反向原子平——guard 反转原仓腿侧，执行件只管
// 成开成/开不成全收回的 G7 语义）。
type CloseSpec struct {
	DecisionID  int64 // G6 闸出件——执行件凭此自验+消费令牌
	StrategyRef int64
	InstID      string
	NotionalUSD float64
	Legs        [2]CloseLeg
}

// CloseLeg 反向腿规格（side=平仓侧；sz 沿原仓腿量——venue 方言换算归执行缝）。
type CloseLeg struct {
	Venue, InstID, Side string
	Sz                  float64
}

// Executor 平仓执行缝——G7 执行件（internal/trading/execution.Runner）由装配层
// 适配接入；未接线 stub 拒执（fail-closed 不假成交不留半成品）。
type Executor interface {
	ExecuteClose(ctx context.Context, spec CloseSpec) (execID int64, err error)
}

type missingExecutor struct{}

func (missingExecutor) ExecuteClose(context.Context, CloseSpec) (int64, error) {
	return 0, errors.New("平仓执行缝未接线——拒执 fail-closed")
}

// Warden 员工「仓巡」——常驻哨戒循环。
type Warden struct {
	pool   *pgxpool.Pool
	cfg    *config.Client
	gate   *risk.Gate
	ex     Executor
	sizer  VenueSizer
	ctVal  func(ctx context.Context, venue, instID string) (float64, bool) // 合约面值现读件（T-0046①——T-0048 后 facts 表废，cmd 注入 venue 现读）
	rej    *rejects.Door
	al     *notify.Alerter
	notes  map[string]time.Time                       // skip 态限频日志时间戳（键级 10min 节流）
	equity func(ctx context.Context) (float64, error) // 净权益现读件（dd 硬线判定时点实读）
}

// AttachSizer 挂平仓量 venue 现读面（cmd 装配注入——未挂=平仓 fail-closed）。
func (w *Warden) AttachSizer(s VenueSizer) { w.sizer = s }

// AttachCtVal 挂合约面值现读面（T-0046①——未挂=SWAP 腿名义读不到 fail-closed）。
func (w *Warden) AttachCtVal(f func(ctx context.Context, venue, instID string) (float64, bool)) {
	w.ctVal = f
}

// AttachAlerter 挂告警器（nil=静默降级不炸）。
func (w *Warden) AttachAlerter(a *notify.Alerter) { w.al = a }

// AttachEquity 挂净权益现读件（装配层接线——dd 硬线判定时点 balances 实读）。
func (w *Warden) AttachEquity(fn func(ctx context.Context) (float64, error)) { w.equity = fn }

func NewWarden(pool *pgxpool.Pool, cfg *config.Client, gate *risk.Gate, ex Executor) *Warden {
	if ex == nil {
		ex = missingExecutor{}
	}
	return &Warden{pool: pool, cfg: cfg, gate: gate, ex: ex,
		rej: rejects.New("guard", "missing_param", "facts_missing", "leg_anomaly",
			"close_denied", "close_failed", "close_gave_up", "write_err",
			"dd_missing_param", "equity_missing", "equity_stale", "dd_no_base", "dd_hardline_tripped", "dd_close_exhausted")}
}

// Run 在岗登记+心跳+哨戒循环（产出窗=fin_guard_close 行/fin_position.closed_at 迁移
// +回撤硬线动作链——T-0025 段一并入本岗）。
func (w *Warden) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, w.pool, "仓巡", "risk", "看护平仓哨——三触发+策略联动+回撤硬线自动退", nil); err != nil {
		errc <- fmt.Errorf("仓巡 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, w.pool, "仓巡", 30*time.Second, errc)
	// 事件化（T-0048：周期环拆——tick/funding/仓行变真事件驱动 evaluate）。
	events.ListenLoop(ctx, w.pool, "guard",
		[]string{"ev_ticks", "ev_funding", "ev_position"}, w.scanOnce)
}

type posRow struct {
	id, execID, stratRef int64
	inst                 string
	notional             float64
	opened               time.Time
	attempts             int64
	stratState           string
}

// scanOnce 一轮哨戒：回撤硬线先判（账本级——触线即落闸全平，优先于单仓触发），
// 再扫在仓行（close_failed 按 guard.retry_* 节拍重试、超上限不再扫=升级留痕不沉默），
// 逐仓判触发→close intent 过闸→执行缝→状态迁移。
func (w *Warden) scanOnce(ctx context.Context) error {
	w.ddScan(ctx)
	backoff, maxAtt, err := w.retryKeys()
	if err != nil {
		return err
	}
	rows, err := w.pool.Query(ctx, `
		SELECT p.id, p.execution_id, p.strategy_ref, p.inst_id, p.notional_usd, p.opened_at,
		       COALESCE(g.attempts,0), COALESCE(s.state,'')
		FROM fin_position p
		LEFT JOIN fin_guard_close g ON g.position_id = p.id
		LEFT JOIN fin_strategy s ON s.id = p.strategy_ref
		WHERE p.closed_at IS NULL
		  AND COALESCE(g.state,'open') <> 'closed'
		  AND COALESCE(g.attempts,0) < $1
		  AND (g.last_try_at IS NULL OR now() - g.last_try_at >= make_interval(secs => $2))`,
		maxAtt, backoff)
	if err != nil {
		return fmt.Errorf("scan query: %w", err)
	}
	var batch []posRow
	for rows.Next() {
		var p posRow
		if err := rows.Scan(&p.id, &p.execID, &p.stratRef, &p.inst, &p.notional, &p.opened,
			&p.attempts, &p.stratState); err != nil {
			rows.Close()
			return fmt.Errorf("scan row: %w", err)
		}
		batch = append(batch, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("scan rows: %w", err)
	}
	for _, p := range batch {
		w.tend(ctx, p, maxAtt, w.evaluate(ctx, p, w.legs(ctx, p.execID)))
	}
	return nil
}

// tend 单仓看护：按既定触发（evaluate 判得或 dd_30d 强制）→标 closing（节拍账
// attempts+1）→闸→反向双腿执行→状态迁移。trig 空=无触发零动作。
func (w *Warden) tend(ctx context.Context, p posRow, maxAtt int64, trig string) {
	legs := w.legs(ctx, p.execID)
	if trig == "" {
		return
	}
	// 腿形态契约外（缺腿/>2 腿）无法反向原子平——升级留痕不盲发畸形 intent。
	if legs.one {
		w.rej.Inc("leg_anomaly")
		log.Printf("[guard] WARN 仓 %d 触发 %s 但腿形态异常——不可反向平，人工介入", p.id, trig)
		return
	}
	if err := w.markClosing(ctx, p.id, trig); err != nil {
		return
	}
	in := risk.Intent{
		StrategyRef: p.stratRef, Venue: legs.venue, InstID: p.inst,
		Legs: legs.labels(), NotionalUSD: p.notional, Kind: "close",
	}
	dec := w.gate.Check(ctx, in)
	if !dec.Allow {
		w.rej.Inc("close_denied")
		w.markFailed(ctx, p.id)
		log.Printf("[guard] WARN 仓 %d 平仓被闸拒（%s/%s）——升级留痕", p.id, trig, dec.Reason)
		return
	}
	cls, ok := w.sizedCloseLegs(ctx, legs.reverse())
	if !ok {
		w.rej.Inc("facts_missing")
		w.note("sizer_fail", fmt.Sprintf("仓 %d 平仓量 venue 现读败/对齐零——fail-closed 不发 intent", p.id))
		return
	}
	execID, err := w.ex.ExecuteClose(ctx, CloseSpec{
		DecisionID: dec.DecisionID, StrategyRef: p.stratRef, InstID: p.inst,
		NotionalUSD: p.notional, Legs: cls,
	})
	if err != nil {
		w.rej.Inc("close_failed")
		w.markFailed(ctx, p.id)
		if p.attempts+1 >= maxAtt {
			w.rej.Inc("close_gave_up")
			w.al.Alert(ctx, "close_gave_up", "平仓重试耗尽——人工介入")
			log.Printf("[guard] ERROR 仓 %d 平仓重试耗尽（%s，%d 次）——人工介入", p.id, trig, p.attempts+1)
			return
		}
		log.Printf("[guard] WARN 仓 %d 平仓败（%s：%v）——按节拍重试", p.id, trig, err)
		return
	}
	if err := w.markClosed(ctx, p.id, execID); err != nil {
		log.Printf("[guard] WARN 仓 %d 平仓成交但态迁写败：%v", p.id, err)
		return
	}
	log.Printf("[guard] 仓 %d 已平（触发=%s，decision=%d，close_exec=%d）", p.id, trig, dec.DecisionID, execID)
}

// evaluate 触发判序（单仓单触发，先到先平）：funding 翻面→delta 漂移→持仓超龄
// →策略降级/退役联动。判据键缺=fail-closed 跳过+计数；事实行缺=计数不误判。
func (w *Warden) evaluate(ctx context.Context, p posRow, legs legSet) string {
	// ① funding 翻面：最新 funding 与开仓方向异号且超 guard.flip_bps。
	if flipBps, err := w.getF("guard.flip_bps"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:flip", fmt.Sprintf("仓 %d guard.flip_bps 读败——判据链跳过: %v", p.id, err))
	} else if rate, ok := w.latestFunding(ctx, p.inst); !ok {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:funding", fmt.Sprintf("仓 %d 无 funding 事实行——翻面判据跳过", p.id))
	} else if sign, ok := perpSign(legs, p.inst); !ok {
		w.rej.Inc("leg_anomaly")
		w.note("leg_anomaly:perpSign", fmt.Sprintf("仓 %d perp 侧不可辨——翻面判据跳过", p.id))
	} else if rate*sign < 0 && math.Abs(rate)*10000 > flipBps {
		return "funding_flip"
	} else if nr, ok := w.nextFunding(ctx, p.inst); ok && nr*sign < 0 && math.Abs(nr)*10000 > flipBps {
		return "funding_flip_preempt" // T-0084：预测负值前置平——不等实收先交一刀
	}
	// ② delta 漂移（T-0064 语义正名）：漂移=对**开时基线**的偏移非绝对失衡——
	// |imb_now − imb_open| > band（出生粒度残差不杀，canon
	// lotsz-granularity-vs-hedge-band 裁决案）；双闸：imb_now>500bps 绝对
	// 兜底（防 sizer 失守放裸腿级失衡）。基线=NULL=首拍实算落列（0033 列）。
	if bandBps, err := w.getF("exec.hedge_band_bps"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:band", fmt.Sprintf("仓 %d exec.hedge_band_bps 读败——漂移判据链跳过: %v", p.id, err))
	} else if legs.one {
		w.rej.Inc("leg_anomaly")
		w.note("leg_anomaly:legs", fmt.Sprintf("仓 %d 腿形态异常——漂移判据跳过", p.id))
	} else if n1, ok1 := w.notionalNow(ctx, legs.a); !ok1 {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:notional", fmt.Sprintf("仓 %d 腿 %s 名义读败（tick/ctVal 缺）——漂移判据跳过", p.id, legs.a.inst))
	} else if n2, ok2 := w.notionalNow(ctx, legs.b); !ok2 {
		w.rej.Inc("facts_missing")
		w.note("facts_missing:notional", fmt.Sprintf("仓 %d 腿 %s 名义读败（tick/ctVal 缺）——漂移判据跳过", p.id, legs.b.inst))
	} else if avg := (n1 + n2) / 2; avg > 0 && w.driftTriggered(ctx, p.id, (n1-n2)/avg*10000, bandBps) {
		return "delta_drift"
	}
	// ③ 持仓超龄：age > guard.max_hold_hours（T-0111：收益失效判——live 且
	// trailing 24h 实收 funding>0 的仓免超龄闸；无收入仓保留年龄闸——僵尸
	// 杀手机制不丢，盈利 carry 仓不再制度性误杀）。
	if maxH, err := w.getF("guard.max_hold_hours"); err != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:maxhold", fmt.Sprintf("仓 %d guard.max_hold_hours 读败——超龄判据跳过: %v", p.id, err))
	} else if time.Since(p.opened) > time.Duration(maxH*float64(time.Hour)) {
		if w.trailingIncomePositive(ctx, p.id) {
			w.note("max_hold_exempt", fmt.Sprintf("仓 %d 超龄但 trailing 24h 实收>0——收益失效判豁免", p.id))
		} else {
			return "max_hold"
		}
	}
	// ④ 策略降级/退役联动（状态机事实直读，无键面——degraded/retired 必平）。
	switch p.stratState {
	case "degraded":
		return "strategy_degraded"
	case "retired":
		return "strategy_retired"
	}
	w.note("no_trigger", fmt.Sprintf("仓 %d 全判据无触发——无触发也是决策（可观测）", p.id))
	return ""
}

// trailingIncomePositive 收益失效判实读（T-0111）：fin_cashflow 归仓行
// kind=funding 近 24h 实收合计>0=仓仍在赚——免超龄处决。读败/无行=false
// 保守侧（证据缺不豁免——fail-closed 回年龄判据）。
func (w *Warden) trailingIncomePositive(ctx context.Context, posID int64) bool {
	var sum float64
	if err := w.pool.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0) FROM fin_cashflow
		WHERE position_ref=$1 AND kind='funding_settle' AND occurred_at>now()-interval '24 hours'`,
		posID).Scan(&sum); err != nil {
		log.Printf("WARN [guard] 仓 %d trailing 实收读败（豁免判据回退年龄闸）: %v", posID, err)
		return false
	}
	return sum > 0
}
