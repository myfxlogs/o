// tsops.go —— ts 判桥表（ticks/mark_index/funding/signal/dcd）的通道绑定与
// 扫描面：水印后谓词切片查询（md 正本）→参数行（SG upsert）。numeric 文本搬运
// 保精度；活跃集/dcd 集经 Runner 快照取（set 钩）。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// tsProjection ts 判桥表的通道绑定（query $1=水印 $2=键集；queryDual=dualwrite
// 旁写泵无谓词变体——$2 不传；set=nil=无谓词全量镜像道零占位参——T-0272
// 信号道撤水印）。
type tsProjection struct {
	wmKey     string
	query     string
	queryDual string
	upsert    string
	set       func(*Runner) []string
	scan      func(pgx.Rows) ([][]any, time.Time, error)
}

var tsProjections = map[string]tsProjection{
	"ev_ticks": {"md_ticks", qTicks, qTicksDual, uTicks,
		func(r *Runner) []string { return r.activeList() },
		func(rows pgx.Rows) ([][]any, time.Time, error) {
			defer rows.Close()
			var out [][]any
			var max time.Time
			for rows.Next() {
				var ts time.Time
				var venue, inst string
				var last, bid, ask, vol *string
				var recv time.Time
				if err := rows.Scan(&ts, &venue, &inst, &last, &bid, &ask, &vol, &recv); err != nil {
					return nil, max, err
				}
				out = append(out, []any{ts, venue, inst, last, bid, ask, vol, recv})
				if ts.After(max) {
					max = ts
				}
			}
			return out, max, rows.Err()
		}},
	"ev_markidx": {"md_mark_index", qMarkIdx, qMarkIdxDual, uMarkIdx,
		func(r *Runner) []string { return r.activeList() },
		func(rows pgx.Rows) ([][]any, time.Time, error) {
			defer rows.Close()
			var out [][]any
			var max time.Time
			for rows.Next() {
				var ts time.Time
				var venue, inst, channel string
				var px *string
				var recv time.Time
				if err := rows.Scan(&ts, &venue, &inst, &channel, &px, &recv); err != nil {
					return nil, max, err
				}
				out = append(out, []any{ts, venue, inst, channel, px, recv})
				if ts.After(max) {
					max = ts
				}
			}
			return out, max, rows.Err()
		}},
	"ev_funding": {"md_funding", qFunding, qFundingDual, uFunding,
		func(r *Runner) []string { return r.activeList() },
		func(rows pgx.Rows) ([][]any, time.Time, error) {
			defer rows.Close()
			var out [][]any
			var max time.Time // 水印=recv_ts（ts=未来结算拍——非单调追加序，T-0169 实证）
			for rows.Next() {
				var ts time.Time
				var venue, inst string
				var rate, nextRate *string
				var nextTs, recv *time.Time
				if err := rows.Scan(&ts, &venue, &inst, &rate, &nextRate, &nextTs, &recv); err != nil {
					return nil, max, err
				}
				out = append(out, []any{ts, venue, inst, rate, nextRate, nextTs, recv})
				if recv != nil && recv.After(max) {
					max = *recv
				}
			}
			return out, max, rows.Err()
		}},
	"ev_signal": {"md_signal", qSignal, qSignalDual, uSignal,
		nil, // T-0272：全量镜像道——无活跃集键谓词
		func(rows pgx.Rows) ([][]any, time.Time, error) {
			defer rows.Close()
			var out [][]any
			var max time.Time
			for rows.Next() {
				var kind, key string
				var val *string
				var meta []byte
				var ts time.Time
				if err := rows.Scan(&kind, &key, &val, &meta, &ts); err != nil {
					return nil, max, err
				}
				out = append(out, []any{kind, key, val, meta, ts})
				if ts.After(max) {
					max = ts
				}
			}
			return out, max, rows.Err()
		}},
	"ev_dcd": {"md_dcd_product", qDcd, qDcdDual, uDcd,
		func(r *Runner) []string { return r.dcdSet() },
		func(rows pgx.Rows) ([][]any, time.Time, error) {
			defer rows.Close()
			var out [][]any
			var max time.Time
			for rows.Next() {
				var ts time.Time
				var venue, surface, uly, product, direction string
				var strike, ann, minSz, maxSz, spot *string
				var expiry *time.Time
				var raw []byte
				if err := rows.Scan(&ts, &venue, &surface, &uly, &product, &direction,
					&strike, &expiry, &ann, &minSz, &maxSz, &spot, &raw); err != nil {
					return nil, max, err
				}
				out = append(out, []any{ts, venue, surface, uly, product, direction,
					strike, expiry, ann, minSz, maxSz, spot, raw})
				if ts.After(max) {
					max = ts
				}
			}
			return out, max, rows.Err()
		}},
}

// queryTs mode 感知查询（set=nil=无谓词全量道零参；serve=谓词切片；
// dualwrite=无谓词旁写变体）。
func (r *Runner) queryTs(ctx context.Context, p tsProjection, wm time.Time) (pgx.Rows, error) {
	if p.set == nil { // T-0272：无谓词全量镜像道（信号道撤水印——查询零占位参）
		return r.md.Query(ctx, p.query)
	}
	if r.dualwrite {
		return r.md.Query(ctx, p.queryDual, wm)
	}
	return r.md.Query(ctx, p.query, wm, p.set(r))
}

// tsMinPassInterval T-0193：ts 表族过拍最小间隔闸注册表——ticks/markidx/
// funding ≥3s（高变桥追平式全量捞塌陷醒零损=拍率供给挤兑根修），
// signal/dcd ≥1s。New 装配默认读此册（Runner.tsMinPass 实例化）。
var tsMinPassInterval = map[string]time.Duration{
	"ev_ticks": 3 * time.Second, "ev_markidx": 3 * time.Second,
	"ev_funding": 3 * time.Second, "ev_signal": time.Second, "ev_dcd": time.Second,
}

// projectTsTable 单 ts 桥表谓词切片投影（水印后增量）。
func (r *Runner) projectTsTable(ctx context.Context, ch string) error {
	p, ok := tsProjections[ch]
	if !ok {
		return nil
	}
	// T-0193 过拍最小间隔闸：不足即 return nil——每拍本就 wm 追平式
	// 全量捞（塌陷醒零损=cap1 合流语义同族），wm 幂等无行可丢。
	if min := r.tsMinPass[ch]; min > 0 {
		r.wmMu.Lock()
		if r.tsLastPass == nil {
			r.tsLastPass = map[string]time.Time{}
		}
		if last, ok := r.tsLastPass[ch]; ok && time.Since(last) < min {
			r.wmMu.Unlock()
			return nil
		}
		r.tsLastPass[ch] = time.Now()
		r.wmMu.Unlock()
	}
	wm := r.getWm(p.wmKey)
	rows, err := r.queryTs(ctx, p, wm)
	if err != nil {
		return fmt.Errorf("%s 查询: %w", p.wmKey, err)
	}
	batch, max, err := p.scan(rows)
	if err != nil {
		return fmt.Errorf("%s 扫描: %w", p.wmKey, err)
	}
	if err := batchExec(ctx, r.sg, p.upsert, batch); err != nil {
		return fmt.Errorf("%s 投影: %w", p.wmKey, err)
	}
	r.bumpWm(p.wmKey, max)
	return nil
}
