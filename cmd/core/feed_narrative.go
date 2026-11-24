// feed_narrative.go —— O·动 unit 聚合叙事层（工单O-36963 业主令「要故事线
// 非流水账」）：同 unit 会话窗（在办→会话收口）把窗内动作行收成一段白话
// 叙事——「唤知会话·尾6559 醒了，给「fix/x」签了复审、处理了报警 #a，
// 活了 18 秒干完一轮」。窗态持久化 fin_push_ledger(feed_narr_brackets)
// ——重启续跟不丢行；收口缺=超时兜底退逐行；无 unit 锚/窗外交行=现行逐行
// 渲染（叙事=加成非替换，不消灭任何信息面）。只做呈现层：ops_event 写面/
// 升件语义/探测判词零触。
package main

import (
	"context"
	"encoding/json"
	"regexp"
	"time"
)

// feedNarrTimeout 会话窗超时兜底——无闭标（会话暴毙/丢收口）窗内抑制行
// 按本限退逐行不丢行。取值=生产会话龄分布 p90≈4min 的 7 倍余量
// （max 实测 ~71min——长会话活到收口仍聚合，超时只兜「永不闭」）。
const feedNarrTimeout = 30 * time.Minute

// feedNarrLedgerKey 窗态账键（fin_push_ledger sys_status 域同 action_feed）。
const feedNarrLedgerKey = "feed_narr_brackets"

// feedNarrCloseDB 全闭标集（库查首闭判别用——会话收口 feed 可见交 Phase B
// 行序解；唤起果/唤起败 feed 剔集须库查补解）。
var feedNarrCloseDB = []string{"会话收口", "唤起果", "唤起败"}

// feedNarrAbsorb 窗内吸收事件集——会话工作产物类（醒/领/立/审/完/销/记）。
// 升件/恢复/部署/落地/判词/参数调整等系统事件不入聚合：要事不延迟，
// 窗内到达仍逐行即时推（feedPierce 语义不破）。
var feedNarrAbsorb = map[string]bool{
	"在办": true, "认领": true, "处置": true, "复审": true, "复审过": true,
	"完工回执": true, "工单完工": true, "候审回执": true, "会话收口": true,
	"工单": true, "记事": true, "挂起": true, "败": true, "勘误": true,
	"等锁": true,
}

var feedUnitRe = regexp.MustCompile(`unit=(o-[A-Za-z0-9_-]+)`)

// feedUnitOf 首枚 unit= 锚（无=""）。
func feedUnitOf(note string) string {
	if m := feedUnitRe.FindStringSubmatch(note); m != nil {
		return m[1]
	}
	return ""
}

// feedBracket 开窗态——unit 会话锚+开口行 id/时戳。
type feedBracket struct {
	Unit   string    `json:"u"`
	OpenID int64     `json:"i"`
	OpenTS time.Time `json:"t"`
}

// feedClosedRange 已闭段（唤起果/唤起败闭合专用）——闭标行 feed 剔集不
// 入 pending，其窗成员行后续批次仍按 id 序到达，留戒区抑制二次逐行渲。
type feedClosedRange struct {
	Unit    string `json:"u"`
	OpenID  int64  `json:"o"`
	CloseID int64  `json:"c"`
}

// feedPendingClose 剔集闭段待实证（O-37510①）——叙事块已产待水位实证：
// SendText 败=窗保开下拍重发叙事，非发前即闭+成员行被抑的整段静默。
// BatchHi=载叙事批行 id 上界——wm≥BatchHi=该批送成实证（闭标行 feed
// 剔集不入水位域，实证锚取批界非闭标 id）。
type feedPendingClose struct {
	Unit    string `json:"u"`
	OpenID  int64  `json:"o"`
	CloseID int64  `json:"c"`
	BatchHi int64  `json:"h"`
}

// feedNarrState 窗态持久件（fin_push_ledger msg_id JSON）。
type feedNarrState struct {
	Open    []feedBracket      `json:"o,omitempty"`
	Closed  []feedClosedRange  `json:"c,omitempty"`
	Pending []feedPendingClose `json:"p,omitempty"`
}

// narrativeOn 叙事模式闸——feed.narrative_mode 默认开（业主消费面钦定默认态；
// 键缺/坏形=开，显式 false=回逐行）。cfg 缺席（桩测裸件）=关不炸。
func (s *feedPush) narrativeOn(ctx context.Context) bool {
	if s.cfg == nil {
		return false
	}
	en, err := s.cfg.GetBool("feed.narrative_mode")
	return err != nil || en
}

