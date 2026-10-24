// okxunwind —— 事故回卷件：卖 PENGU→USDT→ Earn 复位停泊。
// 限价保护卖（floor=bid×0.998）分批，批间实读余量；全部成 USDT 后
// trading→funding→Earn purchase 复原停泊态。
// 用法：okxunwind -dsn <dsn> -key <keyfile> -ccy PENGU
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

func main() {
	dsn := flag.String("dsn", "", "")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	ccy := flag.String("ccy", "PENGU", "")
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
			die(err)
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			die(err)
		}
		*dst = p
	}
	cli := okxexec.NewClient(creds, false)
	inst := *ccy + "-USDT"

	tra0, _ := cli.TradingAvailBal(ctx, "USDT") // 台账基线：回卷前 USDT 存量
	avail := detailAvail(ctx, cli, *ccy)
	fmt.Printf("在册 %s availBal=%.6f\n", *ccy, avail)

	// 限价保护卖循环：floor=bid×0.998；每轮卖出实读余量，最多 8 轮
	for round := 1; round <= 8 && avail > 0; round++ {
		m, err := cli.InstrumentMeta(ctx, "SPOT", inst)
		if err != nil {
			die(err)
		}
		if avail < m.MinSz {
			fmt.Printf("余量 %.6f < minSz %g——残尘保留\n", avail, m.MinSz)
			break
		}
		tk, _ := cli.Get(ctx, "/api/v5/market/ticker?instId="+inst)
		bid := f(jget(tk, "bidPx"))
		floor := bid * 0.998
		floor = float64(int64(floor*1e7)) / 1e7 // tickSz 粒度粗截
		sz := fmtAmt(avail)
		clid := fmt.Sprintf("unw%d", time.Now().UnixNano()%1e12)
		or, err := cli.PlaceOrder(ctx, okxexec.OrderParams{InstID: inst, TdMode: "cash",
			Side: "sell", OrdType: "limit", Sz: sz, Px: fmtAmt(floor), ClOrdID: clid})
		if err != nil {
			die(fmt.Errorf("unwind sell r%d: %w", round, err))
		}
		fmt.Printf("r%d limit sell %s %s @%.7f ordId=%s\n", round, sz, *ccy, floor, jget(or, "ordId"))
		for w := 0; w < 12; w++ {
			time.Sleep(1200 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
			st, _ := cli.ResolveOrder(ctx, inst, clid)
			if st == okxexec.StateFilled {
				fmt.Printf("  r%d filled\n", round)
				break
			}
			if st == okxexec.StateLive || st == okxexec.StatePartial {
				if w == 5 { // 挂 7s 未全成→撤了重挂贴近价
					fmt.Printf("  r%d 未全成(%s)——撤单换价\n", round, st)
					cli.CancelOrder(ctx, inst, "", clid)
					time.Sleep(800 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
				}
			}
		}
		avail = detailAvail(ctx, cli, *ccy)
		fmt.Printf("  r%d 后 %s availBal=%.6f\n", round, *ccy, avail)
	}

	// 摆回：trading USDT→funding→Earn（复原原停泊态）——台账制：只许动本次卖出所得
	tra, _ := cli.TradingAvailBal(ctx, "USDT")
	proceeds := tra - tra0
	fmt.Printf("USDT 回笼: trading_avail=%.4f（本次卖出所得 %.4f，基线 %.4f）\n", tra, proceeds, tra0)
	var parked float64
	// 复审 860b8c1：台账封顶无条件——proceeds<=0（零所得/无货可卖路径）=零摆回
	// 零复购，fail-open 分支物理消失（原 if tra0>0&&proceeds>0 卫=同族残留）。
	back := backCap(tra, proceeds)
	if back > 0 {
		cltid, err := cli.AssetTransfer(ctx, "USDT", back, okxexec.AcctTrading, okxexec.AcctFunding)
		if err != nil {
			die(fmt.Errorf("transfer back: %w (clTId=%s)", err, cltid))
		}
		parked = back
		time.Sleep(2 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	}
	fund, _ := cli.AssetBalances(ctx)
	var fu float64
	for _, b := range fund {
		if b.Ccy == "USDT" {
			fu = f(b.AvailBal)
		}
	}
	fmt.Printf("funding_avail=%.4f → Earn purchase\n", fu)
	pu := fu - 0.2
	if pu > parked {
		pu = parked // 只复购本工具刚摆回的额度——存量 funding 禁碰
	}
	if pu > 1 {
		if err := cli.SavingsPurchase(ctx, "USDT", pu); err != nil {
			die(fmt.Errorf("savings repurchase: %w", err))
		}
	}
	// 终态实读
	fmt.Printf("终态: %s avail=%.6f  USDT trading=%.4f funding=%s EarnUSDT=%.4f\n",
		*ccy, detailAvail(ctx, cli, *ccy),
		func() float64 { v, _ := cli.TradingAvailBal(ctx, "USDT"); return v }(),
		func() string {
			fb, _ := cli.AssetBalances(ctx)
			for _, b := range fb {
				if b.Ccy == "USDT" {
					return b.AvailBal
				}
			}
			return "0"
		}(),
		savingsUSDT(ctx, cli))
}

func detailAvail(ctx context.Context, cli *okxexec.Client, ccy string) float64 {
	raw, err := cli.Get(ctx, "/api/v5/account/balance?ccy="+ccy)
	if err != nil {
		return 0
	}
	var rows []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if json.Unmarshal(raw, &rows) != nil || len(rows) == 0 {
		return 0
	}
	for _, d := range rows[0].Details {
		if d.Ccy == ccy {
			return f(d.AvailBal)
		}
	}
	return 0
}
func savingsUSDT(ctx context.Context, cli *okxexec.Client) float64 {
	raw, err := cli.SavingsBalance(ctx)
	if err != nil {
		return 0
	}
	var rows []struct {
		Ccy string `json:"ccy"`
		Amt string `json:"amt"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		return 0
	}
	for _, r := range rows {
		if r.Ccy == "USDT" {
			return f(r.Amt)
		}
	}
	return 0
}
func jget(raw json.RawMessage, keys ...string) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	m, _ := v.([]any)
	if len(m) == 0 {
		return ""
	}
	d, _ := m[0].(map[string]any)
	for _, k := range keys {
		if s, ok := d[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}
func f(s string) float64 {
	v, _ := json.Number(s).Float64()
	return v
}
func fmtAmt(x float64) string {
	s := fmt.Sprintf("%.8f", x)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}
func die(err error) { fmt.Fprintln(os.Stderr, "FAIL:", err); os.Exit(1) }

// backCap 台账封顶（复审 860b8c1 钦定形）：可摆回= min(trading 闲余-缓冲,
// max(本次卖出所得,0))——零所得=零摆回，封顶无条件无卫分支。
func backCap(tra, proceeds float64) float64 {
	return min(tra-0.2, max(proceeds, 0))
}
