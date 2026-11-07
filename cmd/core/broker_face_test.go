// broker_face_test.go —— T-0407/T-0430 桩测：①Origin 同源闸（缺席/跨源=
// 403）；②形闸（venue 坏形/proto∉mt5|cex 两值〔mt4=拒——业主令 2026-11-07〕/
// cex 册外 venue/cex 缺件/login 非数字/password·server 空=400 且零凭据落库
// 零审计行）；③红桩 verify-then-write（假 Session connect 败/AccountSummary
// 败/cex 只读端败=400 即回携原因——app_secrets 零行落零审计行）；④真形桩
// （fake Session 喂真 connect 回包→<broker>.mt5.* 四件密文落〔master key 回
// 解=原值〕+行态 verified+审计 esc_scope——password 永禁入注）；⑤CEX 桩
// （cexBalance 桩返 totalEq→<venue>.cex.* 三件落+verified+凭据件零出面）；
// ⑥mt4 提交拒+存量 .mt4. 键读面兼容显形；⑦phase 三源派生（败径>管道态>
// 提交审计；cex 落库即 verified；无凭据台账行 phase 空）；⑧brokerCredKeySplit
// 键切分（单例 mt5.*/<b>.mt5.*/<v>.cex.*/<b>.mt4.*/存量 okx.*/坏形）。
// 全走 testutil 独立 schema+connecthttp 真挂载——桩测注假在 seam 注（fake
// Session/cexBalance），禁桩真 venue；不假数据不打生产。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/signutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
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
const goodBody = `{"venue":"exness","proto":"mt5","login":"3001157","password":"pw-桩","server":"Exness-MT5Trial9","host":"mt5-demo.roboforex.com:443","note":"业主手填桩"}`

// fakeSession 桩验连会话（真形回包喂入——Connect/AccountSummary 可编程；
// mt5VerifySession 窄面注假，禁桩真 venue）。
type fakeSession struct {
	connErr     error
	sumErr      error
	eq, fm, bal float64
}

func (f *fakeSession) Connect(_ context.Context, _ mt5.Credentials) error { return f.connErr }
func (f *fakeSession) AccountSummary(context.Context) (float64, float64, float64, error) {
	if f.sumErr != nil {
		return 0, 0, 0, f.sumErr
	}
	return f.eq, f.fm, f.bal, nil
}
func (f *fakeSession) Disconnect(context.Context) {}
func (f *fakeSession) Close()                     {}

// brokerFaceMux 带验连缝挂载（earnFaceMux 同构——fake Session/cexBalance 注
// 假面，nil 缝=生产径本测试不用）。
func brokerFaceMux(t *testing.T, dial func(string) (mt5VerifySession, error),
	cexBal func(context.Context, signutil.Credentials) (map[string]float64, error),
) (http.Handler, *config.Client, *pgxpool.Pool, context.Context) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, coreService{
		pool: pool, cfg: cfg, dialSession: dial, cexBalance: cexBal})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	return mux, cfg, pool, ctx
}

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

