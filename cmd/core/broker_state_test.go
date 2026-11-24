package main

// broker_state_test.go —— O-37450 生命周期面桩测（roster upsert 同事务/
// SetBrokerState 迁移机全谱/零持仓核闸；HTTP 径走 earnOriginGate 同源闸，
// verify 走 cexBalance seam 禁桩真 venue）。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"o/internal/platform/signutil"
)

const stateProc = "/o.dashboard.v1.CoreService/SetBrokerState"

func statePost(t *testing.T, mux http.Handler, venue, state, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, stateProc,
		strings.NewReader(`{"venue":"`+venue+`","state":"`+state+`"}`))
	req.Host = "ui.local"
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// TestSetBrokerStateTransition 迁移机全谱桩：registered→disabled→
// removed（零持仓核过+凭据族删）→终态拒；disabled→active→connecting；
// 非 disabled→removed 跳档拒。
func TestSetBrokerStateTransition(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t, nil, nil)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_roster(venue,proto,cred_ns,roles,state)
		VALUES('tstlife','cex','tstlife.cex','{collect}','registered')
		ON CONFLICT (venue) DO UPDATE SET state='registered',cred_ns='tstlife.cex'`); err != nil {
		t.Fatalf("roster 造行: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue='tstlife'`)
	defer pool.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE 'tstlife%'`)
	// 非 disabled→removed=跳档拒（FailedPrecondition→HTTP 400 域）。
	rec := statePost(t, mux, "tstlife", "removed", "http://ui.local")
	if rec.Code == 200 {
		t.Fatal("registered→removed 跳档应拒")
	}
	// registered→disabled 过（落定 state=disabled）。
	rec = statePost(t, mux, "tstlife", "disabled", "http://ui.local")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"state":"disabled"`) {
		t.Fatalf("disable: %d %s", rec.Code, rec.Body)
	}
	// disabled→disabled 幂等（200 现态）。
	if rec := statePost(t, mux, "tstlife", "disabled", "http://ui.local"); rec.Code != 200 {
		t.Fatalf("幂等 disable: %d %s", rec.Code, rec.Body)
	}
	// disabled→active=connecting 复活。
	rec = statePost(t, mux, "tstlife", "active", "http://ui.local")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"state":"connecting"`) {
		t.Fatalf("复活: %d %s", rec.Code, rec.Body)
	}
	// 再 disabled→removed：零持仓核过→凭据族删+墓碑。
	statePost(t, mux, "tstlife", "disabled", "http://ui.local")
	if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES('tstlife.cex.api_key','\\x00')`); err != nil {
		t.Fatalf("凭据造行: %v", err)
	}
	rec = statePost(t, mux, "tstlife", "removed", "http://ui.local")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"state":"removed"`) {
		t.Fatalf("removed: %d %s", rec.Code, rec.Body)
	}
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets WHERE key LIKE 'tstlife.cex.%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("凭据族应删净余 %d", n)
	}
	// removed→任何=终态拒。
	if rec := statePost(t, mux, "tstlife", "disabled", "http://ui.local"); rec.Code == 200 {
		t.Fatal("removed→disabled 应拒")
	}
}

// TestSetBrokerStateZeroPositionGate 零持仓核闸：策略非退役在=拒。
func TestSetBrokerStateZeroPositionGate(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t, nil, nil)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_roster(venue,proto,cred_ns,roles,state)
		VALUES('tstpos','cex','tstpos.cex','{collect}','disabled')
		ON CONFLICT (venue) DO UPDATE SET state='disabled'`); err != nil {
		t.Fatalf("roster 造行: %v", err)
	}
	defer pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue='tstpos'`)
	defer pool.Exec(ctx, `DELETE FROM fin_strategy WHERE venue='tstpos'`)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('tstpos-i','t','live',1,'tstpos')`); err != nil {
		t.Fatalf("策略造行: %v", err)
	}
	rec := statePost(t, mux, "tstpos", "removed", "http://ui.local")
	if rec.Code == 200 || !strings.Contains(rec.Body.String(), "零持仓核") {
		t.Fatalf("有策略应拒: %d %s", rec.Code, rec.Body)
	}
}

// TestSubmitBrokerCredRosterUpsert roster upsert 同事务桩：cex 提交后
// roster 行在（cred_ns 推导+state=connecting+roles={collect}）。
func TestSubmitBrokerCredRosterUpsert(t *testing.T) {
	brokerMk(t) // Seal master key 桩（同 VerifyThenWrite 先例）
	mux, _, pool, ctx := brokerFaceMux(t, nil,
		func(context.Context, signutil.Credentials) (map[string]float64, error) {
			return map[string]float64{"USDT": 1.0}, nil
		})
	defer pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue='okx2'`)
	defer pool.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE 'okx2%'`)
	rec := subPost(t, mux, `{"venue":"okx2","proto":"cex","api_key":"k","api_secret":"s","passphrase":"p"}`, "http://ui.local")
	if rec.Code != 200 {
		t.Fatalf("submit: %d %s", rec.Code, rec.Body)
	}
	var credNS, state string
	var roles []string
	err := pool.QueryRow(ctx, `SELECT cred_ns,state,roles FROM fin_venue_roster WHERE venue='okx2'`).Scan(&credNS, &state, &roles)
	if err != nil {
		t.Fatalf("roster 行缺: %v", err)
	}
	if credNS != "okx2.cex" {
		t.Fatalf("cred_ns=%s 应=okx2.cex", credNS)
	}
	if state != "connecting" {
		t.Fatalf("state=%s 应=connecting", state)
	}
	if len(roles) != 1 || roles[0] != "collect" {
		t.Fatalf("roles=%v 应={collect}", roles)
	}
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets WHERE key LIKE 'okx2.cex.%'`).Scan(&n)
	if n != 3 {
		t.Fatalf("凭据键应 3 件余 %d", n)
	}
}

// TestBrokerFaceRosterPrimary 读面主源化桩：roster 行直出（login 列免
// 解密/state=roster 态）+cred_ns 缺件→phase=cred_missing。
func TestBrokerFaceRosterPrimary(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t, nil, nil)
	defer pool.Exec(ctx, `DELETE FROM fin_venue_roster WHERE venue='tstface'`)
	// 名册行 cred_ns 三键全缺→cred_missing。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_roster(venue,proto,cred_ns,login,roles,state)
		VALUES('tstface','cex','tstface.cex',NULL,'{collect}','active')`); err != nil {
		t.Fatalf("roster 造行: %v", err)
	}
	rec := listGet(t, mux)
	if rec.Code != 200 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"venue":"tstface"`) ||
		!strings.Contains(rec.Body.String(), `"phase":"cred_missing"`) {
		t.Fatalf("cred_missing 显形缺: %s", rec.Body)
	}
	// 补齐凭据三键→phase=verified（active 生命周期映射）。
	for _, f := range []string{"api_key", "api_secret", "passphrase"} {
		if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,'\\x00')`, "tstface.cex."+f); err != nil {
			t.Fatalf("凭据造行 %s: %v", f, err)
		}
	}
	defer pool.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE 'tstface%'`)
	rec = listGet(t, mux)
	if !strings.Contains(rec.Body.String(), `"phase":"verified"`) {
		t.Fatalf("active+凭据齐应 verified: %s", rec.Body)
	}
}
