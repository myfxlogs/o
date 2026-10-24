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

	"phase/internal/guard"
	"phase/internal/ledger"
	"phase/internal/mt5"
	"phase/internal/risk"
	"phase/internal/trading/okxexec"
)

// attachVenueReaders venue 现读件装配（T-0048 返修②口径钉死——快照行全域出局）：
//   - okx 权益/闲余=md_account_events 最新 account/balance_and_position 帧解
//     （details eqUsd 总权益/availBal 闲余；push-on-change——帧龄不判陈，陈度归 WS 活性件管）。
//   - okx 对账聚合=事件帧交易户 eq+REST /asset/balances(funding)+/finance/savings/balance(amt)
//     现读合 {okx,USDT}——资金/停泊户无推送面（skill 在案）。
//   - mt5 权益=AccountSummary.Equity 现读。
//
// 凭据缺席=boot 明示 WARN 不静默缺岗；各闭包判 nil 不 panic。
func attachVenueReaders(riskGate *risk.Gate, wd *guard.Warden, ledgerR *ledger.Runner,
	pool *pgxpool.Pool, okcli *okxexec.Client, openMt5 func(ctx context.Context) (*mt5.Client, error)) {
	// frameEq/frameAvail 事件帧现读（okx 交易户——数据域落行的同源读，REST 非判据源）。
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
					if d.Ccy == "USDT" {
						eq = parseNum(d.EqUsd)
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
				if d.Ccy == "USDT" {
					eq = parseNum(d.EqUsd)
					avail = parseNum(d.AvailBal)
				}
			}
			return eq, avail, nil
		}
		return parseNum(top2.EqUsd), parseNum(top2.AvailBal), nil
	}
	if okcli == nil {
		log.Printf("WARN: venue 现读件未挂接（okx 凭据缺席）——risk equity/dd 硬线/对账 fail-closed 让位")
		return
	}
	// okx 权益=事件帧现读（push-on-change——不判帧龄）。
	liveEq := func(ctx context.Context) (float64, error) {
		eq, _, err := frameVals(ctx)
		return eq, err
	}
	riskGate.WithEquity(func(ctx context.Context, venue string) (float64, error) {
		if venue == "mt5" {
			if openMt5 == nil {
				return 0, fmt.Errorf("mt5 权益件缺席")
			}
			cl, err := openMt5(ctx)
			if err != nil {
				return 0, err
			}
			defer cl.Close()
			e, _, _, err := cl.AccountSummary(ctx)
			return e, err
		}
		if venue != "okx" {
			return 0, fmt.Errorf("venue 无权益件: %s", venue)
		}
		// T-0050 停泊态口径：venue 权益=交易户帧 eq+funding 户+Earn 停泊（D-0006
		// 资本可及面——闲资默认停 Earn，只读交易户 eq=停泊态下集中度闸永拒）。
		eq, _, err := frameVals(ctx)
		if err != nil {
			return 0, err
		}
		if okcli != nil {
			if bals, err := okcli.AssetBalances(ctx); err == nil {
				for _, b := range bals {
					eq += okxexec.ParseF(b.Bal)
				}
			}
			if sav, err := okcli.SavingsBalance(ctx); err == nil {
				var rows []struct {
					Amt string `json:"amt"`
				}
				if json.Unmarshal(sav, &rows) == nil {
					for _, r2 := range rows {
						eq += okxexec.ParseF(r2.Amt)
					}
				}
			}
		}
		return eq, nil
	})
	wd.AttachEquity(liveEq)
	// 对账聚合=帧 eq+REST funding+savings（无推送户）+mt5 AccountSummary。
	ledgerR.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		if eq, _, err := frameVals(ctx); err == nil && eq != 0 {
			out[[2]string{"okx", "USDT"}] += eq
		}
		if okcli != nil {
			if bals, err := okcli.AssetBalances(ctx); err == nil {
				for _, b := range bals {
					out[[2]string{"okx", b.Ccy}] += okxexec.ParseF(b.Bal)
				}
			}
			if sav, err := okcli.SavingsBalance(ctx); err == nil {
				var rows []struct {
					Ccy string `json:"ccy"`
					Amt string `json:"amt"`
				}
				if json.Unmarshal(sav, &rows) == nil {
					for _, r2 := range rows {
						out[[2]string{"okx", r2.Ccy}] += okxexec.ParseF(r2.Amt)
					}
				}
			}
		}
		if openMt5 != nil {
			if cl, err := openMt5(ctx); err == nil {
				if e, _, _, err := cl.AccountSummary(ctx); err == nil {
					out[[2]string{"mt5", "USD"}] = e
				}
				cl.Close()
			}
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
func okxAvailFromEvents(ctx context.Context, pool *pgxpool.Pool) (float64, error) {
	var data json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel='account' ORDER BY ts DESC LIMIT 1`).Scan(&data); err != nil {
		return 0, err
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
