package legset

import (
	"testing"

	"o/internal/platform/execfam"
)

// TestInstFitOKFamilyContract 工单#15943 锚：族→inst 构型契约全谱——
// funding-hedge/xvenue-carry=USDT 永续形；basis 两族=季度远期形；
// dcd/vol_sell 族=DCD 期权形（call/put/strangle 细分）。
func TestInstFitOKFamilyContract(t *testing.T) {
	cases := []struct {
		tmpl, inst string
		want       bool
	}{
		// funding-hedge：USDT 本位永续=合；DCD 期权形/币本位 SWAP/现货对/伪键=拒
		{"funding-hedge", "BTC-USDT-SWAP", true},
		{"funding-hedge", "ETH-USDT-SWAP", true},
		{"funding-hedge", "BETH-USDT-261113-2800-C", false}, // 病灶实证形
		{"funding-hedge", "BTC-USD-SWAP", false},            // D-0018 USDT 计价闸
		{"funding-hedge", "BTC-USDT", false},                // 裸现货对非主腿形
		{"funding-hedge", "okx/earn-flex", false},           // 伪键形
		// xvenue-carry/dir_perp_mom：okx 主腿同 USDT 永续形
		{"xvenue-carry", "BTC-USDT-SWAP", true},
		{"xvenue-carry", "BETH-USDT-261113-2800-C", false},
		{"dir_perp_mom", "BTC-USDT-SWAP", true},
		{"dir_perp_mom", "BTC-USDT-261225-95000-C", false},
		// basis 两族：季度远期形（币本位/USDT 本位皆可）
		{"quarter_basis_long", "BTC-USD-270326", true},
		{"quarter_basis_long", "BTC-USDT-261225", true},
		{"quarter_basis_long", "BTC-USDT-SWAP", false},
		{"perp_quarter_spread", "BTC-USD-261225", true},
		{"perp_quarter_spread", "BETH-USDT-261113-2800-C", false},
		// dcd 族：期权形+类型细分（QUOTE 域 RLUSD 等不收窄——实测在案）
		{"dcd_call", "BTC-USDT-261225-95000-C", true},
		{"dcd_call", "BTC-RLUSD-261026-80250-C", true},
		{"dcd_call", "BTC-USDT-261225-68000-P", false},
		{"dcd_call", "BTC-USDT-SWAP", false},
		{"dcd_put", "BTC-USDT-261225-68000-P", true},
		{"dcd_put", "BTC-USDT-261225-95000-C", false},
		{"vol_sell_single_call", "DOGE-USDT-261030-0.0752-C", true},
		{"vol_sell_single_call", "BETH-USDT-261030-2200-P", false},
		{"vol_sell_strangle", "BETH-USDT-261029-2325-C+ETH-RLUSD-261029-2330-P", true},
		{"vol_sell_strangle", "BETH-USDT-261029-2325-C", false}, // strangle=双腿复合形
		// sprd-atomic-basis：sprdId 双腿 '_' 连接形（工单#26268——官方
		// docs 实证 BTC-USDT_BTC-USDT-SWAP/ETH-USD-SWAP_ETH-USD-231229）
		{"sprd-atomic-basis", "BTC-USDT_BTC-USDT-SWAP", true},            // 现货×永续
		{"sprd-atomic-basis", "ETH-USD-SWAP_ETH-USD-231229", true},       // 永续×远期（USD 结算 inverse 形不收窄）
		{"sprd-atomic-basis", "BTC-USDT_BTC-USD-SWAP", false},            // 异结算币（USDT vs USD）拒
		{"sprd-atomic-basis", "BTC-USDT_ETH-USDT-SWAP", false},           // 跨基腿币拒
		{"sprd-atomic-basis", "BTC-USDT_BTC-USDT", false},                // 同腿退化形拒
		{"sprd-atomic-basis", "BTC-USDT", false},                         // 单腿非价差形拒
		{"sprd-atomic-basis", "BTC-USDT_BTC-USDT-SWAP_EXTRA", false},     // 三段腿拒
		{"sprd-atomic-basis", "BTC-USDT_BTC-USDT-261225-95000-C", false}, // 期权腿拒
		{"sprd-atomic-basis", "BTC-USDT_btc-usdt-swap", false},           // 腿内小写形拒
		// stable-park-usdg：okx:USDG 资产级伪键=合；他币伪键/交易对/SWAP=拒
		{"stable-park-usdg", "okx:USDG", true},
		{"stable-park-usdg", "okx:USDT", false},  // 伪键他币非本族标的
		{"stable-park-usdg", "cmc:USDG", false},  // venue 前缀钉 okx
		{"stable-park-usdg", "USDG-USDT", false}, // 交易对非伪键形
		{"stable-park-usdg", "USDG-USDT-SWAP", false},
		// staking-defi-parking：okx:ETH/okx:SOL 资产级伪键=合；他币/异 venue/交易对=拒
		{"staking-defi-parking", "okx:ETH", true},
		{"staking-defi-parking", "okx:SOL", true},
		{"staking-defi-parking", "okx:BTC", false},  // 卡面钦定两币收窄
		{"staking-defi-parking", "cmc:ETH", false},  // venue 前缀钉 okx
		{"staking-defi-parking", "ETH-USDT", false}, // 交易对非伪键形
		{"staking-defi-parking", "ETH-USDT-SWAP", false},
		// signal-bot-relay：信号载体=order-algo instIds USDT 永续形=合；币本位/现货/远期/伪键=拒
		{"signal-bot-relay", "BTC-USDT-SWAP", true},
		{"signal-bot-relay", "BTC-USD-SWAP", false},   // D-0018 USDT 计价闸
		{"signal-bot-relay", "BTC-USDT", false},       // 现货对非载体形
		{"signal-bot-relay", "BTC-USD-261225", false}, // 远期形非 instIds 域
		{"signal-bot-relay", "okx:BTC", false},        // 伪键非载体形
		// mt5_dir_mom：venue='mt5' CFD 官名形=合（formOther 域非 OKX inst 形非伪键）；
		// OKX inst 形/伪键/期权形=拒（O-35163 构型闸）
		{"mt5_dir_mom", "XAUUSDm", true},
		{"mt5_dir_mom", "BTCUSDm", true},
		{"mt5_dir_mom", "US30_x10m", true},
		{"mt5_dir_mom", "BTC-USDT-SWAP", false},  // OKX 永续形非 CFD 官名域
		{"mt5_dir_mom", "BTC-USD-261225", false}, // 远期形拒
		{"mt5_dir_mom", "okx:TRX", false},        // venue:ccy 伪键拒
		// O-35489 三族骨架：USDT 本位永续构型=合；现货对/币本位/远期/伪键=拒
		{"xsec-perp-mom-basket", "BTC-USDT-SWAP", true},
		{"xsec-perp-mom-basket", "BTC-USD-SWAP", false}, // 币本位拒（D-0018）
		{"xsec-perp-mom-basket", "BTC-USDT", false},     // 现货对非 perp 构型
		{"funding-extreme-fade", "ETH-USDT-SWAP", true},
		{"funding-extreme-fade", "ETH-USD-SWAP", false},
		{"mark-last-disloc-fade", "SOL-USDT-SWAP", true},
		{"mark-last-disloc-fade", "okx:SOL", false}, // 伪键非构型
		// 未注册族 fail-closed
		{"unsupported-template", "BTC-USDT-SWAP", false},
	}
	for _, c := range cases {
		if got := InstFitOK(c.tmpl, c.inst); got != c.want {
			t.Errorf("InstFitOK(%q,%q)=%v want %v", c.tmpl, c.inst, got, c.want)
		}
	}
}

