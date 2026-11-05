package dcd

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetirementRequiresMoreThanNFailuresAndSurvivesRestart(t *testing.T) {
	ctx, r := setup(t)
	// fin_0112 seeds N=3. Exactly N failures do not retire the arm.
	r.failStreak = 3
	if err := r.maybeAutoRetire(ctx, errors.New("read failure")); err != nil {
		t.Fatal(err)
	}
	if r.retired || count(t, ctx, r, `SELECT count(*) FROM fin_condition_gap
		WHERE surface='dcd-fail-streak'`) != 0 {
		t.Fatal("N failures must not trigger strict >N retirement")
	}

	// N+1 atomically writes a persistent stop request and an escalation event.
	r.failStreak = 4
	if err := r.maybeAutoRetire(ctx, errors.New("read failure")); err != nil {
		t.Fatal(err)
	}
	if !r.retired {
		t.Fatal("N+1 failures did not retire DCD")
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_condition_gap
		WHERE kind='permission' AND venue='okx' AND surface='dcd-fail-streak'
		AND state='pending'`); n != 1 {
		t.Fatalf("persistent retirement request count=%d want 1", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM ops_event
		WHERE ticket='T-0353' AND event='升件' AND seat='research/dcd'`); n != 1 {
		t.Fatalf("escalation event count=%d want 1", n)
	}

	// A fresh Runner observes the SG request and does not enumerate products.
	next := NewRunner(r.pool, r.cfg).AttachFin(r.pool)
	calls := 0
	next.AttachFeed(func(context.Context, Surface) ([]Product, error) {
		calls++
		return nil, nil
	})
	if err := next.scanTracked(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || !next.retired {
		t.Fatalf("restart ignored pending retirement: calls=%d retired=%v", calls, next.retired)
	}

	// Reject/dismiss keep the stop; explicit owner fulfillment releases it.
	if _, err := r.pool.Exec(ctx, `UPDATE fin_condition_gap SET state='rejected'
		WHERE surface='dcd-fail-streak' AND state='pending'`); err != nil {
		t.Fatal(err)
	}
	if err := next.scanTracked(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("rejected retirement request must keep DCD stopped")
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_condition_gap SET state='fulfilled'
		WHERE surface='dcd-fail-streak' AND state='rejected'`); err != nil {
		t.Fatal(err)
	}
	if err := next.scanTracked(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || next.retired || next.failStreak != 0 {
		t.Fatalf("fulfilled request did not rearm cleanly: calls=%d retired=%v streak=%d",
			calls, next.retired, next.failStreak)
	}
}

func TestThrottledNoopDoesNotClearFailureStreak(t *testing.T) {
	ctx, r := setup(t)
	r.failStreak = 2
	r.lastAt = time.Now()
	if err := r.scanTracked(ctx); err != nil {
		t.Fatal(err)
	}
	if r.failStreak != 2 {
		t.Fatalf("throttled no-op falsely cleared failure streak: %d", r.failStreak)
	}
}
