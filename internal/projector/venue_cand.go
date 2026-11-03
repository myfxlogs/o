// venue_cand.go —— fin_venue_candidate 台账桥频道拆分件（T-0301——行数闸
// 拆分；md 正本→SG 镜像 venue 自然键全量 upsert）。
package projector

import (
	"context"
	"fmt"
	"time"
)

// projectVenueCandidates 候选台账桥投影（T-0301——md 正本→SG 镜像上行，
// venue 自然键全量 upsert：metrics/state/last_request_at 三列随行不丢列；
// 10 行小表全量镜像零水印——md_signal 同款 latest-value 形）。
func (r *Runner) projectVenueCandidates(ctx context.Context) error {
	rows, err := r.md.Query(ctx, `SELECT id,ts,venue,kind,endpoints,maker_bps,taker_bps,
		min_notional_usd,kyc_ok,api_ok,withdraw_limit,metrics,evidence,state,last_request_at
		FROM fin_venue_candidate`)
	if err != nil {
		return fmt.Errorf("venue_candidate 查询: %w", err)
	}
	var batch [][]any
	var rec int
	for rows.Next() {
		var id int64
		var ts time.Time
		var venue, kind, state string
		var endpoints, metrics, evidence []byte
		var maker, taker, minN *string
		var kyc, api bool
		var wd string
		var lastReq *time.Time
		if err := rows.Scan(&id, &ts, &venue, &kind, &endpoints, &maker, &taker, &minN,
			&kyc, &api, &wd, &metrics, &evidence, &state, &lastReq); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, []any{id, ts, venue, kind, endpoints, maker, taker, minN,
			kyc, api, wd, metrics, evidence, state, lastReq})
		if state == "recommended" {
			rec++
		}
	}
	rows.Close()
	if err := batchExec(ctx, r.sg, uVenueCand, batch); err != nil {
		return fmt.Errorf("venue_candidate 投影: %w", err)
	}
	// T-0312 第二刀②：recommended>0 唤 ev_venue_req（空批不唤 R2 同款）。
	if rec > 0 {
		if _, err := r.sg.Exec(ctx, "SELECT pg_notify('ev_venue_req','')"); err != nil {
			return fmt.Errorf("venue_req notify: %w", err)
		}
	}
	return nil
}

const uVenueCand = `INSERT INTO fin_venue_candidate(id,ts,venue,kind,endpoints,maker_bps,taker_bps,
	min_notional_usd,kyc_ok,api_ok,withdraw_limit,metrics,evidence,state,last_request_at)
	VALUES($1,$2::timestamptz,$3,$4,$5::jsonb,$6::numeric,$7::numeric,$8::numeric,$9,$10,$11,$12::jsonb,$13::jsonb,$14,$15::timestamptz)
	ON CONFLICT (venue) DO UPDATE SET ts=EXCLUDED.ts,kind=EXCLUDED.kind,endpoints=EXCLUDED.endpoints,
		maker_bps=EXCLUDED.maker_bps,taker_bps=EXCLUDED.taker_bps,min_notional_usd=EXCLUDED.min_notional_usd,
		kyc_ok=EXCLUDED.kyc_ok,api_ok=EXCLUDED.api_ok,withdraw_limit=EXCLUDED.withdraw_limit,
		metrics=EXCLUDED.metrics,evidence=EXCLUDED.evidence,state=EXCLUDED.state,
		last_request_at=EXCLUDED.last_request_at`

// projectGhostTables 两幽灵表升桥投影（T-0303——md 正本→SG 化石表镜像：
// hypothesis=UNIQUE(venue,surface) 自然键全量 upsert（parked/dismissed 等非
// proposed 态随行）；template=uq_opp_template_struct 自然键全量 upsert。
// 均小表全量镜像零水印——venue_cand 同款形）。
func (r *Runner) projectGhostTables(ctx context.Context) error {
	rows, err := r.md.Query(ctx, `SELECT id,ts,venue,surface,family,leg_primitives,requirements,state,evidence
		FROM fin_opp_hypothesis`)
	if err != nil {
		return fmt.Errorf("hypothesis 查询: %w", err)
	}
	var hb [][]any
	for rows.Next() {
		var id int64
		var ts time.Time
		var venue, surface, family, state string
		var legs, reqs, ev []byte
		if err := rows.Scan(&id, &ts, &venue, &surface, &family, &legs, &reqs, &state, &ev); err != nil {
			rows.Close()
			return err
		}
		hb = append(hb, []any{id, ts, venue, surface, family, legs, reqs, state, ev})
	}
	rows.Close()
	if err := batchExec(ctx, r.sg, uHypo, hb); err != nil {
		return fmt.Errorf("hypothesis 投影: %w", err)
	}
	rows2, err := r.md.Query(ctx, `SELECT id,template,venue,params,enabled,surface,state
		FROM fin_opp_template`)
	if err != nil {
		return fmt.Errorf("template 查询: %w", err)
	}
	var tb [][]any
	for rows2.Next() {
		var id int64
		var template, venue string
		var params []byte
		var enabled bool
		var surface, state *string
		if err := rows2.Scan(&id, &template, &venue, &params, &enabled, &surface, &state); err != nil {
			rows2.Close()
			return err
		}
		tb = append(tb, []any{id, template, venue, params, enabled, surface, state})
	}
	rows2.Close()
	return batchExec(ctx, r.sg, uTemplate, tb)
}

const uHypo = `INSERT INTO fin_opp_hypothesis(id,ts,venue,surface,family,leg_primitives,requirements,state,evidence)
	VALUES($1,$2::timestamptz,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9::jsonb)
	ON CONFLICT (venue,surface) DO UPDATE SET ts=EXCLUDED.ts,family=EXCLUDED.family,
		leg_primitives=EXCLUDED.leg_primitives,requirements=EXCLUDED.requirements,
		state=EXCLUDED.state,evidence=EXCLUDED.evidence`

const uTemplate = `INSERT INTO fin_opp_template(id,template,venue,params,enabled,surface,state)
	VALUES($1,$2,$3,$4::jsonb,$5,$6,$7)
	ON CONFLICT (template,venue,(COALESCE(surface,''))) DO UPDATE SET params=EXCLUDED.params,
		enabled=EXCLUDED.enabled,surface=EXCLUDED.surface,state=EXCLUDED.state`
