package mt5

// catalog_test.go —— 全目录粗扫桩测（T-0408①验收锚）：覆盖=全目录非订阅
// 小集（SymbolList 自报面）+GetQuoteMany 快照 mid 供换算（未订阅符
// InPoints 方言经快照 mid 可年化）+meta.mid 快照价落行；SymbolList 败/
// 零行=让位零写；quote 批败=降级 mid=0→InPoints val=NULL 缺席如实。
import (
	"context"
	"errors"
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func catSetup(t *testing.T) (context.Context, *BridgeCollector) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	return ctx, &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "symbols_read_failed", "symbol_group_nil", "tick_write_err")}
}

// 锚①：覆盖=全目录——3 符（含未订阅 XAUUSDm）全落 mt5_swap 行；快照 mid
// 供 InPoints 年化（XAU mid=2400→swap_long=-1800 pts/天→ann≈-4450bps
// −44.5%/年量级——金案口径形）；meta.mid=快照价留痕。
func TestCatalogSweepCoversUnsubscribed(t *testing.T) {
	ctx, c := catSetup(t)
	c.symList = func(ctx context.Context, cl *Client) ([]string, error) {
		return []string{"BTCUSDm", "ETHUSDm", "XAUUSDm"}, nil
	}
	c.quoteMany = func(ctx context.Context, cl *Client, syms []string) map[string]float64 {
		return map[string]float64{"BTCUSDm": 80000, "ETHUSDm": 3000, "XAUUSDm": 2400}
	}
	c.symFetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
		return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
			Symbol: sym, SymbolInfo: siOf(0, 0, 0.01, 1),
			SymbolGroup: &mt5pb.SymGroup{SwapType: mt5pb.SwapType_SwapType_InPoints,
				SwapLong: -1800, SwapShort: -50}}}, nil
	}
	pace := catalogPace
	catalogPace = 0
	defer func() { catalogPace = pace }()
	c.catalogSweep(ctx, nil)
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal WHERE kind='mt5_swap'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("全目录应落 3 行 got %d", n)
	}
	var val float64
	var meta map[string]any
	if err := c.pool.QueryRow(ctx, `SELECT val, meta FROM md_signal
		WHERE kind='mt5_swap' AND key='XAUUSDm'`).Scan(&val, &meta); err != nil {
		t.Fatal("XAUUSDm 行应落:", err)
	}
	want := -50 * 0.01 * 1 * 365 / 2400 * 10000 // 空腿年化（swap_short）≈-760.4bps
	if val < want*1.01 || val > want*0.99 {
		t.Fatalf("XAU ann=%v want≈%v（快照 mid 换算）", val, want)
	}
	if mid, ok := meta["mid"].(float64); !ok || mid != 2400 {
		t.Fatalf("meta.mid 应=快照 2400: %v", meta["mid"])
	}
	// 双腿年化随行——long=-1800pts→≈-27375bps（-273%/yr 量级金案形）。
	wantLong := -1800 * 0.01 * 1 * 365 / 2400 * 10000
	if v, ok := meta["swap_long_ann_bps"].(float64); !ok || v < wantLong*1.01 || v > wantLong*0.99 {
		t.Fatalf("swap_long_ann≈%v want≈%v: %v", meta["swap_long_ann_bps"], wantLong, meta)
	}
}

// 锚②：SymbolList 败/零行=让位零写（fail-closed 不虚构目录）；quote 批败=
// mid 空集降级→InPoints val=NULL 缺席如实（行仍落——事实域到即记）。
func TestCatalogSweepDegradeShapes(t *testing.T) {
	ctx, c := catSetup(t)
	pace := catalogPace
	catalogPace = 0
	defer func() { catalogPace = pace }()
	// 幕1：SymbolList 败→零写。
	c.symList = func(ctx context.Context, cl *Client) ([]string, error) {
		return nil, errors.New("rpc error: EOF")
	}
	c.catalogSweep(ctx, nil)
	var n int
	c.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal WHERE kind='mt5_swap'`).Scan(&n)
	if n != 0 {
		t.Fatalf("list 败应零写 got %d", n)
	}
	// 幕2：quote 批败降级→行落 val NULL（InPoints 无 mid 不可年化如实）。
	c.symList = func(ctx context.Context, cl *Client) ([]string, error) {
		return []string{"XAUUSDm"}, nil
	}
	c.quoteMany = func(ctx context.Context, cl *Client, syms []string) map[string]float64 {
		return map[string]float64{} // 批败空集降级
	}
	c.symFetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
		return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
			Symbol: sym, SymbolInfo: siOf(0, 0, 0.01, 1),
			SymbolGroup: &mt5pb.SymGroup{SwapType: mt5pb.SwapType_SwapType_InPoints,
				SwapLong: -1800, SwapShort: -50}}}, nil
	}
	c.catalogSweep(ctx, nil)
	var val *float64
	if err := c.pool.QueryRow(ctx, `SELECT val FROM md_signal
		WHERE kind='mt5_swap' AND key='XAUUSDm'`).Scan(&val); err != nil {
		t.Fatal("降级行应落:", err)
	}
	if val != nil {
		t.Fatalf("无 mid InPoints 应 val NULL 缺席如实 got %v", *val)
	}
}
