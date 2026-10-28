// alerts.go —— TRX 投票改投/领奖告警渲染 + convertVotes 适配（自 scheduler.go 折分，D-120 行数门禁）。
package trxvote

import "fmt"

// convertVotes 将 map[string]int64（gotron-sdk Votes，TRX 单位）转为 []VoteEntry（链上 vote_count 已是 TRX 无需换算）。
func convertVotes(votes map[string]int64) []VoteEntry {
	out := make([]VoteEntry, 0, len(votes))
	for addr, v := range votes {
		out = append(out, VoteEntry{SRAddress: addr, Votes: float64(v)})
	}
	return out
}

// formatRebalanceAlert 格式化改投告警（设计 §8，D-293 S4 加 SR 变化明细；
// T-0220 扩形——年化差 pp+总票数人话化。CurrentYield/TargetYield 已是年化
// 百分数，差值即 pp）。
func formatRebalanceAlert(txHash string, d RebalanceDecision, currentVotes []VoteEntry, totalTP int64) string {
	short := txHash
	if len(short) > 16 {
		short = short[:16]
	}
	return fmt.Sprintf("[投票改投] %s→%s (年化差%+.2fpp) %s votes tx=%s",
		topSRPrefix(currentVotes), topSRPrefix(d.Target),
		d.TargetYield-d.CurrentYield, formatVotes(totalTP), short)
}

// topSRPrefix 取票数最高 SR 地址前 8 字符（无投票返回 "无"）。
func topSRPrefix(votes []VoteEntry) string {
	if len(votes) == 0 {
		return "无"
	}
	top := votes[0]
	for _, v := range votes[1:] {
		if v.Votes > top.Votes {
			top = v
		}
	}
	addr := top.SRAddress
	if len(addr) > 8 {
		addr = addr[:8]
	}
	return addr
}

// formatClaimAlert 格式化领奖告警（设计 §8；T-0220 扩形——收入额 TRX+USD
// 折算（价源=LastCMCPriceMidWithTime TRXUSDT mid，与 GetTrxYieldCurve
// currentPrice 同源）；priceUSD≤0 无报价省略折算段）。
func formatClaimAlert(txHash string, amount int64, priceUSD float64) string {
	short := txHash
	if len(short) > 16 {
		short = short[:16]
	}
	trx := float64(amount) / 1e6
	msg := fmt.Sprintf("[投票领奖] +%.2f TRX", trx)
	if priceUSD > 0 {
		msg += fmt.Sprintf(" (≈$%.2f)", trx*priceUSD)
	}
	return msg + " tx=" + short
}

// formatVotes 票数人话化（107300→107.3k，1530000→1.53M，不足 1k 原样）。
func formatVotes(v int64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(v)/1e6)
	case v >= 1_000:
		return fmt.Sprintf("%.1fk", float64(v)/1e3)
	default:
		return fmt.Sprintf("%d", v)
	}
}

// formatFloat 格式化浮点数（4 位小数）。
func formatFloat(f float64) string {
	return fmt.Sprintf("%.4f", f)
}
