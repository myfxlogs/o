// closeoutprobe.go —— 施工收口探针（工单#25790）：完工回执落账∧超 30min
// 窗∧无同席会话收口行载该票完工→「票滞留」升件 esc_scope=
// worker_closeout_missing。治「回执/收口词形钉了没人执行」类违规——
// worker_runbook §③④ 钦定回执与收口同行席，但无机检。
//
// 配对口径：回执=seat='施工席' ∧ event∈(处置,记事) ∧ note 邻接形
// 「工单#N[空白]*完工回执」（票号与回执词间仅空白——销票/还票/验真
// 引证行天然排除）；覆盖=施工席 会话收口行 ts≥回执 且 note 完工场载
// 该票号（新契字段形「完工<票表>」∨老形「工单#N…完工」双形 Go 侧
// 解析——完工无/完工回执/挂起还票不误配）。
//
// 窗：票面钦 30min（会话死∨超窗取超窗支——死会话必超窗；活会话回执
// →收口实测间隔 ≤1.3min/30d 全谱，30min 窗零误伤面）。扫面=近 7d 回执
// 窗——更早=历史欠账非新病灶，上线前存量不追溯炸账。
//
// 域界：候选 seat 钉 '施工席'——worker_runbook §④ 契约道（o-worker/
// adhoc-ro 同格）；外派道（seat='外席'）回执历史零收口契（16 条/14d
// 实证无一收口），纳入=误报洪泛，其收口纪律归各派工单自管。
//
// 抑制同 guard/anomaly 同契：同票未销升件在案即抑、同 scope 开工单
// 在案即抑（销票/挂票过期=抑撤条件存自然复火）；查败 fail-open、写败
// 下拍重试。只报不修——补收口/裁假完工归唤知席。
package hunger

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// closeoutEscScope 升件域键（票面钦定）。
	closeoutEscScope = "worker_closeout_missing"
	// closeoutEscClass 升件类（判类册自由词——票滞留类）。
	closeoutEscClass = "票滞留"
	// closeoutMaxList 单行列举上限（超=折叠计数）。
	closeoutMaxList = 10
)

var (
	// receiptRE 回执邻接形（票号与「完工回执」间仅空白）。
	receiptRE = regexp.MustCompile(`工单#(\d+)\s*完工回执`)
	// doneFieldRE 收口新契字段形「完工<票表>」（分隔 ,，、# 空白）。
	doneFieldRE = regexp.MustCompile(`完工[=]?\s*#?\s*(\d+(?:\s*[#,，、]\s*\d+)*)`)
	// doneOldRE 收口老形「工单#N…完工」（同 ; 段邻近；完工回执引证形由 [^回] 挡）。
	doneOldRE = regexp.MustCompile(`工单#(\d+)[^;；]{0,40}完工([^回]|$)`)
	// numRE 票表切分。
	numRE = regexp.MustCompile(`\d+`)
)

// claimedDoneTickets 会话收口 note 的完工认领票集（双形并取）。
func claimedDoneTickets(note string) map[string]bool {
	out := map[string]bool{}
	for _, m := range doneFieldRE.FindAllStringSubmatch(note, -1) {
		for _, n := range numRE.FindAllString(m[1], -1) {
			out[n] = true
		}
	}
	for _, m := range doneOldRE.FindAllStringSubmatch(note, -1) {
		out[m[1]] = true
	}
	return out
}

// ProbeWorkerCloseout 施工收口探针一拍（hungerTick 6h 同槽；未超窗/全
// 覆盖=静默零行）。
func (s *Sensor) ProbeWorkerCloseout(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT id,ts,note FROM ops_event
		WHERE event IN ('处置','记事') AND seat='施工席'
		  AND note ~ '工单#[0-9]+[[:space:]]*完工回执'
		  AND ts < now()-interval '30 minutes'
		  AND ts > now()-interval '7 days'
		ORDER BY id`)
	if err != nil {
		return fmt.Errorf("回执面读败: %w", err)
	}
	type receipt struct {
		id int64
		ts time.Time
	}
	cands := map[string]receipt{}
	for rows.Next() {
		var id int64
		var ts time.Time
		var note string
		if err := rows.Scan(&id, &ts, &note); err != nil {
			rows.Close()
			return fmt.Errorf("回执行扫败: %w", err)
		}
		for _, m := range receiptRE.FindAllStringSubmatch(note, -1) {
			// 同票多回执取最新（最新回执须被覆盖——早者被同收口覆盖即毕）。
			if prev, ok := cands[m[1]]; !ok || ts.After(prev.ts) {
				cands[m[1]] = receipt{id: id, ts: ts}
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("回执行迭代败: %w", err)
	}
	if len(cands) == 0 {
		return nil
	}

	// 收口面（8d 窗=回执 7d 窗+余量）：note 完工场解析在 Go 侧（双形精确）。
	type closeout struct {
		ts     time.Time
		claims map[string]bool
	}
	crows, err := s.pool.Query(ctx, `SELECT ts,note FROM ops_event
		WHERE event='会话收口' AND seat='施工席'
		  AND ts > now()-interval '8 days'`)
	if err != nil {
		return fmt.Errorf("收口面读败: %w", err)
	}
	var closes []closeout
	for crows.Next() {
		var ts time.Time
		var note string
		if err := crows.Scan(&ts, &note); err != nil {
			crows.Close()
			return fmt.Errorf("收口行扫败: %w", err)
		}
		closes = append(closes, closeout{ts: ts, claims: claimedDoneTickets(note)})
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return fmt.Errorf("收口行迭代败: %w", err)
	}

	type missing struct {
		tid  string
		rid  int64
		ageM float64
	}
	var miss []missing
	for tid, r := range cands {
		covered := false
		for _, c := range closes {
			if !c.ts.Before(r.ts) && c.claims[tid] {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		// 同票未销升件在案即抑（销而未修=真再发照火，anomaly 同契）。
		var open bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope=$1
			AND e.note ~ ('工单#'||$2||'([^0-9]|$)')
			AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
				AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`,
			closeoutEscScope, tid).Scan(&open); err != nil {
			open = false // 查败 fail-open——升件优先不吞
		}
		if open {
			continue
		}
		miss = append(miss, missing{tid: tid, rid: r.id,
			ageM: s.now().Sub(r.ts).Minutes()})
	}
	if len(miss) == 0 {
		return nil
	}

	// 票抑同契（anomaly#24838 照 escInsert 票抑段）：同 scope 开工单在案
	// →票管催，探针静默；销票/挂票过期=抑撤条件存自然复火。
	var tick int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)') AND d.note !~ ('工单#'||t.id||' ?接续'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		closeoutEscScope).Scan(&tick); err == nil && tick > 0 {
		return nil // 开工单在案——票管催，探针静默（读败不抑=fail-open 升件优先）
	}

	sort.Slice(miss, func(i, j int) bool { return miss[i].tid < miss[j].tid })
	var parts []string
	for i, m := range miss {
		if i == closeoutMaxList {
			parts = append(parts, fmt.Sprintf("等%d票", len(miss)-closeoutMaxList))
			break
		}
		parts = append(parts, fmt.Sprintf("工单#%s(回执#%d·%.0fmin前)", m.tid, m.rid, m.ageM))
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		"施工收口缺失——"+strings.Join(parts, "；")+
			"；处置=核会话死活补会话收口行∨裁定假完工——探针只报不修",
		closeoutEscClass, closeoutEscScope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
