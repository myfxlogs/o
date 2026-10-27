// readcred.go —— T-0170 认证读面 SG 端（径 A 终裁）/ T-0174 venue 泛化：
// 只读钥解密专口。app_secrets 唯一真源（<venue>_read.* 行族）→ master key
// 解密 → venue-map JSON 直出（仅内存）。venue 集=数据非拓扑：行族按 '_read.'
// 锚枚举（加行即面扩，零代码）+ app_config readcred.entitlement 白名单控
// 出面（fin_0076 种子）。专口 :8095——凭据面与 50051 UI/API 面隔离，host 侧
// loopback pin 由 compose ports 承担（同 50051 形）；dev 隧道 5545 boot 取，
// 零磁盘副本。fail-closed 分级：主钥缺/白名单键缺或形坏/查库败/解密败=503
// （解密败系统性告警，禁 200 空 map 冒充在位）；行族残缺如实透传（SG 无
// venue 凭据形面知识，加行零代码语义不可破）——各臂自校验形面缺件让位。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

const readcredAddr = ":8095"

// readcredEntitlementKey app_config 白名单键（venue 字符串数组——fin_0076
// 种子行；键缺/形坏=503 fail-closed，空数组=显式出面为零）。
const readcredEntitlementKey = "readcred.entitlement"

// readcredRowAnchor 行族锚——app_secrets 键 <venue>_read.<field> 的切分锚
// （交易钥 okx.api_key 无此段，结构上不入枚举面）。
const readcredRowAnchor = "_read."

// startReadcred readcred 专口发射（独立 server 不挂 50051 总线；errc 收
// bind 败——面缺=臂 401 fail-closed 让位自愈，不炸 boot；随 ctx 收停）。
func startReadcred(ctx context.Context, errc chan<- error, pool *pgxpool.Pool, cfg *config.Client) {
	srv := &http.Server{Addr: readcredAddr, Handler: mountReadcred(pool, cfg), ReadHeaderTimeout: 10 * time.Second}
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
func mountReadcred(pool *pgxpool.Pool, cfg *config.Client) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/readcred", readcredMap(pool, cfg))
	return mux
}

// readcredMap GET /v1/readcred——venue-map 直出：白名单内各所 <venue>_read.*
// 行族解密成 {<venue>:{<field>:<明文>}}。
func readcredMap(pool *pgxpool.Pool, cfg *config.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mk, err := config.ReadMasterKey()
		if err != nil {
			http.Error(w, "readcred: master key 缺——fail-closed", http.StatusServiceUnavailable)
			return
		}
		allow, err := readcredEntitlement(cfg)
		if err != nil {
			http.Error(w, "readcred: 白名单缺——fail-closed", http.StatusServiceUnavailable)
			return
		}
		rows, err := pool.Query(r.Context(),
			"SELECT key, ciphertext FROM app_secrets WHERE strpos(key,$1)>0", readcredRowAnchor)
		if err != nil {
			http.Error(w, "readcred: 行族读败——fail-closed", http.StatusServiceUnavailable)
			return
		}
		defer rows.Close()
		family := map[string]map[string][]byte{}
		for rows.Next() {
			var k string
			var ct []byte
			if err := rows.Scan(&k, &ct); err != nil {
				http.Error(w, "readcred: 行族读败——fail-closed", http.StatusServiceUnavailable)
				return
			}
			venue, field, ok := strings.Cut(k, readcredRowAnchor)
			if !ok || venue == "" || field == "" || !allow[venue] {
				continue
			}
			if family[venue] == nil {
				family[venue] = map[string][]byte{}
			}
			family[venue][field] = ct
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "readcred: 行族读败——fail-closed", http.StatusServiceUnavailable)
			return
		}
		out := make(map[string]map[string]string, len(family))
		for venue, fields := range family {
			opened := make(map[string]string, len(fields))
			for field, ct := range fields {
				pt, err := cryptoutil.Open(mk, ct)
				if err != nil {
					http.Error(w, "readcred: 解密败——fail-closed", http.StatusServiceUnavailable)
					return
				}
				opened[field] = pt
			}
			out[venue] = opened
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// readcredEntitlement 白名单读（config 唯一入口按次读——键缺/形坏=错，
// 上层 503 fail-closed；空数组=合法显式空面）。
func readcredEntitlement(cfg *config.Client) (map[string]bool, error) {
	raw, err := cfg.Get(readcredEntitlementKey)
	if err != nil {
		return nil, err
	}
	var venues []string
	if err := json.Unmarshal(raw, &venues); err != nil {
		return nil, fmt.Errorf("%s: 非字符串数组形: %w", readcredEntitlementKey, err)
	}
	allow := make(map[string]bool, len(venues))
	for _, v := range venues {
		allow[v] = true
	}
	return allow, nil
}
