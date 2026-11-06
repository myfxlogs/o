// twotier.go —— T-0401 两级采集制桥面对投影（md 正本→SG 镜像）：
// md_tick_scan（粗扫全谱最新行表，自然键全量 upsert——latest-value 小表
// 同 md_signal 形零水印）+fin_sub_request（晋级请求盒，自然键全量 upsert
// 携 state——open/closed 翻转随行）。挂 ev_funding 帧顺带+boot 首拍+
// 活性对账（venue_cand/ghost 同款挂点——小表全量镜像零独立帧）。
package projector

import (
	"context"
	"fmt"
	"time"
)

// projectTwoTier 两表全量镜像（同一拍——请求盒与粗扫面同源节拍零拆分理由）。
func (r *Runner) projectTwoTier(ctx context.Context) error {
	if err := r.projectTickScan(ctx); err != nil {
		return err
	}
	return r.projectSubRequests(ctx)
}

// projectTickScan 粗扫表全量 upsert（(venue,inst_id) 自然键——扫描臂写正本
// 每 60s 一拍，镜像新鲜度=本拍顺带；行宽固定无列膨胀）。
func (r *Runner) projectTickScan(ctx context.Context) error {
	rows, err := r.md.Query(ctx, `SELECT ts,venue,inst_id,last,bid,ask,recv_ts FROM md_tick_scan`)
	if err != nil {
		return fmt.Errorf("tick_scan 查询: %w", err)
	}
	var batch [][]any
	for rows.Next() {
		var ts, recv time.Time
		var venue, inst string
		var last, bid, ask *string
		if err := rows.Scan(&ts, &venue, &inst, &last, &bid, &ask, &recv); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, []any{ts, venue, inst, last, bid, ask, recv})
	}
	rows.Close()
	if err := batchExec(ctx, r.sg, uTickScan, batch); err != nil {
		return fmt.Errorf("tick_scan 投影: %w", err)
	}
	return nil
}

// projectSubRequests 晋级请求盒全量 upsert（(venue,inst_id,requester) 自然
// 键——state 开闭翻转随行：open→closed 退订语义靠列搬运非行删）。
// T-0412：user_intent 行 SG 正本（dashboard 建单径写）——正向镜像豁免免
// 双源互覆；其 md 副本=reverseSubRequests 反向桥产出非本面搬运域。
func (r *Runner) projectSubRequests(ctx context.Context) error {
	rows, err := r.md.Query(ctx, `SELECT ts,venue,inst_id,requester,state,reason,recv_ts
		FROM fin_sub_request WHERE requester<>'user_intent'`)
	if err != nil {
		return fmt.Errorf("sub_request 查询: %w", err)
	}
	var batch [][]any
	for rows.Next() {
		var ts, recv time.Time
		var venue, inst, requester, state, reason string
		if err := rows.Scan(&ts, &venue, &inst, &requester, &state, &reason, &recv); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, []any{ts, venue, inst, requester, state, reason, recv})
	}
	rows.Close()
	if err := batchExec(ctx, r.sg, uSubRequest, batch); err != nil {
		return fmt.Errorf("sub_request 投影: %w", err)
	}
	return nil
}

const uTickScan = `INSERT INTO md_tick_scan(ts,venue,inst_id,last,bid,ask,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4::numeric,$5::numeric,$6::numeric,$7::timestamptz)
	ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
		bid=EXCLUDED.bid,ask=EXCLUDED.ask,recv_ts=EXCLUDED.recv_ts
	WHERE EXCLUDED.ts > md_tick_scan.ts`

const uSubRequest = `INSERT INTO fin_sub_request(ts,venue,inst_id,requester,state,reason,recv_ts)
	VALUES($1::timestamptz,$2,$3,$4,$5,$6,$7::timestamptz)
	ON CONFLICT (venue,inst_id,requester) DO UPDATE SET ts=EXCLUDED.ts,
		state=EXCLUDED.state,reason=EXCLUDED.reason,recv_ts=EXCLUDED.recv_ts`
