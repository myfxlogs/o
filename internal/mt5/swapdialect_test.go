package mt5

// swapdialect_test.go —— swap 点数方言换算桩测（T-0143 验收锚）：Points 径
// 数值形/PercCurPrice 径/换算链缺环 NULL/非 Points NULL。
import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func sgOf(t mt5pb.SwapType, short float64) *mt5pb.SymGroup {
	return &mt5pb.SymGroup{SwapType: t, SwapShort: short, SwapLong: -1786.5}
}

func siOf(tv, ts, pts, cs float64) *mt5pb.SymbolInfo {
	return &mt5pb.SymbolInfo{TickValue: tv, TickSize: ts, Points: pts, ContractSize: cs}
}

// 锚：PercCurPrice 直年化（swap_short=-0.05%/天→-0.05×365×10⁴=-182500 bps）。
func TestSwapAnnualPercCurPrice(t *testing.T) {
	ann, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_PercCurPrice, -0.05), 3000)
	if !ok || ann != -182500 {
		t.Fatalf("perc 径 ann=%v ok=%v", ann, ok)
	}
}

// 锚：Points 径（二返修正典式——真实 venue 形：tick 双 0 不填，points/cs 实填）。
// ETHUSDm 实拍形：points=0.01/cs=1/tick_value=tick_size=0/swap_short=0→空腿免费 ann=0。
func TestSwapAnnualPointsVenueShape(t *testing.T) {
	ann, ok := SwapAnnualBPS(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, 0), 3000)
	if !ok || ann != 0 {
		t.Fatalf("空腿免费形 ann=%v ok=%v（0 值合法产出非缺席）", ann, ok)
	}
	// swap_long=-1786.5 点/天（实拍 raw）代入 mid=79842 →
	// -1786.5×0.01×1×365/(79842×1)×10⁴≈-816.8 bps/yr 量级断。
	ann, ok = SwapAnnualBPS(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -1786.5), 79842)
	if !ok {
		t.Fatal("tick 双 0 不应阻断（候选佐证非必需环）")
	}
	want := -1786.5 * 0.01 * 1 * 365 / 79842 * 10000
	if ann < want*1.01 || ann > want*0.99 {
		t.Fatalf("points 正典式 ann=%v want≈%v", ann, want)
	}
}

// 锚：换算链缺环（tick_value=0）→false（val=NULL 缺席如实）；非 Points/Perc
// 方言→false；mid 缺（0）→false；SymbolInfo 缺→false。
func TestSwapAnnualMissingLinks(t *testing.T) {
	if _, ok := SwapAnnualBPS(siOf(0, 0, 0, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000); ok {
		t.Fatal("points 缺应 false（真缺环）")
	}
	if _, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 0); ok {
		t.Fatal("mid 缺应 false")
	}
	if _, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_MarginCurrency, -50), 3000); ok {
		t.Fatal("非 Points/Perc 方言应 false")
	}
	if _, ok := SwapAnnualBPS(nil, sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000); ok {
		t.Fatal("SymbolInfo 缺应 false")
	}
}

// 返修锚（T-0143）：swapLoop 随会话 ctx 止——sessCtx cancel→goroutine 退出
// （断连桩锚：bridgeSession defer cancel 同源语义，goroutine 零泄漏）。
func TestSwapLoopStopsWithSession(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "test")}
	sessCtx, cancel := context.WithCancel(context.Background())
	before := runtime.NumGoroutine()
	done := make(chan struct{})
	go func() {
		c.swapLoop(sessCtx, nil) // cl=nil——会话 ctx cancel 即退（poll 零 sym 不触 RPC）
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测期界等待
		t.Fatal("swapLoop 应随会话 ctx 止（零泄漏）")
	}
	if after := runtime.NumGoroutine(); after > before+2 { // 容忍 runtime 波动
		t.Fatalf("goroutine 应归位 before=%d after=%d", before, after)
	}
}

// T-0144 返修①锚：双腿年化——swap_long=-1786.5pts（实拍 raw）代入
// points=0.01/cs=1/mid=79842 → long_ann≈-816bps；short=0→short_ann=0；
// 双端齐 ok。
func TestSwapAnnualBoth(t *testing.T) {
	long, short, ok := SwapAnnualBoth(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, 0), 79842)
	if !ok {
		t.Fatal("双腿应双端齐")
	}
	want := -1786.5 * 0.01 * 365 / 79842 * 10000
	if long < want*1.01 || long > want*0.99 {
		t.Fatalf("long_ann=%v want≈%v", long, want)
	}
	if short != 0 {
		t.Fatalf("short_ann=%v（空腿零成本实拍）", short)
	}
}

