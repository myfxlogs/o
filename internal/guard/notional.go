// notional.go —— 腿名义额量纲消歧读（T-0107 自 facts.go 拆件——行数棘轮）。
package guard

import (
	"context"
	"time"
)

// notionalNow 腿现价名义额（量纲按乐器分派——T-0046 根修）。
// （sz=张数——ctVal 经 cmd 注入 venue 现读件（T-0048 后 facts 表废，D-0008 决策
// 时点现读——okx instruments 静态属性短 TTL 缓存径），禁按币种名猜）。 无 tick/合约缺 ctVal=false（调用方 facts_missing 计数不误判）。
// T-0128 新鲜度闸：md_ticks 状态表化下行恒在、内容可陈——tick 年龄超
// exec.tick_max_age_s（缺省 120s）= 等同缺失 false（陈=缺 fail-closed，
// 一腿陈一腿新不再产价时差型假失衡；#23 误杀实证）。
func (w *Warden) notionalNow(ctx context.Context, l leg) (float64, bool) {
	var last *float64
	var ts time.Time
	if err := w.pool.QueryRow(ctx,
		"SELECT last,ts FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1", l.inst).Scan(&last, &ts); err != nil || last == nil {
		return 0, false
	}
	maxAge := 120.0
	if v, err := w.getF("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	if time.Since(ts) > time.Duration(maxAge*float64(time.Second)) {
		return 0, false // 陈 tick=事实缺——fail-closed 同 facts_missing 计数
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
