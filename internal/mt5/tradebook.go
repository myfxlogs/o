// tradebook.go —— O-33207 手动资金流/手动仓采集（OrderHistory 交易行+
// OpenedOrders 在仓快照两径，同 balanceOpsLoop 逐户会话细胞——同拍距
// 同数据面零新器官）。链：
//
//	OrderHistory 交易行（DealBuy/DealSell 平仓仓）→ fin_cashflow
//	  kind='trade_pnl'（realized=Profit+Swap+Commission+Fee 行级聚合
//	  =venue balance 增量全分量——O 单/手动单同径守恒优先，O 单另经
//	  comment→leg→position 归 position_ref，手动仓 NULL 聚合层豁免）
//	  + fin_venue_position 平仓事实（非 O 单——手动仓簿记面）。
//	OpenedOrders 在仓快照 → fin_venue_position state='open' upsert
//	  （非 O 单）+快照消失→'stale' 清出 mtm——在仓浮盈并入对账净值
//	  （equity 含浮盈 net 不含=幻影残差根修）。
//
// 双簿防线（spec 禁线）：O 自执行单 comment=clOrdID——fin_order_leg/
// fin_risk_decision 双面命中=O 单→不入 fin_venue_position（其仓走
// fin_position+mtm 既有径）；cashflow 守恒不分彼此全量入账。
// 幂等：mt5_deal_uk(account_id,source_table,source_ref) 硬约束+
// manual_dup 容差闸（balanceops 同构——人工先记跳过不双记）。
package mt5

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	mt5pb "o/internal/gen/mtapi/mt5"
)

// bookUnmapped balance 邻类守恒兜底（O-33207——Charge/Correction/
// Commission*/InterestRate 金额带号入 unmapped_bill：venue balance
// 增量必入账否则 ledger_diff 幻影残差；归因待裁=unmapped 语义本色，
// 判词注记并存记 deal 型）。幂等双闸同 bookBalance。
func (c *Collector) bookUnmapped(ctx context.Context, a *acctRun, o *mt5pb.Order, amt float64) {
	occurred, ok := dealTime(o)
	if !ok {
		log.Printf("[mt5] %s 邻类行无时刻跳票=%d amt=%.4f", a.acct.Venue, o.GetTicket(), amt)
		return
	}
	acctID := c.acctIDFor(ctx, a)
	var id int64
	err := c.pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at,account_id)
		SELECT $1,$8,'USD',$2,$3,'mt5_deal',$4,NULLIF($9,'')
		WHERE NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=$9 AND f.source_table='mt5_deal' AND f.source_ref=$3)
		AND NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=$9 AND f.source_table<>'mt5_deal'
			  AND f.amount*$2 > 0
			  AND abs(abs(f.amount)-abs($2)) <= greatest($5, abs($2)*$6)
			  AND abs(extract(epoch from (f.occurred_at - $4))) <= $7)
		RETURNING id`,
		"unmapped_bill", amt, o.GetTicket(), occurred,
		manualAmtAbsTol, manualAmtRelTol, manualMatchWindowSec, a.acct.Venue, acctID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.classifySkip(ctx, a, o, amt)
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		c.classifySkip(ctx, a, o, amt)
		return
	}
	if err != nil {
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 邻类守恒入账写败 票=%d amt=%.4f: %v", a.acct.Venue, o.GetTicket(), amt, err)
		return
	}
	log.Printf("[mt5] %s 邻类守恒入账 id=%d amt=%.4f 票=%d deal=%s comment=%q",
		a.acct.Venue, id, amt, o.GetTicket(), o.GetDealType(), o.GetComment())
}

// creditLikeDeal spec 钦定入金族（O-33207——DealType_Credit/Bonus
// →deposit|withdraw 同 balance 径；Charge/Correction/Commission*/
// InterestRate 归 unmapped_bill 兜底域）。
func creditLikeDeal(dt mt5pb.DealType) bool {
	return dt == mt5pb.DealType_DealType_Credit || dt == mt5pb.DealType_DealType_Bonus
}

// isTradeDeal 平仓仓交易行判定（DealType 0/1=DealBuy/DealSell——实证
// EXR OrderHistory：每仓一行 posTicket 域，DealInternalIn/Out 逐 deal
// 载荷在案）。CanceledBuy/Sell=撤单族非仓——不入此径。
func isTradeDeal(dt mt5pb.DealType) bool {
	return dt == mt5pb.DealType_DealType_DealBuy || dt == mt5pb.DealType_DealType_DealSell
}

// isCancelDeal 撤单族（DealType 13/14）——零盈亏事实行，判词注记域。
func isCancelDeal(dt mt5pb.DealType) bool {
	return dt == mt5pb.DealType_DealType_CanceledBuy || dt == mt5pb.DealType_DealType_CanceledSell
}

// tradeRealized 行级 realized 聚合——venue balance 增量全分量
// （Profit 价差+Swap 隔夜+Commission 佣金+Fee 费；EXR 实证三分量
// 全零仅 Profit 非零，聚合式对 XM 佣金户同构兼容——分量符号=
// venue 报告符（DealInternal money digits 带号域），异号实证差
// 由 ledger_diff 对拍显形不静默）。
func tradeRealized(o *mt5pb.Order) float64 {
	return o.GetProfit() + o.GetSwap() + o.GetCommission() + o.GetFee()
}

// acctIDFor 账户域键解析——login 在=venue:login 显式（fin_account
// 行须在册 FK fail-loud=注册表契约）；login 缺=注册表 CASE 同
// 触发器规则近似解析（空串=交触发器兜底，dedup 检查同值域一致）。
func (c *Collector) acctIDFor(ctx context.Context, a *acctRun) string {
	if a.acct.Login != "" {
		return a.acct.Venue + ":" + a.acct.Login
	}
	var id string
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(
		(SELECT min(account_id) FROM fin_account WHERE venue=$1 AND sub_type='main' AND state='active' HAVING count(*)=1),
		(SELECT min(account_id) FROM fin_account WHERE venue=$1 AND sub_type='main' AND state='disabled' HAVING count(*)=1),
		$1||':legacy')`, a.acct.Venue).Scan(&id)
	return id
}

