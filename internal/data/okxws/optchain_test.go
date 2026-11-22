// optchain_test.go —— O-32540 期权链采集面桩测：
// ①目录解析+活跃子集判据（近N到期日×±20%行权价窗×族帽）
// ②订阅帧构造（opt-summary instFamily 键×4+tickers instId 键）
// ③帧解析（opt-summary/期权 tickers——空串→nil 标缺律）
// ④handle 分流（期权 instId 过闸/期权 ticker→optTick/opt-summary→optSum/
//
//	mark-price 期权形防御拒入/非期权形不误中）
//
// ⑤写节流（optSum 60s vs optTick 3s 分键分隙）
// ⑥落表+快照臂（DB 桩——O_TEST_PG_DSN 缺=skip）
// ⑦E2E 断线重订（fake WS 录订阅帧→断→重连全量重订含期权面）
package okxws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// optStubGet 期权目录桩读件——path 分派四族 instruments+两指数
// index-tickers；calls 计数验 TTL 命中零重读。
func optStubGet(calls *int, instRaw map[string]json.RawMessage, idxRaw map[string]json.RawMessage, err error) ScanGet {
	return func(ctx context.Context, path string) (json.RawMessage, error) {
		*calls++
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(path, "/api/v5/market/index-tickers") {
			for k, v := range idxRaw {
				if strings.HasSuffix(path, "instId="+k) { // 尾锚——BTC-USD 前缀不误中 BTC-USD_UM 同型
					return v, nil
				}
			}
			return json.RawMessage(`[]`), nil
		}
		for k, v := range instRaw {
			if strings.HasSuffix(path, "instFamily="+k) { // 尾锚——族名互为前缀须全等
				return v, nil
			}
		}
		return json.RawMessage(`[]`), nil
	}
}

// optInstJSON 期权 instruments 载荷行构造（测试件——字段形照 venue 实拍）。
func optInstJSON(fam, id, uly, otype, state string, expMs int64, stk string) string {
	return fmt.Sprintf(`{"instId":%q,"instFamily":%q,"uly":%q,"optType":%q,"expTime":%q,"stk":%q,"ctType":"linear","settleCcy":"USD","state":%q,"tickSz":"0.1","lotSz":"1","minSz":"1","ctVal":"1","ctValCcy":"BTC","listTime":"%d"}`,
		id, fam, uly, otype, fmt.Sprint(expMs), stk, state, expMs-86400000)
}

var optIdxRaw = map[string]json.RawMessage{
	"BTC-USD": json.RawMessage(`[{"instId":"BTC-USD","idxPx":"60000"}]`),
	"ETH-USD": json.RawMessage(`[{"instId":"ETH-USD","idxPx":"3000"}]`),
}

// optCat 四族目录桩——每族 2 到期日×4 行权价（近/中/远档）+1 非 live
// 行+1 窗外到期日行。行权价按族参照价定档：BTC 族 ref=60000 窗
// [48000,72000]、ETH 族 ref=3000 窗 [2400,3600]——每族 in/out 各半。
func optCat() map[string]json.RawMessage {
	near := time.Now().UTC().Add(10 * 24 * time.Hour).UnixMilli()
	far := time.Now().UTC().Add(40 * 24 * time.Hour).UnixMilli()
	out := time.Now().UTC().Add(120 * 24 * time.Hour).UnixMilli() // 窗外远月
	mk := func(fam, uly string, in1, in2, in3, out1 string) json.RawMessage {
		var b strings.Builder
		b.WriteString(`[`)
		rows := []string{
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(near), in1), uly, "C", "live", near, in1),
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-P", fam, yymmdd(near), in2), uly, "P", "live", near, in2),
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(near), in3), uly, "C", "live", near, in3),
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(near), out1), uly, "C", "live", near, out1), // 行权价出窗
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(far), in1), uly, "C", "live", far, in1),
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(out), in1), uly, "C", "live", out, in1),      // 到期日出窗
			optInstJSON(fam, fmt.Sprintf("%s-%s-%s-C", fam, yymmdd(near), in2), uly, "C", "suspend", near, in2), // 非 live
		}
		for i, r := range rows {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(r)
		}
		b.WriteString(`]`)
		return json.RawMessage(b.String())
	}
	return map[string]json.RawMessage{
		"BTC-USD":    mk("BTC-USD", "BTC-USD", "59000", "61000", "55000", "40000"),
		"BTC-USD_UM": mk("BTC-USD_UM", "BTC-USD", "59000", "61000", "55000", "40000"),
		"ETH-USD":    mk("ETH-USD", "ETH-USD", "2900", "3100", "2600", "1500"),
		"ETH-USD_UM": mk("ETH-USD_UM", "ETH-USD", "2900", "3100", "2600", "1500"),
	}
}

