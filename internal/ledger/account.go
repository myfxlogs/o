package ledger

// account.go —— 账户维解析面（工单O-39440——「以账户来分」根修）：
// fin_account 注册表=账户身份真源；account_id=venue:account 域键
// （okx:trading/funding/earn/dcd 子户+mt5:<login> 户号+<venue>:legacy
// 兜底）。写入径经 AccountFor 归户——显式 account 优先,缺省按
// (venue,kind) 显式规则,不可判→<venue>:legacy 终身标禁猜。

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountRow fin_account 投影行（读面纯件——写径只INSERT不UPDATE
// 户号,身份不可变铁律在迁移注释+spec 双在案）。
type AccountRow struct {
	ID        string
	Venue     string
	AccountNo string
	Alias     string
	Ccy       string
	SubType   string
	State     string
}

// AccountRows 账户册读面（state 过滤可选——""=全量;"active"=对拍/
// 实读面,legacy frozen 行天然缺席）。读败=err 上抛不静默。
func AccountRows(ctx context.Context, pool *pgxpool.Pool, state string) ([]AccountRow, error) {
	q := `SELECT account_id,venue,account_no,alias,ccy,sub_type,state FROM fin_account`
	args := []any{}
	if state != "" {
		q += ` WHERE state=$1`
		args = append(args, state)
	}
	q += ` ORDER BY account_id`
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountRow
	for rows.Next() {
		var a AccountRow
		if err := rows.Scan(&a.ID, &a.Venue, &a.AccountNo, &a.Alias, &a.Ccy, &a.SubType, &a.State); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// VenueOf account_id→venue 折叠（首冒号前段——与既有 scope 键折叠
// 同式；account_id 本身即 venue 前缀故恒等价,保留函数防散写）。
func VenueOf(accountID string) string {
	if i := strings.IndexByte(accountID, ':'); i >= 0 {
		return accountID[:i]
	}
	return accountID
}

// acctKindRules (venue,kind)→账户 显式映射正本（迁移 fin_0256 回填
// 同规则——两面同源防漂；kind 闭集外→<venue>:legacy）。
var acctKindRules = map[string]map[string]string{
	"okx": {
		"deposit": "okx:funding", "withdraw": "okx:funding",
		"owner_in": "okx:funding", "owner_out": "okx:funding",
		"earn_in": "okx:earn", "earn_out": "okx:earn", "interest": "okx:earn",
	},
}

// AccountFor 账户解析：explicit 非空=直接用（FK 闸兜底合法性）；
// 缺省走 (venue,kind) 规则——dcd_* kind 前缀→<venue>:dcd,okx 余者
// →okx:trading,非 okx/未知→<venue>:legacy。返回值恒非空。
func AccountFor(venue, kind, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if venue == "exness" {
		return "exness:3001157" // 迁移回填同规则——Exness 簿记链实证（O-39440① venue 改名）
	}
	if venue == "xmglobal" {
		return "xmglobal:110660304" // XM 户——roster venue token=xmglobal
	}
	if venue == "okx" {
		if r, ok := acctKindRules["okx"][kind]; ok {
			return r
		}
		if strings.HasPrefix(kind, "dcd_") || kind == "dcd_owner" {
			return "okx:dcd"
		}
		if kind == "recon_adjust" || kind == "unmapped_bill" {
			return "okx:legacy"
		}
		return "okx:trading"
	}
	return venue + ":legacy"
}

// compAccount 对拍组件→账户归属（parked→earn、dcdBase→dcd、
// nativeBase/mtm→trading；venue 无对应 sub_type=不入任何账户——
// 与今日 venue 桶无对应键同果,覆盖不丢不增）。sub='trading' 时
// 单 main 账户 venue 兜底（mt5 类）——多 main=歧义不归属返回 ""。
func compAccount(accts []AccountRow, venue, sub string) string {
	mains := 0
	main := ""
	for _, a := range accts {
		if a.Venue != venue || a.State != "active" {
			continue
		}
		if a.SubType == sub {
			return a.ID
		}
		if a.SubType == "main" {
			mains++
			main = a.ID
		}
	}
	if sub == "trading" && mains == 1 {
		return main
	}
	return ""
}

// foldAcctKeys 账户域键→venue 域折叠（基线武装径用——native_baseline
// intent 域键=venue+ccy 原样；eq 求和、wl 存在性折叠）。
func foldAcctKeys(ven map[[2]string]float64, wl map[[2]string]bool) (venV map[[2]string]float64, wlV map[[2]string]bool) {
	venV = map[[2]string]float64{}
	for k, v := range ven {
		venV[[2]string{VenueOf(k[0]), k[1]}] += v
	}
	wlV = map[[2]string]bool{}
	for k := range wl {
		wlV[[2]string{VenueOf(k[0]), k[1]}] = true
	}
	return venV, wlV
}

// acctParts VenueParts 账户键优先、venue 键兜底（parts 键面=账户域——
// O-39440；旧 venue 键写入方过渡期兼容）。
func acctParts(parts VenueParts, acct, ccy string) []BalPart {
	if p, ok := parts[[2]string{acct, ccy}]; ok {
		return p
	}
	return parts[[2]string{VenueOf(acct), ccy}]
}
