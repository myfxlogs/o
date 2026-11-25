// balanceops.go —— O-35753 充提入账采集细胞（汇市巡同会话面细胞：同拍距
// 同数据面零新器官——docs/resident_units.txt 无需登记）。
// 链：OrderHistory 轮询 → 充提行（DealType_Balance/OrderType_Balance 双标
// 记实证互证）→ fin_cashflow deposit/withdraw（venue='mt5'/ccy='USD'/
// source_table='mt5_deal'/source_ref=deal 票号/occurred_at=deal 时刻）。
// 效果链：业主入金→venue 余额/deal 行→本细胞落 deposit→账本净额追平
// venue→ledger_diff_mt5_USD 归零→无误报；真差异仍告警（本件不动告警层）。
// 幂等双闸（判词行载策略——工单#35753 簿记先例 #35771「采臂落地后须
// dedup 兼容防双记」承接）：
//
//	①精确重放闸——(venue,source_table='mt5_deal',source_ref=票号) 部分
//	  唯一索引（fin_0241）硬约束+WHERE NOT EXISTS 预检双保险；
//	②人工先例冲突闸——同 venue 非 mt5_deal 行（owner_out#34239/deposit
//	  #35771 先例域）、同号、金额容差 max($0.25,0.2%)、occurred_at ±72h
//	  窗内=视为已人工簿记→跳过不双记（manual_dup 日志归因可追）。
//
// 其余 balance 邻类（Credit 单/Bonus/Charge/Correction/Commission 族/
// InterestRate——mt5 proto DealType 实录枚举）不静默丢：ops_event 判词行
// 注记处置（marker mt5deal:<票号> 幂等——账本不猜语义，真差异留告警径）。
package mt5

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/protobuf/proto"
	mt5pb "o/internal/gen/mtapi/mt5"
)

const (
	// balanceBootLookback 首跑回看窗——O-35753 spec 钦定 7 天补漏窗
	// （部署时点追平已漏区间；重启再跑同窗幂等无害）。
	balanceBootLookback = 7 * 24 * time.Hour
	// balancePollEvery 增量拍距——充提前可分钟级，15min 对账延迟可受。
	balancePollEvery = 15 * time.Minute
	// balanceWindowOverlap 增量窗重叠——水位=上拍上界，重叠 2h 防
	// 时钟差/断档漏单（from/to 秒精度截断+出站序抖动皆被重叠吃掉）。
	balanceWindowOverlap = 2 * time.Hour
	// manualMatchWindowSec 人工簿记 occurred_at 容差窗（秒）——实证
	// owner_out#34239 先例：deal 11-21T05:18 vs 人工 occurred_at
	// 11-22T14:27 Δ=33h；deposit#35771 先例 Δ=0.6h——72h 双先例全覆。
	manualMatchWindowSec = 72 * 3600
	// manualAmtAbsTol/manualAmtRelTol 人工簿记金额容差——实证先例
	// deal -3247.31 vs 人工 -3247.41（差 $0.10=0.003%）：绝对下限
	// $0.25 收小额舍入，0.2% 相对项收大额尾差。
	manualAmtAbsTol = 0.25
	manualAmtRelTol = 0.002
)

