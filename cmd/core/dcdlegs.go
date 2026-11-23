// dcdlegs.go —— dcd 簿记机械环写件（O-33362——凭据+counted 现金腿同事务
// 配对；自 dcdrecon.go 拆出守 300 行棘轮）。全幂等：ordId 凭据在册即
// 整组跳过（调用侧前置检查+fin_0233 唯一索引双闸），腿/凭据同生同灭。
package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/trading/okxexec"
)

// dcdCashCcy counted 腿币种闸（O-34287：原生币域开闸——BTC 回款/申购
// 按本币记账，凭据配对语义不变；形闸=ledger.CcyOK 同判）。
func dcdCashCcy(ccy string) bool { return ledger.CcyOK("okx", ccy) }

// dcdOccurTS venue ms 串→occurred_at（订阅锚 cTime/实结锚 settledTime
// 优先、settleTime 兜底；缺位=now()—— occurred_at NOT NULL 闭集）。
func dcdOccurTS(ms ...string) time.Time {
	for _, s := range ms {
		if v := okxexec.ParseF(s); v > 0 {
			return time.UnixMilli(int64(v)).UTC()
		}
	}
	return time.Now().UTC()
}

// dcdCashPost counted 现金腿入账（source_ref=配对凭据行——venue 订单即
// 归因源，凭据=簿记环书面凭据；禁推断记账律下来源行实存）。
func dcdCashPost(ctx context.Context, tx pgx.Tx, kind, ccy string, amt float64,
	credID int64, occurred time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES($1,'okx',$2,$3,$4,'fin_risk_decision',$5)`,
		kind, ccy, amt, credID, occurred)
	return err
}

// dcdBaselineWrite 基线凭据+申购 counted 腿同事务（T-0101 凭据配对模型
// 机械化——形同 id=165/35358 业主手工凭据：decision=allow/reason=transfer/
// kind=open；腿 kind=dcd_out 负额=自有资金入结构产品内部移动）。
func dcdBaselineWrite(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	intent, _ := json.Marshal(map[string]any{
		"op": "dcd_baseline", "kind": "transfer", "venue": "okx",
		"ccy": od.NotionalCcy, "amt": okxexec.ParseF(od.NotionalSz),
		"ordId": od.OrdID, "productId": od.ProductID})
	var credID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES($1,'allow','transfer',1,'open') RETURNING id`, intent).Scan(&credID); err != nil {
		return err
	}
	if amt := okxexec.ParseF(od.NotionalSz); dcdCashCcy(od.NotionalCcy) && amt != 0 {
		if err := dcdCashPost(ctx, tx, "dcd_out", od.NotionalCcy, -amt, credID,
			dcdOccurTS(od.CTime)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// dcdSettleWrite 结算凭据+counted 回款腿同事务（形同 id=35308 业主凭据
// +#33341 手工腿机械化）。腿拆分：yieldCcy=settleCcy∧0<yieldSz<settleSz
// →本金腿 dcd_in(+) +息腿 interest(+)（息=income 真赚域）；不可分拆
// →全额 dcd_in 实收（本金+息一体合法——现金域实收为准）。
// settleCcz∉本位闭集=无 counted 腿（杂币回款结构性不入账，凭据照落）。
func dcdSettleWrite(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	intent, _ := json.Marshal(map[string]any{
		"op": "dcd_settle", "kind": "transfer", "venue": "okx",
		"ccy": od.SettleCcy, "amt": okxexec.ParseF(od.SettleSz),
		"ordId": od.OrdID, "productId": od.ProductID,
		"settlePx": od.SettlePx, "settleTime": od.SettleTime})
	var credID int64
	if err := tx.QueryRow(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
		VALUES($1,'allow','transfer',1,'open') RETURNING id`, intent).Scan(&credID); err != nil {
		return err
	}
	settleSz := okxexec.ParseF(od.SettleSz)
	if !dcdCashCcy(od.SettleCcy) || settleSz == 0 {
		return tx.Commit(ctx)
	}
	occurred := dcdOccurTS(od.SettledTime, od.SettleTime)
	yieldSz := okxexec.ParseF(od.YieldSz)
	if od.YieldCcy == od.SettleCcy && yieldSz > 0 && yieldSz < settleSz {
		if err := dcdCashPost(ctx, tx, "dcd_in", od.SettleCcy, settleSz-yieldSz,
			credID, occurred); err != nil {
			return err
		}
		if err := dcdCashPost(ctx, tx, "interest", od.SettleCcy, yieldSz,
			credID, occurred); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if err := dcdCashPost(ctx, tx, "dcd_in", od.SettleCcy, settleSz,
		credID, occurred); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