// yymmdd ms→YYMMDD（与 optInstRe 尾段同形——测试件自洽）。
func yymmdd(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("060102")
}

// TestOptChainRefreshParse 目录解析+TTL+活跃子集判据。
func TestOptChainRefreshParse(t *testing.T) {
	var calls int
	cat := NewOptChain(optStubGet(&calls, optCat(), optIdxRaw, nil), 0)
	ctx := context.Background()
	fresh, err := cat.Refresh(ctx)
	if err != nil || !fresh {
		t.Fatalf("首刷应 fresh: fresh=%v err=%v", fresh, err)
	}
	// TTL 命中零重读（calls 不增——6 次首刷=4 族 instruments+2 指数）
	callsAfter := calls
	fresh, err = cat.Refresh(ctx)
	if err != nil || fresh {
		t.Fatalf("TTL 内应命中 fresh=false: fresh=%v err=%v", fresh, err)
	}
	if calls != callsAfter {
		t.Fatalf("TTL 命中竟重读 venue: calls %d→%d", callsAfter, calls)
	}
	// 活跃子集：近 2 到期日（near+far——out 在 60d 窗外被 cutoff 断）
	// ×±20% 行权价窗——每族 in 窗 3+1=4 枚，out 档/出窗档/suspend 全滤。
	act := cat.Active()
	if len(act) != 16 {
		t.Fatalf("活跃集应=16（4 族×4 枚）: %d %v", len(act), act)
	}
	for _, id := range act {
		if !isOptInstID(id) {
			t.Fatalf("活跃集混入非期权形: %s", id)
		}
		if strings.Contains(id, "-40000-") || strings.Contains(id, "-1500-") ||
			strings.Contains(id, "-30000-") {
			t.Fatalf("窗外行权价误入活跃集: %s", id)
		}
	}
	joined := strings.Join(act, " ")
	for _, want := range []string{"BTC-USD-", "BTC-USD_UM-", "ETH-USD-", "ETH-USD_UM-",
		"59000", "61000", "2900", "3100"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("活跃集缺 %s: %v", want, act)
		}
	}
	// 四族均有代表（BTC-USD/BTC-USD_UM/ETH-USD/ETH-USD_UM 前缀分桶）
	famN := map[string]int{}
	for _, id := range act {
		for _, f := range OptFamilies {
			if strings.HasPrefix(id, f+"-") {
				famN[f]++
			}
		}
	}
	for _, f := range OptFamilies {
		if famN[f] == 0 {
			t.Fatalf("族 %s 活跃集零代表: %v", f, famN)
		}
	}
	// enum 全态行（suspend 行亦入 AllInstruments——快照全态语义）
	if n := len(cat.AllInstruments()); n < 28 {
		t.Fatalf("enum 行数异常偏少: %d", n)
	}
}

// TestOptChainSubscribeArgs 订阅帧构造——opt-summary×4 族键+
// tickers×活跃集 instId 键（分帧节流同 subscribeAll 律）。
func TestOptChainSubscribeArgs(t *testing.T) {
	var calls int
	cat := NewOptChain(optStubGet(&calls, optCat(), optIdxRaw, nil), 0)
	if _, err := cat.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	args := cat.SubscribeArgs()
	var sumN, tickN int
	for _, a := range args {
		switch a["channel"] {
		case "opt-summary":
			sumN++
			if a["instFamily"] == "" || a["instId"] != "" {
				t.Fatalf("opt-summary 键形错（须 instFamily 非 instId）: %v", a)
			}
		case "tickers":
			tickN++
			if !isOptInstID(a["instId"]) {
				t.Fatalf("tickers 混入非期权 instId: %v", a)
			}
		default:
			t.Fatalf("期权订阅面混入异频道: %v", a)
		}
	}
	if sumN != len(OptFamilies) {
		t.Fatalf("opt-summary 族数=%d 应=%d", sumN, len(OptFamilies))
	}
	if tickN != len(cat.Active()) {
		t.Fatalf("tickers args=%d ≠ 活跃集=%d", tickN, len(cat.Active()))
	}
}

