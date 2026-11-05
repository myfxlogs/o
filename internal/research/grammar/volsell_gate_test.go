package grammar

import "testing"

func TestVolSellBootEnableAndFailStreakConfiguration(t *testing.T) {
	ctx, r := setup(t)
	r.AttachFin(r.pool)
	if err := r.ConfigureVolSell(); err != nil {
		t.Fatal(err)
	}
	if r.volSellEnabled {
		t.Fatal("migration default false must keep volsell disabled")
	}
	if _, err := r.pool.Exec(ctx, `UPDATE app_config SET value='true'::jsonb WHERE key='research.volsell.enabled'`); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.ConfigureVolSell(); err != nil {
		t.Fatal(err)
	}
	if !r.volSellEnabled || r.volSellFailN != 3 {
		t.Fatalf("boot gate config enabled=%t N=%d", r.volSellEnabled, r.volSellFailN)
	}
}

func TestVolSellRetirementRequiresMoreThanNAndFulfilledToResume(t *testing.T) {
	ctx, r := setup(t)
	r.AttachFin(r.pool)
	r.volSellFailN, r.volSellStreak, r.volSellLastErr = 1, 1, "read failure"
	if err := r.maybeRetireVolSell(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap WHERE surface='volsell-fail-streak'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("exactly N failures must not retire: rows=%d err=%v", n, err)
	}
	r.volSellStreak++
	if err := r.maybeRetireVolSell(ctx); err != nil {
		t.Fatal(err)
	}
	if stopped, err := r.volSellRetired(ctx); err != nil || !stopped {
		t.Fatalf("persisted condition must hold: stopped=%t err=%v", stopped, err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_condition_gap SET state='fulfilled' WHERE surface='volsell-fail-streak'`); err != nil {
		t.Fatal(err)
	}
	if stopped, err := r.volSellRetired(ctx); err != nil || stopped {
		t.Fatalf("only fulfilled resumes: stopped=%t err=%v", stopped, err)
	}
}
