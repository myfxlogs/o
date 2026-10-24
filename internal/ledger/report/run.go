// run.go —— 信使岗主循环（T-0032 期界化重构）：废每拍 UPSERT 重算——期界等待
// （睡到下个 UTC 边界=日 00:00/周一/月首三者最近者，边界等待非周期拉取）；
// 边界到=把已关闭上期生成终行+推送一次（fin_push_ledger 幂等锚——重启不重推）；
// 当期零生成零推送（期中 GET /report 读已闭期史行）。日历日推同界（有货才说）。
package report

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
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

// closedBounds 已闭期界（当前时刻的前一整期：日=昨日、周=上周、月=上月）。
func closedBounds(period string, now time.Time) Bounds {
	y, m, d := now.UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	switch period {
	case "weekly":
		mon := today.AddDate(0, 0, -int(now.UTC().Weekday()+6)%7)
		return Bounds{Start: mon.AddDate(0, 0, -7), End: mon}
	case "monthly":
		first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
		return Bounds{Start: first.AddDate(0, -1, 0), End: first}
	}
	return Bounds{Start: today.AddDate(0, 0, -1), End: today}
}

// nextBoundary 下个期界（日/周/月三者最近——睡到点=边界等待非轮询）。
func nextBoundary(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	day := today.AddDate(0, 0, 1)
	mon := today.AddDate(0, 0, -int(now.UTC().Weekday()+6)%7).AddDate(0, 0, 7)
	mo := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	nxt := day
	if mon.Before(nxt) {
		nxt = mon
	}
	if mo.Before(nxt) {
		nxt = mo
	}
	return nxt
}

// Runner 信使岗：期界出行+推送（幂等——push ledger 查账防重）。
type Runner struct {
	pool   *pgxpool.Pool
	pusher *Pusher
	cfg    *config.Client
}

func NewRunner(pool *pgxpool.Pool, pusher *Pusher) *Runner {
	return &Runner{pool: pool, pusher: pusher}
}

// WithConfig 热启键接线（macro 日推 horizon 键——nil=日推静默缺）。
func (r *Runner) WithConfig(cfg *config.Client) *Runner { r.cfg = cfg; return r }

// Run 注册+Beat+期界循环：boot 补推（重启查账漏界）→睡到下界→出行推送。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "信使", "ledger", "经营报表岗——期界出行+推送+GET /report", nil); err != nil {
		errc <- fmt.Errorf("信使 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "信使", beatEvery, errc)
	for {
		r.cycle(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextBoundary(time.Now()))):
		}
	}
}

// cycle 一界：三粒度已闭期终行（未推者）生成+推送；日历日推（有货才说）。
func (r *Runner) cycle(ctx context.Context) {
	now := time.Now()
	for _, p := range []string{"daily", "weekly", "monthly"} {
		b := closedBounds(p, now)
		r.pushClosed(ctx, p, b)
	}
	r.macroDailyPush(ctx, now)
}

// pushClosed 已闭期：行缺=生成终行；未推=推一次+ledger 记账（幂等锚）。
func (r *Runner) pushClosed(ctx context.Context, period string, b Bounds) {
	key := period + ":" + b.Start.Format("2006-01-02")
	var pushed bool
	if err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM fin_push_ledger WHERE scope='report' AND key=$1)", key).Scan(&pushed); err != nil {
		log.Printf("[report] push ledger 查账败 %s: %v", key, err)
		return
	}
	if pushed {
		return // 已推期不重推（连拍/重启幂等）
	}
	if err := Gen(ctx, r.pool, period, b); err != nil {
		log.Printf("[report] %s 终行生成败: %v", key, err)
		return
	}
	var m map[string]any
	var note string
	if err := r.pool.QueryRow(ctx, `SELECT metrics,note FROM fin_report
		WHERE period=$1 AND period_start=$2`, period, b.Start).Scan(&m, &note); err != nil {
		return
	}
	r.pusher.MaybePush(ctx, period, m, note)
	if _, err := r.pool.Exec(ctx,
		"INSERT INTO fin_push_ledger(scope,key) VALUES('report',$1) ON CONFLICT DO NOTHING", key); err != nil {
		log.Printf("[report] push ledger 记账败 %s: %v", key, err)
	}
}

// macroDailyPush 财经日历日推 1 次（T-0032：近窗事件压 ≤6 行一条；无事件静默——
// 事件驱动语义有货才说）。horizon 键=research.macro.horizon_days 复用。
func (r *Runner) macroDailyPush(ctx context.Context, now time.Time) {
	key := "macro:" + now.UTC().Format("2006-01-02")
	var pushed bool
	if err := r.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM fin_push_ledger WHERE scope='macro_daily' AND key=$1)", key).Scan(&pushed); err != nil || pushed {
		return
	}
	var h float64
	if r.cfg != nil {
		if v, err := r.cfg.GetFloat("research.macro.horizon_days"); err == nil {
			h = v
		}
	}
	if h <= 0 {
		return // horizon 键缺=日推静默（fail-closed 不虚构窗）
	}
	rows, err := r.pool.Query(ctx, `SELECT event_date,kind,title FROM md_macro_event
		WHERE event_date>=CURRENT_DATE AND event_date<=CURRENT_DATE+make_interval(days => $1)
		ORDER BY event_date LIMIT 6`, int(h))
	if err != nil {
		return
	}
	type ev struct {
		d    time.Time
		k, t string
	}
	var evs []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.d, &e.k, &e.t); err != nil {
			continue
		}
		evs = append(evs, e)
	}
	rows.Close()
	if len(evs) == 0 {
		return // 无近窗事件=静默不推
	}
	text := "【PHASE 财经日历】近窗高冲击事件：\n"
	for _, e := range evs {
		text += fmt.Sprintf("- %s %s %s\n", e.d.Format("01-02"), e.k, e.t)
	}
	r.pusher.PushText(ctx, text)
	if _, err := r.pool.Exec(ctx,
		"INSERT INTO fin_push_ledger(scope,key) VALUES('macro_daily',$1) ON CONFLICT DO NOTHING", key); err != nil {
		log.Printf("[report] macro ledger 记账败: %v", err)
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
