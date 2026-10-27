// readcred.go —— T-0170 认证读面 SG 端（径 A 终裁）：只读钥解密专口。
// app_secrets 唯一真源（okx_read.* 密文列，与交易钥同店不同架）→ master key
// 解密 → JSON 直出（仅内存）。专口 :8095——凭据面与 50051 UI/API 面隔离，
// host 侧 loopback pin 由 compose ports 承担（同 50051 形）；dev 隧道 5545
// boot 取，零磁盘副本。缺件/解密败=503 fail-closed 不吐部分物料。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

const readcredAddr = ":8095"

// startReadcred readcred 专口发射（独立 server 不挂 50051 总线；errc 收
// bind 败——面缺=臂 401 fail-closed 让位自愈，不炸 boot；随 ctx 收停）。
func startReadcred(ctx context.Context, errc chan<- error, pool *pgxpool.Pool) {
	srv := &http.Server{Addr: readcredAddr, Handler: mountReadcred(pool), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- fmt.Errorf("readcred 专口: %w", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
}

// mountReadcred readcred 专口路由（独立 mux——不挂 50051 总线面）。
func mountReadcred(pool *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/readcred/okx", readcredOKX(pool))
	return mux
}

// readcredOKX GET /v1/readcred/okx——okx_read 三件解密直出（缺一=503）。
func readcredOKX(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mk, err := config.ReadMasterKey()
		if err != nil {
			http.Error(w, "readcred: master key 缺——fail-closed", http.StatusServiceUnavailable)
			return
		}
		var cts [3][]byte
		for i, k := range []string{"okx_read.api_key", "okx_read.api_secret", "okx_read.passphrase"} {
			if err := pool.QueryRow(r.Context(),
				"SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&cts[i]); err != nil {
				http.Error(w, "readcred: 钥物料缺——fail-closed", http.StatusServiceUnavailable)
				return
			}
		}
		key, e1 := cryptoutil.Open(mk, cts[0])
		sec, e2 := cryptoutil.Open(mk, cts[1])
		pp, e3 := cryptoutil.Open(mk, cts[2])
		if e1 != nil || e2 != nil || e3 != nil {
			http.Error(w, "readcred: 解密败——fail-closed", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"api_key": key, "api_secret": sec, "passphrase": pp})
	}
}