// listGet 读面 POST（connect unary JSON 空体——同枚读形）。
func listGet(t *testing.T, mux http.Handler) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, listProc, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSubmitBrokerCredOriginGate(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t,
		func(string) (mt5VerifySession, error) { return &fakeSession{eq: 1, bal: 1}, nil }, nil)
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
	mux, _, pool, ctx := brokerFaceMux(t,
		func(string) (mt5VerifySession, error) { return &fakeSession{eq: 1, bal: 1}, nil }, nil)
	brokerMk(t)
	bad := []struct{ name, body string }{
		{"venue 大写", `{"venue":"Exness","login":"1","password":"p","server":"s"}`},
		{"venue 含点", `{"venue":"ex.ness","login":"1","password":"p","server":"s"}`},
		{"venue 含冒号", `{"venue":"ex:ness","login":"1","password":"p","server":"s"}`},
		{"proto mt4 拒（业主令不处理）", `{"venue":"fxmt4","proto":"mt4","login":"1","password":"p","server":"s"}`},
		{"proto 非二值", `{"venue":"exness","proto":"fix","login":"1","password":"p","server":"s"}`},
		{"login 非数字", `{"venue":"exness","login":"abc","password":"p","server":"s"}`},
		{"password 空", `{"venue":"exness","login":"1","password":"","server":"s"}`},
		{"server 空", `{"venue":"exness","login":"1","password":"p","server":""}`},
		{"cex venue 册外", `{"venue":"binance","proto":"cex","api_key":"k","api_secret":"s","passphrase":"p"}`},
		{"cex 缺件", `{"venue":"okx","proto":"cex","api_key":"k"}`},
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

// TestSubmitBrokerCredVerifyFailRedStake 红桩（验收首锚——验败零落表）：
// MT5 假凭据（connect 败/AccountSummary 败）+CEX 假 key（只读端败）→400
// 即回携原因+app_secrets 零行落+审计行零。
func TestSubmitBrokerCredVerifyFailRedStake(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		dial  func(string) (mt5VerifySession, error)
		cexB  func(context.Context, signutil.Credentials) (map[string]float64, error)
		whyKr string
	}{
		{"MT5 connect 败", goodBody,
			func(string) (mt5VerifySession, error) {
				return &fakeSession{connErr: errors.New("MT5 invalid account")}, nil
			}, nil, "MT5 invalid account"},
		{"MT5 AccountSummary 败", goodBody,
			func(string) (mt5VerifySession, error) {
				return &fakeSession{sumErr: errors.New("summary 空体")}, nil
			}, nil, "summary 空体"},
		{"CEX 假 key", `{"venue":"okx","proto":"cex","api_key":"k","api_secret":"s","passphrase":"p"}`,
			nil,
			func(context.Context, signutil.Credentials) (map[string]float64, error) {
				return nil, errors.New("50111 权限不足")
			}, "50111"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux, _, pool, ctx := brokerFaceMux(t, tc.dial, tc.cexB)
			brokerMk(t)
			rec := subPost(t, mux, tc.body, "http://ui.local")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("验败应 400: code=%d body=%s", rec.Code, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), "验连败") || !strings.Contains(rec.Body.String(), tc.whyKr) {
				t.Fatalf("即回应携原因: %s", rec.Body)
			}
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets`).Scan(&n); err != nil || n != 0 {
				t.Fatalf("验败零落表: n=%d err=%v", n, err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='venue凭据提交'`).Scan(&n); err != nil || n != 0 {
				t.Fatalf("验败零审计行: n=%d err=%v", n, err)
			}
		})
	}
}