// FIX-02 消哑锚：失败形态 fixture 分键（探针实测谱——transport err/in-band err/
// Result nil/SG nil 四形）——symbol_group_nil 独立键留痕（原 177 行静默 continue
// 第二哑径），symbols_read_failed 收前三形；每形返非空败因样本（诊断种子）。
func TestSwapPollFailShapesCounters(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "symbols_read_failed", "symbol_group_nil", "tick_write_err")}
	shapes := []struct {
		name  string
		reply *mt5pb.SymbolParamsReply
		err   error
		key   string
		want  int64 // 门键累计值（rej 计数为跨拍累计——逐形递增断）
	}{
		{"transport_err", nil, errors.New("rpc error: EOF"), "symbols_read_failed", 1},
		{"in_band_err", &mt5pb.SymbolParamsReply{Error: &mt5pb.Error{
			Code: mt5pb.ErrorCode_COMMON_ERROR, Message: "Invalid token"}}, nil, "symbols_read_failed", 2},
		{"result_nil", &mt5pb.SymbolParamsReply{}, nil, "symbols_read_failed", 3},
		{"sg_nil", &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
			SymbolInfo: siOf(0, 0, 0.01, 1)}}, nil, "symbol_group_nil", 1},
	}
	for _, sh := range shapes {
		sh := sh
		c.symFetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
			return sh.reply, sh.err
		}
		why := c.fetchSwap(context.Background(), nil, "BTCUSDm")
		if why == "" {
			t.Fatalf("%s 形应返败因样本", sh.name)
		}
		snap := c.rej.Snapshot()
		if snap["mt5bridge/"+sh.key] != sh.want {
			t.Fatalf("%s 形应 %s=%d got %v", sh.name, sh.key, sh.want, snap)
		}
	}
}

// FIX-02 锚：全灭拍判定——pollSwap 零落行（okN=0）+ 样本携首败 sym 详情
// （swapLoop WARN 触发条件 len(syms)>0 && okN==0 的判定输入面）。
func TestSwapPollAllFailBlind(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "symbols_read_failed")}
	c.symFetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
		return nil, fmt.Errorf("rpc error: EOF")
	}
	okN, sample := c.pollSwap(context.Background(), nil, []string{"BTCUSDm", "ETHUSDm", "SOLUSDm"})
	if okN != 0 {
		t.Fatalf("全灭拍应零落行 got %d", okN)
	}
	if len(sample) < 7 || sample[:7] != "BTCUSDm" {
		t.Fatalf("样本应携首败 sym 详情: %q", sample)
	}
}

// FIX-02 锚：健康实测形（探针实拍 BTCUSDm——InPoints L=-1778.7/points=0.01/cs=1）
// 照常落行+meta 年化链随行——消哑改造零伤正道（单 sym 败仅计数语义不触动）。
func TestSwapPollHealthyShapeLands(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "tick_write_err")}
	c.symFetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
		return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
			Symbol:      "BTCUSDm",
			SymbolInfo:  siOf(0, 0, 0.01, 1),
			SymbolGroup: sgOf(mt5pb.SwapType_SwapType_InPoints, -1778.7)}}, nil
	}
	// mid 基准（midOf 读面——md_ticks venue='mt5' 实行）
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask)
		VALUES(now(),'mt5','BTCUSDm',80000,80000,80000)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=now(),last=EXCLUDED.last,
		bid=EXCLUDED.bid,ask=EXCLUDED.ask,recv_ts=now()`); err != nil {
		t.Fatal(err)
	}
	if why := c.fetchSwap(ctx, nil, "BTCUSDm"); why != "" {
		t.Fatalf("健康形应落行 got %q", why)
	}
	var val *float64
	var meta map[string]any
	if err := pool.QueryRow(ctx,
		`SELECT val, meta FROM md_signal WHERE kind='mt5_swap' AND key='BTCUSDm'`).
		Scan(&val, &meta); err != nil {
		t.Fatalf("行应落: %v", err)
	}
	if val == nil {
		t.Fatal("InPoints 形应年化 val 非 NULL")
	}
	want := -1778.7 * 0.01 * 1 * 365 / 80000 * 10000
	if *val < want*1.01 || *val > want*0.99 {
		t.Fatalf("ann=%v want≈%v", *val, want)
	}
	if meta["ann_ok"] != true {
		t.Fatalf("meta 年化链应随行: %v", meta)
	}
	if mid, ok := meta["mid"].(float64); !ok || mid != 80000 {
		t.Fatalf("meta mid 应=80000: %v", meta["mid"])
	}
}
