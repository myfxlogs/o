package main

// fundsave.go —— funding/savings/DCD 现读聚（自 restlive.go 拆出——棘轮
// 300 行顶线；O-33206 资金户子桶枚举 avail/frozen+DCD productId 分量件）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/trading/okxexec"
)

// fundSaveBalances funding+savings 现读聚（T-0058：任一读败=err 上抛——静默跳腿禁）。
// T-0100 ①③：非本位币只收停泊溯源币（okx:funding/earn 域 FundCcy 成员——
// Earn 利息尘/腿基币渗漏=托管不入 venue 侧键）；usd=true=非 USDT 按 md_ticks
// 估值归一（权益求和径），false=原币量（对拍 ledger 侧按币量建键径）。
func fundSaveBalances(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client,
	out map[[2]string]float64, usd bool, parts ledger.VenueParts) error {
	set, err := ledger.FundCcy(ctx, pool)
	if err != nil {
		return err
	}
	emit := func(ccy string, amt float64, bucket string, detail ...ledger.BalPart) {
		if amt == 0 {
			return
		}
		// O-39440：账户域键=okx:<bucket>（funding/earn/dcd 三子户分账——
		// venue 桶聚合退役，对拍面账户粒度；usd 求和径键面无关照旧）。
		acct := "okx:" + bucket
		if parts != nil {
			k := [2]string{acct, ccy}
			if len(detail) > 0 {
				parts[k] = append(parts[k], detail...)
			} else {
				parts[k] = append(parts[k], ledger.BalPart{Name: bucket, Amount: amt})
			}
		}
		if !usd {
			// O-34287 验收②：对拍径=venue 真值全币发射（原生币各域——
			// 未登记币由基线凭据机制建账本域对拍，不静默跳过；TRX 托管币
			// 对账域照建、业绩豁免在聚合层 ccy 闸）。
			out[[2]string{acct, ccy}] += amt
			return
		}
		if ccy != "USDT" && ccy != "USD" &&
			!set[[2]string{acct, ccy}] {
			return // 非停泊托管币=域外不键（T-0100 ① venue 侧同源排除）
		}
		if ccy != "USDT" && ccy != "USD" {
			amt = valCcy(ctx, pool, ccy, amt)
		}
		out[[2]string{acct, ccy}] += amt
	}
	bals, err := okcli.AssetBalances(ctx)
	if err != nil {
		return fmt.Errorf("funding 读败: %w", err)
	}
	for _, b := range bals {
		// O-33206 资金户子桶枚举（业主令「分得要细一些」）：avail/frozen
		// 分量入 parts——bal=avail+frozen venue 口径,eq 总额不变;
		// TRX 挂单冻结等子位可见化=重锚枚举实据,禁猜锚依据。
		emit(b.Ccy, okxexec.ParseF(b.Bal), "funding",
			ledger.BalPart{Name: "funding_avail", Amount: okxexec.ParseF(b.AvailBal)},
			ledger.BalPart{Name: "funding_frozen", Amount: okxexec.ParseF(b.FrozenBal)})
	}
	sav, err := okcli.SavingsBalance(ctx)
	if err != nil {
		return fmt.Errorf("savings 读败: %w", err)
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if json.Unmarshal(sav, &rows) != nil {
		return fmt.Errorf("savings parse")
	}
	for _, r2 := range rows {
		emit(r2.Ccy, okxexec.ParseF(r2.Amt), "earn")
	}
	// T-0101：dcd 在持产品单无条件入键（venue LIVE 单=已部署资本铁证——
	// 不走 FundCcy 闸：未基线的新购仓 venue 侧照发，基线缺席由对拍面兜差）；
	// 读败=err 上抛不假零。
	// 工单#20904/#22919 权益缝根修（状态机逐段盖，零双计零永漏）：
	//   live            →名义照计（原径）；
	//   pending_settle/redeem →在途名义照计——离 live 集到回款入账之间
	//     整额不计曾产 NAV 假回撤（到期回撤硬线误触面）；
	//   settled         →结算回款币未注册 FundCcy(funding) 前按 settleSz
	//     计（BTC 回款=册外币永久漏计面——注册后由资金户余额面接管，
	//     本键自动退出零双计）。
	ords, err := okcli.DcdOrders(ctx)
	if err != nil {
		return fmt.Errorf("dcd 持仓读败: %w", err)
	}
	addOrd := func(od okxexec.DcdOrder) {
		ccy, amt := od.NotionalCcy, okxexec.ParseF(od.NotionalSz)
		if strings.EqualFold(od.State, "settled") || strings.EqualFold(od.State, "redeemed") {
			if od.SettleCcy == "" || set[[2]string{"okx:funding", od.SettleCcy}] {
				return // 回款币已注册→资金户余额面已计（双计禁）
			}
			ccy, amt = od.SettleCcy, okxexec.ParseF(od.SettleSz)
		}
		if amt == 0 {
			return
		}
		if parts != nil {
			k := [2]string{"okx:dcd", ccy}
			// O-33206 DCD 子桶枚举——分量名挂 productId（在持期权逐项
			// 可见=重锚枚举实据;productId 缺席回退裸 'dcd'）。
			name := "dcd"
			if od.ProductID != "" {
				name = "dcd:" + od.ProductID
			}
			parts[k] = append(parts[k], ledger.BalPart{Name: name, Amount: amt})
		}
		if usd && ccy != "USDT" && ccy != "USD" {
			amt = valCcy(ctx, pool, ccy, amt)
		}
		out[[2]string{"okx:dcd", ccy}] += amt
	}
	var live []okxexec.DcdOrder
	for _, od := range ords {
		switch {
		case strings.EqualFold(od.State, "live"),
			strings.EqualFold(od.State, "pending_settle"),
			strings.EqualFold(od.State, "pending_redeem"),
			strings.EqualFold(od.State, "settled"),
			strings.EqualFold(od.State, "redeemed"):
			addOrd(od)
		}
		if strings.EqualFold(od.State, "live") {
			live = append(live, od)
		}
	}
	dcdSyncEvents(ctx, pool, live)
	dcdSettleReconcile(ctx, pool, ords)
	restOKAt.Store(time.Now().UnixNano())
	return nil
}

// dcdSyncEvents LIVE 集指纹对账（T-0101 到期收口：在持集变动=LIVE→SETTLED
// 迁移/新增部署——md_account_events channel='dcd_live' 落事件行，指纹相同
// 幂等跳写不复读）。写败只 log 不炸对账面。
func dcdSyncEvents(ctx context.Context, pool *pgxpool.Pool, ords []okxexec.DcdOrder) {
	var ids []string
	payload := []map[string]any{}
	for _, od := range ords {
		ids = append(ids, od.OrdID)
		payload = append(payload, map[string]any{
			"ordId": od.OrdID, "productId": od.ProductID, "notionalCcy": od.NotionalCcy,
			"notionalSz": od.NotionalSz, "settleTime": od.SettleTime})
	}
	sort.Strings(ids)
	fp := strings.Join(ids, ",")
	var prev string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(data->>'live_ids','') FROM md_account_events
		WHERE channel='dcd_live' ORDER BY ts DESC LIMIT 1`).Scan(&prev)
	if prev == fp {
		return // 在持集无变——幂等跳写
	}
	data, _ := json.Marshal(map[string]any{"live_ids": fp, "orders": payload})
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(venue,channel,ccy,data,ts)
		VALUES('okx','dcd_live','', $1, now())
		ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
		data=EXCLUDED.data,recv_ts=now()`, data); err != nil {
		log.Printf("[dcd] 持仓集事件行写败: %v", err)
	}
}
