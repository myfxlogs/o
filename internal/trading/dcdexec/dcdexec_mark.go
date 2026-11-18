// dcdexec_mark.go —— DCD 开仓标定族（标定/拒因/映射工具自
// dcdexec.go 拆出——棘轮 300 行顶线）。
package dcdexec

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/trading/okxexec"
)

// isVenueReject venue 业务终拒分型（*APIError=拒绝态非传输态——
// earn transfer 同律:终拒禁补单禁仲裁,如实败）。
func isVenueReject(err error) bool {
	var ae *okxexec.APIError
	return errors.As(err, &ae)
}

// VenueErr venue 拒因双层抽取（T-0259：内层 sCode 在且≠0=真实拒因优先——
// envelope code=1「All operations failed」壳码实证在案 order.go:13；非
// APIError=传输/解析败，原文留 msg、code=NULL 不臆造）。
func VenueErr(err error) (code, msg string) {
	var ae *okxexec.APIError
	if err == nil {
		return "", ""
	}
	if !errors.As(err, &ae) {
		return "", err.Error()
	}
	if ae.SCode != "" && ae.SCode != "0" {
		if ae.SMsg != "" {
			return ae.SCode, ae.SMsg
		}
		return ae.SCode, ae.Msg
	}
	return ae.Code, ae.Msg
}

// Mark exec 失败标定+拒因同 UPDATE（同语句=同原子写——乱序重建不丢证）；
// 写败显形非吞（拒因落库败=取证面再毁）。err 径供 APIError/传输败复用。
func Mark(ctx context.Context, pool *pgxpool.Pool, execID int64, state string, err error) {
	vc, vm := VenueErr(err)
	MarkVE(ctx, pool, execID, state, vc, vm)
}

// MarkVE 显式码文版（venue 响应行 sCode/sMsg 等非 error 形拒因直落）。
func MarkVE(ctx context.Context, pool *pgxpool.Pool, execID int64, state, vc, vm string) {
	if _, err := pool.Exec(ctx, `UPDATE fin_execution SET state=$2,
		venue_err_code=NULLIF($3,''), venue_err_msg=NULLIF($4,'') WHERE id=$1`,
		execID, state, vc, vm); err != nil {
		log.Printf("ALERT [dcdexec] exec %d 标定败（state=%s——拒因落库丢证风险）: %v", execID, state, err)
	}
}

// QuoteReject quote 步上游拦落件（T-0259：exec 行未建 venue 拒——
// fin_eval_reject 同构拒因表；reason 携码 per-(inst,template,direction,code)
// 幂等行，evidence.hits 累计尝试数=取证面拒率分母，全层码文入 evidence 不丢）。
func QuoteReject(ctx context.Context, pool *pgxpool.Pool, spec Spec, vc, vm string) {
	reason := "dcd_quote_reject"
	if vc != "" {
		reason += ":" + vc
	}
	ev, _ := json.Marshal(map[string]any{
		"step": "quote", "venue_code": vc, "venue_msg": vm,
		"notional_usd": spec.NotionalUSD, "strategy_ref": spec.StrategyRef, "hits": 1,
	})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,evidence)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=now(),
		evidence=jsonb_set(EXCLUDED.evidence,'{hits}',
			to_jsonb(COALESCE((fin_eval_reject.evidence->>'hits')::int,0)+1))`,
		spec.InstID, spec.Template, spec.Direction, reason, ev); err != nil {
		log.Printf("ALERT [dcdexec] dcd quote 拒因落件败（%s %s——证毁风险）: %v", spec.InstID, reason, err)
	}
}

// ExecState order-status 态→fin_execution 态映射（T-0160 返修⑤桩锚件——
// 未载态=空串让位（禁默认 open）；歧态势由调用径置 ambiguous）。
func ExecState(vstate string) string {
	return map[string]string{"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"}[vstate]
}

// Sz 数量格式化（venue sz 方言拒 sci——FormatFloat f,-1 去指数形；
// T-0160 返修②实证锚）。
func Sz(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
