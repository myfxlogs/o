package ledger

// diff_dustpx_test.go —— O-38064 非本位键判差锚（spec A1-A6）：
// A1 无鲜价 fail-closed 照报不产出美元数（币量+币名主量）；
// A2 陈价行（>diffPxMaxAge）按无价不误吃；
// A3 鲜价折算 rq*px 对容差+USD 豁免（NFT≈$9.44 实案形）；
// A4 qty:<units> 币量豁免币量空间先扣——rq≈0 不告警不查价、
//    部分扣减残量照折、amount_usd 锚不双计；
// A5 价源查询 err→recon_probe_failed+ok=false（#10063③ 禁吞判例）；
// A6 USDT/USD 键逐字节守恒（diff_human_test/ledger_test 既有锚回归）；
// S2 守卫：qty: 行落 USDT/USD 键=未注册 resolver 回退静态锚语义不变。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// armNFT 白名单键桩——native_baseline amt=0 行使 (venue,ccy) 入对拍域
// （O-34287 机制同源：FundCcy 未登记币的对账域载体；amt=0=纯键位
// 不贡献净额——净额全由 fin_cashflow 现金流腿构成）。须先于 Post
// 落行——compensateLateBaselineBooks 以 arm_ts 为迟簿分界。
func armNativeKey(t *testing.T, pool *pgxpool.Pool, venue, ccy string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO fin_risk_decision
		(intent,decision,reason,config_gen,kind)
		VALUES(jsonb_build_object('op','native_baseline','venue',$1::text,'ccy',$2::text,'amt',0::float8),
		'allow','allow',1,'open')`, venue, ccy); err != nil {
		t.Fatal(err)
	}
}

// nftFixture 非本位键判差夹具：okx/NFT 键入册+deposit 现金流+venue 现读。
// net=deposit 额；eq=venue 实读；返回 rej 快照与告警 note 查询面。
func nftFixture(t *testing.T, dep, eq float64) (context.Context, *Runner, func() int64) {
	t.Helper()
	ctx, r := setup(t)
	armNativeKey(t, r.pool, "okx", "NFT")
	if _, err := Post(ctx, r.pool, Entry{Kind: "deposit", Venue: "okx",
		Ccy: "NFT", Amount: dep, SourceRef: 1, SourceTable: "t",
		AccountID: "okx:trading"}); err != nil { // O-39440：NFT 判差归 trading 户（native_base 归户同键）
		t.Fatal(err)
	}
	ven := map[[2]string]float64{{"okx:trading", "NFT"}: eq}
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, VenueParts, error) {
		return ven, nil, nil
	})
	return ctx, r, func() int64 { return r.rej.Snapshot()["ledger/diff_alert"] }
}

// escNote 取 ledger_diff 升件 note（未升=空串——断言面用）。
func escNote(t *testing.T, ctx context.Context, pool *pgxpool.Pool, scope string) string {
	t.Helper()
	var note string
	err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='升件' AND esc_scope=$1 ORDER BY id DESC LIMIT 1`, scope).Scan(&note)
	if err != nil {
		return ""
	}
	return note
}

// A4a：币量豁免全额覆盖→rq≈0 静默（不告警不查价——无价行缺席下
// 若误查价仍照报 fail-closed，静默=扣减先于折价判词成立）。
func TestDiffQtyExemptFullCover(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 96, 100)                        // diff=4 币量
	seedResid(t, r.pool, "okx", "NFT", 999, "qty:4", "okx:trading") // amount_usd=999 诱饵——不得计 USD 豁免
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() != 0 {
		t.Fatal("币量豁免全额覆盖应静默")
	}
	// trackedExempt 同源面：qty 行 Mode=qty/Value=0——USD 豁免额不含 999。
	_, parts, err := r.trackedExempt(ctx, "okx", "NFT", "okx:trading")
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 || parts[0].Mode != "qty" || parts[0].Value != 0 {
		t.Fatalf("qty 行应 Mode=qty Value=0 got %+v", parts)
	}
	qx, err := r.qtyExempt(ctx, "okx", "NFT", "okx:trading")
	if err != nil || qx != 4 {
		t.Fatalf("qtyExempt 应=4 got=%v err=%v", qx, err)
	}
}

