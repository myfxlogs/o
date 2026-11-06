// beats.go —— 四拍流水（T-0379）：①探测→②映射草拟→③map 闸→④paper 激活。
// 每拍 ops_event「venue接入」行落账；败径 fail-closed stay 'registered' 自报。
package onboard

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"o/internal/mt5"
)

// ── ①探测（registered→probed）───────────────────────────────────────

// beatProbe Connect→AccountSummary→SymbolList→SymbolParams（能力账 canon
// 序——skills/mt5-* 五件不猜）→onboard report 落 ops_event+evidence。
// 三败径（凭据坏/网关不可达/符号表空）=fail-closed 自报不跃迁；败重试
// backoff=probeRetryGap（evidence.attempts/last_attempt_at 锚）。
func (r *Runner) beatProbe(ctx context.Context, b credFamily, ob map[string]any) (bool, error) {
	if b.missing {
		return false, nil // 凭据不在册——安静候（发现拍已落 registered 候凭据）
	}
	login := b.fields["login"]
	if !r.retryDue(ob) {
		return false, nil
	}
	fail := func(why string) (bool, error) {
		ob["attempts"] = num(ob["attempts"]) + 1
		ob["last_attempt_at"] = r.now().UTC().Format(time.RFC3339)
		ob["last_error"] = why
		r.rej.Inc("probe_failed")
		return false, r.trans(ctx, b.venue, login, "", "探测",
			ob, "败径 fail-closed："+why, true)
	}
	if b.broken != "" {
		return fail("凭据坏：" + b.broken)
	}
	addr, err := r.cfgStr(ctx, "mt5.gateway_addr")
	if err != nil {
		return fail("mt5.gateway_addr: " + err.Error())
	}
	loginN, _ := strconv.ParseUint(login, 10, 64)
	sess, err := r.dial(addr)
	if err != nil {
		return fail("网关不可达 dial: " + err.Error())
	}
	defer sess.Close()
	pctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := sess.Connect(pctx, mt5.Credentials{Login: loginN,
		Password: b.fields["password"], Server: b.fields["server"], Host: b.fields["host"]}); err != nil {
		return fail("网关不可达/凭据拒 connect: " + err.Error())
	}
	defer sess.Disconnect(context.Background())
	eq, fm, bal, err := sess.AccountSummary(pctx)
	if err != nil {
		return fail("AccountSummary: " + err.Error())
	}
	syms, err := sess.SymbolList(pctx)
	if err != nil {
		return fail("SymbolList: " + err.Error())
	}
	if len(syms) == 0 {
		return fail("符号表空（SymbolList 零行——venue 侧缺陷 fail-closed）")
	}
	params := map[string]any{}
	for _, s := range mt5.MatchedSymbols(syms, r.okxUniverse(ctx)) {
		if f, err := sess.SymbolFacts(pctx, s); err == nil {
			params[s] = f
		}
		if len(params) >= maxMapPairs {
			break
		}
	}
	ob["login"] = login
	ob["probed_at"] = r.now().UTC().Format(time.RFC3339)
	ob["probe"] = map[string]any{"equity": eq, "free_margin": fm, "balance": bal,
		"symbols_total": len(syms), "symbol_params": params}
	ob["probed_symbols"] = syms
	delete(ob, "last_error")
	return true, r.trans(ctx, b.venue, login, "probed", "探测", ob,
		fmt.Sprintf("Connect ok login=%s eq=%.2f fm=%.2f symbols=%d params=%d——能力账 canon 序实证",
			login, eq, fm, len(syms), len(params)), false)
}

// retryDue 探测重试闸（attempt 未记/过期=放行）。
func (r *Runner) retryDue(ob map[string]any) bool {
	last, _ := ob["last_attempt_at"].(string)
	if last == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, last)
	return err != nil || r.now().Sub(t) >= probeRetryGap
}
