package mt5

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// TestNoMutatingSurface 禁 mutating 面锚（T-0031 验收标准：mt5-trading-mutating 卡
// 禁用——本包源码零 OrderSend/Modify/ClosePosition/NewTradingClient 引用）。
// gen 产物（权威 proto 全谱含 trading service）不计——禁面=业务面不接，非 gen 面不存在。
func TestNoMutatingSurface(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v n=%d", err, len(files))
	}
	banned := []string{"OrderSend", "OrderModify", "ClosePosition", "NewTradingClient", "NewOrdersClient"}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range banned {
			if strings.Contains(string(b), w) {
				t.Fatalf("%s 含 mutating 面引用 %q——mt5-trading-mutating 本票禁用", f, w)
			}
		}
	}
}

// TestSymbolCandidates 候选谱常量锚（方言双形——m 尾+裸名；回确制入口面）。
func TestSymbolCandidates(t *testing.T) {
	if symbolCandidates != "BTCUSDm,ETHUSDm,BTCUSD,ETHUSD" {
		t.Fatalf("候选谱被改: %q", symbolCandidates)
	}
}

// TestProfitDedup OnOrderProfit 帧去重键锚——同值帧不落（push-on-change 事件语义：
// venue ~1Hz 周期推但表只记值变，防 append-only 表被同值帧灌水）。
func TestProfitDedup(t *testing.T) {
	snap := func(eq, fm, bal, m, p, ml float64, open int) profitSnap {
		return profitSnap{eq, fm, bal, m, p, ml, open}
	}
	a := snap(3247.5, 3245.4, 3247.5, 2.02, 0.05, 160800, 1)
	if a != snap(3247.5, 3245.4, 3247.5, 2.02, 0.05, 160800, 1) {
		t.Fatal("同值帧键应相等——去重谓词失效")
	}
	if a == snap(3247.51, 3245.4, 3247.5, 2.02, 0.05, 160800, 1) {
		t.Fatal("equity 变应判变帧")
	}
	if a == snap(3247.5, 3245.4, 3247.5, 2.02, 0.05, 160800, 0) {
		t.Fatal("open 仓数变应判变帧")
	}
}

// TestSnapshotWrite 账户快照落表锚（venue='mt5'/venue_type='mt5'/USD 本位行可查）。
func TestSnapshotWrite(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 直写径验证：snapshotOnce 需要 live client——此处验证落表形状（写径同 SQL）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src)
		VALUES('mt5','mt5','USD',1000.5,900.25,'test')
		ON CONFLICT(venue,venue_type,ccy) DO UPDATE SET equity=EXCLUDED.equity,updated_at=now()`); err != nil {
		t.Fatal(err)
	}
	var eq, av float64
	var src string
	var up time.Time
	if err := pool.QueryRow(ctx, `SELECT equity,avail,src,updated_at FROM fin_venue_account_snapshot
		WHERE venue='mt5' AND venue_type='mt5' AND ccy='USD'`).Scan(&eq, &av, &src, &up); err != nil || eq != 1000.5 || av != 900.25 {
		t.Fatalf("mt5 快照行 eq=%v av=%v err=%v", eq, av, err)
	}
}
