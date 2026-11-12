// flow_rollback.go —— LEGGED 回滚处置（自 flow.go 拆件——棘轮 D-588）。
package execution

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"o/internal/trading/okxexec"
)

// rollback leg1 对向平仓腿（LEGGED 处置）；回滚败=rollback_failed 升级留痕。
func (r *Runner) rollback(ctx context.Context, execID int64, leg Leg, res OrderResult) {
	rev := leg
	if leg.Side == "buy" {
		rev.Side = "sell"
	} else {
		rev.Side = "buy"
	}
	// 回滚方向律（T-0046 ③根修，canon rollback-close-direction）：rev.Close 随原腿
	// 反置——开腿回滚=平（T-0036 语义）；平腿回滚=重开（恒 true=reduceOnly 打空仓
	// 拒单→歧态势，exec5 实证）。语义位按 Intent 上下文派生非恒定值。
	rev.Close = !leg.Close
	rev.Ticket = res.FillTicket
	clOrdID := fmt.Sprintf("ph%drb%d", execID, time.Now().UnixNano()%1e6)
	rb, err := r.exec.Place(ctx, rev, clOrdID)
	// 回滚径拒码留痕（T-0046 ⑤：任何 Place/Resolve 败径都带拒码——envelope 业务拒
	// 先抽码，透传失败面 leg 行）。
	var rvc, rvm string
	var rae *okxexec.APIError
	if errors.As(err, &rae) {
		rvc, rvm = rae.Code, rae.Msg
	}
	if err != nil {
		rb, err = r.exec.Resolve(ctx, rev, clOrdID)
	}
	if err != nil || (rb.State != "filled" && rb.State != "live" && rb.State != "partially_filled") {
		r.setState(ctx, execID, "rollback_failed")
		r.rej.Inc("rollback_failed")
		r.alert(ctx, "rollback_failed", "LEGGED 回滚败——人工介入")
		log.Printf("ALERT [execution] #%d 回滚败——残留 leg1 敞口须人工升级处置（err=%v）", execID, err)
		r.leg(ctx, execID, 1, leg, clOrdID, "ambiguous", rvc, rvm)
		return
	}
	legID := r.legPx(ctx, execID, 1, leg, clOrdID, "rolled_back", "", "", 0, rb.FillSz, "market", rb.FillTicket)
	// 回滚腿=真实现金事件（买回即卖出）——spot 腿同径记账（rev.Side 定向，px 缺=名义锚）。
	r.legCash(ctx, legID, rev, rb.FillPx, rb.FillSz, nil, rb.Fee, rb.FeeCcy)
	r.setState(ctx, execID, "rolled_back")
	log.Printf("[execution] #%d LEGGED→leg1 回滚成", execID)
}

// alert 告警器空守（未接线=静默降级不炸——产线必挂，桩测免挂）。
func (r *Runner) alert(ctx context.Context, kind, msg string) {
	if r.al != nil {
		r.al.Alert(ctx, kind, msg)
	}
}
