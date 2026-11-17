package guard

// anomaly.go —— 腿形态异常定性行（工单#22845 拆件：facts.go 行棘轮顶线）。

import (
	"context"
)

// markLegAnomaly 腿形态异常定性行（工单#22845）：legs.one 仓反向平径恒
// 不可达（零腿=dcd 单产品仓/畸形腿——无腿可反向）——原 WARN 每拍直刷
// 噪音洪泛。定性行=fin_guard_close upsert 一次（state=close_failed 是
// CHECK 集内唯一贴合态；last_error 载病灶定性）；attempts 不动不进
// 末路 auto 径（attempts>=maxAtt 才触发——本件不伪造尝试账）。进程内
// anomalyMarked 去重，重启后幂等重写；写败撤标下拍重试不吞。
func (w *Warden) markLegAnomaly(ctx context.Context, posID int64, trig string) {
	if w.anomalyMarked[posID] {
		return
	}
	_, err := w.pool.Exec(ctx,
		`INSERT INTO fin_guard_close(position_id,state,trigger_kind,last_error,updated_at)
		 VALUES($1,'close_failed',$2,$3,now())
		 ON CONFLICT(position_id) DO UPDATE SET last_error=$3,updated_at=now()`,
		posID, trig, "leg_anomaly: 腿形态异常不可反向平——需专道收殓（dcd 到期结算/人工），守卫反向平径不适用")
	if err != nil {
		w.rej.Inc("write_err")
		return
	}
	w.anomalyMarked[posID] = true
}
