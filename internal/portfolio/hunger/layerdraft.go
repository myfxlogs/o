// layerdraft.go —— T-0508 层扫异常→诊断票自动进水口（CEO自运感知票B）：
// T-0507 层扫窗升件行（seat=MECH event=升件 note=「层扫定性<类目>:<对象>
// ——<详情>」）→fin_ticket_draft 诊断型草案（kind=diagnosis；signature=
// 「层扫:<类目>:<对象>」——票面钉死字面，类目/对象取升件行载荷原文
// 不改写）。物料化走既有 ticketdraft 正径（pending→docs/tasks 候选票）
// 零新件零直写票册；本件只产草案行——根因定性/修法选择/根修票立件
// 全归现行裁定径，本票不发明任何判断（禁线钉死）。
//
// dedup 判据（票面钉）：同对象同类目「在飞/已诊不重复起」——
//
//	①pending 草案在=占位不堆同款（drafterDedup 同例）；
//	②materialized 票在飞（latest 事件未达终态）=不重复起；
//	③materialized 票终态（完成/作废——先例终态词集同源）=已诊，
//	  同 signature 后续升件不再起草（复发≠新草案——消行语义即
//	  签名级永闭，dm_queue 终态出队律同构）；
//	④同签名 24h 内任意态草案=防灌窗（dismissed/merged 边缘面同
//	  drafterDedup③——已诊签名永不复出，窗内闸只管未诊签名）。
//
// 起草拍挂 hungerTick 同槽（DraftTickets 双径编排）——6h 拍律与
// 探针族起草同源，升件行本身 24h dedup 产速天然限流。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// 层扫升件契约常量（T-0507 票面/代码双钉——note 前缀+seat 写席）。
const (
	layerScanPrefix   = "层扫定性"      // note 前缀（ops_esc 去重键同根）
	layerScanSeat     = "MECH"      // 升件写席（T-0506 语义闸钦定）
	layerScanKind     = "diagnosis" // 诊断型草案 kind（票面钉死）
	layerScanWindowH  = 24          // 扫窗——与探针起草拍同窗（升件行自身 24h dedup 产速同源）
	layerScanPairSep  = "——"        // note 内 类目:对象 与 详情 分隔（ops_esc 统一加）
	layerDraftMaxNote = 400         // 末行原文截长（drafterInsert 同阈）
)

// layerSelfDisposeCats 归领自销类目（gateway/escalate.sh 分类归领
// case 同源——机械可证类升件归领当场自销不起票；进水口不再产草案=
// 物料化不占号不烧号，T-0672 断源钉。两表须同扩——归领新增机械
// 自销类目时本表同步）。
var layerSelfDisposeCats = map[string]bool{
	"done-翻漏":   true,
	"幽灵票件":      true,
	"无籍行":       true,
	"done-无收口链": true,
}

// layerScanRow 层扫升件窗行（升件行载荷四元组——id 留证据引、
// ts 留首报时间、note 拆 signature、ticket 原样留证不入签名）。
type layerScanRow struct {
	id     int64
	ts     time.Time
	note   string
	ticket string
	pair   string // 「<类目>:<对象>」——升件行载荷原文（signature 尾段）
	cat    string // 类目（pair 首个 ':' 前——类目词表钉死无冒号）
	obj    string // 对象（pair 首个 ':' 后——票号/稳定键原文）
}

// signature 病灶签名（票面钉死「层扫:<类目>:<对象>」——pair 原文直嵌，
// 对象含冒号等形不改写不丢段）。
func (r *layerScanRow) signature() string { return "层扫:" + r.pair }

// parseLayerScanNote 升件行 note→载荷四元组（fail-closed：前缀/分隔/
// 类目对象三要件缺一即弃——畸形行不起草只留账面原行）。
func parseLayerScanNote(id int64, ts time.Time, ticket, note string) (*layerScanRow, bool) {
	if !strings.HasPrefix(note, layerScanPrefix) {
		return nil, false
	}
	rest := note[len(layerScanPrefix):]
	pair := rest
	if i := strings.Index(rest, layerScanPairSep); i >= 0 {
		pair = rest[:i]
	}
	ci := strings.Index(pair, ":")
	if ci <= 0 || ci == len(pair)-1 {
		return nil, false // 无冒号/空类目/空对象=签名残缺弃行
	}
	return &layerScanRow{
		id: id, ts: ts, note: note, ticket: ticket,
		pair: pair, cat: pair[:ci], obj: pair[ci+1:],
	}, true
}

