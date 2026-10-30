package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/guard"
	"o/internal/ledger"
	"o/internal/mt5"
	"o/internal/risk"
	"o/internal/trading/okxexec"
)

// attachVenueReaders venue 现读件装配（T-0048 返修②口径钉死——快照行全域出局）：
//   - okx 权益/闲余=md_account_events 最新 account/balance_and_position 帧解
//     （details eqUsd 总权益/availBal 闲余；push-on-change——帧龄不判陈，陈度归 WS 活性件管）。
//   - okx 对账聚合=事件帧交易户 eq+REST /asset/balances(funding)+/finance/savings/balance(amt)
//     现读合 {okx,USDT}——资金/停泊户无推送面（skill 在案）；mt5=AccountSummary 现读。
//
// 凭据缺席=boot 明示 WARN 不静默缺岗；各闭包判 nil 不 panic。
func attachVenueReaders(riskGate *risk.Gate, wd *guard.Warden, ledgerR *ledger.Runner,
	pool *pgxpool.Pool, okcli *okxexec.Client, openMt5 func(ctx context.Context) (*mt5.Client, error)) {
	// frameVals 事件帧现读（okx 交易户同源读，REST 非判据源——帧龄不判陈）。
	frameVals := func(ctx context.Context) (eq, avail float64, err error) {
		if pool == nil {
			return 0, 0, fmt.Errorf("pool 缺")
		}
		// account 帧=details[] per-ccy（eqUsd/availBal）；balance_and_position 帧=顶字段。
		var ch string
		var data json.RawMessage
		err = pool.QueryRow(ctx, `SELECT channel,data FROM md_account_events
			WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&ch, &data)
		if err != nil {
			return 0, 0, err
		}
		if ch == "account" {
			var top struct {
				Details []struct {
					EqUsd    string `json:"eqUsd"`
					AvailBal string `json:"availBal"`
					Ccy      string `json:"ccy"`
				} `json:"details"`
			}
			if jerr := json.Unmarshal(data, &top); jerr == nil && len(top.Details) > 0 {
				for _, d := range top.Details {
					eq += parseNum(d.EqUsd) // 全币行求和=交易户总 USD 权益（持仓含现货值）
					if d.Ccy == "USDT" {
						avail = parseNum(d.AvailBal)
					}
				}
				return eq, avail, nil
			}
		}
		var top2 struct {
			EqUsd    string `json:"eqUsd"`
			AvailBal string `json:"availBal"`
			Details  []struct {
				EqUsd    string `json:"eqUsd"`
				AvailBal string `json:"availBal"`
				Ccy      string `json:"ccy"`
			} `json:"details"`
		}
		_ = json.Unmarshal(data, &top2)
		if len(top2.Details) > 0 {
			for _, d := range top2.Details {
				eq += parseNum(d.EqUsd) // 全币行求和=总 USD 权益
				if d.Ccy == "USDT" {
					avail = parseNum(d.AvailBal)
				}
			}
			return eq, avail, nil
		}
		return parseNum(top2.EqUsd), parseNum(top2.AvailBal), nil
	}
	// tradingVals 帧先 REST 兜底读件（T-0058 根修+T-0071 全币求和——帧败/
	// eq=0→/account/balance 兜底；兜底再败=err 上抛 fail-closed 禁部分聚合）。
	tradingVals := func(ctx context.Context) (eq, avail float64, err error) {
		if eq, avail, err = frameVals(ctx); err == nil && eq != 0 {
			return eq, avail, nil
		}
		raw, rerr := okcli.Get(ctx, "/api/v5/account/balance")
		if rerr != nil {
			return 0, 0, fmt.Errorf("帧缺+REST 兜底败: %w", rerr)
		}
		var rows []struct {
			Details []struct {
				Ccy      string `json:"ccy"`
				EqUsd    string `json:"eqUsd"`
				AvailBal string `json:"availBal"`
			} `json:"details"`
		}
		if uerr := json.Unmarshal(raw, &rows); uerr != nil || len(rows) == 0 {
			return 0, 0, fmt.Errorf("balance parse: %v", uerr)
		}
		for _, d := range rows[0].Details {
			eq += parseNum(d.EqUsd) // 全币行求和=总 USD 权益（REST 兜底同口径）
			if d.Ccy == "USDT" {
				avail = parseNum(d.AvailBal)
			}
		}
		if eq == 0 {
			return 0, 0, fmt.Errorf("details 缺席")
		}
		return eq, avail, nil
	}
	if okcli == nil {
		log.Printf("WARN: venue 现读件未挂接（okx 凭据缺席）——risk equity/dd 硬线/对账 fail-closed 让位")
		return
	}
	// okx 权益=三分量聚合单源（okxVenueEq——riskGate/dd 硬线共用同一闭包
	// 禁平行第二口径；T-0264：dd 基准曾只读交易户——停泊态假零硬线空转+
	// 户间划转假回撤误触）。
	okxEq := func(ctx context.Context) (float64, error) {
		return okxVenueEq(ctx,
			func(c context.Context) (float64, error) {
				eq, _, err := tradingVals(c)
				return eq, err
			},
			func(c context.Context, m map[[2]string]float64) error {
				return fundSaveBalances(c, pool, okcli, m, true)
			})
	}
	riskGate.WithEquity(func(ctx context.Context, venue string) (float64, error) {
		if venue == "mt5" {
			if openMt5 == nil {
				return 0, fmt.Errorf("mt5 权益件缺席")
			}
			a, err := mt5SummaryOnce(ctx, pool, nil, openMt5)
			return a.Equity, err
		}
		if venue != "okx" {
			return 0, fmt.Errorf("venue 无权益件: %s", venue)
		}
		return okxEq(ctx)
	})
	wd.AttachEquity(okxEq)
	// T-0055：savings 累计收益读件（利息事件化入账源——earnings 字段真源）。
	ledgerR.WithEarnEarnings(func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		if okcli == nil {
			return out, fmt.Errorf("earnings 件缺席（凭据缺）")
		}
		sav, err := okcli.SavingsBalance(ctx)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Ccy      string `json:"ccy"`
			Earnings string `json:"earnings"`
		}
		if json.Unmarshal(sav, &rows) != nil {
			return nil, fmt.Errorf("savings parse")
		}
		for _, r2 := range rows {
			out[r2.Ccy] = okxexec.ParseF(r2.Earnings)
		}
		return out, nil
	})
	// 对账聚合=帧 eq+REST funding+savings（无推送户）+mt5 AccountSummary。
	// T-0058 根修：三分量任一读败=整读 err（对账轮跳过非假差——
	// 「读件缺席跳过本轮」与「真差超限」两文本分径）。
	// T-0095 v2：交易户贡献=基金口径过滤求和（USDT∪open 腿基币行 eqUsd——
	// 不按全账户取，账外尘口径内天然排除），白名单 FundCcy 同源。
	ledgerR.WithMTM(mtmReader(pool, ctValReader(okcli))) // T-0076 盯市偏移
	ledgerR.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		eq, err := tradingFundEq(ctx, pool, okcli)
		if err != nil {
			return nil, fmt.Errorf("trading 读败: %w", err)
		}
		out[[2]string{"okx", "USDT"}] += eq
		// T-0100 ③：对拍径 usd=false 原币量入键——ledger 侧非本位键按币量
		// 建键（ParkedAmt 台账），venue/ledger 单位同径不错配。
		if err := fundSaveBalances(ctx, pool, okcli, out, false); err != nil {
			return nil, err
		}
		if openMt5 != nil {
			a, cerr := mt5SummaryOnce(ctx, pool, nil, openMt5)
			if cerr != nil {
				return nil, fmt.Errorf("mt5 读败: %w", cerr)
			}
			out[[2]string{"mt5", "USD"}] = a.Equity
		}
		return out, nil
	})
}

func parseNum(s string) float64 {
	var v float64
	_, _ = fmt.Sscanf(s, "%f", &v)
	return v
}

// okxAvailFromEvents 交易户可用额=md_account_events 最新帧 details availBal
// （T-0048 返修③：depositIdle 自 AcctTrading 摆钱——资金户=错户已纠；
// push-on-change 帧龄不判陈，陈度归 WS 活性件管）。
func okxAvailFromEvents(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client) (float64, error) {
	var data json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel='account' ORDER BY ts DESC LIMIT 1`).Scan(&data); err != nil {
		if okcli == nil {
			return 0, err
		}
		return okcli.TradingAvailBal(ctx, "USDT") // 帧缺→REST 兜底（T-0058）
	}
	var top struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return 0, err
	}
	for _, d := range top.Details {
		if d.Ccy == "USDT" {
			return parseNum(d.AvailBal), nil
		}
	}
	if okcli != nil {
		return okcli.TradingAvailBal(ctx, "USDT") // USDT 行缺=REST 兜底
	}
	return 0, fmt.Errorf("USDT 行缺席")
}

