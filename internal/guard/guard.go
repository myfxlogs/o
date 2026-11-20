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

// CloseLeg 反向腿规格（side=平仓侧；sz=下单方言目标量——SWAP 张数/spot 币量
// /mt5 手数，T-0419 量纲消歧后沿腿实得 qty_base 非开腿 sz 额；SzKnown=false=
// 腿账不可定→venue 余额实查备源；Ticket=T-0145 venue 仓凭据——mt5 平仓
// OrderClose 必携）。
type CloseLeg struct {
	Venue, InstID, Side string
	Sz                  float64
	SzKnown             bool
	Ticket              int64
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
	pool          *pgxpool.Pool
	cfg           *config.Client
	gate          *risk.Gate
	ex            Executor
	sizer         VenueSizer
	sizers        map[string]VenueSizer                                           // T-0145：venue 路由现读面（跨 venue 双腿——mt5 腿须 mt5 sizer；缺 venue 路由=fail-closed 不开）
	ctVal         func(ctx context.Context, venue, instID string) (float64, bool) // 合约面值现读件（T-0046①——T-0048 后 facts 表废，cmd 注入 venue 现读）
	rej           *rejects.Door
	al            *notify.Alerter
	notes         map[string]time.Time                       // skip 态限频日志时间戳（键级 10min 节流）
	emg           emergencyState                             // 末路只减仓出口态（工单#22247——exec缝/受令去重/节流）
	anomalyMarked map[int64]bool                             // 腿形态异常定性行已落集（工单#22845——进程内去重,重启幂等重写）
	equity        func(ctx context.Context) (float64, error) // 净权益现读件（dd 硬线判定时点实读）
	ddFacts       map[string]any                             // 本拍回撤实况载荷（dd/阈/peak/cur——T-0415 平仓决策行携实况）
	// 以下三件由 events.ListenLoop 单 goroutine 串行驱动（scanOnce）——不加锁同前提。
	now         func() time.Time // 时钟缝（nil=time.Now；取用走 w.clock()）
	bar         ddBar            // dd 硬线分钟柱在收柱（柱内样本攒内存）
	eqFailSince time.Time        // 权益连续读败起点（成功拍清零）
}

// AttachSizer 挂平仓量 venue 现读面（cmd 装配注入——未挂=平仓 fail-closed）。
func (w *Warden) AttachSizer(s VenueSizer) { w.sizer = s }

// AttachVenueSizer 挂 venue 路由现读面（T-0145——跨 venue 双腿按腿 venue 分派；
// 同 venue 重复挂=后覆前幂等）。w.sizer=okx 默认兜底（未挂路由的 venue）。
func (w *Warden) AttachVenueSizer(venue string, s VenueSizer) {
	if w.sizers == nil {
		w.sizers = map[string]VenueSizer{}
	}
	w.sizers[venue] = s
}

// sizerFor 腿 venue 路由现读件（T-0145）：venue 路由在册=专用件；okx/空=
// 默认 w.sizer 兜底（单 venue 仓不变行为）；外 venue 无专用件=nil
// fail-closed（错 venue 件=错读量——binance 腿抄 okx sizer=异 venue 误读）。
func (w *Warden) sizerFor(venue string) VenueSizer {
	if s, ok := w.sizers[venue]; ok && s != nil {
		return s
	}
	if venue == "okx" || venue == "" {
		return w.sizer
	}
	return nil
}

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
			"close_denied", "close_failed", "close_gave_up", "write_err", "loop_alert",
			"dd_missing_param", "equity_missing", "equity_stale", "dd_no_base", "dd_hardline_tripped", "dd_close_exhausted", "dd_bar_thin",
			"dir_missing_param", "dir_nav_missing", "dir_measure_err", "dir_leg_px_missing", "dir_mark_missing",
			"dir_freeze_err", "dir_loss_breaker_tripped", "dir_close_exhausted",
			"bare_bag"),
		emg:           emergencyState{seen: map[int64]bool{}, last: map[int64]time.Time{}},
		anomalyMarked: map[int64]bool{}}
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

// ScanOnce 一拍哨戒的外部驱动缝（T-0426 执行径演习——simdrill 跨包驱动同一
// 哨戒拍跑产线判定/平仓径；产线路径 Run/ListenLoop 不变）。
func (w *Warden) ScanOnce(ctx context.Context) error { return w.scanOnce(ctx) }

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
	w.dirScan(ctx)   // 方向族亏损预算熔断（业主令⑧——dd 线后族级闸）
	w.loopProbe(ctx) // T-0423 自残探针——closing 滞留/超次升件+熔断（阈键独立缺件不阻主扫描）
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
		WHERE p.state<>'closed'
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
	w.emergencyScan(ctx, maxAtt) // 工单#22247：末路只减仓出口（attempts 耗尽+受令）
	return nil
}

