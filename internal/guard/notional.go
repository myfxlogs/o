// notional.go —— 腿名义额量纲消歧读（T-0107 自 facts.go 拆件——行数棘轮）。
package guard

import (
	"context"
	"time"

	"o/internal/mt5"
	"o/internal/platform/legset"
)

// notionalNow 腿现价名义额（量纲按乐器分派——T-0046 根修）。
// （sz=张数——ctVal 经 cmd 注入 venue 现读件（T-0048 后 facts 表废，D-0008 决策
// 时点现读——okx instruments 静态属性短 TTL 缓存径），禁按币种名猜）。 无 tick/合约缺 ctVal=false（调用方 facts_missing 计数不误判）。
// T-0128 新鲜度闸：md_ticks 状态表化下行恒在、内容可陈——tick 年龄超
// exec.tick_max_age_s（缺省 120s）= 等同缺失 false（陈=缺 fail-closed，
// 一腿陈一腿新不再产价时差型假失衡；#23 误杀实证）。
func (w *Warden) notionalNow(ctx context.Context, l leg) (float64, bool) {
	// 升件#9802③：sim 域显拒——演习夹具以真实 inst_id 播种 venue='sim' 行，
	// 无 venue 谓词=假价污判官径（risk.dcdExposureUSD 同口径先滤）。
	// O-41385：价源=legset.MarkPx 单源（中价∪mark_index union——原平行件收编）。
	px, ts, _, ok := legset.MarkPx(ctx, w.pool, "", l.inst)
	if !ok {
		return 0, false
	}
	maxAge := legset.TickMaxAgeDefS
	if v, err := w.getF("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	if mt5.IsCfdProtoVenue(ctx, w.pool, l.venue) { // 名册 proto='cfd' 真源判——缺席/读败=严闸 fail-closed（O-41382）
		maxAge = legset.SparseTickBoundDefS
		if v, err := w.getF("lifecycle.dir.cfd_tick_max_age_s"); err == nil && v > 0 {
			maxAge = v
		}
	}
	if time.Since(ts) > time.Duration(maxAge*float64(time.Second)) {
		return 0, false // 陈 tick=事实缺——fail-closed 同 facts_missing 计数
	}
	if l.qtyBase > 0 {
		if l.szUnit == "contract" {
			// T-0522 根修：datedFUT qty_base=实填张数（T-0483 钦定——SWAP
			// 域 NULL 不同形）非币量——按基量×px 会虚胖 1/ctVal 倍
			// （BTC 域~1000×）→delta_drift 绝对兜底秒杀全谱 basis 仓。
			ct, ok := w.ctValOf(ctx, l.venue, l.inst)
			if !ok {
				return 0, false
			}
			return l.qtyBase * ct * px, true
		}
		return l.qtyBase * px, true // quote/base/lot 域 qty_base=币量
	}
	switch l.szUnit {
	case "base":
		return l.sz * px, true
	case "contract":
		ct, ok := w.ctValOf(ctx, l.venue, l.inst)
		if !ok {
			return 0, false
		}
		return l.sz * ct * px, true
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
