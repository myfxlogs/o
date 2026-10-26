package report

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// pushFake 假 Sender（报表/日历推送计数+文本留痕）。
type pushFake struct{ texts []string }

// errFake 恒败 Sender（T-0093 发败不记账锚）。
type errFake struct{ n int }

func (f *errFake) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.n++
	return "", fmt.Errorf("stub send fail")
}

func (f *pushFake) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.texts = append(f.texts, text)
	return "mid", nil
}

func setupPush(t *testing.T) (context.Context, *Runner, *pushFake) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"notify.feishu.enabled": "true", "feishu.chat_id": "c1",
		"research.macro.horizon_days": "3"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// CI 定时炸弹根修：迁移 0020 给测试库也种真历（FOMC 10-28 等）——
	// 真时间走过界窗测试即连环炸。「无事件」语义须清全表非只清自种行。
	if _, err := pool.Exec(ctx, "DELETE FROM md_macro_event"); err != nil {
		t.Fatal(err)
	}
	f := &pushFake{}
	r := NewRunner(pool, &Pusher{cfg: cfg, pool: pool, cli: f,
		rej: rejects.New("report", "push_degraded", "push_err")}).WithConfig(cfg)
	return ctx, r, f
}

// TestBoundaryOnceOnly 期界语义锚：已闭期终行生成+仅推 1 次；连拍同 period_start
// 零重推；重启（新 Runner 同库）不重推已推期；期内（当期）零生成。
func TestBoundaryOnceOnly(t *testing.T) {
	ctx, r, f := setupPush(t)
	key := "daily:" + closedBounds("daily", time.Now()).Start.Format("2006-01-02")
	_, _ = r.pool.Exec(ctx, "DELETE FROM fin_push_ledger WHERE scope='report'")
	_, _ = r.pool.Exec(ctx, "DELETE FROM fin_report WHERE period='daily' AND period_start>=now()-interval '2 days'")
	r.cycle(ctx) // 第一界：三粒度已闭期各生成各推 1 条（日/周/月）
	if len(f.texts) != 3 {
		t.Fatalf("首界应三粒度各 1 条: %d %v", len(f.texts), f.texts)
	}
	for _, want := range []string{"日 报", "周 报", "月 报"} {
		hit := false
		for _, tx := range f.texts {
			if strings.Contains(tx, want) {
				hit = true
			}
		}
		if !hit {
			t.Fatalf("缺 %s: %v", want, f.texts)
		}
	}
	var rows int
	_ = r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_report WHERE period='daily' AND period_start>=now()-interval '2 days'").Scan(&rows)
	if rows != 1 {
		t.Fatalf("已闭期终行=%d（应 1）", rows)
	}
	f.texts = nil
	r.cycle(ctx) // 连拍：不重推不重算
	if len(f.texts) != 0 {
		t.Fatalf("连拍重推: %d", len(f.texts))
	}
	// 重启幂等：同库新 Runner 同界——ledger 查账拦（同池：testutil 每 Pool() 新 schema
	// 跨实例重启语义以同池重建 Runner 折算）
	f2 := &pushFake{}
	r2 := NewRunner(r.pool, &Pusher{cfg: r.cfg, pool: r.pool, cli: f2,
		rej: rejects.New("report", "push_degraded", "push_err")}).WithConfig(r.cfg)
	r2.cycle(ctx)
	if len(f2.texts) != 0 {
		t.Fatalf("重启重推已推期: %v", f2.texts)
	}
	// 期内零生成：当期（今日）行不存在——期内不连续重算
	var cur int
	_ = r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_report WHERE period='daily' AND period_start=date_trunc('day',now())").Scan(&cur)
	if cur != 0 {
		t.Fatal("当期行被生成——期界语义破坏（期内禁生成）")
	}
	_ = key
}

