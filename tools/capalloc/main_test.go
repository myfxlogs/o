package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRow struct{ vals []any }

func (r fakeRow) Scan(dest ...any) error {
	for i := range dest {
		switch p := dest[i].(type) {
		case *float64:
			if v, ok := r.vals[i].(float64); ok {
				*p = v
			}
		case *string:
			if v, ok := r.vals[i].(string); ok {
				*p = v
			}
		}
	}
	return nil
}

// fakeStore 键值账面——equity/app_config 由测试播种；exec 记写痕。
type fakeStore struct {
	eq    float64
	eqErr error
	cfg   map[string]float64
	old   map[string]string
	execs []string
}

func (s *fakeStore) QueryRow(_ context.Context, sql string, args ...any) pgxRow {
	if strings.Contains(sql, "md_equity_curve") {
		if s.eqErr != nil {
			return errRow{s.eqErr}
		}
		return fakeRow{[]any{s.eq}}
	}
	k := args[0].(string)
	if strings.Contains(sql, "float8") {
		return fakeRow{[]any{s.cfg[k]}}
	}
	return fakeRow{[]any{s.old[k]}}
}

func (s *fakeStore) Exec(_ context.Context, sql string, _ ...any) (int64, error) {
	s.execs = append(s.execs, sql)
	return 1, nil
}

type errRow struct{ err error }

func (r errRow) Scan(_ ...any) error { return r.err }

func base() *fakeStore {
	return &fakeStore{
		eq:  16000,
		cfg: map[string]float64{"risk.max_exposure_usd": 1300, "risk.max_notional_usd": 1000, "invest.paper.notional_usd": 50},
		old: map[string]string{"risk.max_exposure_usd": "1300"},
	}
}

func TestCapallocWhitelist(t *testing.T) {
	err := run(context.Background(), base(), "risk.kill_switch", 1, "x")
	if err == nil || !strings.Contains(err.Error(), "册外键") {
		t.Fatalf("kill_switch 应册外拒, got %v", err)
	}
	err = run(context.Background(), base(), "risk.max_exposure_usd", 3000, "")
	if err == nil {
		t.Fatal("空证据应拒")
	}
}

func TestCapallocEnvelope(t *testing.T) {
	err := run(context.Background(), base(), "risk.max_exposure_usd", 9000, "e")
	if err == nil || !strings.Contains(err.Error(), "界外") {
		t.Fatalf("9000>0.5×16000 应界外拒, got %v", err)
	}
	err = run(context.Background(), base(), "risk.max_exposure_usd", 3000, "族净正实证")
	if err != nil {
		t.Fatalf("3000 应放行, got %v", err)
	}
}

func TestCapallocSeqGate(t *testing.T) {
	err := run(context.Background(), base(), "risk.max_exposure_usd", 500, "e")
	if err == nil || !strings.Contains(err.Error(), "序闸") {
		t.Fatalf("exp 500 < notional 1000 应序闸拒, got %v", err)
	}
}

func TestCapallocFailClosed(t *testing.T) {
	s := base()
	s.eqErr = errors.New("no rows")
	if err := run(context.Background(), s, "risk.max_exposure_usd", 3000, "e"); err == nil {
		t.Fatal("净值读件缺应 fail-closed 拒")
	}
}

func TestCapallocAuditTrail(t *testing.T) {
	s := base()
	if err := run(context.Background(), s, "risk.max_exposure_usd", 3000, "利用率90%"); err != nil {
		t.Fatal(err)
	}
	if len(s.execs) != 2 || !strings.Contains(s.execs[0], "UPDATE app_config") ||
		!strings.Contains(s.execs[1], "INSERT INTO ops_event") {
		t.Fatalf("写键+落账应序发, got %v", s.execs)
	}
}
