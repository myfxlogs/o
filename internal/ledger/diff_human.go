// diff_human.go —— 账差升件白话账段（工单O-36964——业主令「给出的就不是
// 直白明白的账」：机读底稿段前置人话段，读者三秒知道差多少/差由哪几块
// 组成/是否已豁免/要不要人动手）。三段式：①App实存分量相加（同一次
// venue 读面拼装零扩 API）②差额构成逐行（豁免白话名+实时值/记账基准）
// ③豁后残差+需动手判词。禁机器词表钉死：resolver/baseline/容差/锚/
// 现算/parked/nativeBase 永不出业主面（桩测全文检索断言在案）。
package ledger

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// BalPart venue 实读分量一项（人话段原料——Name=内部代号，白话名映射
// 在 venuePartPlain；Amount=币量同 eq 键单位）。
type BalPart struct {
	Name   string
	Amount float64
}

// VenueParts (venue,ccy)→分量集（O-36964——与 eq 同一次读面拼装；
// nil map/缺键=分量读面缺，白话段降级显总额不虚构分量）。
type VenueParts map[[2]string][]BalPart

// venuePartPlain 分量代号→人话名（未录代号原样透出——宁露不漏同
// frameText 字典律；白话名本身禁机器词已人工核）。
var venuePartPlain = map[string]string{
	"trading": "交易户",
	"funding": "资金户",
	"earn":    "Earn理财",
	"dcd":     "DCD期权",
	"account": "账户",
}

// residPlainName resolver 键→豁免白话名（未录键原样透出——同上字典律）。
var residPlainName = map[string]string{
	"dcd_mtm_drift":    "DCD期权市价漂移",
	"earn_accrual_gap": "Earn应计息未结算",
}

// humanBanned 白话段禁词表（spec 钦定——机读词永不出业主面；
// 「锚/现算/容差」等机读行话换「记账基准/实时值/允许差」）。
var humanBanned = []string{"resolver", "baseline", "容差", "锚", "现算", "parked", "nativeBase"}

