package guard

// anomaly.go —— 腿形态异常定性行（工单#22845 拆件：facts.go 行棘轮顶线）。

import (
	"context"
	"fmt"
	"strconv"
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
	// 工单#20672②：表行外补升件径——fin_guard_close 是态非信号（探测薄壳
	// 律「疼须写成信号行」），原 WARN 刷频零 ops_event=机器不可见实证。
	// dedup=同 scope 未销升件且同仓号在案即抑（销而未修=真再发照火）；
	// 查败 fail-open 不补——定性行已载病灶，升件缺位下拍自重试。
	var open bool
	if err := w.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope='guard_leg_anomaly'
		AND e.note ~ ('仓 #'||$1::text||'([^0-9]|$)')
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`, strconv.FormatInt(posID, 10)).Scan(&open); err != nil || open {
		return
	}
	// 工单#24838：票抑同契补——本径自写 INSERT 原绕 escInsert 票抑闸
	// （销一件火一件三火实证：同 scope OPEN 票在案仍每拍复燃）。
	// 钦定候选①：升件前查同 scope 开工单，SQL 照 escInsert 票抑段
	// 同契——开工单=无处置引「工单#id」∨挂票期限内；销票/挂票过期
	// =抑撤条件存自然复火。读败 fail-open 升件优先不吞。
	var tick int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)') AND d.note !~ ('工单#'||t.id||' ?接续'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		"guard_leg_anomaly").Scan(&tick); err == nil && tick > 0 {
		return // 开工单在案——票管催,探针静默(读败不抑=fail-open升件优先)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('仓巡','升件','',$1,'守卫环','guard_leg_anomaly')`,
		fmt.Sprintf("仓 #%d 腿形态异常不可反向平（定性行已落 fin_guard_close）——需专道收殓：dcd 到期结算∨人工介入，守卫反向平径不适用", posID)); err != nil {
		w.rej.Inc("write_err")
	}
}
