// subreq_test.go —— T-0412 意图晋级径桩测：reconcileSubs 扫尾豁免
// requester='user_intent'（常驻意图非评估判词可裁——超龄/判拒/常驻三闭
// 请件皆不及）；研究提名行照旧三件套扫尾。
package hypothesis

import (
	"testing"
)

// TestReconcileSubsUserIntentExempt user_intent 超龄 open 行豁免扫尾；
// 同窗研究提名行照旧闭请（48h>24h 默认阈）。
func TestReconcileSubsUserIntentExempt(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_sub_request(ts,venue,inst_id,requester,state,reason)
		VALUES(now()-interval '48 hours','okx','TRX-USDT-SWAP','user_intent','open','price_alert#1'),
		      (now()-interval '48 hours','okx','DOGE-USDT-SWAP','xvenue-carry','open','carry_diff_threshold')`); err != nil {
		t.Fatal(err)
	}
	r.reconcileSubs(ctx)
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='TRX-USDT-SWAP' AND requester='user_intent' AND state='open'`); n != 1 {
		t.Fatalf("user_intent 常驻意图应豁免扫尾仍 open: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='DOGE-USDT-SWAP' AND requester='xvenue-carry' AND state='closed'`); n != 1 {
		t.Fatalf("研究提名超龄应照旧闭请: %d", n)
	}
}
