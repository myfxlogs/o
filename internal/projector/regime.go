// regime.go —— T-0312 第九桥（md_regime_fact/memo 正本 PG-md→SG 镜像）：
// append-only 按 id 增量——SG 自对 max(id) 水印（无 wm 注册面；DO NOTHING
// 幂等）。镜像后 SELECT pg_notify('ev_regime',”) 唤推送面（钦定 LISTEN 首选径）。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Runner) projectRegime(ctx context.Context) error {
	for _, p := range []struct{ table, q, u string }{
		{"md_regime_fact", qRegimeFact, uRegimeFact},
		{"md_regime_memo", qRegimeMemo, uRegimeMemo},
	} {
		var wm int64
		if err := r.sg.QueryRow(ctx, fmt.Sprintf("SELECT COALESCE(max(id),0) FROM %s", p.table)).Scan(&wm); err != nil {
			return fmt.Errorf("%s 水印: %w", p.table, err)
		}
		rows, err := r.md.Query(ctx, p.q)
		if err != nil {
			return fmt.Errorf("%s 查询: %w", p.table, err)
		}
		var batch [][]any
		for rows.Next() {
			var id int64
			var vals []any
			if p.table == "md_regime_fact" {
				var dim, stance string
				var conf *float64
				var ev []byte
				var cts, vu *time.Time
				if err := rows.Scan(&id, &dim, &stance, &conf, &ev, &cts, &vu); err != nil {
					rows.Close()
					return err
				}
				vals = []any{id, dim, stance, conf, ev, cts, vu}
			} else {
				var ts *time.Time
				var fid *int64
				var memo string
				if err := rows.Scan(&id, &ts, &fid, &memo); err != nil {
					rows.Close()
					return err
				}
				vals = []any{id, ts, fid, memo}
			}
			if id > wm {
				batch = append(batch, vals)
			}
		}
		rows.Close()
		if err := batchExec(ctx, r.sg, p.u, batch); err != nil {
			return fmt.Errorf("%s 投影: %w", p.table, err)
		}
	}
	// 唤推送面（SG 侧 pg_notify——o-core ListenLoop 'ev_regime' 消费）。
	if _, err := r.sg.Exec(ctx, "SELECT pg_notify('ev_regime','')"); err != nil {
		return fmt.Errorf("regime notify: %w", err)
	}
	return nil
}

var _ = pgx.Rows(nil)