// A1：无鲜价 fail-closed——照报但不产出美元差额（币量+币名主量、
// 「无可用行情价」明写、qty 豁免行白话名不冠 $）。自举回归锚：加
// USD 静态豁免后触发与告警语义不变（豁免只进容差位不再喂强制值）。
func TestDiffPxUnpricedFailClosed(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 90, 100)                        // diff=10
	seedResid(t, r.pool, "okx", "NFT", 999, "qty:4", "okx:trading") // rq=6 币量
	if _, err := r.pool.Exec(ctx, `DELETE FROM md_tick_scan WHERE inst_id='NFT-USDT'`); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() == 0 {
		t.Fatal("无价≠静默——rq>0 必须照报")
	}
	note := escNote(t, ctx, r.pool, "ledger_diff_okx_NFT")
	if note == "" {
		t.Fatal("无价键升件未落")
	}
	for _, want := range []string{"账实差 6 NFT", "无可用行情价", "币量豁免 4 NFT"} {
		if !strings.Contains(note, want) {
			t.Fatalf("无价升件缺 %q —— note=%s", want, note)
		}
	}
	// 币量永不冠 $：机读段不得出现「账实差 $<数>」形（$ 只许出现在
	// 容差/豁免 USD 位）。
	if strings.Contains(note, "账实差 $") {
		t.Fatalf("币量差被冠 $: %s", note)
	}
	// ErrNoRows≠查询败——无价不计 recon_probe_failed。
	if n := r.rej.Snapshot()["ledger/recon_probe_failed"]; n != 0 {
		t.Fatalf("无价行缺席误计探针败 recon_probe_failed=%d", n)
	}
	// 自举回归锚：USD 静态豁免入账——触发与币量差文案不变（旧强制式
	// 下豁免+1 会改写告警额=自举怪圈；新径无价不产出美元数故不变）。
	before := alerts()
	seedResidReason(t, r.pool, "okx", "NFT", 50,
		"升件#38064定性：NFT双簿重复记账已知残差", "", "okx:trading")
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() <= before {
		t.Fatal("USD 静态豁免不得压无价键告警（自举回归）")
	}
	note2 := escNote(t, ctx, r.pool, "ledger_diff_okx_NFT")
	if !strings.Contains(note2, "账实差 6 NFT") || !strings.Contains(note2, "无可用行情价") {
		t.Fatalf("加 USD 豁免后币量差文案应不变: %s", note2)
	}
}

// A2：陈价行（>10min 鲜度界）按无价——三源全陈=fail-closed 照报。
func TestDiffPxStalePriceUnpriced(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 90, 100)
	seedResid(t, r.pool, "okx", "NFT", 999, "qty:4", "okx:trading")
	old := time.Now().Add(-time.Hour)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,last)
		VALUES($1,'okx','NFT-USDT',1.574)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES($1,'okx','NFT-USDT',1.574)`, old); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px)
		VALUES($1,'okx','NFT-USDT','mark',1.574)`, old); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() == 0 {
		t.Fatal("陈价按无价——rq>0 必须照报")
	}
	note := escNote(t, ctx, r.pool, "ledger_diff_okx_NFT")
	if !strings.Contains(note, "无可用行情价") {
		t.Fatalf("陈价应按无价明写: %s", note)
	}
	if strings.Contains(note, "≈$") {
		t.Fatalf("陈价不应折出美元数: %s", note)
	}
}

// A3：鲜价折算——spec 实案数：币量差 51758586.877993 NFT ×
// md_tick_scan 鲜行 0.0000001823 ≈$9.44（取证节§3/§4 同数）。
func TestDiffPxFreshPriceNFT(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 100, 100+51758586.877993)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,last)
		VALUES(now(),'okx','NFT-USDT',0.0000001823)`); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() == 0 {
		t.Fatal("鲜价 rq*px≈9.44>容差 0.5 必须告警")
	}
	note := escNote(t, ctx, r.pool, "ledger_diff_okx_NFT")
	if !strings.Contains(note, "51758586.877993 NFT ≈$9.44") {
		t.Fatalf("鲜价升件缺实案折算额 —— note=%s", note)
	}
}

// A3b：md_tick_scan 缺行而 md_ticks 鲜→取后者（S1 两级取数序次级臂）。
func TestDiffPxFreshPriceTicksFallback(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 90, 100)
	seedResid(t, r.pool, "okx", "NFT", 999, "qty:4", "okx:trading") // rq=6
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','NFT-USDT',1.574)`); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() == 0 {
		t.Fatal("scan 缺行 md_ticks 鲜应照取——rq*px=9.44>容差必告警")
	}
	note := escNote(t, ctx, r.pool, "ledger_diff_okx_NFT")
	if !strings.Contains(note, "6 NFT ≈$9.44") {
		t.Fatalf("次级价源未取 —— note=%s", note)
	}
}

