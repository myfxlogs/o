package ledger

// recon_acct.go —— O-39440 账户粒度对账段（run.go 拆出——棘轮闸）：
// 遍历 fin_account active 行逐账户对拍——读面缺席账户（cred_gap 升件
// 径已覆盖）不判不警不造 0；台账组件 venue 级量经 compAccount 归户；
// 豁免/qty 豁免/brief 全按 account_id 精确扣。

import (
	"context"
	"log"
	"math"
)

// reconRow 对拍行（账户×币——eq=venue 实读/net=账本净额含组件归户；
// reconAccounts 判定段与 rebaseDecree 重锚段同源消费,禁两处数学漂移）。
type reconRow struct {
	acct, venue, ccy string
	eq, net          float64
}

// reconRows 对拍行集装配（原 reconAccounts 前段原样——账户遍历/
// 读面存在性/组件归户/mtm 并入一处；读败键让位不入集）。
func (r *Runner) reconRows(ctx context.Context, accts []AccountRow,
	ven map[[2]string]float64, wl map[[2]string]bool,
	parked, dcdBase, nativeBase map[[2]string]float64) []reconRow {
	// 组件归户预计算（每 venue 的 earn/dcd/trading 账户——compAccount
	// 同规则;无对应 sub_type=该组件不入任何账户对拍）。
	compOf := func(venue, sub string) string {
		for _, a := range accts {
			if a.Venue == venue && a.SubType == sub {
				return a.ID
			}
		}
		return ""
	}
	tradingOf := func(venue string) string {
		if id := compOf(venue, "trading"); id != "" {
			return id
		}
		return compAccount(accts, venue, "trading")
	}
	// 账户读面存在性——ven 键 account_id 集+子户型账户恒 present：
	// section 型（trading/funding/earn/dcd）共享 venue 原子读面（任一腿
	// 读败=整 bal err 上抛走不到此）——零余额子户无键≠读败,台账有量
	// 账户照判补零差（漏报根修语义保存）；main 型=逐户独立会话读面
	// （MT5-proto 经纪商）——无键=该户会话败/cred_gap,不判不警。
	present := map[string]bool{}
	for k := range ven {
		present[k[0]] = true
	}
	for _, a := range accts {
		if a.SubType != "main" {
			present[a.ID] = true
		}
	}
	// T-0076：盯市偏移 venue×ccy 图单取（循环外——原逐键调用同图）。
	var mtmMap map[[2]string]float64
	if r.mtm != nil {
		if mv, merr := r.mtm(ctx); merr == nil {
			mtmMap = mv
		}
	}
	var rs []reconRow
	for _, a := range accts {
		if !present[a.ID] {
			continue // 读面缺席（cred_gap 升件径已覆盖）——不判不警
		}
		v := VenueOf(a.ID)
		for k := range wl {
			if k[0] != a.ID {
				continue
			}
			ccy := k[1]
			var net float64
			// 升件#10063③：判据喂数禁吞错——读败=计数+WARN+本键让位。
			if err := r.pool.QueryRow(ctx,
				"SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE account_id=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')",
				a.ID, ccy).Scan(&net); err != nil {
				r.rej.Inc("recon_probe_failed")
				log.Printf("WARN [ledger] 判据喂数读败 net %s/%s——本键让位: %v", a.ID, ccy, err)
				continue
			}
			// 组件归户（venue 级台账量→账户——compAccount 规则同源）。
			if compOf(v, "earn") == a.ID {
				net += parked[[2]string{v, ccy}]
			}
			if compOf(v, "dcd") == a.ID {
				net += dcdBase[[2]string{v, ccy}]
			}
			if tradingOf(v) == a.ID {
				net += nativeBase[[2]string{v, ccy}] + mtmMap[[2]string{v, ccy}]
			}
			rs = append(rs, reconRow{a.ID, v, ccy, ven[k], net})
		}
	}
	return rs
}

// reconAccounts 账户粒度对拍主循环（run.go reconcile 尾段——
// ven/vparts/wl 均为账户域键 {account_id,ccy}；parked/dcdBase/
// nativeBase 保持 venue 级 {venue,ccy} 原键经归户规则入账）。
func (r *Runner) reconAccounts(ctx context.Context,
	ven map[[2]string]float64, vparts VenueParts,
	wl map[[2]string]bool, parked, dcdBase, nativeBase map[[2]string]float64,
	tol float64) error {
	accts, err := AccountRows(ctx, r.pool, "active")
	if err != nil {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 账户册读败——本轮对账让位: %v", err)
		return nil
	}
	rs := r.reconRows(ctx, accts, ven, wl, parked, dcdBase, nativeBase)
	for _, x := range rs {
		// T-0081 已知残差豁免：未过期豁免额补进容差——只压已定性部分，
		// unexplained 超额仍即报（豁免≠白名单静默一切）。
		var exempt float64
		// 升件#10063③：豁免额读败=exempt 静默失效过警——同判例计数+让位。
		// O-36327：豁免额=静态行+resolver 跟踪行现算（trackedExempt 同源——
		// 破带行贡献归零+升件，回退/静态语义见 residual_tracked.go）。
		// O-39440：豁免按账户精确扣（fin_known_residual.account_id——
		// 未归属行不匹配任何账户=fail-safe 不静默吞差）。
		var exErr error
		exempt, _, exErr = r.trackedExempt(ctx, x.venue, x.ccy, x.acct)
		if exErr != nil {
			r.rej.Inc("recon_probe_failed")
			log.Printf("WARN [ledger] 判据喂数读败 exempt %s/%s——本键让位: %v", x.acct, x.ccy, exErr)
			continue
		}
		diff := Diff(x.net, x.eq)
		over := false
		var dout *diffOut
		if !isBaseCCY(x.ccy) {
			// O-38064 S3：非本位键币量豁免先扣（币量空间不经价）→残量 rq
			// 折价判差；rq≈0 不告警不查价；有鲜价 diffUSD=rq*px 对容差+
			// USD 豁免；无鲜价 fail-closed 照报不产出美元数（删
			// tol+exempt+1 强制式——自举怪圈+隐含价像根修）。
			qx, qerr := r.qtyExempt(ctx, x.venue, x.ccy, x.acct)
			if qerr != nil {
				r.rej.Inc("recon_probe_failed")
				log.Printf("WARN [ledger] qty 豁免读败 %s/%s——本键让位: %v", x.acct, x.ccy, qerr)
				continue
			}
			rq := diff - qx
			if rq <= 1e-9*math.Max(math.Abs(x.net), math.Abs(x.eq)) {
				continue // 币量豁免后残量≈0——不告警不查价
			}
			dout = &diffOut{qty: rq}
			if px, ok := r.diffPx(ctx, x.venue, x.ccy); ok {
				dout.priced, dout.usd = true, rq*px
				over = dout.usd > tol+exempt
			} else {
				over = true // 无鲜价 fail-closed 照报（无价≠静默）
			}
		} else {
			over = diff > tol+exempt
		}
		r.alertLedgerDiff(ctx, x.acct, x.venue, x.ccy, x.eq, x.net, tol, exempt, diff,
			over, acctParts(vparts, x.acct, x.ccy), dout)
	}
	return nil
}
