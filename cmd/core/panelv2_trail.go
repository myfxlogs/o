// panelv2_trail.go —— 决策径读面（panelv2_face.go 拆出——棘轮 300 行件限）。
package main

import (
	"context"
	"log"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// ── 决策径 GetDecisionTrail ──

// GetDecisionTrail 升件→处置链尾流（首条引证处置=判词锚）+条件请求榜+
// 教训矿面。引证谓词与 escalateUnresolved/runbook① 同形不复述。
func (c coreService) GetDecisionTrail(ctx context.Context,
	req *dashboardv1.GetDecisionTrailRequest) (*dashboardv1.GetDecisionTrailResponse, error) {
	lim := int(req.GetLimit())
	if lim <= 0 || lim > 100 {
		lim = 20
	}
	resp := &dashboardv1.GetDecisionTrailResponse{AsOf: timestamppb.Now()}

	// 引用端物化（SG CPU 烧实证同型改——逐升件 LATERAL regex 全表扫→
	// 引用集单程抽出,per-e 首引行=物化集内 min xid,毫秒级）。
	rs, err := c.pool.Query(ctx, `WITH cited AS MATERIALIZED (
		SELECT x.id xid, x.ts xts, x.note xnote,
			(m.mm)[1]::bigint lo, COALESCE((m.mm)[2]::bigint,(m.mm)[1]::bigint) hi
		FROM ops_event x, LATERAL regexp_matches(x.note,$2,'g') m(mm)
		WHERE x.note LIKE '%升件#%')
		SELECT e.id, e.ts, coalesce(e.esc_class,''), coalesce(e.esc_scope,''),
		left(e.note,200), d.xid, d.xts, left(d.xnote,200)
		FROM ops_event e
		LEFT JOIN LATERAL (
			SELECT c.xid, c.xts, c.xnote FROM cited c
			WHERE c.xid>e.id AND e.id BETWEEN c.lo AND c.hi
			ORDER BY c.xid LIMIT 1
		) d ON true
		WHERE e.event='升件' ORDER BY e.id DESC LIMIT $1`, lim, `升件#([0-9]+)(?:-([0-9]+))?`)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	for rs.Next() {
		var eid int64
		var ets time.Time
		var cls, scope, note string
		var did *int64
		var dts *time.Time
		var dnote *string
		if err := rs.Scan(&eid, &ets, &cls, &scope, &note, &did, &dts, &dnote); err != nil {
			return nil, err
		}
		tr := &dashboardv1.TrailRow{EscId: eid, EscTs: timestamppb.New(ets.UTC()),
			EscClass: cls, EscScope: scope, Note: note}
		if did != nil {
			tr.DispId = *did
			tr.DispTs = timestamppb.New(dts.UTC())
			// 判词截「升件#N 销——」前缀（escId 行头已显，裸注重复——审计 P2）。
			v := *dnote
			if i := strings.Index(v, "——"); i >= 0 {
				v = v[i+len("——"):]
			}
			tr.Verdict = v
			tr.AgeS = int64(dts.Sub(ets).Seconds())
		} else {
			tr.Open = true
			tr.AgeS = int64(time.Since(ets).Seconds())
		}
		resp.Trail = append(resp.Trail, tr)
	}
	if err := rs.Err(); err != nil {
		return nil, err
	}

	// 条件请求榜：近 30d 行+open=无引证行（同升件引证谓词但键=条件请求#id）。
	if crs, err := c.pool.Query(ctx, `SELECT e.id, e.ts, left(e.note,200),
		NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('条件请求#'||e.id::text||'([^0-9]|$)')) AS open
		FROM ops_event e WHERE e.event='条件请求' AND e.ts>now()-interval '30 days'
		ORDER BY e.id DESC LIMIT 8`); err == nil {
		for crs.Next() {
			var id int64
			var ts time.Time
			var note string
			var open bool
			if crs.Scan(&id, &ts, &note, &open) == nil {
				resp.Condreqs = append(resp.Condreqs, &dashboardv1.CondReqRow{
					Id: id, Ts: timestamppb.New(ts.UTC()), Note: note, Open: open})
			}
		}
		crs.Close()
	} else {
		log.Printf("[trail] 条件请求榜查询败: %v", err)
	}

	// 器官事实面：ops_event 全事件类近窗行——工位态推导数据源（工单/处置/
	// 收口等器官事实不在升件类，前端按器官关键词自取、取词邻域判态）。
	if frs, err := c.pool.Query(ctx, `SELECT id, ts, event, left(note,220)
		FROM ops_event ORDER BY id DESC LIMIT 600`); err == nil {
		for frs.Next() {
			var fr dashboardv1.FactRow
			var fts time.Time
			if frs.Scan(&fr.Id, &fts, &fr.Event, &fr.Note) == nil {
				fr.Ts = timestamppb.New(fts.UTC())
				resp.Facts = append(resp.Facts, &fr)
			}
		}
		frs.Close()
	} else {
		log.Printf("[trail] 器官事实面查询败: %v", err)
	}

	// 矿面：fin_eng_lesson 状态计数+fin_econ_lesson pending。
	mine := &dashboardv1.MineRow{}
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE status='pending'),
		count(*) FILTER (WHERE status='canon'),
		count(*) FILTER (WHERE status='verdicted') FROM fin_eng_lesson`).
		Scan(&mine.EngPending, &mine.EngCanon, &mine.EngVerdicted); err != nil {
		log.Printf("[trail] eng 矿查询败: %v", err)
	}
	// econ canon 语义在 verdict_ref 前缀（runbook：econ 禁写 'canon' 状态——
	// status='canon' 恒零是测错口径非真空架）。OR 双轨兼容两制。
	if err := c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE status='pending'),
		count(*) FILTER (WHERE status='canon' OR verdict_ref LIKE 'canon:%'),
		count(*) FILTER (WHERE status='verdicted') FROM fin_econ_lesson`).
		Scan(&mine.EconPending, &mine.EconCanon, &mine.EconVerdicted); err != nil {
		log.Printf("[trail] econ 矿查询败: %v", err)
	}
	resp.Mine = mine
	return resp, nil
}
