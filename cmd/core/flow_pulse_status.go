// flow_pulse_status.go —— 需求驱动段态标（T-0350 段表达根修——需求驱动段无
// 拍距可言：提案只在评估出 concrete 才生、执行只在 armed 提案存在才动，渲龄
// =把纪律性空仓渲成停机。自 flow_pulse.go 拆出——行数棘轮 D-077/D-588）。
package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
)

// —— 需求驱动段态标（T-0350 段表达根修——需求驱动段无拍距可言：提案只在
// 评估出 concrete 才生、执行只在 armed 提案存在才动，渲龄=把纪律性空仓渲成
// 停机。分类册=段表 neutral bool（既存消费径=灯恒中性灰，本次增态标消费）；
// 新段默认 false=周期龄语义，显式归类才入态标面）——

// execRealKinds 真执行类白名单（fin_cashflow.kind 闭集内「只在成交/持仓后果
// 中出现」者——票面钦定 interest/fee=被动行剔出；transfer/adjustment 移动类
// 与 supply_income/airdrop 被动收益同理剔。生产枚举实拍 2026-11-05 SG：
// deposit/withdraw/interest/fee/earn_in/out/recon_adjust/spot_buy/sell/
// funding_settle——四词与枚举零冲突；trade_pnl 闭集在册未发生照入白名单）。
var execRealKinds = []string{"trade_pnl", "spot_buy", "spot_sell", "funding_settle"}

// reNum 拒因数字归一（拒因串内嵌快照瞬值——exact-string 众数被数值碎片化，
// 归一后聚合同类拒因；生产实拍 24h 93 拒归「net_apr < 阈」一类 71+）。
var reNum = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

// proposalStatus 提案段态标（armed=活态 fin_proposed+fin_armed 两词——与
// workItems 同口径防两端漂移）：活态>0=「活跃·N armed」；零活态且近窗（24h）
// 评估全拒=「闲·评估全拒（末因 <dominant 短标>）」；余=「闲·无 armed」
// （评估零行/unevaluable 混行不称「全拒」——不虚构因，评估臂健康归评估段
// 周期灯管）。查询败=空标回退龄渲（fail-soft 同段链口径）。
func (c coreService) proposalStatus(ctx context.Context) string {
	var armed int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE state IN ('fin_proposed','fin_armed')`).Scan(&armed); err != nil {
		log.Printf("[flow_pulse] proposal 态标查询败（回退龄渲）: %v", err)
		return ""
	}
	if armed > 0 {
		return fmt.Sprintf("活跃·%d armed", armed)
	}
	var total, rej int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*),
		count(*) FILTER (WHERE verdict='rejected') FROM fin_opp_eval
		WHERE ts>now()-interval '24 hours'`).Scan(&total, &rej); err != nil {
		log.Printf("[flow_pulse] proposal 态标查询败（回退龄渲）: %v", err)
		return ""
	}
	if total > 0 && rej == total {
		if reason := c.dominantRejectReason(ctx); reason != "" {
			return "闲·评估全拒（末因 " + reason + "）"
		}
		return "闲·评估全拒"
	}
	return "闲·无 armed"
}

// dominantRejectReason 近窗（24h）dominant 拒因短标——evidence 真实 reason
// 字段归一后取众数（键序排序 tie 稳定；不虚构不猜因）。
func (c coreService) dominantRejectReason(ctx context.Context) string {
	rs, err := c.pool.Query(ctx, `SELECT evidence->>'reason' FROM fin_opp_eval
		WHERE ts>now()-interval '24 hours' AND verdict='rejected' AND evidence?'reason'`)
	if err != nil {
		log.Printf("[flow_pulse] 拒因查询败（态标降级不带因）: %v", err)
		return ""
	}
	defer rs.Close()
	n := map[string]int{}
	for rs.Next() {
		var r string
		if rs.Scan(&r) != nil {
			continue
		}
		n[reasonShort(r)]++
	}
	keys := make([]string, 0, len(n))
	for k := range n {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	top, best := "", 0
	for _, k := range keys {
		if n[k] > best {
			top, best = k, n[k]
		}
	}
	return top
}

// reasonShort 拒因短标——数字归一+空白剔净+截 30 rune（数字=快照瞬值，
// 短标只留稳定判据骨架；已知拒因形态归一后 ≤30 rune 全容；evidence 原文
// 在库可查不重复载）。
func reasonShort(r string) string {
	s := strings.ReplaceAll(reNum.ReplaceAllString(r, ""), " ", "")
	if runes := []rune(s); len(runes) > 30 {
		s = string(runes[:30]) + "…"
	}
	return s
}

// execStatus 执行段态标：近窗（24h）有真执行类 cashflow=空标（龄标沿用）；
// 零真执行=「闲·无 armed」。查询败=空标回退龄渲。
func (c coreService) execStatus(ctx context.Context) string {
	var n int64
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_cashflow
		WHERE kind = ANY($1) AND occurred_at>now()-interval '24 hours'`,
		execRealKinds).Scan(&n); err != nil {
		log.Printf("[flow_pulse] exec 态标查询败（回退龄渲）: %v", err)
		return ""
	}
	if n > 0 {
		return ""
	}
	return "闲·无 armed"
}
