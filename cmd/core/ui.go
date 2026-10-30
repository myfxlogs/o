// ui.go —— 静态面挂接（自 service.go 拆出：favicon 挂接增量撑行，静态面自成件）。
package main

import (
	"io/fs"
	"net/http"
	"strings"

	"o/web"
)

// mountUI SPA 单页挂接（T-0170：四页单 SPA——路由在客户端，服务端兜底全捕
// 返 index.html；API 前缀 /o.dashboard. 不吃 HTML（错路 404 语义保留）；
// /trx 旧双入口路径自然收敛进 SPA 路由）。
func mountUI(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/o.dashboard.") {
			http.NotFound(w, r) // Connect API 错路——不回 HTML
			return
		}
		b, err := web.Dist.ReadFile("dist/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	// favicon 显式挂接（T-0269——此前 catch-all 把 index.html 当 favicon 回
	// 200 text/html；真件缺席=204 不回落 HTML）。
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		b, err := web.Dist.ReadFile("dist/favicon.ico")
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		_, _ = w.Write(b)
	})
	// assets 子树挂接（fs.Sub——embed FS 全件带 dist/ 前缀，指根=资产 404 病灶）。
	assets, _ := fs.Sub(web.Dist, "dist")
	mux.Handle("/assets/", http.FileServer(http.FS(assets)))
}
