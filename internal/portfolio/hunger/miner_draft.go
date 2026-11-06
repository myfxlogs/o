// miner_draft.go —— 矿工起草装配件（T-0385 minerDraft/载荷/条目名/
// 锚列原 miner.go 同件——check-lines 棘轮拆件）：pending 行落库与
// ops_event 留账；T-0387 载荷扩 target 域+skill_patch 三件在此装。
package hunger

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
)

// minerPayload 起草载荷机读五件（票面钦：签名=列/证据锚 ops_event id 列/
// 定性草案/建议条目名/验证态——验证态为列，本件载其余三件+归类辅证）。
// T-0387 载荷扩 target 域（lesson|skill_patch——表构不动）+skill_patch
// 三件（target_skill/lesion_anchor/patch_draft，lesson 行省缺）。
type minerPayload struct {
	Event          string   `json:"event"`
	Key            string   `json:"key"`
	Target         string   `json:"target"`
	EvidenceIDs    []int64  `json:"evidence_ids"`
	Tickets        []string `json:"tickets"`
	Draft          string   `json:"draft"`
	SuggestedEntry string   `json:"suggested_entry"`
	TargetSkill    string   `json:"target_skill,omitempty"`
	LesionAnchor   string   `json:"lesion_anchor,omitempty"`
	PatchDraft     string   `json:"patch_draft,omitempty"`
}

// minerDraft pending 行落库（verify_state=ai_evaluated——机器聚类稿带实证
// 锚非 unverified；同拍 ops_event 教训矿工行留账=自主≠无账，行不在矿工
// 签名集内零自指）。
func (s *Sensor) minerDraft(ctx context.Context, c *mineCluster, target, skill string, hits []skillHit) error {
	p := minerPayload{Event: c.rows[0].lessonClass(), Target: target}
	tset := map[string]bool{}
	for _, r := range c.rows {
		p.EvidenceIDs = append(p.EvidenceIDs, r.id)
		if r.ticket != "" && !tset[r.ticket] {
			tset[r.ticket] = true
			p.Tickets = append(p.Tickets, r.ticket)
		}
	}
	first := c.rows[0]
	sort.Strings(p.Tickets)
	p.Key = first.ticket
	if p.Key == "" {
		p.Key = first.seat
	}
	if p.Event == "升件" && first.escClass != "" {
		p.Key = first.escClass + ":" + first.escScope
	}
	note := first.note
	if len([]rune(note)) > 140 {
		note = string([]rune(note)[:140]) + "…"
	}
	p.Draft = fmt.Sprintf("事件账同窗(%dh)%s×%d —— 首行: %s",
		minerWindowHours, c.sig, len(c.rows), note)
	p.SuggestedEntry = engEntrySlug(p.Key, c.sig)
	if target == "skill_patch" {
		p.TargetSkill = skill
		p.LesionAnchor = lesionAnchor(hits)
		p.PatchDraft = fmt.Sprintf(
			"目标=skills/%s/SKILL.md；病灶锚=%s；漂移证据锚=ops_event#%s。"+
				"节替换草案：「## 漂移勘正」节下落——canon 与实况漂移：%s"+
				"（签名 %s，同窗 %d 行）。裁定席按 patch_draft+证据行定稿,"+
				"无证据锚拒入（防 LLM 臆改 canon）。",
			skill, p.LesionAnchor, joinIDs(p.EvidenceIDs), note, c.sig, len(c.rows))
	}
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("矿工载荷序列化: %w", err)
	}
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO fin_eng_lesson(signature,cluster_n,verify_state,numbers)
		VALUES($1,$2,'ai_evaluated',$3::jsonb) RETURNING id`,
		c.sig, len(c.rows), b).Scan(&id)
	if err != nil {
		return fmt.Errorf("矿工 pending 落库 signature=%s: %w", c.sig, err)
	}
	ledger := fmt.Sprintf("pending#%d signature=%s n=%d evidence=#%s",
		id, c.sig, len(c.rows), joinIDs(p.EvidenceIDs))
	if target == "skill_patch" {
		ledger += " target=skill_patch skill=" + skill
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'教训矿工',$2,$3)`,
		minerSeat, minerTicket, ledger); err != nil {
		return fmt.Errorf("矿工留账行: %w", err)
	}
	return nil
}

// engEntrySlug 建议条目名（工程经验库 id=ASCII kebab——非 ASCII/空白折叠
// 横线，零词元退 crc32 指纹兜底保唯一可读位）。
func engEntrySlug(key, sig string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	slug := strings.Trim(b.String(), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		slug = fmt.Sprintf("sig-%08x", crc32.ChecksumIEEE([]byte(sig)))
	}
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	return "eng-" + slug
}

func joinIDs(ids []int64) string {
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", id)
	}
	return b.String()
}
