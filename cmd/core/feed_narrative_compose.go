// feed_narrative_compose.go —— O·动叙事成文纯件（工单O-36963）：
// 窗要素（谁/醒了/关键动作择重/活了多久）→一段白话。词面全白话——
// 机词零漏由 feedNarrBanned 验收表逐词查（feed_narrative_test.go）。
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ── 叙事成文（纯件——桩测直打）────────────────────────────────

// feedNarrUnitLabel 会话白话名——族名+尾号（o-cognition-…→唤知会话·尾NNNN）。
func feedNarrUnitLabel(u string) string {
	fam := "会话"
	for p, cn := range map[string]string{
		"o-cognition": "唤知会话", "o-worker": "施工会话", "o-research": "研究会话",
		"o-self": "自省会话", "o-audit": "审计会话", "o-duty": "值守会话",
		"o-feishu": "推送会话", "o-w": "跑批会话",
	} {
		if strings.HasPrefix(u, p+"-") {
			fam = cn
			break
		}
	}
	tail := ""
	for i := len(u) - 1; i >= 0 && u[i] >= '0' && u[i] <= '9'; i-- {
		tail = string(u[i]) + tail
	}
	if len(tail) > 4 {
		tail = tail[len(tail)-4:]
	}
	if tail == "" {
		return fam
	}
	return fam + "·尾" + tail
}

// feedNarrDur 活了多久——秒级精度（18 秒案）→分→时分级读法。
func feedNarrDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	switch {
	case s < 60:
		return fmt.Sprintf("%d 秒", s)
	case s < 3600:
		return fmt.Sprintf("%d 分 %d 秒", s/60, s%60)
	default:
		return fmt.Sprintf("%d 小时 %d 分", s/3600, (s%3600)/60)
	}
}

var (
	feedNarrEscRe     = regexp.MustCompile(`升件#([0-9]+)\s*销`)
	feedNarrTickRe    = regexp.MustCompile(`工单#([0-9]+)\s*销`)
	feedNarrClaimRe   = regexp.MustCompile(`工单#([0-9]+)|工单\s+O-([0-9]+)|\bO-([0-9]+)\b`)
	feedNarrCapClause = 6 // 叙事从句上限——择重不全列（余量并「等N件」）
)

// feedNarrCompose 叙事段成文——要素：谁(族·尾号)+醒了+关键动作择重+
// 活了多久。词面全白话——升件/复审过/销件/完工回执/戳/净合/三维签等
// 机词零漏（验收逐词查）。
func feedNarrCompose(unit string, open, close feedRow, members []feedRow) string {
	var reviews, dones []string
	var escIDs, tickIDs, claimIDs, filedIDs []int64
	misc := 0
	seenEsc := map[int64]bool{}
	seenTick := map[int64]bool{}
	seenClaim := map[int64]bool{}
	seenDone := map[string]bool{}
	for _, m := range members {
		switch m.event {
		case "复审过", "复审":
			if b := feedHumanBranch(m.note); b != "" && !seenDone[b] {
				seenDone[b] = true
				reviews = append(reviews, b)
			} else {
				misc++
			}
		case "处置":
			if mm := feedNarrEscRe.FindStringSubmatch(m.note); mm != nil {
				if id, err := strconv.ParseInt(mm[1], 10, 64); err == nil && !seenEsc[id] {
					seenEsc[id] = true
					escIDs = append(escIDs, id)
				}
			} else if mm := feedNarrTickRe.FindStringSubmatch(m.note); mm != nil {
				if id, err := strconv.ParseInt(mm[1], 10, 64); err == nil && !seenTick[id] {
					seenTick[id] = true
					tickIDs = append(tickIDs, id)
				}
			} else {
				misc++
			}
		case "认领":
			if mm := feedNarrClaimRe.FindStringSubmatch(m.note); mm != nil {
				var id int64
				for _, g := range mm[1:] {
					if g != "" {
						id, _ = strconv.ParseInt(g, 10, 64)
						break
					}
				}
				if id > 0 && !seenClaim[id] {
					seenClaim[id] = true
					claimIDs = append(claimIDs, id)
				}
			} else {
				misc++
			}
		case "完工回执", "工单完工", "候审回执":
			if b := feedHumanBranch(m.note); b != "" && !seenDone[b] {
				seenDone[b] = true
				dones = append(dones, b)
			} else {
				misc++
			}
		case "工单":
			filedIDs = append(filedIDs, m.id)
		case "会话收口", "在办":
			// 闭/续约行——叙事由首尾+时长表达不单列
		default:
			misc++
		}
	}

	// O-37510②：工单引用一律 O-<id> 票面（票号=ops_event id 域别名，
	// 「#N」是事件行簿记位口径——混用=口径混源）；升件#N=esc 异域保 #。
	joinIDs := func(prefix, tag string, ids []int64, cap int) string {
		if len(ids) == 0 {
			return ""
		}
		n := len(ids)
		if n > cap {
			n = cap
		}
		var parts []string
		for _, id := range ids[:n] {
			parts = append(parts, tag+strconv.FormatInt(id, 10))
		}
		s := prefix + " " + strings.Join(parts, "、")
		if len(ids) > cap {
			s += fmt.Sprintf(" 等%d件", len(ids))
		}
		return s
	}
	var acts []string
	if s := joinIDs("领了工单", "O-", claimIDs, 2); s != "" {
		acts = append(acts, s)
	}
	if s := joinIDs("立了工单", "O-", filedIDs, 2); s != "" {
		acts = append(acts, s)
	}
	if len(reviews) == 1 {
		acts = append(acts, "给「"+reviews[0]+"」签了复审")
	} else if len(reviews) > 1 {
		acts = append(acts, fmt.Sprintf("签了%d个复审", len(reviews)))
	}
	if len(dones) == 1 {
		acts = append(acts, "「"+dones[0]+"」完工")
	} else if len(dones) > 1 {
		acts = append(acts, fmt.Sprintf("%d件完工", len(dones)))
	}
	if s := joinIDs("处理了报警", "#", escIDs, 3); s != "" {
		acts = append(acts, s)
	}
	if s := joinIDs("结了工单", "O-", tickIDs, 2); s != "" {
		acts = append(acts, s)
	}
	if misc > 0 {
		acts = append(acts, fmt.Sprintf("另记%d笔", misc))
	}
	// 择重截断——从句超上限折「等N件」
	var clause string
	if len(acts) == 0 {
		clause = "醒了一轮没留大动作"
	} else {
		show := acts
		rest := 0
		if len(show) > feedNarrCapClause {
			rest = len(show) - feedNarrCapClause
			show = show[:feedNarrCapClause]
		}
		clause = strings.Join(show, "，")
		if rest > 0 {
			clause += fmt.Sprintf("，等%d件", rest)
		}
	}
	d := time.Duration(0)
	if !close.ts.IsZero() && !open.ts.IsZero() {
		d = close.ts.Sub(open.ts)
	}
	return fmt.Sprintf("%s：%s，活了 %s干完一轮",
		feedNarrUnitLabel(unit), clause, feedNarrDur(d))
}
