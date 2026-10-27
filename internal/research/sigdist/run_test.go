// run_test.go —— 信号蒸馏桩测（T-0139 验收锚）：basis_ann 年化数值/funding_view
// 三件（当期+预测+7d 均值，缺数据字段缺席）/Fresh 新鲜度闸（陈=缺席如实）/
// depth5 特征值落表形。
package sigdist

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return context.Background(), New(pool)
}

func md(t *testing.T, ctx context.Context, r *Runner, q string, args ...any) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, q, args...); err != nil {
		t.Fatal(err)
	}
}

// 锚：basis_ann=(mark-index)/index×365/days×10⁴ bps——夹具 2027-06-26 到期，
// mark=102 index=100 days≈243 → 年化≈10000×0.02×365/243≈300.4 bps。
// T-0164 血统闸：行 ts=最旧上游观测（index 40s 前/mark 10s 前→ts=index 边）。
// T-0171：现货侧=index-tickers WS 行（spot_last REST 轮询行已退役）。
func TestBasisAnn(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	md(t, ctx, r, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		($1,'BTC-USDT-270626','mark-price',102),
		($2,'BTC-USDT','index-tickers',100),
		($1,'ETH-USDT-999999','mark-price',101)`, // 999999=坏到期——跳过如实
		now.Add(-10*time.Second), now.Add(-40*time.Second))
	if err := r.basisAnn(ctx); err != nil {
		t.Fatal(err)
	}
	var bps float64
	var meta map[string]any
	var rowTs time.Time
	if err := r.pool.QueryRow(ctx, `SELECT val, meta, ts FROM md_signal WHERE kind='basis_ann' AND key='BTC-USDT-270626'`).
		Scan(&bps, &meta, &rowTs); err != nil {
		t.Fatal("basis_ann 行应落:", err)
	}
	want := float64(102-100) / 100 * 365 / 243 * 10000
	if bps < want-1 || bps > want+1 {
		t.Fatalf("年化 bps≈%v want≈%v", bps, want)
	}
	if meta["expiry"] != "2027-06-26" {
		t.Fatalf("meta.expiry=%v", meta["expiry"])
	}
	// 血统：行 ts=最旧上游观测（index-tickers 40s 前——非写拍 now()）。
	if age := time.Since(rowTs); age < 35*time.Second || age > 60*time.Second {
		t.Fatalf("basis ts 应=最旧上游观测（spot 40s 前）got age=%v", age)
	}
	var n int
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM md_signal WHERE kind='basis_ann'`).Scan(&n)
	if n != 1 {
		t.Fatalf("坏到期 inst 应跳过（行数=%d）", n)
	}
}

// 锚：funding_view 三件——当期 val+meta.next_funding_rate+meta.avg_7d；
// 窗外均值不含陈行。
func TestFundingView(t *testing.T) {
	ctx, r := setup(t)
	now := time.Now()
	md(t, ctx, r, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_ts,next_funding_rate) VALUES
		($1,'BTC-USDT-SWAP',0.0001,$2,0.00012),
		($3,'BTC-USDT-SWAP',0.0005,$2,0.00012)`, // 10d 前陈行——均值窗外
		now, now.Add(time.Hour), now.Add(-10*24*time.Hour))
	md(t, ctx, r, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		($1,'ETH-USDT-SWAP',0.0002)`, now.Add(-2*time.Hour)) // 无 next——字段缺席如实
	if err := r.fundingView(ctx); err != nil {
		t.Fatal(err)
	}
	var rate float64
	var avgS, nextS *string
	if err := r.pool.QueryRow(ctx, `SELECT val, meta->>'avg_7d', meta->>'next_funding_rate'
		FROM md_signal WHERE kind='funding_view' AND key='BTC-USDT-SWAP'`).Scan(&rate, &avgS, &nextS); err != nil {
		t.Fatal(err)
	}
	next, _ := strconv.ParseFloat(*nextS, 64)
	if rate != 0.0001 || next != 0.00012 {
		t.Fatalf("当期/预测=%v/%v", rate, next)
	}
	avg, _ := strconv.ParseFloat(*avgS, 64)
	if avg > 0.00011 { // 陈行 0.0005 不入 7d 窗（均=0.0001）
		t.Fatalf("avg_7d=%v（陈行应窗外）", avg)
	}
	var meta map[string]any
	r.pool.QueryRow(ctx, `SELECT meta FROM md_signal WHERE kind='funding_view' AND key='ETH-USDT-SWAP'`).Scan(&meta)
	if _, ok := meta["next_funding_rate"]; ok {
		t.Fatalf("无 next 行应字段缺席: %v", meta)
	}
}

