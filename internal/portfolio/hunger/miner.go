// miner.go —— 教训矿工拍（T-0385）：ops_event 事件账窗聚类→签名级去重→
// 达阈起草工程教训 pending（fin_eng_lesson fin_0120）→裁定席既有无头裁流
// 入工程经验库。病灶对治：KB 沉淀靠人想起来写——升件/驳回/异常/返修/收口
// 偏离签名每天在账里产，无矿工拍=高信号教训随账沉没（T-0164 式「注释自白
// 盲区」案）。只挖不裁不修——矿工零 KB 直写，判词归裁定席；不引外部服务，
// 纯 PG 账内聚类；签名去重+日额防灌+无高信号零产（benny 沉默律）。
package hunger

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
)

// 票面钦定值（T-0385 范围①④——慢拍窗/聚类阈/日额/签名冷却全票面给定，
// 不配键：键缺 fail-closed 语义非矿工域，常量在案同 silence.go 例）。
const (
	minerWindowHours = 12       // 慢拍扫窗（票面建议 12h——记忆面非急诊）
	minerClusterMin  = 2        // 聚类阈——同窗同签名≥2 起稿
	minerDailyCap    = 5        // 防灌日额——当日 pending 草案 ≤5
	minerDedupDays   = 7        // 签名冷却——同签名案 7d 内不重挖（pending 恒挡）
	minerTicket      = "T-0385" // 矿工归属票号（例行行携环票号同 T-0376 例）
	minerSeat        = "决策席"    // 矿工行席别（DM 执行环=决策席同 hungerSeat）
)

// minerRedSet 红线类级集（fin_0111 ops_precedent_trg 同款黑名单——class 或
// scope 命中即单件高危直起稿；判词仍归裁定席，矿工不代裁）。
var minerRedSet = map[string]bool{
	"资金扩投": true, "新平台": true, "席位立换": true, "红线": true,
}

// minerRow 事件账行（矿工视野面——esc 列空值归一 ”）。
type minerRow struct {
	id                 int64
	seat, event        string
	ticket, note       string
	escClass, escScope string
}

// lessonClass 行→签名类（裁决行 note 驳回*=驳回签名类——T-0382 实证双写径；
// 收口行 note 含「偏离」=收口偏离签名类——票面签名集第五件）。
func (r *minerRow) lessonClass() string {
	if r.event == "裁决" {
		return "驳回"
	}
	if r.event == "收口" {
		return "收口偏离"
	}
	return r.event
}

// signature 签名级去重键（票面钦维度）：升件带类=esc_class+esc_scope 等值键；
// 其余=类+票号维度；无票行退席别键（无票无类行以席聚类防签名串并）。
func (r *minerRow) signature() string {
	cls := r.lessonClass()
	if cls == "升件" && r.escClass != "" {
		return "升件:" + r.escClass + ":" + r.escScope
	}
	key := r.ticket
	if key == "" {
		key = r.seat
	}
	return cls + ":" + key
}

// highRisk 单件高危直起稿面（票面「≥2 同类或单件高危」）：升件=已被升件者
// 筛过的高信号（0380 ff-merge 单件在案）；驳回=返修教训最直接信号（0382
// 种键案）；异常=熔断级自报；红线类/域升件=红线撞线件。返修/收口偏离=例程
// 形签名仍走 ≥2 聚类阈（返修为驳回机械后果同窗常并见，不双起稿）。
func (r *minerRow) highRisk() bool {
	switch r.lessonClass() {
	case "升件", "驳回", "异常":
		return true
	}
	return minerRedSet[r.escClass] || minerRedSet[r.escScope]
}

// mineCluster 同签名簇（证据行保序——id 升序落库锚定）。
type mineCluster struct {
	sig  string
	rows []minerRow
}

