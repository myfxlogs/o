// money_card.go —— 钱在哪卡面件（T-0086 ②三分卡起版；T-0095 v2 口径收紧+
// T-0096 venue 分域后=venue→分区→行三级：行组装/在册集/amt 估值）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"o/internal/trading/okxexec"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/ledger"
)

// moneyRow 组装（T-0095 v2：零遗留概念——集合外托管行不组装不返不渲染；
// T-0096：venue 一级+note 旁注〔读件缺等如实标注〕）。
func moneyRow(venue, section, ccy, label string, amt, usd float64) *dashboardv1.MoneyRow {
	return &dashboardv1.MoneyRow{Label: label, Amount: amt, Usd: usd,
		Venue: venue, Section: section, Ccy: ccy}
}

// fundCcy 基金的钱币种集·作用域过滤（T-0100 ①：在册集账户域作用——trading
// 域=USDT∪open 腿基币、funding/earn 域=USDT∪停泊溯源币；ledger.FundCcy
// 唯一真源——域外同币=托管尘非基金钱）。
func (c coreService) fundCcy(ctx context.Context, scope string) map[string]bool {
	out := map[string]bool{}
	set, err := ledger.FundCcy(ctx, c.pool)
	if err != nil {
		return out
	}
	for k := range set {
		if k[0] == scope {
			out[k[1]] = true
		}
	}
	return out
}

// dcdOrds dcd LIVE 单现读共用件（T-0105：钱卡/在干列表双消费——T-0101 60s
// TTL 单飞缓存已防限流撞车；读件缺/读败=(nil,false) 显形径自裁，(ords,true)=
// 在持单真值）。
func (c coreService) dcdOrds(ctx context.Context) ([]okxexec.DcdOrder, bool) {
	if c.dcdLive == nil {
		return nil, false
	}
	ords, err := c.dcdLive(ctx)
	if err != nil {
		log.Printf("WARN: dcd 持仓活读败（卡面/在干显读件缺）: %v", err)
		return nil, false
	}
	return ords, true
}

// dcdRows 结构性持仓区行组装（T-0101：LIVE 单=已部署产品持仓=基金的钱第三
// 类别——productId/ordId/名义/行权/到期/年化全入 label〔note 位=读件缺专用，
// 非空即无额行，持仓行不占用〕；usd=名义×md_ticks 估值入 NAV；读件缺/读败
// =note 行显形不假零）。
func (c coreService) dcdRows(ctx context.Context, o *dashboardv1.OverviewResponse) {
	ords, ok := c.dcdOrds(ctx)
	if !ok {
		o.Money = append(o.Money, &dashboardv1.MoneyRow{
			Venue: "okx", Section: "结构性持仓", Label: "读件缺", Note: "读件缺"})
		return
	}
	for _, od := range ords {
		sz := okxexec.ParseF(od.NotionalSz)
		u := valCcy(ctx, c.pool, od.NotionalCcy, sz)
		o.NavTotal += u
		exp := msDay(od.SettleTime)
		apy := okxexec.ParseF(od.AnnualizedYield) * 100
		o.Money = append(o.Money, moneyRow("okx", "结构性持仓", od.NotionalCcy,
			fmt.Sprintf("%s #%s 行权%s 到期%s 年化%.1f%%",
				od.ProductID, od.OrdID, od.Strike, exp, apy), sz, u))
	}
}

