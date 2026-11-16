// ddbar.go —— 回撤硬线 5 分钟柱聚合件（工单#20799 O-DD1）：采样只进内存，
// 每满一柱（risk.dd_bar_seconds，默认 300s）取中位数落 md_equity_curve 柱行
// （src='guard_dd_bar'），收柱才判撤、峰值只读柱行。
// 柱长 300s 而非 60s 的两条理由：①真实回撤以日计，5 分钟确认窗对保护
// 时滞无感；②权益读径里 DCD 在持单枚举有 60s TTL 缓存
// （okxexec.DcdLiveOrders），结算回款瞬间可出现最长 60s 的整段双计/漏计
// ——60s 柱会整柱被污染（峰值毒化或假回撤），300s 柱里它占不到 1/2，
// 被中位数滤掉。
// 串行前提：push 仅由 events.ListenLoop 单 goroutine 驱动的 scanOnce 调用
// ——不加锁（与 Warden 其余字段同前提）。
package guard

import (
	"sort"
	"time"
)

const (
	ddBarMinSamples = 3    // 中位数抗单点异常的结构下限（非业务阈）
	ddBarSecMin     = 60   // 柱长合法域下界——结构护栏非业务阈（越界=失明告警）
	ddBarSecMax     = 3600 // 柱长合法域上界——结构护栏非业务阈（越界=失明告警）
)

// ddBar 在收柱键——柱内样本攒内存，跨柱键=收上一柱。
type ddBar struct {
	key int64     // 柱键=now.Unix()/barSec（0=未起柱）
	vs  []float64 // 柱内样本
}

// push 收样。now 落入新柱键=收上一柱：返回（上一柱中位数, 上一柱末时刻,
// 上一柱样本数, true），并以本样本起新柱；同柱=追加返回 closed=false。
// 首样本（key==0）只起柱不收柱。barSec 热改导致键跳变=按跨柱处理（收旧柱）。
func (b *ddBar) push(now time.Time, barSec float64, eq float64) (med float64, end time.Time, n int, closed bool) {
	sec := int64(barSec)
	if sec < 1 {
		sec = 1
	}
	k := now.Unix() / sec
	if b.key == 0 || k == b.key {
		b.key = k
		b.vs = append(b.vs, eq)
		return 0, time.Time{}, 0, false
	}
	med = ddMedian(b.vs)
	n = len(b.vs)
	end = time.Unix((b.key+1)*sec, 0)
	b.key = k
	b.vs = append(b.vs[:0], eq)
	return med, end, n, true
}

// ddMedian 中位数——排序副本取中；偶数取中间两数均值。
func ddMedian(vs []float64) float64 {
	cp := append([]float64(nil), vs...)
	sort.Float64s(cp)
	m := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[m]
	}
	return (cp[m-1] + cp[m]) / 2
}
