// sources.go —— 两 venue 采集源实现（okx=account/config+balance+instruments；
// mt5=Account/AccountSummary+SymbolParams 逐只）。facts 数据域包直引 venue 适配件
// （mt5 同域；okxexec=earn 先例边——白名单登记）。
package facts

import (
	"context"
	"encoding/json"
	"fmt"

	"phase/internal/mt5"
	"phase/internal/trading/okxexec"
)

// OkxSource OKX 户层+乐器层（户层=/account/config 实勘——posMode/acctLv 等只读；
// balance=/account/balance ccy 明细；乐器=instruments 按 okx.symbols 逐只）。
type OkxSource struct {
	Cli     *okxexec.Client
	Symbols func() []string // okx.symbols 热启（装配注入）
}

func (o *OkxSource) Venue() string { return "okx" }

func (o *OkxSource) Collect(ctx context.Context) ([]Fact, error) {
	raw, err := o.Cli.Get(ctx, "/api/v5/account/config")
	if err != nil {
		return nil, fmt.Errorf("account/config: %w", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("account/config parse: %w", err)
	}
	cfg := rows[0]
	var out []Fact
	// 户层全谱（实勘键集——posMode/acctLv/level/kycLv/uid 等，值原样入 jsonb）。
	for _, k := range []string{"posMode", "acctLv", "mgnIsoMode", "level", "kycLv", "uid", "ip", "perm", "ctIsoMode", "roleType"} {
		if v, ok := cfg[k]; ok {
			out = append(out, Fact{Kind: "account", Key: k, Value: v})
		}
	}
	if bal, err := okxexec.BalanceEquity(ctx, o.Cli.Creds(), false); err == nil {
		for ccy, eq := range bal {
			out = append(out, Fact{Kind: "account", Key: "balance:" + ccy, Value: eq})
		}
	}
	for _, sym := range o.Symbols() {
		it := "SPOT"
		if len(sym) > 5 && sym[len(sym)-5:] == "-SWAP" {
			it = "SWAP"
		}
		m, err := o.Cli.InstrumentMeta(ctx, it, sym)
		if err != nil {
			continue // 单只缺=计数不连坐（collectAll 面已计总量）
		}
		out = append(out, Fact{Kind: "instrument", Key: "inst:" + sym, Value: map[string]any{
			"ctVal": m.CtVal, "ctValCcy": m.CtValCcy, "lotSz": m.LotSz,
			"minSz": m.MinSz, "tickSz": m.TickSz, "state": m.State,
		}})
	}
	return out, nil
}

// Mt5Source MT5 户层（Account/AccountSummary——leverage/balance/equity）+
// 乐器层（SymbolParams 逐只——ContractSize/MinLots/LotsStep/TradeMode/spread/digits）。
type Mt5Source struct {
	Open    func(ctx context.Context) (*mt5.Client, error) // 独立会话开件（开连+登录；用毕 Disconnect）
	Symbols func() []string                                // mt5.symbols 热启
}

func (m *Mt5Source) Venue() string { return "mt5" }

func (m *Mt5Source) Collect(ctx context.Context) ([]Fact, error) {
	cl, err := m.Open(ctx)
	if err != nil {
		return nil, fmt.Errorf("mt5 会话: %w", err)
	}
	defer cl.Disconnect(context.Background())
	login, lev, ccy, err := cl.AccountFacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("mt5 Account: %w", err)
	}
	eq, fm, bal, err := cl.AccountSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("mt5 AccountSummary: %w", err)
	}
	out := []Fact{
		{Kind: "account", Key: "login", Value: login},
		{Kind: "account", Key: "leverage", Value: lev},
		{Kind: "account", Key: "ccy", Value: ccy},
		{Kind: "account", Key: "equity", Value: eq},
		{Kind: "account", Key: "free_margin", Value: fm},
		{Kind: "account", Key: "balance", Value: bal},
	}
	for _, sym := range m.Symbols() {
		sp, err := cl.SymbolFacts(ctx, sym)
		if err != nil {
			continue
		}
		out = append(out, Fact{Kind: "instrument", Key: "inst:" + sym, Value: sp})
	}
	return out, nil
}
