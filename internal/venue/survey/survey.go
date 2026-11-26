// Package survey —— venue 上岗普查臂（O-41375 业主令「通用化钦定」：
// venue 上岗从手工单迁常驻器官——fin_venue_candidate 状态机驱动五步
// 检查表全读面，证据机写+承接面判词升件，venue-agnostic 零写单禁拍板）。
//
//	触发：state∈(probed|candidate|measuring|mapped) ∧
//	evidence.onboard.probe.symbol_params 空载 ∧ ts 陈>24h ∧
//	evidence.survey.at 退避>24h——幂等：证据列齐不重跑，重跑只补缺列。
//	产物：evidence.survey 实载（account/逐符行/spread谱/swap分布/
//	margin分层/quote活性/逐族判词/map_draft/doc_path）+onboard.probe.
//	symbol_params 补缺列（触发面同列形 exness 齐）+ops_event 升件行+
//	docs 判词 md（O_VENUE_SURVEY_DIR→$HOME/o-surveys）。
//	边界：零写单（只读 RPC 面）；臂只备证不拍板（capital/宇宙纳入归
//	红线面）；venue 名只作键不作分支（roster/凭据全数据驱动——新
//	venue=业主落 token+台账行 seed 臂自己醒）；roster 无户=skip 标
//	不硬试（非 MT5 会话面 venue 归 scout 度量臂管非本臂）。
package survey

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

const (
	seatName  = "上岗普查"
	eventName = "venue普查"
	ticketID  = "O-41375"
)

var (
	surveyInterval   = time.Hour      // 慢拍（spec 30min~1h——var=桩测覆写锚）
	surveyStaleAfter = 24 * time.Hour // 台账 ts 陈闸（spec 钉死值）
	surveyRetryGap   = 24 * time.Hour // attempt/skip 退避（var=桩测覆写锚）
	surveyMaxPerBeat = 3              // 单拍普查户数帽（var=桩测覆写锚——拍长有界）
)

// Runner 普查拍岗。md=PG-md 正本（fin_venue_candidate 写面——bridge 表
// 正本侧）；fin=SG fin（fin_venue_roster/app_secrets/ops_event 正本面——
// OpenSession 会话面同池）；cfg=md 侧 app_config 副本读（gateway 地址等
// 基建键——onboard 同径先例）。
type Runner struct {
	md, fin *pgxpool.Pool
	cfg     *config.Client
	dial    func(ctx context.Context, venue string) (Session, error)
	rej     *rejects.Door
	now     func() time.Time
}

// New 装配（dial 缺省=OpenSession 生产径；桩测覆写）。
func New(md, fin *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{md: md, fin: fin, cfg: cfg,
		dial: openSessionDial(fin, cfg),
		rej:  rejects.New("venuesurvey", "roster_err", "session_err", "collect_err", "write_err", "doc_err"),
		now:  time.Now}
}

// Run 上岗：席位注册+心跳+config_changed 事件脊+慢拍兜底（onboard.Run
// 同构——事件唤醒可选加速 spec 钦定）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.fin, seatName, "venue",
		"venue 上岗普查臂——候选台账驱动五步检查表（全读面零写单）", nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", seatName, err)
		return
	}
	go roster.BeatLoop(ctx, r.fin, seatName, 30*time.Second, errc)
	events.ListenOrDeadline(ctx, r.fin, "venue-survey", []string{"config_changed"},
		func(context.Context) time.Duration { return surveyInterval }, r.scan)
}