// TestOptParseFrames 解析面——实拍字段形→行字段+空串标缺律。
func TestOptParseFrames(t *testing.T) {
	sum := `{"instId":"BTC-USD_UM-270129-86000-P","instType":"OPTION","uly":"BTC-USD","delta":"-0.94","gamma":"0.000008","vega":"28.37","theta":"-9.27","markVol":"0.478","bidVol":"0","askVol":"","realVol":"","volLv":"0.5","fwdPx":"60374","distance":"0.42","buyApr":"","sellApr":"0.2","lever":"0.00003","ts":"1795337568684"}`
	r := parseOptSummary(obj(json.RawMessage(sum)))
	if r == nil || r.instID != "BTC-USD_UM-270129-86000-P" {
		t.Fatalf("opt-summary 解析败: %+v", r)
	}
	if *r.delta != -0.94 || *r.markVol != 0.478 || *r.fwdPx != 60374 {
		t.Fatalf("opt-summary 数值面错: %+v", r)
	}
	if r.askVol != nil || r.buyApr != nil { // 空串→nil 标缺非 0
		t.Fatalf("空串字段应落 nil: askVol=%v buyApr=%v", r.askVol, r.buyApr)
	}
	if r.ts.UnixMilli() != 1795337568684 {
		t.Fatalf("ts 解析错: %v", r.ts)
	}
	tk := `{"instType":"OPTION","instId":"BTC-USD-261123-46000-C","last":"","lastSz":"0","askPx":"0.2285","askSz":"14","bidPx":"0.227","bidSz":"320","vol24h":"0","ts":"1795337389087"}`
	rt := parseOptTickers("BTC-USD-261123-46000-C", obj(json.RawMessage(tk)))
	if rt == nil || rt.last != nil { // last="" 无成交→nil
		t.Fatalf("期权 ticker last 空串应 nil: %+v", rt)
	}
	if *rt.bidPx != 0.227 || *rt.askSz != 14 {
		t.Fatalf("期权 ticker 数值面错: %+v", rt)
	}
	// instId 缺席行拒入
	if parseOptSummary(obj(json.RawMessage(`{"delta":"0.5"}`))) != nil {
		t.Fatal("无 instId 行应拒")
	}
}

// TestIsOptInstID 键域判据——期权形中/既有形零误中。
func TestIsOptInstID(t *testing.T) {
	for _, ok := range []string{
		"BTC-USD-261123-46000-C", "BTC-USD_UM-270129-86000-P",
		"ETH-USD-261125-3000-C", "SOL-USD-261123-100-P",
	} {
		if !isOptInstID(ok) {
			t.Fatalf("期权形漏中: %s", ok)
		}
	}
	for _, no := range []string{
		"BTC-USDT-SWAP", "BTC-USD", "BTC-USD-261225", "ETH-USDT",
		"BTC-USD-261123-46000-X", "BTC-USD-26112-46000-C",
	} {
		if isOptInstID(no) {
			t.Fatalf("非期权形误中: %s", no)
		}
	}
}

// newOptTestCollector handle 桩件（queue/passAt/rej 最小装配——
// handle 只走内存面不触 pool）。
func newOptTestCollector() *Collector {
	return &Collector{
		rej:       rejects.New("okxws", "non_usdt_instid", "parse_fail", "write_err", "univ_expand_err"),
		queue:     make(chan rowItem, 64),
		passAt:    map[string]time.Time{},
		refuseSub: map[string]bool{},
	}
}