func humanBannedHit(s string) bool {
	for _, w := range humanBanned {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// refPrefixRe 静态豁免 reason 头票据锚剥除（「升件#34532定性：」「工单#N」
// 「esc#N」「T-NNNN」「定性#N」前缀族——人话名只留实质短语）。
var refPrefixRe = regexp.MustCompile(`^(升件#\d+定性|升件#\d+|工单#\d+|工单/\S+|esc#\d+|定性#\d+|T-\d+)[：:、，,。\s]*`)

// staticPlainName 静态豁免行白话名：reason 首段（破折号/括号截断）剥票据
// 锚前缀→「已定性残差·<短语>」；空/超长/含禁词→「已定性残差#<id>」兜底
// （宁可丢修饰不泄机读词）。
func staticPlainName(id int64, reason string) string {
	head := reason
	for _, sep := range []string{"——", "（", "("} {
		if i := strings.Index(head, sep); i >= 0 {
			head = head[:i]
		}
	}
	for {
		n := refPrefixRe.ReplaceAllString(strings.TrimSpace(head), "")
		if n == head {
			break
		}
		head = n
	}
	head = strings.TrimSpace(head)
	r := []rune(head)
	if len(r) > 24 {
		head = string(r[:24]) + "…"
	}
	if head == "" || humanBannedHit(head) {
		return fmt.Sprintf("已定性残差#%d", id)
	}
	return "已定性残差·" + head
}

// exemptPlainLine 豁免逐行白话（名+额/模式定性——tracked=实时值、
// fallback=按记账基准兜底、breach=实时值超带归零、static=定性额、
// qty=币量豁免 O-38064——币量+币名不冠 $）。
func exemptPlainLine(p exemptPart, ccy string) string {
	if p.Mode == "qty" {
		line := fmt.Sprintf("币量豁免 %s %s", qtyFmt(p.Raw), strings.ToUpper(ccy))
		if humanBannedHit(line) {
			return fmt.Sprintf("币量豁免#%d", p.ID)
		}
		return line
	}
	var name, suffix string
	if p.Resolver != "" {
		segs := strings.SplitN(p.Resolver, ":", 2)
		if pn, ok := residPlainName[segs[0]]; ok {
			name = pn
		} else {
			name = segs[0] // 未录 resolver 键原样透出（消毒闸兜底）
		}
		if len(segs) == 2 && segs[1] != "" {
			suffix = "（单#" + segs[1] + "）"
		}
	} else {
		name = staticPlainName(p.ID, p.Reason)
	}
	var line string
	switch p.Mode {
	case "tracked":
		line = fmt.Sprintf("%s%s 实时值$%.2f", name, suffix, p.Value)
	case "breach":
		line = fmt.Sprintf("%s%s 实时值$%.2f超带归零", name, suffix, p.Raw)
	case "fallback":
		line = fmt.Sprintf("%s%s 按记账基准$%.2f兜底", name, suffix, p.Value)
	default:
		line = fmt.Sprintf("%s%s $%.2f", name, suffix, p.Value)
	}
	if humanBannedHit(line) {
		return fmt.Sprintf("已定性残差#%d $%.2f", p.ID, p.Value)
	}
	return line
}

// ledgerDiffHuman 白话账段（升件 note 头部——置于机读段前）。
// eq=venue 实读总额、net=账本净（含停泊/dcd/原生基线/mtm 同径）、
// diffUSD=账实差 USD 口径、tol=允许差、exempt=已定性豁免合计、
// parts=同读面分量集（nil=分量读面缺降级显总额）。
// O-38064：dout≠nil=非本位键——币量+币名主量（有价附≈$x.xx/无价写
// 「无价不可估值待定性」）；dout=nil=本位键逐字节守恒。
func (r *Runner) ledgerDiffHuman(ctx context.Context, acct, venue, ccy string,
	eq, net, diffUSD, tol, exempt float64, parts []BalPart, dout *diffOut) string {
	base := isBaseCCY(ccy)
	ccyU := strings.ToUpper(ccy)
	// 数值渲染：本位键 $%.2f 逐字节守恒；非本位=币量+币名（qtyFmt）。
	money := func(v float64) string {
		if base {
			return fmt.Sprintf("$%.2f", v)
		}
		return fmt.Sprintf("%s %s", qtyFmt(v), ccyU)
	}
	var b strings.Builder
	if dout == nil {
		fmt.Fprintf(&b, "%s %s 账实差 $%.2f——", strings.ToUpper(venue), ccy, diffUSD)
	} else {
		var pxNote string
		if dout.priced {
			pxNote = fmt.Sprintf(" ≈$%.2f", dout.usd)
		} else {
			pxNote = "（无可用行情价）"
		}
		fmt.Fprintf(&b, "%s %s 账实差 %s %s%s——", strings.ToUpper(venue), ccyU,
			qtyFmt(dout.qty), ccyU, pxNote)
	}
	// ①App实存分量相加（分量和与总额差>0.01=如实附注不硬凑）。
	// O-38064：非本位键分量=币量——qtyFmt+币名形不冠 $。
	if len(parts) == 0 {
		fmt.Fprintf(&b, "App实读 %s（分量明细缺——见底稿）", money(eq))
	} else {
		var sum float64
		segs := make([]string, 0, len(parts))
		for _, p := range parts {
			if p.Amount == 0 {
				continue
			}
			name, ok := venuePartPlain[p.Name]
			if !ok {
				name = p.Name // 未录分量代号原样透出
			}
			if humanBannedHit(name) {
				name = "其他分量"
			}
			if base {
				segs = append(segs, fmt.Sprintf("%s %.2f", name, p.Amount))
			} else {
				segs = append(segs, fmt.Sprintf("%s %s %s", name, qtyFmt(p.Amount), ccyU))
			}
			sum += p.Amount
		}
		if len(segs) == 0 {
			fmt.Fprintf(&b, "App实读 %s（分量皆零——见底稿）", money(eq))
		} else if base {
			fmt.Fprintf(&b, "App实存 %s=合计%.2f", strings.Join(segs, "+"), sum)
			if d := math.Abs(sum - eq); d > 0.01 {
				fmt.Fprintf(&b, "（分量和与总额读数差$%.2f——底稿见下）", d)
			}
		} else {
			fmt.Fprintf(&b, "App实存 %s=合计%s %s", strings.Join(segs, "+"), qtyFmt(sum), ccyU)
			if d := math.Abs(sum - eq); d > 0.01 {
				fmt.Fprintf(&b, "（分量和与总额读数差%s——底稿见下）", money(d))
			}
		}
	}
	if base {
		fmt.Fprintf(&b, " vs 账本记 %.2f", net)
	} else {
		fmt.Fprintf(&b, " vs 账本记 %s %s", qtyFmt(net), ccyU)
	}
	// ②差额构成逐行（豁免白话名+额——零条=「无豁免条目」行）。
	b.WriteString("；差额构成：")
	_, exparts, err := r.trackedExempt(ctx, venue, ccy, acct)
	if err != nil || len(exparts) == 0 {
		b.WriteString("无豁免条目")
	} else {
		lines := make([]string, 0, len(exparts))
		for _, p := range exparts {
			lines = append(lines, exemptPlainLine(p, ccy))
		}
		b.WriteString(strings.Join(lines, "+"))
	}
	// ③豁后残差+需动手判词（残差≤允许差=已定性无需动手；超=待定性
	// 取证底稿见下段——告警面本身即「待定性」常态，判词如实两态）。
	// O-38064：非本位键残差在 USD 空间判（priced 时=dout.usd−USD豁免；
	// 无价=币量残量如实写「折不出美元金额」不虚构美元数）。
	if dout != nil && !dout.priced {
		fmt.Fprintf(&b, "——币量豁免后仍差 %s %s，无价不可估值，待定性，取证底稿见下段",
			qtyFmt(dout.qty), ccyU)
	} else {
		residual := diffUSD - exempt
		if dout != nil {
			residual = dout.usd - exempt
		}
		if residual <= tol {
			d := residual
			if d < 0 {
				d = 0
			}
			fmt.Fprintf(&b, "——豁后残差≈$%.2f，无需动手（已定性在案）", d)
		} else {
			fmt.Fprintf(&b, "——豁后残差 $%.2f>允许差 $%.2f，待定性，取证底稿见下段", residual, tol)
		}
	}
	seg := b.String()
	if humanBannedHit(seg) {
		// 兜底闸：任一动态件漏消毒→整段降级最小安全形（机读段仍在）。
		if dout == nil {
			return fmt.Sprintf("%s %s 账实差 $%.2f——明细含机读件见下段底稿",
				strings.ToUpper(venue), ccy, diffUSD)
		}
		return fmt.Sprintf("%s %s 账实差 %s %s——明细含机读件见下段底稿",
			strings.ToUpper(venue), ccyU, qtyFmt(dout.qty), ccyU)
	}
	return seg
}
