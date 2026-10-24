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

// TestAccountEventWrite 账户事件行落表锚（复审 18212d4：快照表 0029 废弃——
// 写形状锚改锚 md_account_events append-only 行：venue='mt5'/USD 本位/eq·avail 键域）。
func TestAccountEventWrite(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 直写径验证：snapshotOnce/profitLoop 需要 live client——此处验证行形状（写径同 SQL）。
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,venue,channel,ccy,data)
		VALUES(now(),'mt5','boot','USD',$1)`,
		[]byte(`{"eq":1000.5,"avail":900.25,"src":"boot"}`)); err != nil {
		t.Fatal(err)
	}
	var eq, av float64
	if err := pool.QueryRow(ctx, `SELECT (data->>'eq')::numeric,(data->>'avail')::numeric
		FROM md_account_events WHERE venue='mt5' AND channel='boot' AND ccy='USD'
		ORDER BY ts DESC LIMIT 1`).Scan(&eq, &av); err != nil || eq != 1000.5 || av != 900.25 {
		t.Fatalf("mt5 事件行 eq=%v av=%v err=%v", eq, av, err)
	}
}

// T-0053 锚：Connect 有界化——grpc.NewClient 懒拨号，首个 RPC 无死线=连接建立
// 无限期阻塞（OpenSession 静默悬挂实证）。黑洞址注入：自带死线兜底+调用方
// 更短 budget 继承，两路都必须在死线内返错而非悬挂。
func TestConnectDeadlineBounded(t *testing.T) {
	blackhole := "10.255.255.1:443" // 不可路由——连接建立阻塞直至死线
	// ① 自带死线兜底：调用方 ctx 无 Deadline → connectBudget 兜底返错。
	old := connectBudget
	connectBudget = 300 * time.Millisecond
	defer func() { connectBudget = old }()
	cl, err := Dial(blackhole)
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	start := time.Now()
	if err := cl.Connect(context.Background(), Credentials{Host: blackhole, Login: 1, Password: "x"}); err == nil {
		t.Fatal("黑洞址 Connect 不应成功")
	} else if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("自带死线失效（悬挂）: %v", d)
	}
	// ② 调用方更短 budget 继承不覆写。
	connectBudget = 30 * time.Second
	cctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start = time.Now()
	err = cl.Connect(cctx, Credentials{Host: blackhole, Login: 1, Password: "x"})
	if err == nil {
		t.Fatal("黑洞址 Connect 不应成功")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("调用方短 budget 未继承（悬挂）: %v", d)
	}
}