// TestHandleOptFrames handle 分流——期权 ticker/opt-summary 过闸入队，
// mark-price 期权形防御拒入，非期权 USDT 形不受影响。
func TestHandleOptFrames(t *testing.T) {
	c := newOptTestCollector()
	// opt-summary 帧（arg.instFamily 键——instId 空过闸）
	c.handle([]byte(`{"arg":{"channel":"opt-summary","instFamily":"BTC-USD_UM"},"data":[` +
		`{"instId":"BTC-USD_UM-270129-86000-P","delta":"-0.94","markVol":"0.478","ts":"1795337568684"}]}`))
	// 期权 ticker 帧（非 USDT 计价——optInstRe 放行入 optTick）
	c.handle([]byte(`{"arg":{"channel":"tickers","instId":"BTC-USD-261123-46000-C"},"data":[` +
		`{"last":"","bidPx":"0.227","askPx":"0.2285","ts":"1795337389087"}]}`))
	// mark-price 期权形防御拒入（本件不订——若来不入 md_mark_index）
	c.handle([]byte(`{"arg":{"channel":"mark-price","instId":"BTC-USD-261123-46000-C"},"data":[` +
		`{"markPx":"0.22","ts":"1795337388663"}]}`))
	// 非期权非 USDT 形仍拒（门语义不回归）
	c.handle([]byte(`{"arg":{"channel":"tickers","instId":"XXX-YYY"},"data":[{"last":"1"}]}`))

	var sumN, tickN, markN int
	for i := 0; i < 4; i++ {
		select {
		case it := <-c.queue:
			if it.optSum != nil {
				sumN++
			}
			if it.optTick != nil {
				tickN++
			}
			if it.mark != nil {
				markN++
			}
		case <-time.After(300 * time.Millisecond): // sched-exempt-ok（桩断言轮询期界）
		}
	}
	if sumN != 1 || tickN != 1 || markN != 0 {
		t.Fatalf("分流错：optSum=%d optTick=%d mark=%d（应 1/1/0）", sumN, tickN, markN)
	}
	if c.rej.Snapshot()["okxws/non_usdt_instid"] != 1 {
		t.Fatalf("非 USDT 非期权形应计 1 次拒: %v", c.rej.Snapshot())
	}
}

// TestPassThrottleOpt 分键分隙——optSum 60s/optTick 3s。
func TestPassThrottleOpt(t *testing.T) {
	c := &Collector{passAt: map[string]time.Time{}}
	sum := rowItem{optSum: &optSumRow{instID: "BTC-USD-261123-46000-C"}}
	tick := rowItem{optTick: &optTickRow{instID: "BTC-USD-261123-46000-C"}}
	if !c.passThrottle(sum) {
		t.Fatal("optSum 首帧应过")
	}
	if c.passThrottle(sum) {
		t.Fatal("optSum 60s 内应跳（增量洪节流）")
	}
	if !c.passThrottle(tick) {
		t.Fatal("optTick 首帧应过（与 optSum 分键不互抑）")
	}
	if c.passThrottle(tick) {
		t.Fatal("optTick 3s 内应跳")
	}
	c.passAt["optsum:BTC-USD-261123-46000-C"] = time.Now().Add(-61 * time.Second)
	if !c.passThrottle(sum) {
		t.Fatal("optSum 61s 后应过")
	}
}

// TestOptChainRefreshFailClosed 目录读败上抛不污旧账。
func TestOptChainRefreshFailClosed(t *testing.T) {
	var calls int
	cat := NewOptChain(optStubGet(&calls, optCat(), optIdxRaw, nil), 0)
	ctx := context.Background()
	if _, err := cat.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	bad := NewOptChain(optStubGet(&calls, nil, nil, fmt.Errorf("venue 500")), 0)
	if _, err := bad.Refresh(ctx); err == nil {
		t.Fatal("读败应上抛")
	}
	// 族读败（instFamily 缺席→空目录→live 零行判）
	partial := map[string]json.RawMessage{
		"BTC-USD": json.RawMessage(`[{"instId":"BTC-USD-261123-46000-C","instFamily":"BTC-USD","uly":"BTC-USD","optType":"C","expTime":"1795420800000","stk":"46000","state":"live"}]`),
	}
	cat2 := NewOptChain(optStubGet(&calls, partial, optIdxRaw, nil), 0)
	if _, err := cat2.Refresh(ctx); err == nil {
		t.Fatal("族缺席应 fail-closed 上抛（不虚构目录）")
	}
}

// ── DB 桩面（O_TEST_PG_DSN 缺=skip）────────────────────────────

