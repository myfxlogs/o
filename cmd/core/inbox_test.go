package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 投件道:multipart 落 inbox 目录+名净(剥径/怪字符/藏点)+回包落径。
func TestInboxUpload(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("O_INBOX_DIR", dir)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", `../../etc/邪 恶.png`)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("fake-png-bytes"))
	mw.Close()

	mux := http.NewServeMux()
	mountInbox(mux, nil)
	req := httptest.NewRequest("POST", "/inbox", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("投件败: code=%d body=%s", rec.Code, rec.Body.String())
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 {
		t.Fatalf("落件数错: %d", len(ents))
	}
	got, _ := os.ReadFile(filepath.Join(dir, ents[0].Name()))
	if string(got) != "fake-png-bytes" {
		t.Fatalf("内容串: %q", got)
	}
	if strings.Contains(ents[0].Name(), "/") || strings.HasPrefix(ents[0].Name(), ".") {
		t.Fatalf("名净失守: %q", ents[0].Name())
	}
}

// 非 multipart/缺 file 域=400 不白吞。
func TestInboxBadRequest(t *testing.T) {
	mux := http.NewServeMux()
	mountInbox(mux, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/inbox", strings.NewReader("x")))
	if rec.Code == http.StatusOK {
		t.Fatalf("坏形放行: %d", rec.Code)
	}
}
