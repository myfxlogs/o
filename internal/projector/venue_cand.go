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
	}
	rows.Close()
	if err := batchExec(ctx, r.sg, uVenueCand, batch); err != nil {
		return fmt.Errorf("venue_candidate 投影: %w", err)
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
