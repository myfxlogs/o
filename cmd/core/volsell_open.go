// volsell_open.go —— vol-sell 族 live 开仓 executor（T-0215——假想卡
// id=6/7 钦定名 vol_sell_strangle_executor/vol_sell_single_call_executor）。
// 自 open.go 拆出（check-lines 棘轮）。
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"o/internal/portfolio/lifecycle"
)

// volSellStrangle strangle 双腿买（T-0215——inst_id 复合键「Cprod+Pprod」拆腿，
// 各腿等权名义（notional/2）走 dcdOpen 同三环 quote→trade→status；腿1 成腿2
// 败=半构歧态势——DCD 无撤销面不可回滚，ALERT 留痕人工对账非吞）。
func (h hedgeOpener) volSellStrangle(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	parts := strings.Split(spec.InstID, "+")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, fmt.Errorf("strangle 复合键形不符（预期 Cprod+Pprod）: %q", spec.InstID)
	}
	var first int64
	for i, pid := range parts {
		leg := spec
		leg.InstID = pid
		leg.NotionalUSD = spec.NotionalUSD / 2 // 双腿等权名义——与评估组合 APR 语义对齐
		eid, err := h.dcdOpen(ctx, leg)
		if err != nil {
			if i > 0 {
				log.Printf("ALERT [open] strangle 半构：%s 腿 %s 已买（exec=%d），%s 腿败——歧态势人工对账（DCD 无撤销面）: %v",
					spec.InstID, parts[0], first, pid, err)
				return first, fmt.Errorf("strangle 半构歧态势（leg1 exec=%d 已成，leg2 败——人工对账禁补单）: %w", first, err)
			}
			return eid, err // 腿1 即败=零成交——同 dcdOpen 败径
		}
		if first == 0 {
			first = eid
		}
	}
	log.Printf("[lifecycle] strangle 双腿开成 exec=%d inst=%s notional=%.2f", first, spec.InstID, spec.NotionalUSD)
	return first, nil
}
