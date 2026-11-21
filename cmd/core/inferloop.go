package main

// inferloop.go —— 举一反三消费环+慧量表（工单#27050）。
// 矿工把升件蒸馏成 fin_eng_lesson pending 候选，但「候选→免疫」缺下游
// 消费件=举一反三缺第三棒；selfmodel 帧缺「智慧占比」第四翼。
// 本件两职：①消费环——pending 行终态分流（关联工单已合 master→标
// 已免疫载合点；签名仍复火→升待canon化行归唤知归纳；>7d 零动作→升
// learn_stale）。禁盲自动归纳——语义归纳归会话，本件只数终态。
// ②慧量表——24h 窗升件终局二分（器官自处置 vs 会话介入+复火率），
// 量表行 ticket='wisdom_gauge' gaugeDue 同纹抑。

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	learnStaleDays   = 7              // pending 零动作陈龄→learn_stale
	wisdomGaugeEvery = 20 * time.Hour // 慧量表重报纹闸（同 self_model 族）
	wisdomWindow     = "24 hours"     // 慧量表窗（票面建议 24h）
)

var (
	lessonTicketRe = regexp.MustCompile(`工单#([0-9]+)`)
	escRefRe       = regexp.MustCompile(`升件#([0-9]+)(?:-([0-9]+))?`)
	mergeShaRe     = regexp.MustCompile(`merge[:：]?([0-9a-zA-Z]{7,40}|随车[0-9a-fA-F]+)`)
)

// inferLoopBeat 消费环一拍——pending 全扫终态分流+慧量表发射判。
// 读面件：判据全在 ops_event/fin_eng_lesson 既有账，零迁移零新键。
func (c coreService) inferLoopBeat(ctx context.Context, deadSince map[string]time.Time) {
	c.lessonConsume(ctx, deadSince)
	c.wisdomGaugeBeat(ctx)
}

// lessonRow 矿工候选行（消费环视野面）。
type lessonRow struct {
	id        int64
	signature string
	ts        time.Time
}

// lessonConsume pending 候选终态分流——三态各一去向，互不连带：
// 已免疫（UPDATE 终态）/待canon化（learn_canon 升件）/learn_stale 升件。
func (c coreService) lessonConsume(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT id,signature,ts FROM fin_eng_lesson
		WHERE status='pending' ORDER BY id`)
	if err != nil {
		return // 读件败本拍让位
	}
	var pend []lessonRow
	for rows.Next() {
		var l lessonRow
		if rows.Scan(&l.id, &l.signature, &l.ts) == nil {
			pend = append(pend, l)
		}
	}
	rows.Close()
	for _, l := range pend {
		// ① 关联工单已合 master→标「已免疫」载合点（status→verdicted 走
		// ops_decision_lesson_verdict 触发器落裁决行=终态账在案）。
		if sha, ok := c.lessonMergeRef(ctx, l); ok {
			if _, err := c.pool.Exec(ctx, `UPDATE fin_eng_lesson
				SET status='verdicted', verdict_ref=$2
				WHERE id=$1 AND status='pending'`,
				l.id, "已免疫@"+sha); err == nil {
				continue
			}
		}
		// ② 签名仍复火（仅升件 cls:scope 形可机判——余形无复火源）→
		// 升「待canon化」行：说明类升件，唤知拾起归纳，禁盲自动归纳。
		if cls, scope, isEsc := splitEscSig(l.signature); isEsc {
			if n, err := c.escSigRefire(ctx, cls, scope, l.ts); err == nil && n > 0 {
				c.escEdgeUV(ctx, deadSince, "learn_canon", "lesson:"+l.signature, "",
					true, func() string {
						return fmt.Sprintf("待canon化 lesson#%d signature=%s——签名簇窗内复火 %d 件,语义归纳归唤知会话(禁盲自动归纳,本件只数终态)",
							l.id, l.signature, n)
					})
			} else {
				c.escEdgeUV(ctx, deadSince, "learn_canon", "lesson:"+l.signature, "",
					false, func() string { return "" })
			}
		}
		// ③ pending 超 7d 零动作→升件 learn_stale（积压陈尸点名）。
		stale := time.Since(l.ts) > learnStaleDays*24*time.Hour
		c.escEdgeUV(ctx, deadSince, "learn_stale", "lesson:"+l.signature, "",
			stale, func() string {
				return fmt.Sprintf("learn_stale lesson#%d signature=%s——pending 超 %dd 零动作,教训候选积压点名",
					l.id, l.signature, learnStaleDays)
			})
	}
}

// splitEscSig 签名拆「升件:<cls>:<scope>」形——cls:scope 等值键为矿工
// 钦定签名维（miner.go signature() 同源）；余形返 isEsc=false。
func splitEscSig(sig string) (cls, scope string, isEsc bool) {
	if !strings.HasPrefix(sig, "升件:") {
		return "", "", false
	}
	rest := sig[len("升件:"):]
	i := strings.Index(rest, ":")
	if i < 0 {
		return "", "", false // 「升件:<ticket/seat>」键形——无 cls:scope 源
	}
	return rest[:i], rest[i+1:], true
}

// escSigRefire 签名簇复火计数——同 esc_class+esc_scope 升件在 lesson.ts
// 后新火数（等值键同 miner 签名维；0=簇已熄）。
func (c coreService) escSigRefire(ctx context.Context, cls, scope string, since time.Time) (int, error) {
	var n int
	err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2 AND ts>$3`,
		cls, scope, since).Scan(&n)
	return n, err
}