// layerScanRows 层扫升件窗行（seat=MECH+event=升件+note 前缀钉死——
// 非本契约行一律不入本径；esc_class/esc_scope 空=探针起草径结构性
// 不相交，双向零串味）。已认领升件不进起草口（后随同票终态词
// {裁决,完成,驳回,挂起,作废}或 note 引「升件#id」处置行=归领已销——
// 已销签名再起草=白烧号，T-0672 断源钉）。
func (s *Sensor) layerScanRows(ctx context.Context) ([]layerScanRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT e.id,e.ts,e.note,e.ticket FROM ops_event e
		WHERE e.seat=$1 AND e.event='升件' AND e.note LIKE $2||'%'
		  AND e.ts>now()-make_interval(hours=>$3)
		  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts AND x.seat<>'gateway'
		    AND ((x.ticket=e.ticket AND e.ticket<>'' AND x.event IN ('裁决','完成','驳回','挂起','作废'))
		      OR x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))
		ORDER BY e.id`,
		layerScanSeat, layerScanPrefix, layerScanWindowH)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []layerScanRow
	for rows.Next() {
		var id int64
		var ts time.Time
		var note, ticket string
		if err := rows.Scan(&id, &ts, &note, &ticket); err != nil {
			return nil, err
		}
		if r, ok := parseLayerScanNote(id, ts, ticket, note); ok {
			out = append(out, *r)
		}
	}
	return out, rows.Err()
}

// draftLayerScans 层扫起草径一拍：窗行聚类→签名 dedup→pending 草案。
// 返回本拍起草数（含探针径由编排面合并计数——本函数只管层扫签名族）。
func (s *Sensor) draftLayerScans(ctx context.Context) (int, error) {
	rows, err := s.layerScanRows(ctx)
	if err != nil {
		return 0, fmt.Errorf("层扫起草拍扫窗: %w", err)
	}
	clusters := map[string][]layerScanRow{}
	for _, r := range rows {
		sig := r.signature()
		clusters[sig] = append(clusters[sig], r)
	}
	var sigs []string
	for sig := range clusters {
		sigs = append(sigs, sig)
	}
	sort.Strings(sigs) // 定排——同日多签名起草序可复现（drafter 同例）
	produced := 0
	var errs []error
	for _, sig := range sigs {
		if layerSelfDisposeCats[clusters[sig][0].cat] {
			continue // 归领自销类——当场销不起票不进草案（T-0672 号耗断源）
		}
		skip, skipto, err := s.layerDedup(ctx, sig)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s 层扫去重: %w", sig, err))
			continue
		}
		if skip {
			if err := s.drafterSkipRow(ctx, sig, skipto); err != nil {
				errs = append(errs, fmt.Errorf("%s 层扫dedup跳过行: %w", sig, err))
			}
			continue
		}
		if err := s.layerInsert(ctx, clusters[sig]); err != nil {
			errs = append(errs, fmt.Errorf("%s 层扫起草: %w", sig, err))
			continue
		}
		produced++
	}
	return produced, errors.Join(errs...)
}

// layerDedup 签名级去重（票面钉：在飞/已诊不重复起）——
// ①pending 草案占位；②materialized 票 latest 事件终态词集
// {完成,作废}（drafterDedup 同源）→已诊永闭；非终态→在飞占位；
// ③同签名 24h 任意态草案=防灌窗（未诊签名边缘面）。
func (s *Sensor) layerDedup(ctx context.Context, sig string) (skip bool, skipto string, err error) {
	var pendingID int64
	err = s.pool.QueryRow(ctx, `SELECT id FROM fin_ticket_draft
		WHERE signature=$1 AND status='pending' ORDER BY id DESC LIMIT 1`, sig).Scan(&pendingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, "", err
	}
	if pendingID > 0 {
		return true, fmt.Sprintf("pending=draft#%d", pendingID), nil
	}
	var ticket string
	err = s.pool.QueryRow(ctx, `SELECT ticket FROM fin_ticket_draft
		WHERE signature=$1 AND status='materialized'
		ORDER BY ts DESC,id DESC LIMIT 1`, sig).Scan(&ticket)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, "", err
	}
	if ticket != "" {
		var last string
		lerr := s.pool.QueryRow(ctx, `SELECT event FROM ops_event
			WHERE ticket=$1 ORDER BY ts DESC,id DESC LIMIT 1`, ticket).Scan(&last)
		if lerr != nil && !errors.Is(lerr, pgx.ErrNoRows) {
			return false, "", fmt.Errorf("在飞/已诊判读败 ticket=%s: %w", ticket, lerr)
		}
		if last == "完成" || last == "作废" {
			return true, "已诊票=" + ticket, nil // 终态=已诊——复发≠新草案（票面消行语义）
		}
		return true, "在飞票=" + ticket, nil // 在飞占位（含无事件链/挂起/审中——非终态全占位）
	}
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature=$1 AND ts>now()-make_interval(hours=>$2)`,
		sig, layerScanWindowH).Scan(&n); err != nil {
		return false, "", err
	}
	if n > 0 {
		return true, "", nil // 同窗已起（含 dismissed/merged 边缘面）——24h 防灌
	}
	return false, "", nil
}

