// ingest_native.go —— O-34287 原生币腿入账件（自 ingest.go 拆出守 300
// 行棘轮）。SPOT/MARGIN 每币独立 bill 行=对称两腿之基币侧。
package funding

import (
	"context"
	"strings"
	"time"
)

// spotNative SPOT/MARGIN 原生币成交腿入账（O-34287 验收④）：bill 每币
// 独立行=对称两腿之基币侧——毛额腿（balChg−fee=sz 域）+fee 腿各记本币。
// kind=spot_buy/spot_sell 按币角色定向（inst BASE-QUOTE：基币入=买/出=卖，
// 计价币入=卖/出=买）；币对不上双段=unmapped_bill 残差如实落。
func (i *Ingester) spotNative(ctx context.Context, b Bill, ccy string, srcID int64, occurred time.Time) (int, error) {
	gross := b.BalChg - b.Fee // balChg=sz+fee 净额实测定案——毛额=净−费
	kind := ""
	if parts := strings.SplitN(b.InstID, "-", 2); len(parts) == 2 {
		inBase := ccy == parts[0]
		if inBase || ccy == parts[1] {
			buy := (inBase && gross > 0) || (!inBase && gross < 0)
			if buy {
				kind = "spot_buy"
			} else {
				kind = "spot_sell"
			}
		}
	}
	n := 0
	if kind != "" && gross != 0 {
		if err := i.post(ctx, kind, gross, b, ccy, srcID, occurred); err != nil {
			return n, err
		}
		n++
	}
	if b.Fee != 0 { // 手续费腿按扣款币（bill.cyc=费币——嵌于 balChg 内拆出不双计）
		if err := i.post(ctx, "fee", b.Fee, b, ccy, srcID, occurred); err != nil {
			return n, err
		}
		n++
	}
	if kind == "" { // 币对解不出=残差守恒兜底（净额真值不丢）
		if err := i.post(ctx, "unmapped_bill", b.BalChg, b, ccy, srcID, occurred); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