// valCcy 代币数→USD 估值（T-0086 ① amt估值）：USDT/USD 直入；其余=md_ticks
// 最新价×量（无 tick=0——卡面仍展示，NAV 不虚计 51M NFT 级原生数）。
func valCcy(ctx context.Context, pool *pgxpool.Pool, ccy string, units float64) float64 {
	if ccy == "USDT" || ccy == "USD" {
		return units
	}
	var last *float64
	_ = pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1`,
		ccy+"-USDT").Scan(&last)
	if last == nil {
		return 0
	}
	return units * *last
}

// moneyRows NAV 与三区卡面组装（T-0100 ①② 基金的钱口径账户域作用——
// FundCcy 唯一真源：交易户=USDT∪open 腿基币、资金户/Earn=USDT∪停泊溯源
// 币，域外同币=托管尘不返不渲染不入 NAV；T-0096 MT5 户逐真实账号行；
// T-0101 dcd 结构性持仓区）。交易户=WS 帧 eqUsd，资金户/Earn=REST 活读
// amt 估值（缺户=0 不误报）；读件缺/读败=显「读件缺」不假零。
func (c coreService) moneyRows(ctx context.Context, o *dashboardv1.OverviewResponse) {
	// T-0100 ①②：NAV 与三区卡面=基金的钱口径账户域作用（FundCcy 唯一真源）
	// ——交易户=USDT∪open 腿基币；资金户/Earn=USDT∪停泊溯源币（residual_park
	// 凭据）；域外同币（Earn 利息尘/腿基币渗漏）=托管尘不返不渲染不入 NAV。
	// 交易户=WS 帧真源 eqUsd，资金户/Earn=REST 活读 amt 估值（缺户=0 不误报）。
	fundT := c.fundCcy(ctx, "okx:trading")
	fundF := c.fundCcy(ctx, "okx:funding")
	fundE := c.fundCcy(ctx, "okx:earn")
	var frame json.RawMessage
	if err := c.pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&frame); err == nil {
		var top struct {
			Details []struct {
				Ccy   string `json:"ccy"`
				EqUsd string `json:"eqUsd"`
				Eq    string `json:"eq"` // T-0114 甲A：原生持币量（卡面量纲归一原生数非美元值）
			} `json:"details"`
		}
		if json.Unmarshal(frame, &top) == nil {
			for _, d := range top.Details {
				v, _ := strconv.ParseFloat(d.EqUsd, 64)
				if v == 0 || !fundT[d.Ccy] {
					continue
				}
				nat, _ := strconv.ParseFloat(d.Eq, 64)
				if nat == 0 {
					nat = v // eq 字段缺席=回退 eqUsd（帧方言兜底）
				}
				o.NavTotal += v
				o.Money = append(o.Money, moneyRow("okx", "交易户", d.Ccy,
					d.Ccy, nat, v))
			}
		}
	}
	// 资金户/Earn 行=原生代币数（展示如实）；NAV 计值=amt估值（T-0086 ①）——
	// USDT/USD 直入，其余按 md_ticks 最新价折算（无 tick=0 不入 NAV 仍上卡）。
	if c.fundingLive != nil {
		if fb, err := c.fundingLive(ctx); err == nil {
			for ccy, v := range fb {
				if v == 0 || !fundF[ccy] {
					continue
				}
				u := valCcy(ctx, c.pool, ccy, v)
				o.NavTotal += u
				o.Money = append(o.Money, moneyRow("okx", "资金户", ccy,
					ccy, v, u))
			}
		}
	}
	if c.earnLive != nil {
		if eb, err := c.earnLive(ctx); err == nil {
			for ccy, v := range eb {
				if v == 0 || !fundE[ccy] {
					continue
				}
				u := valCcy(ctx, c.pool, ccy, v)
				o.NavTotal += u
				o.Money = append(o.Money, moneyRow("okx", "Earn", ccy,
					ccy, v, u))
			}
		}
	}
	// T-0096：MT5 入户（专属券商户全计）——逐真实账号一行；读件缺/读败
	// =「读件缺」行不藏（禁假零）；nav_total += MT5 净值合计。
	miss := &dashboardv1.MoneyRow{Venue: "mt5", Section: "户", Label: "读件缺", Note: "读件缺"}
	var accts []mt5Acct
	var aerr error
	if c.mt5Live != nil {
		accts, aerr = c.mt5Live(ctx)
	}
	if c.mt5Live == nil || aerr != nil {
		if aerr != nil {
			log.Printf("WARN: MT5 户活读败（钱在哪卡显读件缺）: %v", aerr)
		}
		o.Money = append(o.Money, miss)
	} else {
		for _, a := range accts {
			o.NavTotal += a.Equity
			o.Money = append(o.Money, moneyRow("mt5", "户", "USD",
				"户 "+a.Login, a.Equity, a.Equity))
		}
	}
	// T-0101：结构性持仓区（dcd LIVE 单=已部署产品持仓=基金的钱第三类别，
	// 行组装见 money_card.go dcdRows——区 0 行不误导，读件缺显形不假零）。
	c.dcdRows(ctx, o)
}
