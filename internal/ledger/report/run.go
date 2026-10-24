package report

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/roster"
)

const beatEvery = 30 * time.Second

// BoundsOf UTC 期界：daily=当日 00:00 起、weekly=本周一、monthly=本月 1 号。
func BoundsOf(period string, now time.Time) Bounds {
	y, m, d := now.UTC().Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch period {
	case "weekly":
		return Bounds{Start: day.AddDate(0, 0, -int(now.UTC().Weekday()+6)%7), End: now}
	case "monthly":
		return Bounds{Start: time.Date(y, m, 1, 0, 0, 0, 0, time.UTC), End: now}
	}
	return Bounds{Start: day, End: now}
}

// Runner 信使岗：周期到点各粒度幂等出行（每拍判到点——UTC 切点过则补出）。
type Runner struct{ pool *pgxpool.Pool }

func NewRunner(pool *pgxpool.Pool) *Runner { return &Runner{pool: pool} }

// Run 注册+Beat+每拍出期（幂等——重跑 UPDATE）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "信使", "ledger", "经营报表岗——日/周/月聚合单源行+GET /report", nil); err != nil {
		errc <- fmt.Errorf("信使 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "信使", beatEvery, errc)
	for {
		for _, p := range []string{"daily", "weekly", "monthly"} {
			if err := Gen(ctx, r.pool, p, BoundsOf(p, time.Now())); err != nil {
				select {
				case errc <- err:
				default:
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(beatEvery):
		}
	}
}

// Handler GET /report?period=daily|weekly|monthly&n= ——业主 curl 即得（同径供前端）。
func Handler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "daily"
		}
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 || n > 90 {
			n = 8
		}
		rows, err := pool.Query(r.Context(), `SELECT period,period_start,period_end,metrics,note,generated_at
			FROM fin_report WHERE period=$1 ORDER BY period_start DESC LIMIT $2`, period, n)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var out []map[string]any
		for rows.Next() {
			var p, note string
			var s, e, g time.Time
			var m json.RawMessage
			if err := rows.Scan(&p, &s, &e, &m, &note, &g); err != nil {
				continue
			}
			out = append(out, map[string]any{"period": p, "period_start": s, "period_end": e,
				"metrics": json.RawMessage(m), "note": note, "generated_at": g})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