// TestInstFitOKCoversExecFam 扩族覆盖断言：execfam 册每族必有合法 inst
// 构型（漏登=本表无键=红——扩族单点登记语义延伸至构型核）。
func TestInstFitOKCoversExecFam(t *testing.T) {
	valid := map[string]string{
		"funding-hedge":           "BTC-USDT-SWAP",
		"xvenue-carry":            "BTC-USDT-SWAP",
		"dir_perp_mom":            "BTC-USDT-SWAP",
		"perp-trend-daily":        "BTC-USDT-SWAP",
		"quarter_basis_long":      "BTC-USD-270326",
		"perp_quarter_spread":     "BTC-USD-261225",
		"quarter-calendar-spread": "BTC-USD-270327",
		"dcd_call":                "BTC-USDT-261225-95000-C",
		"dcd_put":                 "BTC-USDT-261225-68000-P",
		"vol_sell_single_call":    "DOGE-USDT-261030-0.0752-C",
		"vol_sell_strangle":       "BETH-USDT-261029-2325-C+ETH-RLUSD-261029-2330-P",
		"stake-carry":             "okx:TRX",
		"grid_spot":               "BTC-USDT",
		"sprd-atomic-basis":       "BTC-USDT_BTC-USDT-SWAP",
		"stable-park-usdg":        "okx:USDG",
		"staking-defi-parking":    "okx:ETH",
		"signal-bot-relay":        "BTC-USDT-SWAP",
		"mt5_dir_mom":             "XAUUSDm",
		"xsec-perp-mom-basket":    "BTC-USDT-SWAP",
		"funding-extreme-fade":    "BTC-USDT-SWAP",
		"mark-last-disloc-fade":   "BTC-USDT-SWAP",
	}
	for _, tmpl := range execfam.All() {
		inst, ok := valid[tmpl]
		if !ok {
			t.Fatalf("execfam 族 %q 无 inst 构型契约登记——扩族须同步 legset.InstFitOK", tmpl)
		}
		if !InstFitOK(tmpl, inst) {
			t.Fatalf("execfam 族 %q 契约 inst %q 被自家闸拒", tmpl, inst)
		}
	}
}
