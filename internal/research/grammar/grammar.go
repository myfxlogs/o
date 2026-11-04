// grammar.go —— 模板文法（T-0120/D-0016 Ⅱ）：腿原语×结算原语×组合律
// 枚举——族定义=闭集词汇表（非自由发明，同 T-0106 假想官纪律）；与
// fin_capability_surface enabled 面 join 生成 candidate 模板行。
package grammar

// LegPrim 腿原语（闭集——新增须随族定义同 commit 入库）。
type LegPrim string

const (
	SpotLong        LegPrim = "spot_long"
	SpotShort       LegPrim = "spot_short"
	PerpLong        LegPrim = "perp_long"
	PerpShort       LegPrim = "perp_short"
	QFutLong        LegPrim = "quarter_future_long"
	QFutShort       LegPrim = "quarter_future_short"
	LockedCcyCall   LegPrim = "locked_ccy_sell_call"
	LockedUsdtPut   LegPrim = "locked_usdt_sell_put"
	EarnDeposit     LegPrim = "earn_deposit"
	EarnTermDeposit LegPrim = "earn_term_deposit"
	LendingBorrow   LegPrim = "lending_borrow"
	SpotLongStaked  LegPrim = "spot_long_staked" // T-0320：质押态长现货（非裸持）
	CfdShort        LegPrim = "cfd_short"        // T-0320：CFD 空单（swap 付费 venue）
)

// Family 组合律族——模板名+腿原语集+结算原语+所需能力面（venue→面清单，
// 全位 enabled 才枚举）+executor 在位否（缺=假想卡自动落）。
type Family struct {
	Template    string
	Direction   string
	Kind        string // carry/basis/vol-sell
	Legs        []LegPrim
	Settle      string
	NeedFace    map[string][]string // venue→required enabled surface names
	Executor    string              // 空=缺 executor→假想卡
	Existing    bool                // 已落 enabled 模板（funding-hedge/dcd——文法映射不重复）
	PrimaryVen  string
	PrimarySurf string
}

