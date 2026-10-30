// open_dcd_reject.go —— T-0259 dcdOpen venue 拒因持久化面：quote 步上游拦
// （exec 行未建——fin_execution.state='open' 行被 risk/earn 当在途敞口消费，
// 前建行=幻影敞口+闸拒废行，禁此径）→fin_eval_reject 同构拒件；trade 步
// （exec 行在）→fin_execution.venue_err_* 列随标定同 UPDATE 原子写——
// 持久化先于或同事务于失败标定，容器任意时点重建不丢证。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"o/internal/portfolio/lifecycle"
	"o/internal/trading/okxexec"
)

// dcdVenue dcd 执行三步缝（T-0259 桩测注伪位——产线件=*okxexec.Client
// 同签名实现；nil 回退 cli 即产线形，装配面零改）。
type dcdVenue interface {
	DcdQuote(ctx context.Context, productID, notionalSz, notionalCcy string) (string, error)
	DcdTrade(ctx context.Context, quoteID string) (json.RawMessage, error)
	DcdOrderStatus(ctx context.Context, ordID string) (json.RawMessage, error)
}

func (h hedgeOpener) dcdCli() dcdVenue {
	if h.dcd != nil {
		return h.dcd
	}
	return h.cli
}

// dcdVenueErr venue 拒因双层抽取（T-0259：内层 sCode 在且≠0=真实拒因优先——
// envelope code=1「All operations failed」壳码实证在案 order.go:13；非
// APIError=传输/解析败，原文留 msg、code=NULL 不臆造）。
func dcdVenueErr(err error) (code, msg string) {
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

// dcdMark exec 失败标定+拒因同 UPDATE（同语句=同原子写——乱序重建不丢证）；
// 写败显形非吞（拒因落库败=取证面再毁）。err 径供 APIError/传输败复用。
func (h hedgeOpener) dcdMark(ctx context.Context, execID int64, state string, err error) {
	vc, vm := dcdVenueErr(err)
	h.dcdMarkVE(ctx, execID, state, vc, vm)
}

// dcdMarkVE 显式码文版（venue 响应行 sCode/sMsg 等非 error 形拒因直落）。
func (h hedgeOpener) dcdMarkVE(ctx context.Context, execID int64, state, vc, vm string) {
	if _, err := h.pool.Exec(ctx, `UPDATE fin_execution SET state=$2,
		venue_err_code=NULLIF($3,''), venue_err_msg=NULLIF($4,'') WHERE id=$1`,
		execID, state, vc, vm); err != nil {
		log.Printf("ALERT [open] exec %d 标定败（state=%s——拒因落库丢证风险）: %v", execID, state, err)
	}
}

// dcdQuoteReject quote 步上游拦落件（T-0259：exec 行未建 venue 拒——
// fin_eval_reject 同构拒因表；reason 携码 per-(inst,template,direction,code)
// 幂等行，evidence.hits 累计尝试数=取证面拒率分母，全层码文入 evidence 不丢）。
func (h hedgeOpener) dcdQuoteReject(ctx context.Context, spec lifecycle.OpenSpec, vc, vm string) {
	reason := "dcd_quote_reject"
	if vc != "" {
		reason += ":" + vc
	}
	ev, _ := json.Marshal(map[string]any{
		"step": "quote", "venue_code": vc, "venue_msg": vm,
		"notional_usd": spec.NotionalUSD, "strategy_ref": spec.StrategyRef, "hits": 1,
	})
	if _, err := h.pool.Exec(ctx, `INSERT INTO fin_eval_reject(inst_id,template,direction,reason,evidence)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (inst_id,template,direction,reason) DO UPDATE SET ts=now(),
		evidence=jsonb_set(EXCLUDED.evidence,'{hits}',
			to_jsonb(COALESCE((fin_eval_reject.evidence->>'hits')::int,0)+1))`,
		spec.InstID, spec.Template, spec.Direction, reason, ev); err != nil {
		log.Printf("ALERT [open] dcd quote 拒因落件败（%s %s——证毁风险）: %v", spec.InstID, reason, err)
	}
}

// dcdExecState order-status 态→fin_execution 态映射（T-0160 返修⑤桩锚件——
// 未载态=空串让位（禁默认 open）；歧态势由调用径置 ambiguous）。
func dcdExecState(vstate string) string {
	return map[string]string{"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"}[vstate]
}

// dcdSz 数量格式化（venue sz 方言拒 sci——FormatFloat f,-1 去指数形；
// T-0160 返修②实证锚）。
func dcdSz(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
