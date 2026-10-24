// okxverify —— OKX 真户全谱验证：REST 读面矩阵 + 公开/私有 WS 流面 + mutating
// 实单谱（资金摆渡→开仓→定态→平仓→回转+停泊复位）。凭据自 app_secrets 密文列
// 解（master key 文件径），零落盘零回显。owner 令：真金实测每一项。
// mutating 段收尾必复位：仓全平+余资摆回 funding→Earn（来路回滚）。
// 用法：okxverify -dsn <dsn> -key <keyfile> [-sim] [-mut] [-mvusd 20]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

var (
	passN, failN, diffN int
	mu                  sync.Mutex
)

type row struct{ sec, item, got, want, note, st string }

var rows []row

func rec(sec, item string, got any, want, note string, ok bool) {
	st := "PASS"
	if !ok {
		st = "DIFF"
	}
	mu.Lock()
	rows = append(rows, row{sec, item, fmt.Sprint(got), want, note, st})
	switch st {
	case "PASS":
		passN++
	case "DIFF":
		diffN++
	}
	mu.Unlock()
}
func recFail(sec, item, note string) {
	mu.Lock()
	rows = append(rows, row{sec, item, "ERR", "", note, "FAIL"})
	failN++
	mu.Unlock()
}
func jget(raw json.RawMessage, keys ...string) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	m, _ := v.([]any)
	if len(m) == 0 {
		return "∅"
	}
	d, _ := m[0].(map[string]any)
	for _, k := range keys {
		if s, ok := d[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}
func f64(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }

func main() {
	dsn := flag.String("dsn", "", "phase DSN")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	sim := flag.Bool("sim", false, "demo 户头")
	mut := flag.Bool("mut", false, "跑 mutating 实单谱（真钱）")
	mvusd := flag.Float64("mvusd", 8, "划转往返 USDT 额")
	restore := flag.String("restore", "", "复位目标币（前序测试残留的币对清回——如 PENGU）")
	restoremax := flag.Float64("restoremax", 25, "显式 -restore 买回的 USDT 封顶（防扫存量资金）")
	flag.Parse()
	ctx := context.Background()

	mk, err := config.ReadMasterKeyFrom(*keyf)
	if err != nil {
		die(err)
	}
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		die(err)
	}
	defer pool.Close()
	var creds okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &creds.Key, "okx.api_secret": &creds.Secret, "okx.passphrase": &creds.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			die(fmt.Errorf("%s 读败: %w", k, err))
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			die(fmt.Errorf("%s 解密: %w", k, err))
		}
		*dst = p
	}
	cli := okxexec.NewClient(creds, *sim)
	fmt.Println("== 凭据解密 ok（值零回显）==")

	// ============ REST 读面矩阵 ============
	type rd struct {
		item, path string
		keys       []string // 首行抽查字段
		note       string
	}
	readMx := []rd{
		{"balance", "/api/v5/account/balance", []string{"totalEq"}, "unified 交易户总权益"},
		{"positions", "/api/v5/account/positions", nil, "在仓（∅=全平常态判 PASS）"},
		{"config", "/api/v5/account/config", []string{"posMode", "acctLv"}, "持仓模式+账户等级"},
		{"maxSize", "/api/v5/account/max-size?instId=BTC-USDT-SWAP&tdMode=cross", []string{"maxBuy"}, "最大可开"},
		{"maxAvailSize", "/api/v5/account/max-avail-size?instId=BTC-USDT-SWAP&tdMode=cross", []string{"availBuy"}, "可用可开"},
		{"leverage", "/api/v5/account/leverage-info?instId=BTC-USDT-SWAP&mgnMode=cross", []string{"lever"}, "杠杆档位"},
		{"ordersPending", "/api/v5/trade/orders-pending", nil, "挂单（∅=无挂常态判 PASS）"},
		{"ordersHistory", "/api/v5/trade/orders-history?instType=SPOT", []string{"ordId"}, "现货单史"},
		{"ordersHistorySwap", "/api/v5/trade/orders-history?instType=SWAP", []string{"ordId"}, "合约单史"},
		{"fillsHistory", "/api/v5/trade/fills-history?instType=SPOT", []string{"tradeId"}, "成交流水"},
		{"bills", "/api/v5/account/bills?limit=20", []string{"billId"}, "账单（fee/funding 源）"},
		{"assetBalances", "/api/v5/asset/balances", []string{"ccy"}, "资金户"},
		{"savingsBalance", "/api/v5/finance/savings/balance", []string{"ccy"}, "Earn 持仓"},
		// transfer-state 由 mutating 段真 clTId 实证（虚参返 51000=校验拒，非可达性证）
		{"depositHistory", "/api/v5/asset/deposit-history?limit=5", []string{"ccy"}, "充币史"},
		{"withdrawalHistory", "/api/v5/asset/withdrawal-history?limit=5", []string{"ccy"}, "提币史"},
		{"currencies", "/api/v5/asset/currencies?ccy=USDT", []string{"chain"}, "币种链信息"},
		{"instrumentsSpot", "/api/v5/public/instruments?instType=SPOT&instId=BTC-USDT", []string{"ctVal", "minSz"}, "现货目录"},
		{"instrumentsSwap", "/api/v5/public/instruments?instType=SWAP&instId=BTC-USDT-SWAP", []string{"ctVal", "minSz", "ctValCcy"}, "合约目录"},
		{"ticker", "/api/v5/market/ticker?instId=BTC-USDT-SWAP", []string{"last"}, "行情 ticker"},
		{"fundingRate", "/api/v5/public/funding-rate?instId=BTC-USDT-SWAP", []string{"fundingRate"}, "资金费现率"},
		{"indexTickers", "/api/v5/market/index-tickers?instId=BTC-USDT", []string{"idxPx"}, "指数价"},
		{"markPrice", "/api/v5/public/mark-price?instType=SWAP&instId=BTC-USDT-SWAP", []string{"markPx"}, "标记价"},
	}
	for _, r := range readMx {
		raw, err := cli.Get(ctx, r.path)
		if err != nil {
			recFail("rest", r.item, r.path+" → "+err.Error())
			continue
		}
		if r.keys == nil {
			rec("rest", r.item, jget(raw, "x"), "errfree", r.note, true)
			continue
		}
		got := jget(raw, r.keys...)
		rec("rest", r.item, got, "非空/errfree", r.note, got != "" && got != "∅")
	}
	rs, err := okxexec.LendingRateSummary(ctx, "USDT")
	if err != nil {
		recFail("rest", "lendingRate", err.Error())
	} else {
		rec("rest", "lendingRate", rs.AvgRate, ">0", "Earn 灵活现率（年化）", rs.AvgRate > 0)
	}
	posMode := ""
	if raw, err := cli.Get(ctx, "/api/v5/account/config"); err == nil {
		posMode = jget(raw, "posMode")
	}

	// ============ 公开 WS 流面 ============
	pubFrames := wsCollect(ctx, okxPublicURL, []map[string]string{
		{"channel": "tickers", "instId": "BTC-USDT-SWAP"},
		{"channel": "books5", "instId": "BTC-USDT-SWAP"},
		{"channel": "funding-rate", "instId": "BTC-USDT-SWAP"},
		{"channel": "mark-price", "instId": "BTC-USDT-SWAP"},
	}, 15*time.Second)
	for ch, n := range pubFrames {
		rec("pubws", ch, n, ">0帧", "公开流实测", n > 0)
	}

	// ============ 私有 WS 流面（贯穿 mutating 段观测） ============
	privCh := make(chan string, 512)
	privCtx, privCancel := context.WithCancel(ctx)
	defer privCancel()
	go privStream(privCtx, creds, *sim, privCh)
	time.Sleep(3 * time.Second) // login+订阅稳定窗 // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环

	// ============ mutating 实单谱 ============
	if *mut {
		mutateAll(ctx, cli, *mvusd, posMode, privCh, *restore, *restoremax)
	} else {
		fmt.Println("(mutating 段跳过——-mut 未置)")
	}
	privCancel()
	time.Sleep(500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	drainPriv(privCh, "")

	// ============ 汇总 ============
	fmt.Println("\n=== OKX 真户全谱验证 ===")
	fmt.Printf("%-7s %-20s %-22s %-18s %s\n", "SEC", "ITEM", "GOT", "WANT", "NOTE")
	for _, r := range rows {
		fmt.Printf("%-7s %-20s %-22.22s %-18.18s %s %s\n", r.sec, r.item, r.got, r.want, r.st, r.note)
	}
	fmt.Printf("\n合计: PASS=%d DIFF=%d FAIL=%d\n", passN, diffN, failN)
}