// ownOrder O 单判别——comment=clOrdID 幂等键（mtexec placeLeg 钉死
// Comment=clOrdID）：fin_order_leg.cl_ord_id ∨ fin_risk_decision
// intent.cl_ord_id 双面命中=O 自执行单。comment 空=手动/外部单.
func (c *Collector) ownOrder(ctx context.Context, comment string) bool {
	if comment == "" {
		return false
	}
	var own bool
	_ = c.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fin_order_leg WHERE cl_ord_id=$1)
		    OR EXISTS(SELECT 1 FROM fin_risk_decision WHERE intent->>'cl_ord_id'=$1)`,
		comment).Scan(&own)
	return own
}

// bookTrade 平仓仓行总处置：realized→trade_pnl 现金行（全量守恒）+
// 非 O 单→fin_venue_position 平仓事实。零额行跳（open-only 形不
// 入 OrderHistory——防御分支）。
func (c *Collector) bookTrade(ctx context.Context, a *acctRun, o *mt5pb.Order) {
	amt := tradeRealized(o)
	if amt == 0 {
		log.Printf("[mt5] %s 交易行零额跳票=%d sym=%s comment=%q", a.acct.Venue, o.GetTicket(), o.GetSymbol(), o.GetComment())
		return
	}
	occurred, ok := dealTime(o)
	if !ok {
		log.Printf("[mt5] %s 交易行无时刻跳票=%d amt=%.4f", a.acct.Venue, o.GetTicket(), amt)
		return
	}
	acctID := c.acctIDFor(ctx, a)
	if acctID == "" {
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 交易行账户解析空跳票=%d（fin_account 册缺——注册表契约）", a.acct.Venue, o.GetTicket())
		return
	}
	c.bookTradePnl(ctx, a, o, amt, occurred, acctID)
	if !c.ownOrder(ctx, o.GetComment()) {
		c.venuePosClose(ctx, a, o, amt, occurred, acctID)
	}
}

// bookTradePnl realized→fin_cashflow kind='trade_pnl'——幂等双闸同
// balance 径（mt5_deal_uk 硬约束+manual_dup 容差窗）。position_ref
// 仅 O 单归仓（comment→cl_ord_id→leg→position）；手动仓=NULL——
// 业绩豁免在聚合层（position_ref 缺席天然不入策略统计域）。
func (c *Collector) bookTradePnl(ctx context.Context, a *acctRun, o *mt5pb.Order,
	amt float64, occurred time.Time, acctID string) {
	var posRef *int64
	if cm := o.GetComment(); cm != "" && c.ownOrder(ctx, cm) {
		var pid int64
		if err := c.pool.QueryRow(ctx,
			`SELECT p.id FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
			 WHERE l.cl_ord_id=$1 ORDER BY p.id DESC LIMIT 1`, cm).Scan(&pid); err == nil {
			posRef = &pid
		}
	}
	var id int64
	err := c.pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at,account_id)
		SELECT 'trade_pnl',$5,'USD',$1,$2,$3,'mt5_deal',$4,NULLIF($6,'')
		WHERE NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=$6 AND f.source_table='mt5_deal' AND f.source_ref=$3)
		AND NOT EXISTS (
			SELECT 1 FROM fin_cashflow f
			WHERE f.account_id=$6 AND f.source_table<>'mt5_deal'
			  AND f.amount*$1 > 0
			  AND abs(abs(f.amount)-abs($1)) <= greatest($7, abs($1)*0.002)
			  AND abs(extract(epoch from (f.occurred_at - $4))) <= $8)
		RETURNING id`,
		amt, posRef, strconv.FormatInt(o.GetTicket(), 10), occurred,
		a.acct.Venue, acctID, manualAmtAbsTol, manualMatchWindowSec).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		c.classifySkip(ctx, a, o, amt) // 幂等跳过非写败——不计拒因
		return
	case err != nil:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.classifySkip(ctx, a, o, amt) // 唯一索引兜底命中——并发/重放竞态按重放收
			return
		}
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 交易盈亏入账写败 票=%d amt=%.4f: %v", a.acct.Venue, o.GetTicket(), amt, err)
		return
	}
	log.Printf("[mt5] %s 交易盈亏入账 id=%d amt=%.4f 票=%d sym=%s occurred=%s comment=%q",
		a.acct.Venue, id, amt, o.GetTicket(), o.GetSymbol(), occurred.Format(time.RFC3339), o.GetComment())
}
