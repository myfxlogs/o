// remnant.go —— 残仓类级枚举件（T-0327：策略死/产品活幽灵态——venue 在持但
// 无活策略归属的仓兜底枚举。全集化在持面=策略簿记仓（fin_position）∪残仓
// （venue 实拍−归属集）；归属集=未平仓 inst ∪ 未平仓腿 inst ∪ venue 实态
// 策略（probe/live/degraded）inst——paper/proposed 不出资不归属。
// venue 源=OKX account/positions + dcd 产品单 + MT5 在持定单，各源独立
// fail-soft：读件缺=missing 名段显形不静默（静默=幽灵态复辟）；注入件缺
// （nil）=该面缺席不记缺（与 dcdOrds nil→读件缺语义分野——残仓臂是兜底面
// 非主读面，桩测未注入面不造假缺警）。
package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"o/internal/gen/dashboardv1"
)

// remnantItem 残仓行（src 区分消费面去重——doing 的 dcd 臂已 venue 直读承载，
// 消费方按 src 过滤防同品双行）。
type remnantItem struct {
	src   string     // dcd|okx|mt5
	ref   string     // 展示名（DCD <productId> / OKX <instId> / MT5 <symbol>#<ticket>）
	since *time.Time // venue 开仓时点（可缺=nil）
}

// msToTime 毫秒时戳串→*time.Time（空/解析败=nil——venue 时戳缺席不冒零时点）。
func msToTime(ms string) *time.Time {
	v, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || v <= 0 {
		return nil
	}
	t := time.UnixMilli(v).UTC()
	return &t
}

// holdingAttr 残仓归属判定集（fin_position 未平仓 ∪ 其腿 inst ∪ venue 实态
// 策略 inst——「活策略」=probe/live/degraded 出资态；paper/proposed 簿记态
// 无 venue 仓不归属，retired=策略死正是幽灵类）。
func (c coreService) holdingAttr(ctx context.Context) (map[string]bool, error) {
	rows, err := c.pool.Query(ctx, `SELECT inst_id FROM fin_position WHERE closed_at IS NULL
		UNION SELECT l.inst_id FROM fin_order_leg l
			JOIN fin_position p ON p.execution_id=l.execution_id WHERE p.closed_at IS NULL
		UNION SELECT inst_id FROM fin_strategy WHERE state IN ('probe','live','degraded')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out[s] = true
	}
	return out, rows.Err()
}

// venueRemnants 残仓枚举（venue 在持 − 归属集=残仓；missing=各源读件缺名段
// ——归属集查败=整臂显缺（attr 不可信时宁缺不冒判，全 venue 仓误判残仓=误报
// 洪峰比漏报更劣）。
func (c coreService) venueRemnants(ctx context.Context) (items []remnantItem, missing []string) {
	attr, err := c.holdingAttr(ctx)
	if err != nil {
		return nil, []string{"归属集"}
	}
	// ①dcd 产品单（venue 直读——策略死产品活正是本案）。
	if ords, ok := c.dcdOrds(ctx); !ok {
		missing = append(missing, "dcd")
	} else {
		for _, od := range ords {
			if !attr[od.ProductID] {
				items = append(items, remnantItem{src: "dcd",
					ref: "DCD " + od.ProductID, since: msToTime(od.CTime)})
			}
		}
	}
	// ②OKX 账户持仓（合约/期权/杠杆——策略死腿残仓；腿 inst 已在归属集防
	// 对冲仓腿误判）。
	if c.posLive != nil {
		if ps, perr := c.posLive(ctx); perr != nil {
			missing = append(missing, "okx 持仓")
		} else {
			for _, p := range ps {
				if !attr[p.InstID] {
					items = append(items, remnantItem{src: "okx",
						ref: "OKX " + p.InstID, since: msToTime(p.CTime)})
				}
			}
		}
	}
	// ③MT5 在持定单（同会话捎读件——读败=ordersErr 显缺）。
	if c.mt5Live != nil {
		if accts, aerr := c.mt5Live(ctx); aerr != nil {
			missing = append(missing, "mt5")
		} else {
			for _, a := range accts {
				if a.OrdersErr != nil {
					missing = append(missing, "mt5 持仓")
					continue
				}
				for _, od := range a.Orders {
					if !attr[od.Symbol] {
						it := remnantItem{src: "mt5",
							ref: "MT5 " + od.Symbol + " #" + strconv.FormatInt(od.Ticket, 10)}
						if !od.At.IsZero() {
							t := od.At.UTC()
							it.since = &t
						}
						items = append(items, it)
					}
				}
			}
		}
	}
	return items, missing
}

// doingRemnants 在持面残仓臂（T-0327——非 dcd 残仓行+非 dcd 缺显缺行入列：
// dcd 残仓由 doing 既有 venue 直读臂承载按 src 过滤防双行，dcd 缺同理由
// 上臂「读件缺」行承载。返回入行数供静持真空判据并吞——venue 仓未拍全
// 不冒「无开仓」）。
func (c coreService) doingRemnants(ctx context.Context, o *dashboardv1.OverviewResponse) int {
	rems, rmiss := c.venueRemnants(ctx)
	n := 0
	for _, it := range rems {
		if it.src == "dcd" {
			continue
		}
		st := time.Now().UTC().Format(time.RFC3339)
		if it.since != nil {
			st = it.since.Format(time.RFC3339)
		}
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Stamp: st,
			Text: fmt.Sprintf("%s — 残仓在持，无活策略归属", it.ref)})
		n++
	}
	for _, m := range rmiss {
		if m == "dcd" {
			continue
		}
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Stamp: time.Now().UTC().Format(time.RFC3339),
			Text: fmt.Sprintf("残仓面 %s——读件缺（venue 在持未拍全，不冒零）", m)})
		n++
	}
	return n
}
