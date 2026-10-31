package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"o/web"
)

// T-0268 锚③：/assets/* vite hash 件 immutable 年长缓存；index.html（SPA 兜底
// 同径）no-cache 不吃长缓存——鉴别处理双断言。
func TestMountUICacheHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mountUI(mux)

	// index.html=no-cache（入口壳每回重验证——hash 资产发新版壳不陈）
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("index.html 缓存头错: %q", cc)
	}
	if rec.Code != http.StatusOK ||
		!strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("index.html 兜底败: code=%d ct=%s", rec.Code, rec.Header().Get("Content-Type"))
	}
	// 深链 SPA 兜底（如 /cockpit）同吃 no-cache——同径同规
	recD := httptest.NewRecorder()
	mux.ServeHTTP(recD, httptest.NewRequest("GET", "/cockpit", nil))
	if cc := recD.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("SPA 深链缓存头错: %q", cc)
	}

	// /assets/* 实件=immutable 年长缓存（指纹件——内容变即文件名变，长缓存不陈）
	ents, err := fs.Glob(web.Dist, "dist/assets/*")
	if err != nil || len(ents) == 0 {
		t.Fatalf("dist 资产件缺: %v n=%d", err, len(ents))
	}
	asset := ents[0]
	for _, e := range ents {
		if strings.HasSuffix(e, ".js") {
			asset = e
			break
		}
	}
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, httptest.NewRequest("GET", "/"+strings.TrimPrefix(asset, "dist/"), nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("资产件未达: %s code=%d", asset, rec2.Code)
	}
	cc := rec2.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") || !strings.Contains(cc, "max-age=31536000") {
		t.Fatalf("assets 缓存头错: %q", cc)
	}
}
