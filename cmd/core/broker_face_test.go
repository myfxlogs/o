// broker_face_test.go —— T-0407 桩测：①Origin 同源闸（缺席/跨源=403）；
// ②形闸（venue 坏形/proto 非二值/login 非数字/password·server 空=400 且
// 零凭据落库零审计行）；③同源提交→app_secrets <venue>.mt5.{login,password,
// server} 三行密文落（master key 回解=原值——零明文落库钉）+ops_event
// venue凭据提交 审计行（note 零 password 钉）；④ListBrokerOnboard=凭据族
// login 解密出面+台账态+末拍注记/败径+策略数并集（password 永不出面钉）；
// ⑤mt4 族=<venue>.mt4.* 命名空间+审计注明示链限；⑥brokerCredKeySplit
// 键切分（单例 mt5.*/<b>.mt5.*/<b>.mt4.*/坏形）。
// 全走 testutil 独立 schema+connecthttp 真挂载——不假数据不打生产。
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"o/internal/platform/cryptoutil"
)

// brokerMk 桩测 master key 落临时件+env 指（ReadMasterKey 正径——写面/读面
// 同一钥，跨测试 t.Setenv 隔离）。
func brokerMk(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "mk")
	if err := os.WriteFile(f, []byte("broker-face-test-master-key-0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("O_MASTER_KEY_FILE", f)
	return "broker-face-test-master-key-0123456789"
}

const subProc = "/o.dashboard.v1.CoreService/SubmitBrokerCred"
const listProc = "/o.dashboard.v1.CoreService/ListBrokerOnboard"
const goodBody = `{"venue":"exness","proto":"mt5","login":"3001157","password":"pw-桩","server":"Exness-MT5Trial9","note":"业主手填桩"}`

func subPost(t *testing.T, mux http.Handler, body, origin string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, subProc, strings.NewReader(body))
	req.Host = "ui.local"
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSubmitBrokerCredOriginGate(t *testing.T) {
	mux, _, pool, ctx := earnFaceMux(t)
	brokerMk(t)
	if rec := subPost(t, mux, goodBody, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("Origin 缺席应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	if rec := subPost(t, mux, goodBody, "https://evil.example"); rec.Code != http.StatusForbidden {
		t.Fatalf("跨源应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("全拒后凭据应零行: n=%d err=%v", n, err)
	}
}

func TestSubmitBrokerCredShapeGate(t *testing.T) {
	mux, _, pool, ctx := earnFaceMux(t)
	brokerMk(t)
	bad := []struct{ name, body string }{
		{"venue 大写", `{"venue":"Exness","login":"1","password":"p","server":"s"}`},
		{"venue 含点", `{"venue":"ex.ness","login":"1","password":"p","server":"s"}`},
		{"venue 含冒号", `{"venue":"ex:ness","login":"1","password":"p","server":"s"}`},
		{"proto 非二值", `{"venue":"exness","proto":"fix","login":"1","password":"p","server":"s"}`},
		{"login 非数字", `{"venue":"exness","login":"abc","password":"p","server":"s"}`},
		{"password 空", `{"venue":"exness","login":"1","password":"","server":"s"}`},
		{"server 空", `{"venue":"exness","login":"1","password":"p","server":""}`},
	}
	for _, b := range bad {
		if rec := subPost(t, mux, b.body, "http://ui.local"); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s 应 400: code=%d body=%s", b.name, rec.Code, rec.Body)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("形闸全拒后凭据应零行: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='venue凭据提交'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("形闸全拒后审计行应零: n=%d err=%v", n, err)
	}
}

func TestSubmitBrokerCredWriteAndList(t *testing.T) {
	mux, _, pool, ctx := earnFaceMux(t)
	mk := brokerMk(t)
	rec := subPost(t, mux, goodBody, "http://ui.local")
	if rec.Code != http.StatusOK {
		t.Fatalf("同源提交应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var out struct {
		Venue         string `json:"venue"`
		FieldsWritten int    `json:"fieldsWritten"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("回包解析败: %v body=%s", err, rec.Body)
	}
	if out.Venue != "exness" || out.FieldsWritten != 3 {
		t.Fatalf("回包应 venue=exness fields=3: %+v", out)
	}
	// 三键密文落库——回解=原值；password 永不明文。
	want := map[string]string{
		"exness.mt5.login":    "3001157",
		"exness.mt5.password": "pw-桩",
		"exness.mt5.server":   "Exness-MT5Trial9",
	}
	for k, w := range want {
		var ct []byte
		if err := pool.QueryRow(ctx, `SELECT ciphertext FROM app_secrets WHERE key=$1`, k).Scan(&ct); err != nil {
			t.Fatalf("%s 行缺: %v", k, err)
		}
		if strings.Contains(string(ct), w) {
			t.Fatalf("%s 密文含明文——Seal 失效", k)
		}
		if pt, err := cryptoutil.Open([]byte(mk), ct); err != nil || pt != w {
			t.Fatalf("%s 回解应 %q: pt=%q err=%v", k, w, pt, err)
		}
	}
	// 审计行——venue/login/server/备注在注，password 永禁入注。
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event WHERE event='venue凭据提交'`).Scan(&note); err != nil {
		t.Fatalf("审计行缺: %v", err)
	}
	if !strings.Contains(note, "venue=exness") || !strings.Contains(note, "备注=业主手填桩") {
		t.Fatalf("审计 note 应携 venue+备注: %q", note)
	}
	if strings.Contains(note, "pw-桩") {
		t.Fatalf("审计 note 禁含 password: %q", note)
	}
	// 台账+末拍+策略种子——读面并集。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state)
		VALUES('exness','cfd-broker','probed')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('接市官','venue接入','T-0379','beat=探测 key=exness:3001157 Connect ok','接入败','exness:3001157')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('EURUSD','tpl', 'paper', 100, 'exness'),('BTC-USDT-SWAP','tpl','retired',100,'exness')`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, listProc, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	lrec := httptest.NewRecorder()
	mux.ServeHTTP(lrec, req)
	if lrec.Code != http.StatusOK {
		t.Fatalf("读面应 200: code=%d body=%s", lrec.Code, lrec.Body)
	}
	var list struct {
		Rows []struct {
			Venue      string `json:"venue"`
			Proto      string `json:"proto"`
			Login      string `json:"login"`
			State      string `json:"state"`
			BeatNote   string `json:"beatNote"`
			BeatFailed bool   `json:"beatFailed"`
			Strategies string `json:"strategies"` // protojson int64=string 编码
		} `json:"rows"`
	}
	if err := json.Unmarshal(lrec.Body.Bytes(), &list); err != nil {
		t.Fatalf("读面包解析败: %v body=%s", err, lrec.Body)
	}
	var r *struct {
		Venue      string `json:"venue"`
		Proto      string `json:"proto"`
		Login      string `json:"login"`
		State      string `json:"state"`
		BeatNote   string `json:"beatNote"`
		BeatFailed bool   `json:"beatFailed"`
		Strategies string `json:"strategies"`
	}
	for i := range list.Rows {
		if list.Rows[i].Venue == "exness" {
			r = &list.Rows[i]
		}
	}
	if r == nil {
		t.Fatalf("exness 行缺（迁移种子行并存——凭据族应并集出面）: %+v", list.Rows)
	}
	if r.Proto != "mt5" || r.Login != "3001157" ||
		r.State != "probed" || !r.BeatFailed || !strings.Contains(r.BeatNote, "Connect ok") ||
		r.Strategies != "1" {
		t.Fatalf("行面不符: %+v", r)
	}
	if strings.Contains(lrec.Body.String(), "pw-桩") {
		t.Fatalf("读面禁含 password 明文: %s", lrec.Body)
	}
}

func TestSubmitBrokerCredMT4(t *testing.T) {
	mux, _, pool, ctx := earnFaceMux(t)
	brokerMk(t)
	rec := subPost(t, mux,
		`{"venue":"fxmt4","proto":"mt4","login":"777","password":"p4","server":"FX-MT4"}`, "http://ui.local")
	if rec.Code != http.StatusOK {
		t.Fatalf("mt4 提交应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets
		WHERE key LIKE 'fxmt4.mt4.%'`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("mt4 族应三键: n=%d err=%v", n, err)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event WHERE event='venue凭据提交'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "MT4 族凭据在册") {
		t.Fatalf("mt4 审计注应明示链限: %q", note)
	}
}

func TestBrokerCredKeySplit(t *testing.T) {
	cases := []struct {
		k            string
		venue, proto string
		field        string
		ok           bool
	}{
		{"mt5.login", "mt5", "mt5", "login", true},
		{"exness.mt5.password", "exness", "mt5", "password", true},
		{"fx.mt4.server", "fx", "mt4", "server", true},
		{"Bad.mt5.login", "", "", "", false},
		{"ex:ness.mt5.login", "", "", "", false},
		{".mt5.login", "", "", "", false},
		{"exness.mt5.", "", "", "", false},
		{"okx.api_key", "", "", "", false},
	}
	for _, c := range cases {
		v, p, f, ok := brokerCredKeySplit(c.k)
		if ok != c.ok || (ok && (v != c.venue || p != c.proto || f != c.field)) {
			t.Fatalf("%s 切分=%v,%v,%v,%v 应 %v,%v,%v,%v", c.k, v, p, f, ok, c.venue, c.proto, c.field, c.ok)
		}
	}
}
