// notional.go —— 腿名义额量纲消歧读（T-0107 自 facts.go 拆件——行数棘轮）。
package guard

import (
	"context"
)

// notionalNow 腿现价名义额（量纲按乐器分派——T-0046 根修）。
// （sz=张数——ctVal 经 cmd 注入 venue 现读件（T-0048 后 facts 表废，D-0008 决策
// 时点现读——okx instruments 静态属性短 TTL 缓存径），禁按币种名猜）。 无 tick/合约缺 ctVal=false（调用方 facts_missing 计数不误判）。
func (w *Warden) notionalNow(ctx context.Context, l leg) (float64, bool) {
	var last *float64
	if err := w.pool.QueryRow(ctx,
		"SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1", l.inst).Scan(&last); err != nil || last == nil {
		return 0, false
	}
	if l.qtyBase > 0 {
		return l.qtyBase * *last, true
	}
	switch l.szUnit {
	case "base":
		return l.sz * *last, true
	case "contract":
		ct, ok := w.ctValOf(ctx, l.venue, l.inst)
		if !ok {
			return 0, false
		}
		return l.sz * ct * *last, true
	}
	return 0, false // quote 缺 base 量/量纲未消歧——禁再误读
}

// ctValOf 合约面值（cmd 注入件——缺件/读败/零值=false fail-closed）。
func (w *Warden) ctValOf(ctx context.Context, venue, inst string) (float64, bool) {
	if w.ctVal == nil {
		return 0, false
	}
	return w.ctVal(ctx, venue, inst)
}