// 锚：Fresh 闸——鲜=ok；陈（>2×蒸馏拍）=缺席如实；缺行=缺席。
func TestFreshGate(t *testing.T) {
	ctx, r := setup(t)
	md(t, ctx, r, `INSERT INTO md_signal(kind,key,val,ts) VALUES
		('funding_view','BTC-USDT-SWAP',0.0001,now()),
		('funding_view','ETH-USDT-SWAP',0.0002,now()-interval '5 minutes')`)
	if _, ok, err := Fresh(ctx, r.pool, "funding_view", "BTC-USDT-SWAP"); err != nil || !ok {
		t.Fatalf("鲜信号应 ok（err=%v）", err)
	}
	if _, ok, _ := Fresh(ctx, r.pool, "funding_view", "ETH-USDT-SWAP"); ok {
		t.Fatal("陈信号应缺席（禁陈信号当事实）")
	}
	if _, ok, _ := Fresh(ctx, r.pool, "funding_view", "NOPE"); ok {
		t.Fatal("缺行应缺席")
	}
}

// 锚（验收⑥）：行数有界——同键 upsert 重拍不增行（禁时间序列化 D-0018）。
func TestBoundedRows(t *testing.T) {
	ctx, r := setup(t)
	for i := 0; i < 3; i++ {
		if err := r.basisAnn(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM md_signal`).Scan(&n)
	if n != 0 {
		t.Fatalf("零夹具拍行数应 0（有界）got %d", n)
	}
	md(t, ctx, r, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		(now(),'BTC-USDT-270626','mark-price',102),(now(),'BTC-USDT','index-tickers',100)`)
	for i := 0; i < 3; i++ {
		if err := r.Distill(ctx); err != nil {
			t.Fatal(err)
		}
	}
	r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM md_signal`).Scan(&n)
	if n > 2 {
		t.Fatalf("三拍后行数应有界（≤2）got %d", n)
	}
}

// T-0140 surface 语义映射锚：funding 面→funding_view 榜（|val| 降序 top3）；
// quarter/basis 面→basis_ann；非信号敏感面→零绑定；陈信号零绑定。
func TestSurfaceSignalsMapping(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts) VALUES
		('funding_view','BTC-USDT-SWAP',0.0002,'{}',now()),
		('funding_view','ETH-USDT-SWAP',0.0005,'{}',now()),
		('funding_view','SOL-USDT-SWAP',0.0003,'{}',now()),
		('funding_view','XRP-USDT-SWAP',0.0004,'{}',now()),
		('funding_view','STALE-USDT-SWAP',0.9,'{}',now()-interval '5 minutes'),
		('basis_ann','BTC-USDT-270626',300,'{}',now())`); err != nil {
		t.Fatal(err)
	}
	sigs, ok, err := r.SurfaceSignals(ctx, "okx", "market-funding-rate")
	if err != nil || !ok {
		t.Fatalf("funding 面应有绑定 err=%v", err)
	}
	if len(sigs) != 3 || sigs[0].Key != "ETH-USDT-SWAP" { // |val| 降序 top3，陈行不录入
		t.Fatalf("榜 top3 降序+陈行排除: %v", sigs)
	}
	if bs, ok, _ := r.SurfaceSignals(ctx, "okx", "quarter-basis"); !ok || len(bs) != 1 || bs[0].Kind != "basis_ann" {
		t.Fatalf("quarter 面→basis_ann 绑定: %v", bs)
	}
	if _, ok, _ := r.SurfaceSignals(ctx, "okx", "savings-balance"); ok {
		t.Fatal("非信号敏感面应零绑定")
	}
}

// AttachConfigForTest 测试注 config 键（app_config 直植+Load——热启同径）。
func (r *Runner) AttachConfigForTest(ctx context.Context, kv map[string]any) error {
	for k, v := range kv {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES($1,$2)
			ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, k, string(raw)); err != nil {
			return err
		}
	}
	r.cfg = config.New(r.pool)
	return r.cfg.Load(ctx)
}