// TestOptStoreAndSnapshot 落表+快照臂——insertBatch 两桶+optSnapshot
// 追加（幂等冲突更新/同拍点 DO NOTHING）。
func TestOptStoreAndSnapshot(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	f := func(v float64) *float64 { return &v }
	c := &Collector{pool: pool, rej: rejects.New("okxws", "write_err"), Opt: NewOptChain(nil, 0)}
	// 活跃集注入（OptChain.act 私有——测试同包直写）
	c.Opt.act = []string{"BTC-USD-261123-46000-C", "BTC-USD-261123-47000-P"}

	inst := "BTC-USD-261123-46000-C"
	ts := time.Now().UTC().Truncate(time.Second)
	if err := c.insertBatch(ctx, []rowItem{
		{optSum: &optSumRow{instID: inst, ts: ts, delta: f(0.5), markVol: f(0.4), fwdPx: f(60000)}},
		{optTick: &optTickRow{instID: inst, ts: ts, bidPx: f(0.2), askPx: f(0.21)}},
	}); err != nil {
		t.Fatal(err)
	}
	var mv, d float64
	if err := pool.QueryRow(ctx,
		`SELECT mark_vol,delta FROM md_opt_summary WHERE inst_id=$1`, inst).Scan(&mv, &d); err != nil {
		t.Fatalf("md_opt_summary 未落行: %v", err)
	}
	if mv != 0.4 || d != 0.5 {
		t.Fatalf("summary 值面错: mv=%v d=%v", mv, d)
	}
	var bp, ap float64
	if err := pool.QueryRow(ctx,
		`SELECT bid_px,ask_px FROM md_opt_ticker WHERE inst_id=$1`, inst).Scan(&bp, &ap); err != nil {
		t.Fatalf("md_opt_ticker 未落行: %v", err)
	}
	// upsert 幂等——重发不膨胀
	if err := c.insertBatch(ctx, []rowItem{
		{optSum: &optSumRow{instID: inst, ts: ts.Add(time.Second), delta: f(0.6)}},
	}); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_opt_summary WHERE inst_id=$1`, inst).Scan(&n)
	if n != 1 {
		t.Fatalf("summary upsert 膨胀: n=%d", n)
	}
	// 快照臂：summary 行→snapshot 追加（同拍点重跑 DO NOTHING 不膨胀）
	c.optSnapshot(ctx)
	c.optSnapshot(ctx)
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM md_opt_snapshot WHERE inst_id=$1`, inst).Scan(&n)
	if n != 1 {
		t.Fatalf("snapshot 应恰 1 行（双拍幂等）: n=%d", n)
	}
	var snapTS time.Time
	var srcTS time.Time
	_ = pool.QueryRow(ctx,
		`SELECT snapshot_ts,src_ts FROM md_opt_snapshot WHERE inst_id=$1`, inst).Scan(&snapTS, &srcTS)
	if srcTS.IsZero() {
		t.Fatal("snapshot src_ts 应载源行 venue ts")
	}
	// enum upsert
	c.Opt.inst = map[string][]OptInstrument{
		"BTC-USD": {{InstID: inst, InstFamily: "BTC-USD", Uly: "BTC-USD",
			OptType: "C", State: "live", Strike: 46000, ExpTime: ts}},
	}
	c.optStoreInstruments(ctx)
	var fam string
	if err := pool.QueryRow(ctx,
		`SELECT inst_family FROM md_opt_instruments WHERE inst_id=$1`, inst).Scan(&fam); err != nil || fam != "BTC-USD" {
		t.Fatalf("instruments 未落行: fam=%q err=%v", fam, err)
	}
}

// ── E2E：fake WS 录订阅帧+断线重订（O_TEST_PG_DSN 缺=skip）────────

// optFakeWS 假 venue——每 conn：先推业务帧→读循环录 subscribe 帧入
// subs chan；一号 conn 在订阅帧收齐+推帧后短暂挂起即断（客户端
// read err→退避重连→二号 conn 同径全量重订——断线重订实证面）。
func optFakeWS(t *testing.T, subs chan<- []byte, conns *int32, frames []string) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		id := atomic.AddInt32(conns, 1)
		for _, f := range frames {
			_ = conn.WriteMessage(websocket.TextMessage, []byte(f))
		}
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				select {
				case subs <- append([]byte(nil), data...):
				default:
				}
			}
		}
		_ = id
	}))
}