// layerInsert pending 草案落库+起草行留账（drafterInsert 同构——
// 物料化件=层扫行载荷成形：类目/对象/首报时间/累计计数轨迹；不编
// 根因不编修法——domain_hint 只指认层扫类目定义面不预设病灶）。
func (s *Sensor) layerInsert(ctx context.Context, rows []layerScanRow) error {
	first := rows[0]
	last := rows[len(rows)-1]
	sig := first.signature()
	// 首报时间/计数轨迹=签名全期聚合（'——' 定界 LIKE 主径+无分隔
	// 畸形行 exact 兜底——pair 前缀撞形不误并；窗内行数另计载窗轨迹）。
	// LIKE 元字符转义——pair 来自行载荷理论可含 %/_，聚合计数只影响
	// 轨迹文案不入 dedup。
	var total int
	var firstTS *time.Time
	escPair := strings.NewReplacer(
		`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(first.pair)
	like := layerScanPrefix + escPair + layerScanPairSep + "%"
	exact := layerScanPrefix + first.pair
	if err := s.pool.QueryRow(ctx, `SELECT count(*),min(ts) FROM ops_event
		WHERE seat=$1 AND event='升件' AND (note LIKE $2 OR note=$3)`,
		layerScanSeat, like, exact).Scan(&total, &firstTS); err != nil {
		return fmt.Errorf("签名聚合读败: %w", err)
	}
	traj := fmt.Sprintf("签名全期升件 %d 行", total)
	if firstTS != nil {
		traj = fmt.Sprintf("首报 %s｜%s", firstTS.UTC().Format(time.RFC3339), traj)
	}
	traj += fmt.Sprintf("｜近%d h窗 %d 行", layerScanWindowH, len(rows))
	note := last.note
	if len([]rune(note)) > layerDraftMaxNote {
		note = string([]rune(note)[:layerDraftMaxNote]) + "…"
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.id)
	}
	p := draftPayload{
		Slug:         layerSlug(sig),
		Title:        fmt.Sprintf("层扫异常诊断票:%s@%s", first.cat, first.obj),
		Signature:    sig,
		Kind:         layerScanKind,
		LesionKey:    first.pair,
		EvidenceIDs:  ids,
		EvidenceNote: fmt.Sprintf("%s｜%s", traj, note),
		EvidenceQuery: fmt.Sprintf(
			"SELECT id,ts,seat,note FROM ops_event WHERE seat='%s' AND event='升件' AND note LIKE '%s' ORDER BY id DESC LIMIT 20",
			layerScanSeat, strings.ReplaceAll(like, "'", "''")),
		DomainHint: fmt.Sprintf(
			"层扫类目「%s」——T-0507 层扫窗机械枚举不变量破例（类目定义面=layer_scan.sh）；根因定性/修法/根修票立件归现行裁定径，本票=诊断票不预设根因", first.cat),
	}
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

// layerSlug 物料化票件 slug（ASCII kebab+签名 crc 尾——materialize
// 幂等 glob 按 slug 全域匹配，同 slug 异签名会误并票号；crc 尾钉死
// 签名级唯一，base 取 draftSlug 可读面截 33 字符让位）。
func layerSlug(sig string) string {
	suffix := fmt.Sprintf("-%06x", crc32.ChecksumIEEE([]byte(sig))&0xffffff)
	base := draftSlug(sig)
	maxBase := 50 - len("layerscan-") - len(suffix)
	if len(base) > maxBase {
		base = strings.Trim(base[:maxBase], "-")
	}
	return "layerscan-" + base + suffix
}
