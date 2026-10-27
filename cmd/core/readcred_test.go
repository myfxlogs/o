// readcred_test.go —— T-0170/T-0174 桩锚：readcred 专口 venue-map 面——
// 行族枚举+白名单控面（伪所加行零代码入包=验收锚「新所上架径=两件加行」
// SG 半边）/venue 残缺缺席不吐部分物料/解密败·白名单缺·主钥缺 503
// fail-closed/旧 venue 段路径 404（直接切）。testutil 独立 schema 不打生产。
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func readcredSetup(t *testing.T) (*pgxpool.Pool, *config.Client) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	mk := []byte("0123456789abcdef0123456789abcdef") // 32B 原样用（cryptoutil 归一）
	kf := filepath.Join(t.TempDir(), "mk")
	if err := os.WriteFile(kf, mk, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("O_MASTER_KEY_FILE", kf)
	if _, err := pool.Exec(ctx, `TRUNCATE app_secrets`); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"okx_read.api_key":    "rk-test",
		"okx_read.api_secret": "rs-test",
		"okx_read.passphrase": "rp-test",
		"okx.api_key":         "noise", // 同店不同架——无 _read. 锚，结构上不入枚举面
	} {
		seedSecret(t, pool, k, v, mk)
	}
	if err := setEntitlement(t, pool, []string{"okx"}); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return pool, cfg
}

func seedSecret(t *testing.T, pool *pgxpool.Pool, k, v string, mk []byte) {
	t.Helper()
	ct, err := cryptoutil.Seal(mk, v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2) ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext", k, ct); err != nil {
		t.Fatal(err)
	}
}

// setEntitlement 白名单行改（fin_0075 种子行在位——测试覆写后须 cfg.Load 刷快照）。
func setEntitlement(t *testing.T, pool *pgxpool.Pool, venues []string) error {
	t.Helper()
	v, err := json.Marshal(venues)
	if err != nil {
		return err
	}
	_, err = pool.Exec(context.Background(),
		"INSERT INTO app_config(key,value) VALUES('readcred.entitlement',$1::jsonb) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", v)
	return err
}

func readcredGet(t *testing.T, pool *pgxpool.Pool, cfg *config.Client, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mountReadcred(pool, cfg).ServeHTTP(rec, req)
	return rec
}

// 锚①：三件齐+白名单→200 venue-map 直出（解密回读一致/no-store；交易钥
// noise 行不入面；旧 venue 段路径 404——直接切实证）。
func TestReadcredMapSealed(t *testing.T) {
	pool, cfg := readcredSetup(t)
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control=%q", rec.Header().Get("Cache-Control"))
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Fatalf("venue 面应恰 okx（noise 行不入）: %v", m)
	}
	for k, want := range map[string]string{"api_key": "rk-test", "api_secret": "rs-test", "passphrase": "rp-test"} {
		if m["okx"][k] != want {
			t.Fatalf("okx.%s=%q want %q", k, m["okx"][k], want)
		}
	}
	if len(m["okx"]) != 3 {
		t.Fatalf("okx 多余键面: %v", m["okx"])
	}
	if rec := readcredGet(t, pool, cfg, "/v1/readcred/okx"); rec.Code != http.StatusNotFound {
		t.Fatalf("旧路径应 404（直接切）, status=%d", rec.Code)
	}
}

// 桩锚/验收锚：伪所 bnb_read.* 加行+白名单加元素→回包含新所件，零代码
// （新所上架径=「app_secrets 加行+白名单加行」两件，dev 零 env 变更）。
func TestReadcredMapNewVenueByRowsOnly(t *testing.T) {
	pool, cfg := readcredSetup(t)
	ctx := context.Background()
	mk := []byte("0123456789abcdef0123456789abcdef")
	for k, v := range map[string]string{
		"bnb_read.api_key":    "bk-test",
		"bnb_read.api_secret": "bs-test",
		"bnb_read.passphrase": "bp-test",
	} {
		seedSecret(t, pool, k, v, mk)
	}
	if err := setEntitlement(t, pool, []string{"okx", "bnb"}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["bnb"]["api_key"] != "bk-test" || m["bnb"]["api_secret"] != "bs-test" || m["bnb"]["passphrase"] != "bp-test" {
		t.Fatalf("bnb 件缺或错: %v", m["bnb"])
	}
}

// 锚②：行族在而白名单未授权→该所缺席（entitlement 控出面——凭据在库不等于出面）。
func TestReadcredMapEntitlementGates(t *testing.T) {
	pool, cfg := readcredSetup(t)
	ctx := context.Background()
	mk := []byte("0123456789abcdef0123456789abcdef")
	seedSecret(t, pool, "bnb_read.api_key", "bk-test", mk)
	// bnb 未入白名单→缺席；空数组=显式空面。
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	var m map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["bnb"]; ok {
		t.Fatalf("未授权所不应出面: %v", m)
	}
	if _, ok := m["okx"]; !ok {
		t.Fatal("授权所应在面")
	}
	if err := setEntitlement(t, pool, []string{}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var empty map[string]map[string]string // 新 map 承接——Unmarshal 复用非空 map 不清底
	rec = readcredGet(t, pool, cfg, "/v1/readcred")
	if err := json.Unmarshal(rec.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("显式空面应 {}： %v", empty)
	}
}

// 锚③：行族残缺（删一钥）→如实透传残面（SG 无 venue 凭据形面知识，加行
// 零代码语义不可破）——形面缺件由消费臂 fail-closed（research 端 venueCreds
// 缺件锚对拍：残面进不了任何臂）。
func TestReadcredMapPartialFamilyPassthrough(t *testing.T) {
	pool, cfg := readcredSetup(t)
	if _, err := pool.Exec(context.Background(), "DELETE FROM app_secrets WHERE key='okx_read.passphrase'"); err != nil {
		t.Fatal(err)
	}
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if len(m["okx"]) != 2 || m["okx"]["passphrase"] != "" {
		t.Fatalf("残缺所应如实两件透传: %v", m["okx"])
	}
}

// 锚④：解密败（密文坏）→503 系统性 fail-closed（禁 200 空 map 冒充在位）。
func TestReadcredMapBadCiphertext(t *testing.T) {
	pool, cfg := readcredSetup(t)
	if _, err := pool.Exec(context.Background(),
		"UPDATE app_secrets SET ciphertext=$1 WHERE key='okx_read.api_key'", []byte("garbage-not-sealed")); err != nil {
		t.Fatal(err)
	}
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 锚⑤：白名单键缺→503（键缺≠空面——fail-closed 不谎报在位）。
func TestReadcredMapEntitlementMissing(t *testing.T) {
	pool, cfg := readcredSetup(t)
	if _, err := pool.Exec(context.Background(), "DELETE FROM app_config WHERE key='readcred.entitlement'"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 锚⑥：主钥缺→503。
func TestReadcredMapNoMasterKey(t *testing.T) {
	pool, cfg := readcredSetup(t)
	t.Setenv("O_MASTER_KEY_FILE", filepath.Join(t.TempDir(), "absent"))
	rec := readcredGet(t, pool, cfg, "/v1/readcred")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 锚⑦：非 GET 拒（mux 方法钉——专口路由面）。
func TestReadcredRouterMethod(t *testing.T) {
	mux := mountReadcred(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/readcred", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}