// scan 一拍：候选台账枚举→due 过滤→逐户普查（户败不阻他户——逐户错误
// 边界=本函数内 recover 式隔离经 surveyVenue 内部分段落账）。
func (r *Runner) scan(ctx context.Context) error {
	rows, err := r.md.Query(ctx, `SELECT venue,state,ts,evidence
		FROM fin_venue_candidate
		WHERE state IN ('probed','candidate','measuring','mapped')
		ORDER BY ts`)
	if err != nil {
		return fmt.Errorf("候选台账枚举: %w", err)
	}
	type cand struct {
		venue string
		ts    time.Time
		ev    map[string]any
	}
	var due []cand
	for rows.Next() {
		var c cand
		var ev []byte
		var state string
		if err := rows.Scan(&c.venue, &state, &c.ts, &ev); err != nil {
			rows.Close()
			return err
		}
		_ = json.Unmarshal(ev, &c.ev)
		if c.ev == nil {
			c.ev = map[string]any{}
		}
		if surveyDue(c.ev, c.ts, r.now()) {
			due = append(due, c)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for i, c := range due {
		if i >= surveyMaxPerBeat {
			break
		}
		if err := r.surveyVenue(ctx, c.venue); err != nil {
			log.Printf("[venuesurvey] %s 普查败: %v", c.venue, err)
		}
	}
	return nil
}

// surveyDue 触发判据（spec 钉死三面：symbol_params 空载∧ts 陈>24h∧
// survey.at 退避>24h——幂等面：列齐不重跑/重跑只补缺列由 evidence 合并
// 语义保证）。
func surveyDue(ev map[string]any, rowTS, now time.Time) bool {
	ob, _ := ev["onboard"].(map[string]any)
	probe, _ := ob["probe"].(map[string]any)
	if sp, ok := probe["symbol_params"].(map[string]any); ok && len(sp) > 0 {
		return false // 证据列齐不重跑（exness mapped 齐证态天然免疫）
	}
	if now.Sub(rowTS) < surveyStaleAfter {
		return false
	}
	if sv, ok := ev["survey"].(map[string]any); ok {
		if at, ok := sv["at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, at); err == nil && now.Sub(t) < surveyRetryGap {
				return false
			}
		}
	}
	return true
}

// surveyVenue 单户普查全链：roster 闸→会话→五步检查表→判词→证据+升件+
// 文档。各段败=分段落账（attempt/skip/collect/write_err）不静默吞。
func (r *Runner) surveyVenue(ctx context.Context, venue string) error {
	// roster 闸（venue-agnostic 数据判据非名单——roster 无户/停用=非
	// MT5 会话面 venue 归 scout 度量臂管，本臂 skip 标不硬试）。
	var one int
	err := r.fin.QueryRow(ctx, `SELECT 1 FROM fin_venue_roster
		WHERE venue=$1 AND state NOT IN ('disabled','removed')`, venue).Scan(&one)
	if err != nil {
		return r.markSurvey(ctx, venue, map[string]any{
			"at": r.now().UTC().Format(time.RFC3339), "skipped": "no_roster"})
	}
	sess, err := r.dial(ctx, venue)
	if err != nil {
		r.rej.Inc("session_err")
		if werr := r.markSurvey(ctx, venue, map[string]any{
			"at":         r.now().UTC().Format(time.RFC3339),
			"last_error": fmt.Sprintf("session: %v", err)}); werr != nil {
			return werr
		}
		return r.beatRow(ctx, venue, fmt.Sprintf("会话败 fail-closed：%v", err), true)
	}
	defer sess.Close()
	d, err := r.collect(ctx, sess)
	if err != nil {
		r.rej.Inc("collect_err")
		if werr := r.markSurvey(ctx, venue, map[string]any{
			"at":         r.now().UTC().Format(time.RFC3339),
			"last_error": fmt.Sprintf("collect: %v", err)}); werr != nil {
			return werr
		}
		return r.beatRow(ctx, venue, fmt.Sprintf("检查表败 fail-closed：%v", err), true)
	}
	verdicts, vcounts, pairs, err := r.verdicts(ctx, d.Symbols)
	if err != nil {
		return fmt.Errorf("判词面: %w", err)
	}
	d.FamilyVerdict = verdicts
	d.VerdictCount = vcounts
	d.MapDraft = mapDraft(venue, d.Symbols, pairs)
	docPath, err := writeDoc(venue, d, r.now())
	if err != nil {
		r.rej.Inc("doc_err")
		log.Printf("WARN [venuesurvey] %s 判词文档写败（证据不丢仅镜像缺）: %v", venue, err)
	} else {
		d.MapDraft["doc_path"] = docPath
	}
	if err := r.writeEvidence(ctx, venue, d); err != nil {
		r.rej.Inc("write_err")
		return err
	}
	return r.escalate(ctx, venue, d)
}