// TestWeeklyMonthlyBoundaries 周月界形状锚（closedBounds：周=上周一整周、月=上月整月）。
func TestWeeklyMonthlyBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC) // 周六
	w := closedBounds("weekly", now)
	if w.Start.Weekday() != time.Monday || w.End.Sub(w.Start) != 7*24*time.Hour {
		t.Fatalf("周界 %v→%v", w.Start, w.End)
	}
	m := closedBounds("monthly", now)
	if m.Start.Day() != 1 || m.End.Day() != 1 || !m.End.After(m.Start) {
		t.Fatalf("月界 %v→%v", m.Start, m.End)
	}
	d := closedBounds("daily", now)
	if d.Start.Day() != 23 || d.End.Day() != 24 || d.End.Hour() != 0 {
		t.Fatalf("日界 %v→%v", d.Start, d.End)
	}
	// nextBoundary：最近界=次日 00:00
	nb := nextBoundary(now)
	if nb.Day() != 25 || nb.Hour() != 0 {
		t.Fatalf("nextBoundary=%v", nb)
	}
}

// TestMacroDailyPush 日历日推锚：有近窗事件=日界一条（≤6 行）；无事件静默；
// 同日重拍不重推。
func TestMacroDailyPush(t *testing.T) {
	ctx, r, f := setupPush(t)
	_, _ = r.pool.Exec(ctx, "DELETE FROM fin_push_ledger WHERE scope='macro_daily'")
	_, _ = r.pool.Exec(ctx, "DELETE FROM md_macro_event WHERE source='t32'")
	// 无事件日：静默
	r.cycle(ctx)
	for _, tx := range f.texts {
		if strings.Contains(tx, "财经日历") {
			t.Fatal("无事件日误推日历")
		}
	}
	// 近窗事件：推一条
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_macro_event(event_date,kind,title,source)
		VALUES(CURRENT_DATE,'fomc','FOMC 利率决议','t32'),(CURRENT_DATE+1,'cpi','美国 CPI','t32')`); err != nil {
		t.Fatal(err)
	}
	f.texts = nil
	r.cycle(ctx)
	n := 0
	for _, tx := range f.texts {
		if strings.Contains(tx, "财经日历") {
			n++
			if !strings.Contains(tx, "FOMC") || !strings.Contains(tx, "CPI") {
				t.Fatalf("日历内容缺事件: %s", tx)
			}
		}
	}
	if n != 1 {
		t.Fatalf("日历推=%d（应 1）", n)
	}
	// 同日重拍：不重推
	f.texts = nil
	r.cycle(ctx)
	for _, tx := range f.texts {
		if strings.Contains(tx, "财经日历") {
			t.Fatal("同日重推日历")
		}
	}
}

// T-0093 序根修锚：SendText 返 err → fin_push_ledger 无行+push_err+1；换成功
// sender 下界重试 → 行落（幂等语义=已送达非已尝试，瞬时败≠永久丢报）。
func TestPushFailNoLedger(t *testing.T) {
	ctx, r, _ := setupPush(t)
	_, _ = r.pool.Exec(ctx, "DELETE FROM fin_push_ledger")
	_, _ = r.pool.Exec(ctx, "DELETE FROM fin_report WHERE period_start>=now()-interval '30 days'")
	fail := &errFake{}
	r.pusher.cli = fail
	r.cycle(ctx)
	var n int
	_ = r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_push_ledger WHERE scope='report'").Scan(&n)
	if n != 0 {
		t.Fatalf("发败记账了: %d 行（行=已送达语义破）", n)
	}
	if r.pusher.rej.Snapshot()["report/push_err"] == 0 {
		t.Fatal("push_err 未计数")
	}
	r.pusher.cli = &pushFake{}
	r.cycle(ctx) // 下界重试：送达→行落+msg_id 记
	_ = r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_push_ledger WHERE scope='report'").Scan(&n)
	if n != 3 {
		t.Fatalf("重试送达后 ledger 行=%d（日/周/月应 3）", n)
	}
	var mid string
	_ = r.pool.QueryRow(ctx,
		"SELECT msg_id FROM fin_push_ledger WHERE scope='report' LIMIT 1").Scan(&mid)
	if mid != "mid" {
		t.Fatalf("msg_id 未记: %q", mid)
	}
}
