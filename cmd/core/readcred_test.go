// readcred_test.go —— T-0170 桩锚：readcred 专口（真钥解密直出/缺件 503/
// 主钥缺 503——fail-closed 不吐部分物料）。testutil 独立 schema 不打生产。
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

	"o/internal/platform/cryptoutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func readcredSetup(t *testing.T) *pgxpool.Pool {
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
		"okx.api_key":         "noise", // 同店不同架——readcred 只吐 okx_read 面
	} {
		ct, err := cryptoutil.Seal(mk, v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2) ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext", k, ct); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func readcredGet(t *testing.T, pool *pgxpool.Pool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/readcred/okx", nil)
	rec := httptest.NewRecorder()
	readcredOKX(pool)(rec, req)
	return rec
}

// 锚①：三件齐→200 JSON 直出（解密回读一致；Cache-Control no-store）。
func TestReadcredOKXSealed(t *testing.T) {
	pool := readcredSetup(t)
	rec := readcredGet(t, pool)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control=%q", rec.Header().Get("Cache-Control"))
	}
	var v map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"api_key": "rk-test", "api_secret": "rs-test", "passphrase": "rp-test"} {
		if v[k] != want {
			t.Fatalf("%s=%q want %q", k, v[k], want)
		}
	}
	if len(v) != 3 {
		t.Fatalf("多余键面: %v", v)
	}
}

// 锚②：钥物料缺（删一钥）→503 不吐部分物料。
func TestReadcredOKXMissingFailClosed(t *testing.T) {
	pool := readcredSetup(t)
	if _, err := pool.Exec(context.Background(), "DELETE FROM app_secrets WHERE key='okx_read.passphrase'"); err != nil {
		t.Fatal(err)
	}
	rec := readcredGet(t, pool)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" {
		t.Fatal("503 空体——应带 fail-closed 缘由")
	}
}

// 锚③：主钥缺→503。
func TestReadcredOKXNoMasterKey(t *testing.T) {
	pool := readcredSetup(t)
	t.Setenv("O_MASTER_KEY_FILE", filepath.Join(t.TempDir(), "absent"))
	rec := readcredGet(t, pool)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 锚④：非 GET 拒（mux 方法钉——专口路由面）。
func TestReadcredRouterMethod(t *testing.T) {
	mux := mountReadcred(nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/readcred/okx", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}