// loadNarrState 窗态读——读败/缺席/坏形返空态（下拍窗内行按无窗逐行=
// 不丢行降级；已抑行的窗随坏形丢失=罕见件，超时径不兜已消费段——记档）。
func (s *feedPush) loadNarrState(ctx context.Context) *feedNarrState {
	var raw string
	err := s.pool.QueryRow(ctx,
		`SELECT msg_id FROM fin_push_ledger WHERE scope='sys_status' AND key=$1`,
		feedNarrLedgerKey).Scan(&raw)
	if err != nil || raw == "" {
		return &feedNarrState{}
	}
	var st feedNarrState
	if json.Unmarshal([]byte(raw), &st) != nil {
		return &feedNarrState{}
	}
	return &st
}

// saveNarrState 窗态写——发前持久（先账后推律同水位：窗态=抑制行的消费
// 记录，先落库再发文，发败窗态仍真下拍续跟/兜底）。
func (s *feedPush) saveNarrState(ctx context.Context, st *feedNarrState) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO fin_push_ledger(scope,key,msg_id) VALUES('sys_status',$1,$2)
		 ON CONFLICT (scope,key) DO UPDATE SET msg_id=EXCLUDED.msg_id,pushed_at=now()`,
		feedNarrLedgerKey, string(raw))
	return err
}

// narrDBCloseID 库查首闭标——返 (闭行id, 是否feed剔集闭)。首闭=会话收口
// 返 feedClose=true（Phase B 行序解，防预解后该行二次逐行=双渲+叙事错位）；
// 首闭=唤起果/唤起败 返 feedClose=false（剔集补解径）。无闭标/读败=0,true
// （按未闭让位超时兜底，不阻推送主径）。
func (s *feedPush) narrDBCloseID(ctx context.Context, unit string, afterID int64) (int64, bool) {
	var id int64
	var ev string
	if err := s.pool.QueryRow(ctx,
		`SELECT id,event FROM ops_event
		 WHERE id>$1 AND event = ANY($2)
		   AND note ~ ('unit='||$3||'([^0-9A-Za-z_-]|$)')
		 ORDER BY id LIMIT 1`, afterID, feedNarrCloseDB, unit).Scan(&id, &ev); err != nil {
		return 0, true
	}
	return id, ev == "会话收口"
}

// narrRowByID 单行回取（开/闭标行——pending 批不含者补全叙事要素）。
func (s *feedPush) narrRowByID(ctx context.Context, id int64) (feedRow, error) {
	var r feedRow
	err := s.pool.QueryRow(ctx,
		`SELECT id,seat,event,left(note,500),COALESCE(esc_class,''),COALESCE(esc_scope,''),ts
		 FROM ops_event WHERE id=$1`, id).
		Scan(&r.id, &r.seat, &r.event, &r.note, &r.escClass, &r.escScope, &r.ts)
	return r, err
}

// narrRange 他窗占用段——lo/hi=id 区间（hi=0=开区间未闭）。窗员归属=内层
// 最近开窗：仅 lo>b.OpenID 的他窗段可抢本窗无标行（更早开的窗不抢——
// 本窗是其内层）。
type narrRange struct {
	lo, hi int64
}

// narrBracketRows 窗内吸收行——(openID,hi] 开区间（hi<=0 取 uptoID 上限）
// +unit 标本窗或无标（异 unit 标行属他窗叙事不吸）。others=他窗段——
// 嵌套窗内无标行归内层窗叙事，本窗不吸（防跨窗双渲/误吞）。
func (s *feedPush) narrBracketRows(ctx context.Context, b feedBracket, closeID, uptoID int64, others []narrRange) ([]feedRow, error) {
	hi := closeID
	if hi <= 0 {
		hi = uptoID
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id,seat,event,left(note,500),COALESCE(esc_class,''),COALESCE(esc_scope,''),ts
		 FROM ops_event
		 WHERE id>$1 AND id<=$2
		   AND (note NOT LIKE '%unit=%' OR note ~ ('unit='||$3||'([^0-9A-Za-z_-]|$)'))
		 ORDER BY id`, b.OpenID, hi, b.Unit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []feedRow
	for rows.Next() {
		var r feedRow
		if err := rows.Scan(&r.id, &r.seat, &r.event, &r.note, &r.escClass, &r.escScope, &r.ts); err != nil {
			return out, err
		}
		if !feedNarrAbsorb[r.event] {
			continue // 系统事件不入叙事（窗内已逐行推过）
		}
		claimed := false
		for _, o := range others { // 内层他窗段内行归他窗（防误吞）
			if o.lo > b.OpenID && r.id > o.lo && (o.hi == 0 || r.id <= o.hi) {
				claimed = true
				break
			}
		}
		if !claimed {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

// feedNarrBlock 叙事块输出件——key=排序锚（闭行 id；逐行块=行 id；兜底
// 冲刷块=0 前置），text=成品文。
type feedNarrBlock struct {
	key  int64
	text string
}