// lessonMergeRef 关联工单合点取证——签名直含 工单#N 或经升件簇
// （esc_class+esc_scope 等值）→升件# 引证行→工单#N→销行 merge:sha。
// 返合点 sha；无链/无 merge 证=ok=false（不判死——只数终态）。
func (c coreService) lessonMergeRef(ctx context.Context, l lessonRow) (string, bool) {
	tickets := map[int64]bool{}
	for _, m := range lessonTicketRe.FindAllStringSubmatch(l.signature, -1) {
		if id, err := strconv.ParseInt(m[1], 10, 64); err == nil {
			tickets[id] = true
		}
	}
	// 升件簇链：cls:scope→在案升件 id 集→升件# 引证行→工单#N。
	if cls, scope, isEsc := splitEscSig(l.signature); isEsc {
		escRows, err := c.pool.Query(ctx, `SELECT id,ts FROM ops_event
			WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
			AND ts>now()-interval '180 days' ORDER BY id DESC LIMIT 50`, cls, scope)
		if err == nil {
			var escIDs []int64
			var minTS time.Time
			for escRows.Next() {
				var id int64
				var ts time.Time
				if escRows.Scan(&id, &ts) == nil {
					escIDs = append(escIDs, id)
					if minTS.IsZero() || ts.Before(minTS) {
						minTS = ts
					}
				}
			}
			escRows.Close()
			if len(escIDs) > 0 {
				// 引证行=升件 id 后行（销案语义——同 escalateUnresolved 口径）；
				// 范围引用 升件#lo-hi 覆盖判。
				refs, err := c.pool.Query(ctx, `SELECT note FROM ops_event
					WHERE note LIKE '%升件#%' AND ts>$1
					AND ts>now()-interval '180 days'`, minTS.Add(-time.Hour))
				if err == nil {
					for refs.Next() {
						var note string
						if refs.Scan(&note) != nil {
							continue
						}
						for _, rm := range escRefRe.FindAllStringSubmatch(note, -1) {
							lo, _ := strconv.ParseInt(rm[1], 10, 64)
							hi := lo
							if rm[2] != "" {
								hi, _ = strconv.ParseInt(rm[2], 10, 64)
							}
							for _, e := range escIDs {
								if e >= lo && e <= hi {
									for _, tm := range lessonTicketRe.FindAllStringSubmatch(note, -1) {
										if id, err := strconv.ParseInt(tm[1], 10, 64); err == nil {
											tickets[id] = true
										}
									}
								}
							}
						}
					}
					refs.Close()
				}
			}
		}
	}
	for id := range tickets {
		if sha, ok := c.ticketMergeSHA(ctx, id); ok {
			return sha, true
		}
	}
	return "", false
}

