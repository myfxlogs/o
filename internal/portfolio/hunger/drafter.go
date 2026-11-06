// drafter.go —— 起草拍（T-0393）：探针升件行→候选票草案自动起草。
// 架构断点根修——探针族在长大（饥饿拍/T-0384 族沉默/T-0392 腿覆盖），
// 升件行→立项靠人/席注意（业主反复当「注意到的人」）；本拍挂 hungerTick
// 同槽补「发现→立项」断点：扫 ops_event 探针升件行→病灶签名聚类→签名级
// 去重（同键 24h 防灌/在飞·已立票不重复起草）→fin_ticket_draft pending
// （fin_0123）+ops_event 起草行。物料化=tools/ticketdraft（dm_tick 零 LLM
// 尾段）取号→docs/tasks 候选态票件。只产候选草案不派发——发起权=DM/业主
// （T-0376 权序红线）；起草=确定性模板填充，LLM 不入安全径。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
)

// 票面钦定值（扫窗=签名同窗去重 24h 同源——升件行自带 24h 同窗去重，本拍
// 同窗扫即签名级天然去重；fin_ticket_draft 层再钉一道防灌）。
const (
	drafterWindowHours = 24       // 扫窗——探针升件行近窗面
	drafterDedupHours  = 24       // 同病灶键草案去重（任何态——防灌闸）
	drafterTicket      = "T-0393" // 起草拍归属票号（例行行携环票号同 T-0376 例）
	drafterSeat        = "决策席"    // 起草行席别（DM 执行环=决策席同 hungerSeat）
)

// drafterProbeRules 探针升件类册（病灶签名规范 T-0393 件②：kind=探针类
// =升件 esc_class；key=病灶键=esc_scope）。等值匹配；"*"=任意病灶键。
// 只收探针族自报类——席位/巡检/判例面升件各走己径不入本拍（fail-closed
// 宁漏不猜：类册外签名一律不起草，升件行本身仍在案上行）。
var drafterProbeRules = []struct {
	class string // 探针类（升件 esc_class 等值）
	scope string // 病灶键域（esc_scope 等值；"*"=任意）
}{
	{"观测盲", "*"},               // T-0384 族沉默（scope=template 族名）+T-0392 腿覆盖失明（scope=inst 腿键）同款类
	{"写败", "LIFECYCLE-HUNGER"}, // 饥饿环败情/acted=false 类（hungerEscalate 自报）
	{"写败", "LIFECYCLE-MINER"},  // 教训矿工拍败情（minerEscalate 自报）
	{"账链断", "ops_event"},       // T-0386 验链拍断链/伪戳病灶
}

// drafterRow 探针升件行（起草拍视野面——esc 列非空已滤）。
type drafterRow struct {
	id    int64
	note  string
	class string
	scope string
}

// signature 病灶签名（kind:key 等值键——票面「同一病灶键」机读形）。
func (r *drafterRow) signature() string { return r.class + ":" + r.scope }

// inProbeRegistry 探针类册命中（fail-closed——册外类不碰）。
func (r *drafterRow) inProbeRegistry() bool {
	for _, rule := range drafterProbeRules {
		if rule.class == r.class && (rule.scope == "*" || rule.scope == r.scope) {
			return true
		}
	}
	return false
}

// draftPayload 物料化载荷（确定性模板填充料——docs/templates/ticket-draft.md
// 槽位逐名对应；起草非诊断，疑似根域=类册静态映射非推断）。
type draftPayload struct {
	Slug          string  `json:"slug"`
	Title         string  `json:"title"`
	Signature     string  `json:"signature"`
	Kind          string  `json:"kind"`
	LesionKey     string  `json:"lesion_key"`
	EvidenceIDs   []int64 `json:"evidence_ids"`
	EvidenceNote  string  `json:"evidence_note"`
	EvidenceQuery string  `json:"evidence_query"`
	DomainHint    string  `json:"domain_hint"`
}