// collectSubArgs 收窗内 subscribe 帧 args 汇总（断言面——
// 返回全部订阅 arg 扁平表）。
func collectSubArgs(subs <-chan []byte, d time.Duration) []map[string]any {
	var out []map[string]any
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		select {
		case b := <-subs:
			var m struct {
				Op   string           `json:"op"`
				Args []map[string]any `json:"args"`
			}
			if json.Unmarshal(b, &m) != nil || m.Op != "subscribe" {
				continue
			}
			out = append(out, m.Args...)
		case <-time.After(150 * time.Millisecond): // sched-exempt-ok（桩断言轮询期界）
		}
	}
	return out
}

func hasOptSub(args []map[string]any) (sum, tick bool) {
	for _, a := range args {
		if a["channel"] == "opt-summary" && a["instFamily"] != "" {
			sum = true
		}
		if a["channel"] == "tickers" {
			if id, _ := a["instId"].(string); isOptInstID(id) {
				tick = true
			}
		}
	}
	return
}

// TestOptE2EReSubscribe 断线重订实证——conn#1 订阅帧含期权面+帧落
// md_opt_*；conn 断→重连 conn#2 订阅帧同含期权面（重连=全量重订
// 同径——optSubscribe 在 dialLoop subscribeSet 后恒跑）。
func TestOptE2EReSubscribe(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('okx.symbols','["BTC-USDT-SWAP"]') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}

	optSumFrame := `{"arg":{"channel":"opt-summary","instFamily":"BTC-USD"},"data":[` +
		`{"instId":"BTC-USD-261123-46000-C","delta":"0.5","markVol":"0.4","ts":"1795337568684"}]}`
	optTickFrame := `{"arg":{"channel":"tickers","instId":"BTC-USD-261123-46000-C"},"data":[` +
		`{"last":"","bidPx":"0.2","askPx":"0.21","ts":"1795337389087"}]}`
	subs := make(chan []byte, 512)
	var conns int32
	srv := optFakeWS(t, subs, &conns, []string{optSumFrame, optTickFrame})
	defer srv.Close()

	var calls int
	c := NewCollector(pool, cfg)
	c.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	c.bizURL = "ws://127.0.0.1:1" // kline 臂死端点——防真 venue 外呼+conns 计数纯 public 面
	c.Opt = NewOptChain(optStubGet(&calls, optCat(), optIdxRaw, nil), 0)
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, 8)
	go c.Run(ctx2, errc)

	// conn#1 订阅帧断言（期权面在账）
	args1 := collectSubArgs(subs, 8*time.Second)
	sum1, tick1 := hasOptSub(args1)
	if !sum1 || !tick1 {
		t.Fatalf("conn#1 订阅帧缺期权面：opt-summary=%v optTick=%v", sum1, tick1)
	}

	// 帧落表实证
	inst := "BTC-USD-261123-46000-C"
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var n, tn int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_opt_summary WHERE inst_id=$1`, inst).Scan(&n)
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_opt_ticker WHERE inst_id=$1`, inst).Scan(&tn)
		if n > 0 && tn > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond) // sched-exempt-ok（桩断言轮询期界）
	}
	var n, tn int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_opt_summary WHERE inst_id=$1`, inst).Scan(&n)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM md_opt_ticker WHERE inst_id=$1`, inst).Scan(&tn)
	if n == 0 || tn == 0 {
		t.Fatalf("期权帧未落库 summary=%d ticker=%d", n, tn)
	}

	// 断线重订：等 conn#1 到期断（4s 窗）→重连 conn#2→订阅帧再采
	deadline = time.Now().Add(15 * time.Second)
	for atomic.LoadInt32(&conns) < 2 && time.Now().Before(deadline) {
		time.Sleep(300 * time.Millisecond) // sched-exempt-ok（桩断言轮询期界）
	}
	if atomic.LoadInt32(&conns) < 2 {
		t.Fatalf("未触发重连（conns=%d）——断线重订面不可验", conns)
	}
	// conn#2 订阅帧续采（subs chan 持续收——conn#1 帧已消费，余量即 #2）
	args2 := collectSubArgs(subs, 8*time.Second)
	sum2, tick2 := hasOptSub(args2)
	if !sum2 || !tick2 {
		t.Fatalf("conn#2 重订帧缺期权面：opt-summary=%v optTick=%v（断线重订回归）", sum2, tick2)
	}
}