// balanceOpsLoop 充提轮询环——sessCtx 绑定会话命（流死随会话收，重连
// 由外层 session() 重生；水位 balWM 存 acctRun 跨会话续命不回看重跑）。
// 首拍即跑（boot 7d 回看补漏窗），此后 15min 增量。a=按户运行件
// （O-37448——venue=户 token，水位按户隔离）。
func (c *Collector) balanceOpsLoop(ctx context.Context, cl *Client, a *acctRun) {
	for {
		if err := c.balancePollOnce(ctx, cl, a); err != nil && ctx.Err() == nil {
			c.rej.Inc("balance_ops_err")
			log.Printf("[mt5] %s 充提轮询败（下拍重试）: %v", a.acct.Venue, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(balancePollEvery): // sched-exempt-ok 会话内增量拍（非独立轮询器——同会话细胞随流生死）
		}
	}
}

// balanceWindow 轮询窗计算（纯函数——桩测钉边界）：水位 0=首跑 7d 回看；
// 水位>0=wm-2h 重叠增量窗。to 恒=now。
func balanceWindow(wm, now time.Time) (from, to time.Time) {
	to = now
	if wm.IsZero() {
		return now.Add(-balanceBootLookback), to
	}
	return wm.Add(-balanceWindowOverlap), to
}

// balancePollOnce 单拍：OrderHistory 拉窗→逐行分类入账→水位推到本拍
// 上界（拉取败不推水位=下拍同窗重试不漏）。
func (c *Collector) balancePollOnce(ctx context.Context, cl *Client, a *acctRun) error {
	now := time.Now()
	var wm time.Time
	if v := a.balWM.Load(); v > 0 {
		wm = time.Unix(0, v)
	}
	from, to := balanceWindow(wm, now)
	rep, err := cl.MT5().OrderHistory(cl.Authed(ctx), &mt5pb.OrderHistoryRequest{
		Id:        cl.Token(),
		From:      from.UTC().Format("2006-01-02T15:04:05"),
		To:        to.UTC().Format("2006-01-02T15:04:05"),
		Ascending: proto.Bool(true),
	})
	if err != nil {
		return err
	}
	if rep.GetError() != nil {
		return fmt.Errorf("OrderHistory venue err: %v", rep.GetError())
	}
	for _, o := range rep.GetResult() {
		c.balanceDeal(ctx, a, o)
	}
	// 水位=上拍上界（deal 序不保证——窗连续由 from=wm-overlap 承接；
	// 时钟回拨不推水位防窗倒退）。
	for {
		old := a.balWM.Load()
		if now.UnixNano() <= old || a.balWM.CompareAndSwap(old, now.UnixNano()) {
			break
		}
	}
	return nil
}

// balanceDeal 单行分类处置：充提→fin_cashflow venue=户 token；balance 邻类→
// 判词注记；交易行（Buy/Sell 族）=非充提域静默略（其盈亏归 trade_pnl 他件管）。
func (c *Collector) balanceDeal(ctx context.Context, a *acctRun, o *mt5pb.Order) {
	dealBal := o.GetDealType() == mt5pb.DealType_DealType_Balance
	ordBal := o.GetOrderType() == mt5pb.OrderType_OrderType_Balance
	amt := o.GetProfit()
	if amt == 0 && o.GetDealInternalIn() != nil {
		amt = o.GetDealInternalIn().GetProfit() // 兜底——行顶 Profit 缺时内嵌 deal 实载
	}
	switch {
	case dealBal || (ordBal && !balanceLikeDeal(o.GetDealType())):
		if amt == 0 {
			log.Printf("[mt5] %s 充提行零额跳票=%d deal=%s comment=%q", a.acct.Venue, o.GetTicket(), o.GetDealType(), o.GetComment())
			return
		}
		c.bookBalance(ctx, a, o, amt)
	case o.GetOrderType() == mt5pb.OrderType_OrderType_Credit || ordBal || balanceLikeDeal(o.GetDealType()):
		c.noteBalanceLike(ctx, a, o)
	}
}

// balanceLikeDeal balance 邻类 DealType 集（proto 实录枚举：Credit/Charge/
// Correction/Bonus/Commission 族/InterestRate）——判词注记域非入账域。
func balanceLikeDeal(dt mt5pb.DealType) bool {
	switch dt {
	case mt5pb.DealType_DealType_Credit, mt5pb.DealType_DealType_Charge,
		mt5pb.DealType_DealType_Correction, mt5pb.DealType_DealType_Bonus,
		mt5pb.DealType_DealType_Commission, mt5pb.DealType_DealType_DailyCommission,
		mt5pb.DealType_DealType_MonthlyCommission,
		mt5pb.DealType_DealType_DailyAgentCommission,
		mt5pb.DealType_DealType_MonthlyAgentCommission,
		mt5pb.DealType_DealType_InterestRate:
		return true
	}
	return false
}

// dealTime deal 时刻——CloseTime Timestamp 优先（实证与 CloseTimestampUTC
// 毫秒互洽），回落 int64 毫秒（KB 实证单位），再回落 Open*；全缺=不可
// 定 occurred_at（NOT NULL 列——宁可跳不猜）。
func dealTime(o *mt5pb.Order) (time.Time, bool) {
	if ts := o.GetCloseTime(); ts != nil {
		if t := ts.AsTime(); !t.IsZero() {
			return t.UTC(), true
		}
	}
	if ms := o.GetCloseTimestampUTC(); ms > 0 {
		return time.UnixMilli(ms).UTC(), true
	}
	if ms := o.GetOpenTimestampUTC(); ms > 0 {
		return time.UnixMilli(ms).UTC(), true
	}
	return time.Time{}, false
}

// bookBalance 充提入账——kind 按符号定（正=deposit 负=withdraw，与
// OKX funding 径同约定：withdraw 存负值）。venue=户 token（O-37448——
// (venue,source_table,source_ref) 唯一索引天然按户分键不互踩）。双闸幂等见包注。
func (c *Collector) bookBalance(ctx context.Context, a *acctRun, o *mt5pb.Order, amt float64) {
	occurred, ok := dealTime(o)
	if !ok {
		log.Printf("[mt5] %s 充提行无时刻跳票=%d amt=%.4f", a.acct.Venue, o.GetTicket(), amt)
		return
	}
	kind := "withdraw"
	if amt > 0 {
		kind = "deposit"
	}
	var id int64
	// O-39440：account_id=venue:login 显式归户（跨户 ticket 命名空间独立
	// ——Exness/XM 同号不互斥；login 缺席=NULLIF→触发器兜底映射）。
	acctID := ""
	if a.acct.Login != "" {
		acctID = a.acct.Venue + ":" + a.acct.Login
	}
	err := c.pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at,account_id)
		SELECT $1,$8,'USD',$2,$3,'mt5_deal',$4,NULLIF($9,'')
		WHERE NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=COALESCE(NULLIF($9,''),'mt5:3001157') AND f.source_table='mt5_deal' AND f.source_ref=$3)
		AND NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=COALESCE(NULLIF($9,''),'mt5:3001157') AND f.source_table<>'mt5_deal'
			  AND f.amount*$2 > 0
			  AND abs(abs(f.amount)-abs($2)) <= greatest($5, abs($2)*$6)
			  AND abs(extract(epoch from (f.occurred_at - $4))) <= $7)
		RETURNING id`,
		kind, amt, o.GetTicket(), occurred,
		manualAmtAbsTol, manualAmtRelTol, manualMatchWindowSec, a.acct.Venue, acctID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.classifySkip(ctx, a, o, amt) // 幂等跳过非写败——不计拒因
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		c.classifySkip(ctx, a, o, amt) // 唯一索引兜底命中——并发/重放竞态按重放收
		return
	}
	if err != nil {
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 充提入账写败 票=%d amt=%.4f: %v", a.acct.Venue, o.GetTicket(), amt, err)
		return
	}
	log.Printf("[mt5] %s 充提入账 id=%d kind=%s amt=%.4f 票=%d occurred=%s comment=%q",
		a.acct.Venue, id, kind, amt, o.GetTicket(), occurred.Format(time.RFC3339), o.GetComment())
}

// classifySkip 幂等跳过归因——精确重放（mt5_deal 行已在）静默 vs 人工
// 先例冲突（非 mt5_deal 行命中容差窗）manual_dup 日志，判词可追。
func (c *Collector) classifySkip(ctx context.Context, a *acctRun, o *mt5pb.Order, amt float64) {
	var exact bool
	acctID := ""
	if a.acct.Login != "" {
		acctID = a.acct.Venue + ":" + a.acct.Login
	}
	_ = c.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fin_cashflow WHERE account_id=COALESCE(NULLIF($1,''),'mt5:3001157')
		 AND source_table='mt5_deal' AND source_ref=$2)`, acctID, o.GetTicket()).Scan(&exact)
	if exact {
		return // 精确重放——静默幂等（高频窗重叠常态）
	}
	log.Printf("[mt5] %s 充提行人工簿记在先跳票=%d amt=%.4f deal=%s comment=%q（manual_dup——不双记）",
		a.acct.Venue, o.GetTicket(), amt, o.GetDealType(), o.GetComment())
}

// noteBalanceLike balance 邻类判词注记——marker=<venue>deal:<票号> 幂等
// （'mt5'→'mt5deal:' 存量形不变；他户票号域独立不互踩——同票号跨户
// 不撞 marker）。轮询重叠窗重扫不重复注记。
func (c *Collector) noteBalanceLike(ctx context.Context, a *acctRun, o *mt5pb.Order) {
	marker := fmt.Sprintf("%sdeal:%d", a.acct.Venue, o.GetTicket())
	note := fmt.Sprintf("[mt5] %s balance邻类待处置 %s order=%s deal=%s amt=%.4f comment=%q occurred_ts=%d",
		a.acct.Venue, marker, o.GetOrderType(), o.GetDealType(), o.GetProfit(), o.GetComment(), o.GetCloseTimestampUTC())
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		 SELECT '汇市巡','簿记','',$1
		 WHERE NOT EXISTS (SELECT 1 FROM ops_event WHERE note LIKE '%'||$2||'%')`,
		note, marker); err != nil {
		c.rej.Inc("write_err")
		log.Printf("[mt5] balance邻类判词注记写败 票=%d: %v", o.GetTicket(), err)
	}
}
