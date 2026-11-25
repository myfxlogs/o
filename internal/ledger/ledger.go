// Package ledger —— 账本域（G9 单源货币真相）。
// 铁律：fin_cashflow 每行强制 source_ref+source_table（禁推断记账）；kind 闭集
// +ccy 原生币形闸（O-34287 业主令：各币各域零换算——OKX 原生币全收，
// MT5=USD 单币照旧）；净额=Σcashflow；
// 对账拿 venue 私有流快照交叉，差即告警计数；账本不触发钱动作。
package ledger

import (
	"context"
	"fmt"
	"math"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
)

var kinds = map[string]bool{
	"fee": true, "funding_settle": true, "trade_pnl": true,
	"rollback_cost": true, "deposit": true, "withdraw": true,
	// Earn 停泊款移动（D-0006）：earn_out=出交易户进停泊（amount 负），earn_in=赎回回拢（amount 正）。
	"earn_in": true, "earn_out": true,
	// T-0055：interest=Earn 利息（earnings 累计值事件化入账）；
	// recon_adjust=基线校准一次性行（书面归因后真值校准——禁日常使用）。
	"interest": true, "recon_adjust": true,
	// T-0083 腿级现金流对称：spot_buy=现货腿买入流出毛额（负）、spot_sell=
	// 卖出流入毛额（正）——两腿各记净额自然=真价差；损益派生读面不单写。
	"spot_buy": true, "spot_sell": true,
	// T-0364：owner_in/owner_out=业主域资产收/支（dcd 收归独立仓——venue
	// 共户托管账面归口；class=transfer 非策略线，earn_* 排除面不含=照入
	// venue 净额对拍）。
	"owner_in": true, "owner_out": true,
	// 升件#14671：bill 未映射子型域 kind（fin_0075 册载而 DB/Go 闭集缺位
	// 零写者病灶）——dust_*=残币归口(Easy convert)、supply_income=理财收益、
	// airdrop=空投/奖励、unmapped_bill=残差守恒落账位（adjustment 待归因）。
	"dust_in": true, "dust_out": true, "supply_income": true,
	"airdrop": true, "unmapped_bill": true,
	// O-33362：dcd_out/dcd_in=双币赢申购扣款/结算回款本金腿（transfer
	// counted——earn_* 排除面外照入 net；与 dcd_baseline 凭据配对互抵）。
	"dcd_out": true, "dcd_in": true,
}

// ccyRE 原生币种形（O-34287「各币各域」——OKX 原生币全收：BTC/DOGE/
// 1000PEPE 等 venue 真报币种按本域入账零换算；形闸防垃圾串）。
var ccyRE = regexp.MustCompile(`^[A-Z0-9]{2,20}$`)

// CcyOK 币种域合法判（O-34287：venue 原生币各记各域——mt5 恒 USD 单币
// 照旧；其余 venue=原生币形全放。库约束 fin_0235 同形镜像）。
func CcyOK(venue, ccy string) bool {
	if venue == "exness" || venue == "xmglobal" {
		return ccy == "USD" // cfd 经纪商 USD 单币域（O-39440①——exness/xmglobal 同律）
	}
	return ccyRE.MatchString(ccy)
}

// ccyAllowed Post 写径币闭集（注册表驱动版——roster proto='cfd'
// 判 USD 单域;CcyOK 存量签名留 okx 专用径 dcdlegs 兼容）。
func ccyAllowed(ctx context.Context, pool *pgxpool.Pool, venue, ccy string) (bool, error) {
	var cfd bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fin_venue_roster WHERE venue=$1 AND proto='cfd')`,
		venue).Scan(&cfd); err != nil {
		return ccyRE.MatchString(ccy), nil // 名册读败 fail-open 通用闭集
	}
	if cfd {
		return ccy == "USD", nil
	}
	return ccyRE.MatchString(ccy), nil
}

// Entry 入账入参。
type Entry struct {
	Kind        string
	Venue       string
	Ccy         string
	Amount      float64
	PositionRef *int64
	SourceRef   int64
	SourceTable string
	OccurredAt  string // timestamptz 文本（测试直控）
	// AccountID 账户域键（O-39440——空=按 (venue,kind) 规则归户;
	// 知户写径直填如 'mt5:3001157',禁再落无账户行）。
	AccountID string
}

// Post 入账：闭集+来源行校验（source_ref<=0 或杂币/kind 非法=拒写上抛）。
func Post(ctx context.Context, pool *pgxpool.Pool, e Entry) (int64, error) {
	if !kinds[e.Kind] {
		return 0, fmt.Errorf("kind %q 非法闭集外", e.Kind)
	}
	if ok, err := ccyAllowed(ctx, pool, e.Venue, e.Ccy); err != nil {
		return 0, err
	} else if !ok {
		return 0, fmt.Errorf("ccy %q 非法——各币各域闭集外", e.Ccy)
	}
	if e.SourceRef <= 0 || e.SourceTable == "" {
		return 0, fmt.Errorf("缺 source_ref/source_table——禁推断记账")
	}
	var id int64
	var ts any
	if e.OccurredAt != "" {
		ts = e.OccurredAt
	}
	var acct any // 显式 account_id 直通；空=NULL 交触发器注册表归户
	if e.AccountID != "" {
		acct = e.AccountID
	}
	err := pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at,account_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,COALESCE($8,now()),$9) RETURNING id`,
		e.Kind, e.Venue, e.Ccy, e.Amount, e.PositionRef, e.SourceRef, e.SourceTable, ts,
		acct).Scan(&id)
	return id, err
}

// NetAmount venue×ccy 净额=Σcashflow（一仓一判决书=position_ref 过滤同式）。
func NetAmount(ctx context.Context, pool *pgxpool.Pool, venue, ccy string, positionRef *int64) (float64, error) {
	q := "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE venue=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')"
	args := []any{venue, ccy}
	if positionRef != nil {
		q += " AND position_ref=$3"
		args = append(args, *positionRef)
	}
	var net float64
	err := pool.QueryRow(ctx, q, args...).Scan(&net)
	return net, err
}

// Breakdown 构成明细（判决书=净额+kind 分桶构成可查）。
func Breakdown(ctx context.Context, pool *pgxpool.Pool, positionRef int64) (map[string]float64, error) {
	rows, err := pool.Query(ctx,
		"SELECT kind, sum(amount) FROM fin_cashflow WHERE position_ref=$1 AND ccy IN ('USDT','USD') GROUP BY kind", positionRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var k string
		var v float64
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Diff 对账差（venue 快照 equity − 账本净额绝对差）。
func Diff(ledgerNet, venueEquity float64) float64 {
	return math.Abs(ledgerNet - venueEquity)
}