// ctValReader 合约面值现读件（T-0046①——T-0048 三令后 facts 表废，改 okx
// instruments 决策时点现读；静态乐器属性 10min TTL 缓存：guard 逐 tick 事件评
// 估，无缓存=限速自杀。读败/零值=false fail-closed 漂移判据跳过不误判）。
func ctValReader(okcli *okxexec.Client) func(ctx context.Context, venue, instID string) (float64, bool) {
	var mu sync.Mutex
	type ent struct {
		v  float64
		at time.Time
	}
	cache := map[string]ent{}
	return func(ctx context.Context, venue, instID string) (float64, bool) {
		if okcli == nil || venue != "okx" || !strings.HasSuffix(instID, "-SWAP") {
			return 0, false
		}
		mu.Lock()
		e, ok := cache[instID]
		mu.Unlock()
		if ok && time.Since(e.at) < 10*time.Minute {
			return e.v, e.v > 0
		}
		m, err := okcli.InstrumentMeta(ctx, "SWAP", instID)
		if err != nil || m.CtVal <= 0 {
			return 0, false
		}
		mu.Lock()
		cache[instID] = ent{m.CtVal, time.Now()}
		mu.Unlock()
		return m.CtVal, true
	}
}

// okxFundingAvailUSDT funding 户 USDT 闲余（T-0149 ②——/asset/balances REST
// 低频件；入金天然落 funding 面钦定补漏；无行=0 不误报）。
func okxFundingAvailUSDT(ctx context.Context, okcli *okxexec.Client) (float64, error) {
	rows, err := okcli.AssetBalances(ctx)
	if err != nil {
		return 0, err
	}
	for _, r := range rows {
		if r.Ccy == "USDT" {
			var v float64
			fmt.Sscanf(r.AvailBal, "%f", &v)
			return v, nil
		}
	}
	return 0, nil
}