// MineLessons 矿工一拍：扫窗聚类→签名去重→达阈/高危起草 pending。
// 返回本拍起草数（环级日志面——零产即零，沉默律不落空行）。
func (s *Sensor) MineLessons(ctx context.Context) (int, error) {
	rows, err := s.minerScan(ctx)
	if err != nil {
		return 0, fmt.Errorf("矿工扫窗: %w", err)
	}
	clusters := map[string]*mineCluster{}
	for _, r := range rows {
		c := clusters[r.signature()]
		if c == nil {
			c = &mineCluster{sig: r.signature()}
			clusters[r.signature()] = c
		}
		c.rows = append(c.rows, r)
	}
	// 定排（count desc→sig asc——日额截断面可复现）。
	var sigs []string
	for sig := range clusters {
		sigs = append(sigs, sig)
	}
	sort.Slice(sigs, func(i, j int) bool {
		a, b := clusters[sigs[i]], clusters[sigs[j]]
		if len(a.rows) != len(b.rows) {
			return len(a.rows) > len(b.rows)
		}
		return a.sig < b.sig
	})
	produced := 0
	for _, sig := range sigs {
		c := clusters[sig]
		hit := len(c.rows) >= minerClusterMin
		for _, r := range c.rows {
			hit = hit || r.highRisk()
		}
		if !hit {
			continue // 未达阈且无高危——零产
		}
		ok, err := s.minerCapLeft(ctx)
		if err != nil {
			return produced, err
		}
		if !ok {
			break // 日额闸截断——今日稿额尽
		}
		dup, err := s.minerDup(ctx, sig)
		if err != nil {
			return produced, err
		}
		if dup {
			continue // 同签名重扫=去重不产
		}
		if err := s.minerDraft(ctx, c); err != nil {
			return produced, err
		}
		produced++
	}
	return produced, nil
}

// minerScan 事件账窗行（签名集：升件/驳回/异常/返修＋裁决 note 驳回*双写件
// ＋收口 note 含「偏离」——票面钦五类签名全覆盖）。
func (s *Sensor) minerScan(ctx context.Context) ([]minerRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,seat,event,ticket,note,
		COALESCE(esc_class,''),COALESCE(esc_scope,'') FROM ops_event
		WHERE ts>now()-make_interval(hours=>$1)
		  AND (event IN ('升件','驳回','异常','返修')
		       OR (event='裁决' AND note LIKE '驳回%')
		       OR (event='收口' AND note LIKE '%偏离%'))
		ORDER BY id`, minerWindowHours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []minerRow
	for rows.Next() {
		var r minerRow
		if err := rows.Scan(&r.id, &r.seat, &r.event, &r.ticket, &r.note,
			&r.escClass, &r.escScope); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// minerCapLeft 日额闸（当日已产行 < minerDailyCap 才放行——本拍已落库行
// 随查询自见勿重复计；天然日界用 PG 日界与落库 ts 同源）。
func (s *Sensor) minerCapLeft(ctx context.Context) (bool, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_eng_lesson
		WHERE ts>=date_trunc('day',now())`).Scan(&n); err != nil {
		return false, fmt.Errorf("矿工日额读败: %w", err)
	}
	return n < minerDailyCap, nil
}

// minerDup 签名级去重：同签名 pending 恒挡；任意态 7d 冷却内重扫不产
// （verdicted 案冷却期外复发=新案可再挖——判词行不挡新病灶）。
func (s *Sensor) minerDup(ctx context.Context, sig string) (bool, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_eng_lesson
		WHERE signature=$1 AND (status='pending'
		    OR ts>now()-make_interval(days=>$2))`,
		sig, minerDedupDays).Scan(&n); err != nil {
		return false, fmt.Errorf("矿工签名去重读败: %w", err)
	}
	return n > 0, nil
}

// minerPayload 起草载荷机读五件（票面钦：签名=列/证据锚 ops_event id 列/
// 定性草案/建议条目名/验证态——验证态为列，本件载其余三件+归类辅证）。
// T-0388 扩第六件 repro_hint：可机判病灶类附桩建议文（非可编译码——工人
// 据以写真桩；omitempty 非机判类零载不硬塞）。
type minerPayload struct {
	Event          string   `json:"event"`
	Key            string   `json:"key"`
	EvidenceIDs    []int64  `json:"evidence_ids"`
	Tickets        []string `json:"tickets"`
	Draft          string   `json:"draft"`
	SuggestedEntry string   `json:"suggested_entry"`
	ReproHint      string   `json:"repro_hint,omitempty"`
}

// minerDraft pending 行落库（verify_state=ai_evaluated——机器聚类稿带实证
// 锚非 unverified；同拍 ops_event 教训矿工行留账=自主≠无账，行不在矿工
// 签名集内零自指）。
func (s *Sensor) minerDraft(ctx context.Context, c *mineCluster) error {
	p := minerPayload{Event: c.rows[0].lessonClass()}
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
	p.ReproHint = minerReproHint(c)
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
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'教训矿工',$2,$3)`,
		minerSeat, minerTicket,
		fmt.Sprintf("pending#%d signature=%s n=%d evidence=#%s",
			id, c.sig, len(c.rows), joinIDs(p.EvidenceIDs))); err != nil {
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
