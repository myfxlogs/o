package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"phase/internal/guard"
	"phase/internal/ledger"
	"phase/internal/mt5"
	"phase/internal/risk"
	"phase/internal/trading/okxexec"
)

// attachVenueReaders venue 现读件装配（T-0048：快照行全域出局——决策/判定/对账
// 时点实读；okcli 缺=凭据未位，risk equity/dd 硬线/对账各面 fail-closed 让位）。
func attachVenueReaders(riskGate *risk.Gate, wd *guard.Warden, ledgerR *ledger.Runner,
	okcli *okxexec.Client, openMt5 func(ctx context.Context) (*mt5.Client, error)) {
	if okcli == nil {
		// 缺岗明示（T-0048 返修①：静默缺岗禁——凭据缺席期三读件全让位但可闻）。
		log.Printf("WARN: venue 现读件未挂接（okx 凭据缺席）——risk equity/dd 硬线/对账 fail-closed 让位")
		return
	}
	liveEq := func(ctx context.Context) (float64, error) {
		m, err := okxexec.BalanceEquity(ctx, okcli.Creds(), false)
		if err != nil {
			return 0, err
		}
		var tot float64
		for _, v := range m {
			tot += v
		}
		return tot, nil
	}
	riskGate.WithEquity(func(ctx context.Context, venue string) (float64, error) {
		if venue != "okx" {
			return 0, fmt.Errorf("venue 无权益件: %s", venue)
		}
		return liveEq(ctx)
	})
	wd.AttachEquity(liveEq)
	ledgerR.WithBalances(func(ctx context.Context) (map[[2]string]float64, error) {
		out := map[[2]string]float64{}
		eq, err := okxexec.BalanceEquity(ctx, okcli.Creds(), false)
		if err != nil {
			return nil, err
		}
		for c, v := range eq {
			out[[2]string{"okx", c}] += v
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
		// T-0048 返修③：mt5 侧=AccountSummary equity {mt5,USD}（双 venue 对账面）。
		if openMt5 != nil {
			if cl, err := openMt5(ctx); err == nil {
				defer cl.Close()
				if eq5, _, _, err := cl.AccountSummary(ctx); err == nil {
					out[[2]string{"mt5", "USD"}] = eq5
				}
			}
		}
		return out, nil
	})
}