// DraftTickets 起草拍一拍：扫窗聚类→签名去重→pending 草案+起草行。
// 返回本拍起草数（零产即零，benny 沉默律不落空行）。
func (s *Sensor) DraftTickets(ctx context.Context) (int, error) {
	rows, err := s.drafterScan(ctx)
	if err != nil {
		return 0, fmt.Errorf("起草拍扫窗: %w", err)
	}
	clusters := map[string][]drafterRow{}
	for _, r := range rows {
		sig := r.signature()
		clusters[sig] = append(clusters[sig], r)
	}
	var sigs []string
	for sig := range clusters {
		sigs = append(sigs, sig)
	}
	sort.Strings(sigs) // 定排——同日多签名起草序可复现
	produced := 0
	var errs []error
	for _, sig := range sigs {
		skip, skipto, err := s.drafterDedup(ctx, sig)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s 去重: %w", sig, err))
			continue
		}
		if skip {
			if err := s.drafterSkipRow(ctx, sig, skipto); err != nil {
				errs = append(errs, fmt.Errorf("%s dedup跳过行: %w", sig, err))
			}
			continue
		}
		if err := s.drafterInsert(ctx, clusters[sig]); err != nil {
			errs = append(errs, fmt.Errorf("%s 起草: %w", sig, err))
			continue
		}
		produced++
	}
	return produced, errors.Join(errs...)
}

// drafterScan 探针升件窗行（event=升件+esc 两列非空——fail-closed 签名
// 不明行本步即滤；再经类册等值筛——册外类不起草）。
func (s *Sensor) drafterScan(ctx context.Context) ([]drafterRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,note,esc_class,esc_scope FROM ops_event
		WHERE event='升件' AND ts>now()-make_interval(hours=>$1)
		  AND esc_class IS NOT NULL AND esc_class<>''
		  AND esc_scope IS NOT NULL AND esc_scope<>'' ORDER BY id`,
		drafterWindowHours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []drafterRow
	for rows.Next() {
		var r drafterRow
		if err := rows.Scan(&r.id, &r.note, &r.class, &r.scope); err != nil {
			return nil, err
		}
		if r.inProbeRegistry() {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

// drafterDedup 签名级去重三闸（票面钦③）：①pending 草案在=不重起（草案
// 自身即占位，物料化停摆期不堆同款）；②同键任意态 24h 内已起=防灌窗；
// ③同键已立票在飞（materialized 票 latest 事件未达终态=在飞）→跳过携
// 在飞票号。终态词集=完成/作废——驳回/裁决返修另生子票不释键（防重
// 立票与返修子票撞单；释键保守面见票面回填申报）。
func (s *Sensor) drafterDedup(ctx context.Context, sig string) (skip bool, skipto string, err error) {
	var pendingID int64
	err = s.pool.QueryRow(ctx, `SELECT id FROM fin_ticket_draft
		WHERE signature=$1 AND status='pending' ORDER BY id DESC LIMIT 1`, sig).Scan(&pendingID)
	if err != nil && err.Error() != "no rows in result set" {
		return false, "", err
	}
	if pendingID > 0 {
		return true, fmt.Sprintf("pending=draft#%d", pendingID), nil // pending 占位——物料化停摆期不堆同款
	}
	var ticket string
	err = s.pool.QueryRow(ctx, `SELECT ticket FROM fin_ticket_draft
		WHERE signature=$1 AND status='materialized'
		ORDER BY ts DESC LIMIT 1`, sig).Scan(&ticket)
	if err != nil && err.Error() != "no rows in result set" {
		return false, "", err
	}
	if ticket != "" {
		var last string
		if err := s.pool.QueryRow(ctx, `SELECT event FROM ops_event
			WHERE ticket=$1 ORDER BY ts DESC,id DESC LIMIT 1`, ticket).Scan(&last); err != nil {
			return false, "", fmt.Errorf("在飞判读败 ticket=%s: %w", ticket, err)
		}
		if last != "完成" && last != "作废" {
			return true, "在飞票=" + ticket, nil // 在飞票——不重复起草
		}
	}
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature=$1 AND ts>now()-make_interval(hours=>$2)`,
		sig, drafterDedupHours).Scan(&n); err != nil {
		return false, "", err
	}
	if n > 0 {
		return true, "", nil // 同窗已起（含 pending 占位）——24h 防灌
	}
	return false, "", nil
}