// tend 单仓看护：按既定触发（evaluate 判得或 dd_30d 强制）→标 closing（节拍账
// attempts+1）→闸→反向双腿执行→状态迁移。trig.kind 空=无触发零动作。
func (w *Warden) tend(ctx context.Context, p posRow, maxAtt int64, trig closeTrig) {
	legs := w.legs(ctx, p.execID)
	if trig.kind == "" {
		return
	}
	// 腿形态契约外（缺腿/>2 腿）无法反向平——升级留痕不盲发畸形 intent。
	// 单腿（方向族契约）走同径：labels 一签/reverse 一空槽/execwiring 派
	// ExecuteDir——反向语义在腿级（side 取反）非腿集级（棒5c）。
	if legs.one {
		w.rej.Inc("leg_anomaly")
		// 工单#22845：腿形态异常仓（零腿=dcd 单产品仓等）反向平径恒不可达——
		// 原 WARN 每拍直刷=噪音洪泛；定性行落 fin_guard_close 一次（进程内
		// 去重+upsert 幂等），日志走 note 10min 节流，态不沉默不误报。
		w.markLegAnomaly(ctx, p.id, trig.kind)
		w.note(fmt.Sprintf("leg_anomaly:%d", p.id),
			fmt.Sprintf("仓 %d 触发 %s 腿形态异常——不可反向平（定性行已落 fin_guard_close；10min 节流）", p.id, trig.kind))
		return
	}
	if err := w.markClosing(ctx, p.id, trig.kind); err != nil {
		return
	}
	in := risk.Intent{
		StrategyRef: p.stratRef, Venue: legs.venue, InstID: p.inst,
		Legs: legs.labels(), NotionalUSD: p.notional, Kind: "close",
	}
	dec := w.gate.Check(ctx, in)
	if !dec.Allow {
		w.rej.Inc("close_denied")
		w.markFailed(ctx, p.id, "plan:gate:"+dec.Reason) // T-0428 败因落表——闸拒=计划期败
		log.Printf("[guard] WARN 仓 %d 平仓被闸拒（%s/%s）——升级留痕", p.id, trig.kind, dec.Reason)
		return
	}
	cls, settled, ok := w.sizedCloseLegs(ctx, legs.reverse())
	if !ok {
		w.rej.Inc("facts_missing")
		w.noteAttemptFail(ctx, p.id, "plan:sizer:平仓量现读败/对齐零") // T-0428——Earn 收走余额类计划期败
		w.note("sizer_fail", fmt.Sprintf("仓 %d 平仓量 venue 现读败/对齐零——fail-closed 不发 intent", p.id))
		return
	}
	// 工单#21745：venue 实读对齐零腿=外部已结（residual_clear 外扫/外部
	// 移动等外因清场）——不构平腿不发残量单；幸存腿前置归并（execwiring
	// Legs[1]==""→ExecuteDir 单腿径契约位），残量走挂账显形。
	if cls[0].InstID == "" && cls[1].InstID != "" {
		cls[0], cls[1] = cls[1], CloseLeg{}
	}
	if len(settled) > 0 {
		names := make([]string, 0, len(settled))
		for _, s := range settled {
			w.noteSettledLeg(ctx, p, s)
			names = append(names, s.instID)
		}
		if trig.facts == nil {
			trig.facts = map[string]any{}
		}
		trig.facts["externally_settled"] = names
	}
	if cls[0].InstID == "" {
		// 全腿外结=venue 已平——无执行件可发，直迁 closed（决策行
		// close_exec_id=0 显式记零执行=外结事实非成交）。
		if err := w.markClosed(ctx, p, 0, trig); err != nil {
			log.Printf("[guard] WARN 仓 %d 全腿外结但态迁写败：%v", p.id, err)
			return
		}
		log.Printf("[guard] 仓 %d 全腿外部已结（触发=%s）——直迁 closed", p.id, trig.kind)
		return
	}
	execID, err := w.ex.ExecuteClose(ctx, CloseSpec{
		DecisionID: dec.DecisionID, StrategyRef: p.stratRef, InstID: p.inst,
		NotionalUSD: p.notional, Legs: cls,
	})
	if err != nil {
		w.rej.Inc("close_failed")
		w.markFailed(ctx, p.id, "venue:"+err.Error()) // T-0428——venue 期败返回码落表
		if p.attempts+1 >= maxAtt {
			w.rej.Inc("close_gave_up")
			w.al.Alert(ctx, "close_gave_up", "平仓重试耗尽——人工介入")
			log.Printf("[guard] ERROR 仓 %d 平仓重试耗尽（%s，%d 次）——人工介入", p.id, trig.kind, p.attempts+1)
			return
		}
		log.Printf("[guard] WARN 仓 %d 平仓败（%s：%v）——按节拍重试", p.id, trig.kind, err)
		return
	}
	// T-0419 ②③：残袋显形对拍——markClosed 前落（settle 见挂账列=实现额非全损）。
	w.bareBags(ctx, p, legs, cls, execID)
	if err := w.markClosed(ctx, p, execID, trig); err != nil {
		log.Printf("[guard] WARN 仓 %d 平仓成交但态迁写败：%v", p.id, err)
		return
	}
	log.Printf("[guard] 仓 %d 已平（触发=%s，decision=%d，close_exec=%d）", p.id, trig.kind, dec.DecisionID, execID)
}