// TestSubmitBrokerCredVerifyThenWrite 真形桩：fake Session 喂真 connect 回包
// →<broker>.mt5.* 四件密文落（回解=原值）+审计 esc_scope+行态 verified。
func TestSubmitBrokerCredVerifyThenWrite(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t,
		func(string) (mt5VerifySession, error) { return &fakeSession{eq: 1000, fm: 900, bal: 1000}, nil }, nil)
	mk := brokerMk(t)
	rec := subPost(t, mux, goodBody, "http://ui.local")
	if rec.Code != http.StatusOK {
		t.Fatalf("同源提交应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var out struct {
		Venue         string `json:"venue"`
		FieldsWritten int    `json:"fieldsWritten"`
		Verified      bool   `json:"verified"`
		VerifyNote    string `json:"verifyNote"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("回包解析败: %v body=%s", err, rec.Body)
	}
	if out.Venue != "exness" || out.FieldsWritten != 4 || !out.Verified ||
		!strings.Contains(out.VerifyNote, "eq=1000.00") {
		t.Fatalf("回包应 venue=exness fields=4 verified eq=1000: %+v", out)
	}
	// 四件密文落库——回解=原值；password 永不明文。
	want := map[string]string{
		"exness.mt5.login":    "3001157",
		"exness.mt5.password": "pw-桩",
		"exness.mt5.server":   "Exness-MT5Trial9",
		"exness.mt5.host":     "mt5-demo.roboforex.com:443",
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
	// 审计行——esc_scope=venue（phase 派生源）+venue/login/verify 摘要在注，
	// password 永禁入注。
	var note, escScope string
	if err := pool.QueryRow(ctx, `SELECT note,esc_scope FROM ops_event WHERE event='venue凭据提交'`).Scan(&note, &escScope); err != nil {
		t.Fatalf("审计行缺: %v", err)
	}
	if escScope != "exness" || !strings.Contains(note, "login=3001157") || !strings.Contains(note, "connect ok") {
		t.Fatalf("审计 note 应携 esc_scope+login+verify 摘要: scope=%q note=%q", escScope, note)
	}
	if strings.Contains(note, "pw-桩") {
		t.Fatalf("审计 note 禁含 password: %q", note)
	}
	// 行态 verified——读面 phase 派生（提交审计在册→verified）。
	lrec := listGet(t, mux)
	var list struct {
		Rows []struct {
			Venue string `json:"venue"`
			Phase string `json:"phase"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(lrec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	var ph string
	for _, r := range list.Rows {
		if r.Venue == "exness" {
			ph = r.Phase
		}
	}
	if ph != "verified" {
		t.Fatalf("行态应 verified: %q", ph)
	}
}

// TestSubmitBrokerCredCEX CEX 桩：cexBalance 桩返 totalEq→<venue>.cex.* 三件
// 落+verified；api_key 三件零出面（响应/审计零回显）。
func TestSubmitBrokerCredCEX(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t, nil,
		func(context.Context, signutil.Credentials) (map[string]float64, error) {
			return map[string]float64{"USDT": 7.89, "BTC": 0}, nil
		})
	mk := brokerMk(t)
	rec := subPost(t, mux,
		`{"venue":"okx","proto":"cex","api_key":"ak-桩","api_secret":"as-桩","passphrase":"pp-桩","note":"cex 桩"}`,
		"http://ui.local")
	if rec.Code != http.StatusOK {
		t.Fatalf("cex 提交应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var out struct {
		Venue         string `json:"venue"`
		FieldsWritten int    `json:"fieldsWritten"`
		Verified      bool   `json:"verified"`
		VerifyNote    string `json:"verifyNote"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Venue != "okx" || out.FieldsWritten != 3 || !out.Verified ||
		!strings.Contains(out.VerifyNote, "totalEq=7.89") {
		t.Fatalf("cex 回包应 fields=3 verified totalEq=7.89: %+v", out)
	}
	want := map[string]string{
		"okx.cex.api_key":    "ak-桩",
		"okx.cex.api_secret": "as-桩",
		"okx.cex.passphrase": "pp-桩",
	}
	for k, w := range want {
		var ct []byte
		if err := pool.QueryRow(ctx, `SELECT ciphertext FROM app_secrets WHERE key=$1`, k).Scan(&ct); err != nil {
			t.Fatalf("%s 行缺: %v", k, err)
		}
		if pt, err := cryptoutil.Open([]byte(mk), ct); err != nil || pt != w {
			t.Fatalf("%s 回解应 %q: pt=%q err=%v", k, w, pt, err)
		}
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event WHERE event='venue凭据提交'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "proto=cex") || !strings.Contains(note, "totalEq=7.89") {
		t.Fatalf("cex 审计注应携 proto+摘要: %q", note)
	}
	for _, secret := range []string{"ak-桩", "as-桩", "pp-桩"} {
		if strings.Contains(note, secret) {
			t.Fatalf("审计注禁含凭据件 %q: %q", secret, note)
		}
		if strings.Contains(rec.Body.String(), secret) {
			t.Fatalf("响应禁回显凭据件 %q: %s", secret, rec.Body)
		}
	}
	// mt5 族零行——cex 提交不串族。
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets WHERE key LIKE 'okx.mt5.%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("cex 提交禁串 mt5 族: n=%d", n)
	}
	// 行态 verified——cex 落库即 verified（提交门禁构造保证）。
	lrec := listGet(t, mux)
	var list struct {
		Rows []struct {
			Venue string `json:"venue"`
			Proto string `json:"proto"`
			Phase string `json:"phase"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(lrec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	for _, r := range list.Rows {
		if r.Venue == "okx" && (r.Proto != "cex" || r.Phase != "verified") {
			t.Fatalf("okx 行应 cex/verified: %+v", r)
		}
	}
}

// TestSubmitBrokerCredMT4Reject mt4 提交闸拒（业主令不处理）+存量 .mt4. 键
// 读面兼容显形（在册如实显形不假冒接入态）。
func TestSubmitBrokerCredMT4Reject(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t,
		func(string) (mt5VerifySession, error) { return &fakeSession{eq: 1, bal: 1}, nil }, nil)
	mk := brokerMk(t)
	if rec := subPost(t, mux,
		`{"venue":"fxmt4","proto":"mt4","login":"777","password":"p4","server":"FX-MT4"}`,
		"http://ui.local"); rec.Code != http.StatusBadRequest {
		t.Fatalf("mt4 提交应 400（业主令不处理）: code=%d body=%s", rec.Code, rec.Body)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM app_secrets`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("mt4 拒后应零行: n=%d", n)
	}
	// 存量 .mt4. 键读面兼容——直插密文行（SQL 径存量形），列表如实显 proto=mt4。
	for f, v := range map[string]string{"login": "777", "password": "p4", "server": "FX-MT4"} {
		ct, err := cryptoutil.Seal([]byte(mk), v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)`,
			"fxmt4.mt4."+f, ct); err != nil {
			t.Fatal(err)
		}
	}
	lrec := listGet(t, mux)
	if lrec.Code != http.StatusOK {
		t.Fatalf("读面应 200: %d %s", lrec.Code, lrec.Body)
	}
	var list struct {
		Rows []struct {
			Venue string `json:"venue"`
			Proto string `json:"proto"`
			Phase string `json:"phase"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(lrec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range list.Rows {
		if r.Venue == "fxmt4" {
			found = true
			if r.Proto != "mt4" || r.Phase != "" {
				t.Fatalf("mt4 遗留行应 proto=mt4 phase 空（链未接显形）: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("fxmt4 遗留行缺（读面兼容）: %+v", list.Rows)
	}
}

// TestListBrokerOnboardPhase phase 三源派生对拍：管道已过探测=verified；
// 台账 registered 无审计=connecting；仅提交审计=verified；末拍败径=failed
// 携败因注；cex 落库即 verified；无凭据台账行 phase 空 raw 态；存量 okx.*
// 遗留键并入 cex 族。
func TestListBrokerOnboardPhase(t *testing.T) {
	mux, _, pool, ctx := brokerFaceMux(t,
		func(string) (mt5VerifySession, error) { return &fakeSession{eq: 1, bal: 1}, nil }, nil)
	mk := brokerMk(t)
	seedCred := func(venue string) {
		t.Helper()
		ct, err := cryptoutil.Seal([]byte(mk), "3001157")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)`,
			venue+".mt5.login", ct); err != nil {
			t.Fatal(err)
		}
	}
	// 管道 verified：probed 态。
	seedCred("exness")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state)
		VALUES('exness','cfd-broker','probed')`); err != nil {
		t.Fatal(err)
	}
	// connecting：registered 态无审计无拍。
	seedCred("waitbro")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state)
		VALUES('waitbro','cfd-broker','registered')`); err != nil {
		t.Fatal(err)
	}
	// failed：registered+末拍败径。
	seedCred("badbro")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state)
		VALUES('badbro','cfd-broker','registered')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('接市官','venue接入','T-0379','beat=探测 key=badbro:9 败径 fail-closed：凭据拒','接入败','badbro:9')`); err != nil {
		t.Fatal(err)
	}
	// verified：仅提交审计（凭据落库台账候发现拍）。
	seedCred("newbro")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_scope)
		VALUES('dashboard-ui','venue凭据提交','T-0407','venue=newbro proto=mt5 connect ok eq=1','newbro')`); err != nil {
		t.Fatal(err)
	}
	// 无凭据台账行（scout 榜 venue）——phase 空 raw 态。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state)
		VALUES('ghostex','cex-perp','candidate')`); err != nil {
		t.Fatal(err)
	}
	// 存量 okx.* 遗留键——cex 族并入同表。
	for f, v := range map[string]string{"api_key": "ak", "api_secret": "as", "passphrase": "pp"} {
		ct, err := cryptoutil.Seal([]byte(mk), v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)`, "okx."+f, ct); err != nil {
			t.Fatal(err)
		}
	}
	lrec := listGet(t, mux)
	if lrec.Code != http.StatusOK {
		t.Fatalf("读面应 200: %d %s", lrec.Code, lrec.Body)
	}
	var list struct {
		Rows []struct {
			Venue      string `json:"venue"`
			Proto      string `json:"proto"`
			State      string `json:"state"`
			Phase      string `json:"phase"`
			BeatNote   string `json:"beatNote"`
			BeatFailed bool   `json:"beatFailed"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(lrec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	type row struct {
		Proto, State, Phase, BeatNote string
		BeatFailed                    bool
	}
	byVenue := map[string]row{}
	for _, r := range list.Rows {
		byVenue[r.Venue] = row{r.Proto, r.State, r.Phase, r.BeatNote, r.BeatFailed}
	}
	if r := byVenue["exness"]; r.Proto != "mt5" || r.Phase != "verified" {
		t.Fatalf("exness 应 mt5/verified: %+v", r)
	}
	if r := byVenue["waitbro"]; r.Phase != "connecting" {
		t.Fatalf("waitbro 应 connecting: %+v", r)
	}
	if r := byVenue["badbro"]; r.Phase != "failed" || !r.BeatFailed || !strings.Contains(r.BeatNote, "凭据拒") {
		t.Fatalf("badbro 应 failed 携败因: %+v", r)
	}
	if r := byVenue["newbro"]; r.Phase != "verified" {
		t.Fatalf("newbro 应 verified（提交审计）: %+v", r)
	}
	if r := byVenue["ghostex"]; r.Phase != "" || r.State != "candidate" {
		t.Fatalf("ghostex 应 phase 空 raw 态: %+v", r)
	}
	if r := byVenue["okx"]; r.Proto != "cex" || r.Phase != "verified" {
		t.Fatalf("okx 遗留应 cex/verified: %+v", r)
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
		{"okx.cex.api_key", "okx", "cex", "api_key", true},
		{"okx.cex.passphrase", "okx", "cex", "passphrase", true},
		{"okx.api_key", "okx", "cex", "api_key", true},
		{"okx.api_secret", "okx", "cex", "api_secret", true},
		{"okx.passphrase", "okx", "cex", "passphrase", true},
		{"fx.mt4.server", "fx", "mt4", "server", true},
		{"Bad.mt5.login", "", "", "", false},
		{"ex:ness.mt5.login", "", "", "", false},
		{".mt5.login", "", "", "", false},
		{"exness.mt5.", "", "", "", false},
		{"exness.cex.", "", "", "", false},
		{"okx.login", "", "", "", false},
		{"ibkr.paper_user", "", "", "", false},
	}
	for _, c := range cases {
		v, p, f, ok := brokerCredKeySplit(c.k)
		if ok != c.ok || (ok && (v != c.venue || p != c.proto || f != c.field)) {
			t.Fatalf("%s 切分=%v,%v,%v,%v 应 %v,%v,%v,%v", c.k, v, p, f, ok, c.venue, c.proto, c.field, c.ok)
		}
	}
}
