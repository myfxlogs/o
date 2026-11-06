// subreq_test.go —— T-0412 意图晋级径桩测：reconcileSubs 扫尾豁免
// requester='user_intent'（常驻意图非评估判词可裁——超龄/判拒/常驻三闭
// 请件皆不及）；研究提名行照旧三件套扫尾。
package hypothesis

import (
	"context"
	"errors"
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

// TestNominateSubCatalogGate T-0416 提名 catalog 闸桩：inst ∉ 宇宙目录=
// 拒提名零落行+nom_offcatalog 计（MUU 型假符进盒前拒掉病灶闸）；目录员
// 提名照常落 open；目录读败=提名让位 fail-closed 零落行。
func TestNominateSubCatalogGate(t *testing.T) {
	ctx, r := setup(t)
	r.AttachCatalog(func(context.Context) (map[string]bool, error) {
		return map[string]bool{"QQQ-USDT-SWAP": true}, nil
	})
	// 目录员提名→open 行落。
	r.nominateSub(ctx, "QQQ-USDT-SWAP", "xvenue-carry", "carry_diff_threshold")
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='QQQ-USDT-SWAP' AND state='open'`); n != 1 {
		t.Fatalf("目录员提名应落 open: %d", n)
	}
	// 假符（MUU 型）目录外→拒零落行。
	r.nominateSub(ctx, "MUU-USDT", "xvenue-carry", "carry_diff_threshold")
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='MUU-USDT'`); n != 0 {
		t.Fatalf("假符应拒提名零落行: %d", n)
	}
	if n := r.rej.Snapshot()["research/nom_offcatalog"]; n != 1 {
		t.Fatalf("nom_offcatalog 应计 1: %d", n)
	}
	// 目录读败=让位 fail-closed——目录员提名亦不落。
	r.AttachCatalog(func(context.Context) (map[string]bool, error) {
		return nil, errors.New("instruments 面断")
	})
	r.nominateSub(ctx, "ZZZ-USDT-SWAP", "xvenue-carry", "carry_diff_threshold")
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='ZZZ-USDT-SWAP'`); n != 0 {
		t.Fatalf("目录读败应让位零落行: %d", n)
	}
}