// A3c：鲜价但折算额≤容差+USD 豁免→静默（判据仍 USD 口径——币量残量
// 折价后对 tol+exempt，非币量直对 $0.5 旧病灶复发面）。
func TestDiffPxPricedWithinTol(t *testing.T) {
	ctx, r, alerts := nftFixture(t, 90, 100)
	seedResid(t, r.pool, "okx", "NFT", 999, "qty:4", "okx:trading")
	// rq=6 × px=0.01 = $0.06 < tol 0.5 → 静默
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,last)
		VALUES(now(),'okx','NFT-USDT',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if alerts() != 0 {
		t.Fatal("折价后 $0.06<容差应静默")
	}
}

// A5：价源查询 err→recon_probe_failed+ok=false（禁吞判例——旧径 `_ =`
// 静默归零）。closed pool=确定性查询败。
func TestDiffPxProbeErr(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	pool.Close()
	r := trkRunner(pool)
	if px, ok := r.diffPx(ctx, "okx", "NFT"); ok || px != 0 {
		t.Fatalf("查询败应 ok=false got px=%v ok=%v", px, ok)
	}
	if n := r.rej.Snapshot()["ledger/recon_probe_failed"]; n == 0 {
		t.Fatal("查询败未计 recon_probe_failed")
	}
}

// S2 守卫：qty: 行落本位键=未注册 resolver 回退静态锚（币量豁免语义
// 只属非本位键——USDT 键 amount_usd 照旧计 USD 豁免不双计不漏计）。
func TestQtyResolverBaseCCYFallback(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	seedResid(t, pool, "okx", "USDT", 7, "qty:5")
	total, parts, err := r.trackedExempt(ctx, "okx", "USDT", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 7 || len(parts) != 1 || parts[0].Mode != "fallback" {
		t.Fatalf("本位键 qty 行应回退静态锚 got total=%.2f parts=%+v", total, parts)
	}
	if qx, err := r.qtyExempt(ctx, "okx", "USDT", ""); err != nil || qx != 5 {
		t.Fatalf("qtyExempt 本位键不该被 run.go 消费（防御面仍应如实读）got=%v err=%v", qx, err)
	}
}

// A6 文案锚：非本位键白话段币量+币名主量、零 $ 冠币量；BTC 0.0217
// 与 0.0200 可辨（spec 取证§5「0.02 vs 0.02」抹平病灶根修）。
func TestHumanSegmentNonBaseQty(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	seg := r.ledgerDiffHuman(ctx, "", "okx", "BTC",
		0.02, 0.0417, 0, 0.5, 0, nil, &diffOut{qty: 0.0217})
	if !strings.Contains(seg, "0.0217 BTC") {
		t.Fatalf("BTC 币量差应保真 8 位内小数: %s", seg)
	}
	if strings.Contains(seg, "$") {
		t.Fatalf("非本位键白话段不得出现 $: %s", seg)
	}
	seg2 := r.ledgerDiffHuman(ctx, "", "okx", "BTC",
		0.02, 0.04, 0, 0.5, 0, nil, &diffOut{qty: 0.02})
	if !strings.Contains(seg2, "账实差 0.02 BTC") {
		t.Fatalf("0.0200 应印 0.02 BTC: %s", seg2)
	}
	if strings.Contains(seg2, "0.0217") {
		t.Fatalf("两值应可辨: %s", seg2)
	}
}

// qtyFmt 钉死：≤8 位去尾零——BTC 0.0217 与 0.02 可辨（spec 文案验收）。
func TestQtyFmt(t *testing.T) {
	for in, want := range map[float64]string{
		0.0217:             "0.0217",
		0.02:               "0.02",
		51758586.877993:    "51758586.877993",
		6:                  "6",
		100:                "100",
		0:                  "0",
		0.000000001:        "0",
		123456789.12345678: "123456789.12345678",
	} {
		if got := qtyFmt(in); got != want {
			t.Fatalf("qtyFmt(%v)=%q want %q", in, got, want)
		}
	}
}