// Families 闭集族表（funding-hedge/dcd 映射文法项——Existing 面不重生行）。
func Families() []Family {
	need := func(s ...string) map[string][]string { return map[string][]string{"okx": s} }
	return []Family{
		// ① carry 族（funding/借贷/Earn 利差）
		{Template: "funding-hedge", Direction: "short_perp_long_spot", Kind: "carry",
			Legs: []LegPrim{SpotLong, PerpShort}, Settle: "funding_8h",
			NeedFace: need("market-funding-rate", "trade-order"), Executor: "funding",
			Existing: true},
		{Template: "lending_spread", Direction: "lending_spread", Kind: "carry",
			Legs: []LegPrim{LendingBorrow, EarnDeposit}, Settle: "accrual",
			NeedFace: need("savings-balance", "savings-rate")},
		{Template: "earn_term_spread", Direction: "earn_term_spread", Kind: "carry",
			Legs: []LegPrim{EarnDeposit, EarnTermDeposit}, Settle: "accrual",
			NeedFace: need("savings-balance", "savings-rate")},
		{Template: "dcd_put", Direction: "dcd_put", Kind: "vol-sell",
			Legs: []LegPrim{LockedUsdtPut}, Settle: "expiry",
			NeedFace: need("dcd"), Executor: "dcd", Existing: true},
		{Template: "dcd_call", Direction: "dcd_call", Kind: "vol-sell",
			Legs: []LegPrim{LockedCcyCall}, Settle: "expiry",
			NeedFace: need("dcd"), Executor: "dcd", Existing: true},
		// ② basis 族（季度期货基差）
		{Template: "quarter_basis_long", Direction: "basis_quarter", Kind: "basis",
			Legs: []LegPrim{SpotLong, QFutShort}, Settle: "expiry",
			NeedFace: need("market-instruments", "market-tickers"),
			Executor: "quarter_basis_long_executor"}, // T-0216 executor 落位
		{Template: "quarter_basis_short", Direction: "basis_quarter_short", Kind: "basis",
			Legs: []LegPrim{SpotShort, QFutLong}, Settle: "expiry",
			// T-0216 条件缺口：spot_short 腿=保证金借币卖——margin-borrow 面在册
			// 但 enabled=false（读面待实测启用）→族不枚举=恒缺腿如实呈。
			NeedFace: need("market-instruments", "market-tickers", "margin-borrow-repay")},
		{Template: "perp_quarter_spread", Direction: "perp_quarter_spread", Kind: "basis",
			Legs: []LegPrim{PerpLong, QFutShort}, Settle: "expiry",
			NeedFace: need("market-funding-rate", "market-instruments", "market-tickers"),
			Executor: "perp_quarter_spread_executor"}, // T-0216 executor 落位；T-0216 返修：direction 去分叉——原与 quarter_basis_long 共用 basis_quarter，同 inst 双族撞去重闸 (inst_id,direction) 键=第二族提案被 md_1054 静默吞（提案去重语叉根修）
		// ③ vol-sell 族（DCD 外构型——T-0215 executor 落位：假想卡 id=6/7
		// 钦定名 vol_sell_strangle_executor/vol_sell_single_call_executor）
		{Template: "vol_sell_strangle", Direction: "vol_sell", Kind: "vol-sell",
			Legs: []LegPrim{LockedCcyCall, LockedUsdtPut}, Settle: "expiry",
			NeedFace: need("dcd"), Executor: "vol_sell_strangle_executor"},
		{Template: "vol_sell_single_call", Direction: "vol_sell", Kind: "vol-sell",
			Legs: []LegPrim{LockedCcyCall}, Settle: "expiry",
			NeedFace: need("dcd"), Executor: "vol_sell_single_call_executor"},
		// ④ mt5 面映射（catalog 注册但 enabled=false——面关=族不生锚）
		{Template: "mt5_swap_carry", Direction: "gap_carry", Kind: "carry",
			Legs: []LegPrim{SpotLong}, Settle: "swap_daily",
			NeedFace:   map[string][]string{"mt5": {"catalog", "symbols"}},
			PrimaryVen: "mt5", PrimarySurf: "catalog"},
		// ⑤ stake_carry 双头族（T-0320——质押现货+空单付费双腿同构业主
		// TRX 自营盘；NeedFace=二维积具体面：质押维注册+空单付费面注册。
		// executor 缺=假想卡（逐资产实例枚举归假想官 stakeCarryScan 臂）；
		// Kind 自立词——通用 carry 评估器不覆写（实例级评估归双头臂）。
		{Template: "stake_carry", Direction: "stake_carry", Kind: "stake_carry",
			Legs: []LegPrim{SpotLongStaked, CfdShort}, Settle: "swap_daily",
			NeedFace:   map[string][]string{"tron": {"staking-TRX"}, "cmc": {"swap-carry-TRX"}},
			PrimaryVen: "cmc", PrimarySurf: "swap-carry-TRX"},
	}
}

// Enumerate 族×enabled 面 join（注册即被枚举——全需面在=候选行；
// 任一需面关=族不生）。
func Enumerate(enabled map[string]bool) []Family {
	var out []Family
	for _, f := range Families() {
		ok := true
		for ven, sfs := range f.NeedFace {
			for _, sf := range sfs {
				if !enabled[ven+"/"+sf] {
					ok = false
				}
			}
		}
		if ok {
			if f.PrimaryVen == "" {
				f.PrimaryVen = "okx"
				for ven := range f.NeedFace {
					f.PrimaryVen = ven
					break
				}
			}
			if f.PrimarySurf == "" {
				for _, sfs := range f.NeedFace {
					if len(sfs) > 0 {
						f.PrimarySurf = sfs[0]
					}
				}
			}
			out = append(out, f)
		}
	}
	return out
}