// drafterSkipRow dedup 跳过行落账（票面「ops_event 记 dedup 跳过行」——
// 同签名 24h 至多一记，跳过行自身不暴）。
func (s *Sensor) drafterSkipRow(ctx context.Context, sig, skipto string) error {
	mark := "dedup skip signature=" + sig
	note := mark
	if skipto != "" {
		note += " " + skipto
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT $1,'起草',$2,$3 WHERE NOT EXISTS (SELECT 1 FROM ops_event
		WHERE event='起草' AND note LIKE $4 AND ts>now()-interval '24 hours')`,
		drafterSeat, drafterTicket, note, mark+"%")
	return err
}

// drafterInsert pending 草案落库+起草行留账（同拍——自主≠无账；起草行
// ticket=环票号，物料化落号另记 `起草` 行携新票号串链）。
func (s *Sensor) drafterInsert(ctx context.Context, rows []drafterRow) error {
	first := rows[0]
	p := draftPayload{
		Slug:       draftSlug(first.class + "-" + first.scope),
		Title:      fmt.Sprintf("探针升件自动起草：%s@%s", first.class, first.scope),
		Signature:  first.signature(),
		Kind:       first.class,
		LesionKey:  first.scope,
		DomainHint: domainHint(first),
	}
	for _, r := range rows {
		p.EvidenceIDs = append(p.EvidenceIDs, r.id)
	}
	last := rows[len(rows)-1]
	note := last.note
	if len([]rune(note)) > 400 {
		note = string([]rune(note)[:400]) + "…"
	}
	p.EvidenceNote = note
	p.EvidenceQuery = fmt.Sprintf(
		"SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='%s' AND esc_scope='%s' ORDER BY id DESC LIMIT 20",
		strings.ReplaceAll(first.class, "'", "''"), strings.ReplaceAll(first.scope, "'", "''"))
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("载荷序列化: %w", err)
	}
	var id int64
	if err := s.pool.QueryRow(ctx, `INSERT INTO fin_ticket_draft
		(signature,kind,lesion_key,payload) VALUES($1,$2,$3,$4::jsonb)
		RETURNING id`, p.Signature, p.Kind, p.LesionKey, b).Scan(&id); err != nil {
		return fmt.Errorf("pending 落库: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'起草',$2,$3)`,
		drafterSeat, drafterTicket,
		fmt.Sprintf("draft#%d signature=%s n=%d evidence=#%s",
			id, p.Signature, len(rows), joinIDs(p.EvidenceIDs))); err != nil {
		return fmt.Errorf("起草行: %w", err)
	}
	return nil
}

// draftSlug 票件名 slug（ASCII kebab——非 ASCII/空白折叠横线，零词元退
// crc32 指纹兜底；engEntrySlug 同族形但无前缀——票件命名域）。
func draftSlug(key string) string {
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
		slug = fmt.Sprintf("sig-%08x", crc32.ChecksumIEEE([]byte(key)))
	}
	if len(slug) > 50 {
		slug = strings.Trim(slug[:50], "-")
	}
	return slug
}

// domainHint 疑似根域静态映射（确定性——起草非诊断，DM 裁时可改判；
// 类册外类不会到此）。
func domainHint(r drafterRow) string {
	switch {
	case r.class == "观测盲" && strings.Contains(r.scope, "-USD"):
		return "行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）"
	case r.class == "观测盲":
		return "评估×信号链面（fin_opp_eval×md_signal——T-0384 族沉默探针域）"
	case r.class == "写败" && r.scope == "LIFECYCLE-HUNGER":
		return "lifecycle 饥饿拍写径（hungerEscalate 自报面）"
	case r.class == "写败" && r.scope == "LIFECYCLE-MINER":
		return "lifecycle 教训矿工拍写径（minerEscalate 自报面）"
	case r.class == "账链断":
		return "ops_event 账链面（fin_0121 opschain 验链域）"
	}
	return "待裁——疑似域不定（fail-closed 让位不猜）"
}
