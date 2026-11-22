// closeoutprobe.go —— 施工认领配对探针（工单#26306 谓词放宽——原#25790
// 「回执→收口」配对改「认领→回执」配对）：完工回执=收口同义认，
// 配对率真源是回执不是单独收口行。施工席「认领」行超 72h∧票未销
// ∧无完工回执配对∧未挂起还票→「票滞留」升件 esc_scope=
// worker_closeout_missing。
//
// 配对口径：候选=seat='施工席' event='认领' 行（票号取 note 首个
// 「工单#N」——认领契逐票一行自号领头）；配对=认领 ts 后任一
// seat='施工席' 处置/记事行 note 邻接形「工单#N[空白]*完工回执」。
// 会话收口行不再作配对要件（收口=簿记契由 runbook+壳 EXIT trap
// 兜底保证，探针不再审它）；还票判据=收口「挂起还票<票表>」字段
// ∨任意行「工单#N…挂起条件=」邻近形（ts>认领，且晚于末次
// 「挂起解=」复办——复办后再滞留照火）；票销=ops_event_act
// act='close'（ts>认领）=生命周期终结非滞留。
//
// 窗：claim→receipt 30d 实测 p99=38h/max=40h——72h=最大值 1.8×
// 余量。扫面=近 7d 认领窗——更早=历史欠账非新病灶，上线前存量
// 不追溯炸账；证据面 8d 窗恒覆盖（认领窗内票的回执/还票/复办
// 必落窗内——ts>认领>now-7d）。
//
// 域界：候选 seat 钉 '施工席'——worker_runbook §④ 契约道；外派道
// （seat='外席'）收口纪律归各派工单自管，纳入=误报洪泛。
//
// 抑制同契：同票未销升件在案即抑、同 scope 开工单在案即抑；查败
// fail-open、写败下拍重试。只报不修——补回执/还票/裁定归唤知席。
package hunger

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	// claimRE 认领行票号（首个「工单#N」=自号领头——认领契逐票一行）。
	claimRE = regexp.MustCompile(`工单#(\d+)`)
	// receiptRE 完工回执邻接形（票号与「完工回执」间仅空白——销票/
	// 还票/验真引证行天然排除）。
	receiptRE = regexp.MustCompile(`工单#(\d+)[\s　]*完工回执`)
	// returnFieldRE 收口还票字段「挂起还票<票表>」（分隔 ,，、# 空白；
	// 「无/；/：」形天然不配）。
	returnFieldRE = regexp.MustCompile(`挂起还票[=]?[\s　]*#?[\s　]*(\d+(?:[\s　]*[#,，、][\s　]*\d+)*)`)
	// suspendAdjRE 处置还票邻近形「工单#N…挂起条件=」（≤20 非段字符——
	// 「挂起条件已解」无「=」不配=非还票；认领行尾缀同形亦不配）。
	suspendAdjRE = regexp.MustCompile(`工单#(\d+)[^;；\n]{0,20}挂起条件=`)
	// resumeAdjRE 复办邻近形「工单#N…挂起解=」——复办撤销在先还票。
	resumeAdjRE = regexp.MustCompile(`工单#(\d+)[^;；\n]{0,20}挂起解=`)
	// numRE 票表切分。
	numRE = regexp.MustCompile(`\d+`)
)

