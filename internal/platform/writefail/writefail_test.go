package writefail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakePool struct {
	calls int
	sql   string
	args  []any
	err   error
}

func (f *fakePool) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.calls++
	f.sql = sql
	f.args = arguments
	return pgconn.NewCommandTag("INSERT 0 1"), f.err
}

func TestReportNilPoolSilent(t *testing.T) {
	SetPool(nil)
	Report(context.Background(), "s", "t", "SCOPE", "boom") // 不产调用不炸
}

func TestReportWritesEscalation(t *testing.T) {
	fp := &fakePool{}
	SetPool(fp)
	defer SetPool(nil)
	Report(context.Background(), "research/dirmom", "工单#22873", "RESEARCH-TICK/dirmom", "拍败: pg 42P10")
	if fp.calls != 1 {
		t.Fatalf("calls=%d want 1", fp.calls)
	}
	for _, want := range []string{"'升件'", "'写败'", "NOT EXISTS", "interval '24 hours'"} {
		if !strings.Contains(fp.sql, want) {
			t.Fatalf("sql 缺 %q: %s", want, fp.sql)
		}
	}
	if fp.args[0] != "research/dirmom" || fp.args[3] != "RESEARCH-TICK/dirmom" {
		t.Fatalf("args=%v", fp.args)
	}
	if !strings.Contains(fp.args[2].(string), "RESEARCH-TICK/dirmom 写败: 拍败: pg 42P10") {
		t.Fatalf("note=%v", fp.args[2])
	}
}

func TestReportSelfWriteFailSilent(t *testing.T) {
	fp := &fakePool{err: errors.New("denied")}
	SetPool(fp)
	defer SetPool(nil)
	Report(context.Background(), "s", "t", "SCOPE", "boom") // 自报写败=静默退不返错不炸
	if fp.calls != 1 {
		t.Fatalf("calls=%d want 1", fp.calls)
	}
}

func TestReportDedupSameScopeWindow(t *testing.T) {
	// 同窗去重=SQL 层 NOT EXISTS（24h）——两次 Report 各发一语句，
	// 去重判据在库侧；此处只证语句携 scope 判例键供 WHERE 命中。
	fp := &fakePool{}
	SetPool(fp)
	defer SetPool(nil)
	Report(context.Background(), "s", "t", "SCOPE-A", "e1")
	Report(context.Background(), "s", "t", "SCOPE-B", "e2")
	if fp.calls != 2 {
		t.Fatalf("calls=%d want 2（异 scope 不互斥——去重键=esc_scope）", fp.calls)
	}
}
