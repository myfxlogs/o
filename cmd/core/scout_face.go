// scout_face.go —— venue scout 榜单读面拆分件（T-0300 stub——行数闸拆分）。
package main

import (
	"context"
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"

	"o/internal/gen/scoutv1"
)

// Board venue scout 榜单 stub（T-0300：fin_venue_candidate 直出 top-20——
// metrics jsonb 为 scout 日拍度量榜行；前端表非本票锚）。
func (c coreService) Board(ctx context.Context, _ *scoutv1.BoardRequest) (*scoutv1.BoardResponse, error) {
	rows, err := c.pool.Query(ctx, `SELECT venue,kind,state,metrics FROM fin_venue_candidate
		ORDER BY ts DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resp := &scoutv1.BoardResponse{}
	for rows.Next() {
		var v, k, st string
		var m []byte
		if err := rows.Scan(&v, &k, &st, &m); err != nil {
			return nil, err
		}
		st_, _ := structpb.NewStruct(map[string]any{})
		var mMap map[string]any
		_ = json.Unmarshal(m, &mMap)
		if s, err := structpb.NewStruct(mMap); err == nil {
			st_ = s
		}
		resp.Rows = append(resp.Rows, &scoutv1.VenueRow{Venue: v, Kind: k, State: st, Metrics: st_})
	}
	return resp, rows.Err()
}