// ProbeWorkerCloseout 施工认领配对探针一拍（hungerTick 6h 同槽；窗内
// 全配对=静默零行）。
func (s *Sensor) ProbeWorkerCloseout(ctx context.Context) error {
	// ① 认领面（72h 超龄窗×7d 扫面）。
	rows, err := s.pool.Query(ctx, `SELECT id,ts,note FROM ops_event
		WHERE event='认领' AND seat='施工席'
		  AND ts < now()-interval '72 hours'
		  AND ts > now()-interval '7 days'
		ORDER BY id`)
	if err != nil {
		return fmt.Errorf("认领面读败: %w", err)
	}
	type claim struct {
		cid int64
		ts  time.Time
		tid string
	}
	var claims []claim
	for rows.Next() {
		var cid int64
		var ts time.Time
		var note string
		if err := rows.Scan(&cid, &ts, &note); err != nil {
			rows.Close()
			return fmt.Errorf("认领行扫败: %w", err)
		}
		if m := claimRE.FindStringSubmatch(note); m != nil {
			claims = append(claims, claim{cid: cid, ts: ts, tid: m[1]})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("认领行迭代败: %w", err)
	}
	if len(claims) == 0 {
		return nil
	}

	// ② 回执面（8d 窗）：邻接形双闸——SQL ~ 粗筛+Go 精解析。
	receipts := map[string][]time.Time{}
	rrows, err := s.pool.Query(ctx, `SELECT ts,note FROM ops_event
		WHERE event IN ('处置','记事') AND seat='施工席'
		  AND note ~ '工单#[0-9]+[[:space:]　]*完工回执'
		  AND ts > now()-interval '8 days'`)
	if err != nil {
		return fmt.Errorf("回执面读败: %w", err)
	}
	for rrows.Next() {
		var ts time.Time
		var note string
		if err := rrows.Scan(&ts, &note); err != nil {
			rrows.Close()
			return fmt.Errorf("回执行扫败: %w", err)
		}
		for _, m := range receiptRE.FindAllStringSubmatch(note, -1) {
			receipts[m[1]] = append(receipts[m[1]], ts)
		}
	}
	rrows.Close()
	if err := rrows.Err(); err != nil {
		return fmt.Errorf("回执行迭代败: %w", err)
	}

	// ③ 还票/复办面（8d 窗全事件——收口挂起还票字段+处置邻近形并取）。
	returns := map[string][]time.Time{}
	resumes := map[string][]time.Time{}
	lrows, err := s.pool.Query(ctx, `SELECT ts,note FROM ops_event
		WHERE seat='施工席' AND ts > now()-interval '8 days'`)
	if err != nil {
		return fmt.Errorf("还票面读败: %w", err)
	}
	for lrows.Next() {
		var ts time.Time
		var note string
		if err := lrows.Scan(&ts, &note); err != nil {
			lrows.Close()
			return fmt.Errorf("还票行扫败: %w", err)
		}
		for _, m := range returnFieldRE.FindAllStringSubmatch(note, -1) {
			for _, n := range numRE.FindAllString(m[1], -1) {
				returns[n] = append(returns[n], ts)
			}
		}
		for _, m := range suspendAdjRE.FindAllStringSubmatch(note, -1) {
			returns[m[1]] = append(returns[m[1]], ts)
		}
		for _, m := range resumeAdjRE.FindAllStringSubmatch(note, -1) {
			resumes[m[1]] = append(resumes[m[1]], ts)
		}
	}
	lrows.Close()
	if err := lrows.Err(); err != nil {
		return fmt.Errorf("还票行迭代败: %w", err)
	}

	// ④ 票销面：act='close' 末次 ts（生命周期终结=非滞留）。
	closes := map[string]time.Time{}
	crows, err := s.pool.Query(ctx, `SELECT rel,max(ts) FROM ops_event_act
		WHERE act='close' GROUP BY rel`)
	if err != nil {
		return fmt.Errorf("销票面读败: %w", err)
	}
	for crows.Next() {
		var rel int64
		var ts time.Time
		if err := crows.Scan(&rel, &ts); err != nil {
			crows.Close()
			return fmt.Errorf("销票行扫败: %w", err)
		}
		closes[strconv.FormatInt(rel, 10)] = ts
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		return fmt.Errorf("销票行迭代败: %w", err)
	}

	maxTs := func(m map[string][]time.Time, tid string) time.Time {
		var t time.Time
		for _, x := range m[tid] {
			if x.After(t) {
				t = x
			}
		}
		return t
	}
	type missing struct {
		tid  string
		cid  int64
		ageH float64
	}
	var miss []missing
	for _, c := range claims {
		paired := false
		for _, rts := range receipts[c.tid] {
			if rts.After(c.ts) {
				paired = true
				break
			}
		}
		if paired {
			continue
		}
		// 还票在案且晚于末次复办=仍处还票态；复办后再滞留照火。
		lastR, lastU := maxTs(returns, c.tid), maxTs(resumes, c.tid)
		if lastR.After(c.ts) && lastR.After(lastU) {
			continue
		}
		if ct, ok := closes[c.tid]; ok && ct.After(c.ts) {
			continue
		}
		// 同票未销升件在案即抑（销而未修=真再发照火，anomaly 同契）。
		var open bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope=$1
			AND e.note ~ ('工单#'||$2||'([^0-9]|$)')
			AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
				AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`,
			closeoutEscScope, c.tid).Scan(&open); err != nil {
			open = false // 查败 fail-open——升件优先不吞
		}
		if open {
			continue
		}
		miss = append(miss, missing{tid: c.tid, cid: c.cid,
			ageH: s.now().Sub(c.ts).Hours()})
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
		parts = append(parts, fmt.Sprintf("工单#%s(认领#%d·%.0fh前)", m.tid, m.cid, m.ageH))
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		"施工认领未配对——"+strings.Join(parts, "；")+
			"；处置=核会话死活补完工回执（活干完）∨挂起条件=还票（没干完）——探针只报不修",
		closeoutEscClass, closeoutEscScope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
