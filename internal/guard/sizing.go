// sizing.go —— 平仓目标量量纲消歧+venue 现读对齐（T-0419 自 facts.go 拆件——行数棘轮）。
package guard

import (
	"context"
	"math"
	"strings"
)

// closeWant 平仓目标量（下单方言量纲消歧——T-0419 根修：开腿 sz 量纲随
// sz_unit 漂移〔quote=美元额〕，裸拷当 base 发=错平——生产实证 24.99 美元额
// 只清 1.3% 币量）：
//
//	SWAP=sz 张数；qty_base>0=venue 实填币量（唯一真源）；sz_unit=base=sz；
//	quote 腿 qty_base 缺/量纲未消歧=(0,false)——sizedCloseLegs 备源 venue
//	余额实查（钦定「读腿/读持仓余额」双径），美元额禁当 base 卖。
func (l leg) closeWant() (float64, bool) {
	if strings.HasSuffix(l.inst, "-SWAP") {
		return l.sz, true
	}
	if l.qtyBase > 0 {
		return l.qtyBase, true
	}
	if l.szUnit == "base" {
		return l.sz, true
	}
	return 0, false
}

// VenueSizer 平仓量 venue 现读面（cmd 装配注入——guard 不识 venue 方言）。
// D-0008 钦定径（canon cex-close-sz-fee-in-base）：再操作量从 venue 实况推
// ——现货平仓量=availBal（taker 费基准币内扣致实持<委托）、合约=posCcy 张数； 禁抄腿行委托量。读败=fail-closed 不构平仓腿。
type VenueSizer interface {
	SpotAvailBal(ctx context.Context, instID string) (float64, error)
	ContractPos(ctx context.Context, instID string) (float64, error) // 净持仓张数 abs
	LotSz(ctx context.Context, instID string) (float64, error)       // 数量步长
}

// alignCloseSz 平仓量对齐：min(腿记录, venue 实读) 按 lotSz 网格向下取整； 对齐后≤0=false（fail-closed 不发残量单）。
func alignCloseSz(want, avail, lotSz float64) (float64, bool) {
	sz := math.Min(want, avail)
	if lotSz > 0 {
		sz = math.Floor(sz/lotSz+1e-9) * lotSz
	}
	if sz <= 0 {
		return 0, false
	}
	return sz, true
}

// sizedCloseLegs 反向腿 venue 实读定量（T-0046 ②根修）——任一腿读败/对齐零= false（fail-closed：不构 intent 不发半仓平仓）。
func (w *Warden) sizedCloseLegs(ctx context.Context, cls [2]CloseLeg) ([2]CloseLeg, bool) {
	if w.sizer == nil {
		return cls, false // 未接线=禁发（现读面缺席——腿行推导禁线）
	}
	for i, cl := range cls {
		var avail float64
		var err error
		if strings.HasSuffix(cl.InstID, "-SWAP") {
			avail, err = w.sizer.ContractPos(ctx, cl.InstID)
		} else {
			avail, err = w.sizer.SpotAvailBal(ctx, cl.InstID)
		}
		if err != nil {
			return cls, false
		}
		lot, err := w.sizer.LotSz(ctx, cl.InstID)
		if err != nil {
			return cls, false
		}
		// T-0419：腿账不可定（SzKnown=false——quote 腿 qty_base 缺/量纲未消歧）
		// →目标量备源=venue 余额实查（钦定双径之「读持仓余额」）；腿账可定
		// =min(腿实得,venue 实余) 双界对齐。美元额绝不当 base 发。
		want := cl.Sz
		if !cl.SzKnown {
			want = avail
		}
		sz, ok := alignCloseSz(want, avail, lot)
		if !ok {
			return cls, false
		}
		cls[i].Sz = sz
		cls[i].SzKnown = true
	}
	return cls, true
}