// ticketMergeSHA 工单合点取证——销行形「工单#N 销——merge:<sha>」：
// note 先锚工单#N 引用、其后须有「销」字、mergeShaRe 取 sha
// （升件销行引根因票≠本票销——工单#N 后的销须同段，防误配）。
func (c coreService) ticketMergeSHA(ctx context.Context, ticketID int64) (string, bool) {
	rows, err := c.pool.Query(ctx, `SELECT note FROM ops_event
		WHERE event='处置' AND note ~ ('工单#'||$1||'([^0-9]|$)')
		AND note ~ '销' AND note ~ 'merge'
		AND ts>now()-interval '400 days' ORDER BY id DESC LIMIT 10`,
		strconv.FormatInt(ticketID, 10))
	if err != nil {
		return "", false
	}
	defer rows.Close()
	ref := "工单#" + strconv.FormatInt(ticketID, 10)
	for rows.Next() {
		var note string
		if rows.Scan(&note) != nil {
			continue
		}
		i := strings.Index(note, ref)
		if i < 0 || !strings.Contains(note[i:], "销") {
			continue
		}
		if m := mergeShaRe.FindStringSubmatch(note); m != nil {
			return m[1], true
		}
	}
	return "", false
}

// wisdomGaugeBeat 慧量表——24h 窗升件终局二分：会话介入=升件后有
// session 席 处置/认领/销号/工单 行引证（升件#id∨升件#lo-hi 覆盖）；
// 器官自处置=无此关联（含机房自愈/未触）；复火=同 esc_scope 窗内≥2 火。
// 产「慧=自处置a%/会话b%/复火c%」量表行，gaugeDue 同纹抑。
func (c coreService) wisdomGaugeBeat(ctx context.Context) {
	var total, sess, refire int
	err := c.pool.QueryRow(ctx, `WITH w AS (
		SELECT e.id, e.esc_scope,
			(SELECT count(*) FROM ops_event e2 WHERE e2.event='升件'
				AND COALESCE(e2.esc_scope,'')=COALESCE(e.esc_scope,'')
				AND e2.ts>now()-interval '`+wisdomWindow+`') AS scope_n,
			EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
				AND x.event ~ '^(处置|认领|销号|工单)$'
				AND x.seat ~ '^(唤知|唤知席|唤知机|外席|外审|施工|施工席|施工机|处置席|复审席|复审者|自我|自我机)$'
				AND (x.note ~ ('升件#'||e.id||'([^0-9]|$)')
					OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
						WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint))) AS sess
		FROM ops_event e WHERE e.event='升件' AND e.ts>now()-interval '`+wisdomWindow+`')
		SELECT count(*), count(*) FILTER (WHERE sess),
			count(*) FILTER (WHERE scope_n>=2) FROM w`).
		Scan(&total, &sess, &refire)
	if err != nil || total == 0 {
		return // 读件败/零升件窗——让位不产空行
	}
	selfN := total - sess
	fp := fmt.Sprintf("%d|%d|%d|%d", total, sess, selfN, refire)
	if !gaugeDue(ctx, c, "wisdom_gauge", fp, wisdomGaugeEvery) {
		return
	}
	pct := func(n int) int { return n * 100 / total }
	note := fmt.Sprintf("v=%s 慧=自处置%d%%/会话%d%%/复火%d%%——24h窗升件N=%d 处置/认领/销号/工单引证分派(器官自处置=无会话席关联;复火=同esc_scope窗内≥2火);智慧占比第四翼只述事实不定性",
		fp, pct(selfN), pct(sess), pct(refire), total)
	_, _ = c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','wisdom_gauge',$1)`, note)
}
