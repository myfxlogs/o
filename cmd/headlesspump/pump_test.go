// pump_test.go —— 票级态机单测（fake writer 缝——不连库验锚行→五态派生链）。
package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeW struct {
	n    int
	args []any
}

func (f *fakeW) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	f.n++
	f.args = args
	return pgconn.CommandTag{}, nil
}

func TestDerive(t *testing.T) {
	cases := []struct {
		name string
		tk   tk
		want string
	}{
		{"done 压一切", tk{doneF: true, procAlive: true, stalled: true, overdue: true}, "done"},
		{"死压停滞", tk{procAlive: false, stalled: true}, "dead"},
		{"停滞压逾期", tk{procAlive: true, stalled: true, overdue: true}, "stalled"},
		{"逾期", tk{procAlive: true, overdue: true}, "overdue"},
		{"在飞", tk{procAlive: true}, "alive"},
	}
	for _, c := range cases {
		if got := c.tk.derive(); got != c.want {
			t.Errorf("%s: derive=%q want %q", c.name, got, c.want)
		}
	}
}

// append 续写日志文件并驱动 onLog（与 fsnotify 写帧同径）。
func append(t *testing.T, p *pump, tk *tk, line string) {
	t.Helper()
	f, err := os.OpenFile(tk.logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line); err != nil {
		t.Fatal(err)
	}
	f.Close()
	p.onLog(tk)
}

func TestMarkerChain(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-headless-T-9999.log")
	if err := os.WriteFile(lp, []byte("[10-30 11:00:00] 无头 T-9999 spawn（devin -p）\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fw := &fakeW{}
	p := newPump(fw, dir)
	tk := p.ensure("T-9999", "Dev", lp)

	p.onLog(tk)
	if tk.state != "alive" {
		t.Fatalf("spawn 后 state=%q want alive", tk.state)
	}
	append(t, p, tk, "some devin output line\n")
	if tk.state != "alive" {
		t.Fatalf("普通写帧后 state=%q want alive", tk.state)
	}
	append(t, p, tk, "[10-30 12:00:00] T-9999 收口待审命中（/x）——.done 已落，环将转审计\n")
	if tk.state != "done" {
		t.Fatalf("收口命中后 state=%q want done", tk.state)
	}
	if fw.n == 0 {
		t.Fatal("emit 零次——upsert 未发")
	}
	if fw.args[3] != "done" {
		t.Fatalf("末次 upsert state=%v want done", fw.args[3])
	}
}

func TestNoDoneExitDead(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-headless-T-8888.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头 T-8888 spawn（devin -p）\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("T-8888", "Dev", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 无头 T-8888 进程退出 rc=1\n")
	append(t, p, tk, "[10-30 12:00:01] ⚠ T-8888 退出但全树无收口待审标记——需人工跟进\n")
	if tk.state != "dead" {
		t.Fatalf("退出无收口后 state=%q want dead", tk.state)
	}
}

func TestAuditSeatAndExit(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-audit-T-7777.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头审计 T-7777 spawn\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("T-7777", "DM", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 无头审计 T-7777 退出 rc=0\n")
	if tk.state != "done" {
		t.Fatalf("审计退出后 state=%q want done（审计 pass 完工）", tk.state)
	}
}

func TestReportLine(t *testing.T) {
	for ln, want := range map[string]string{
		"[10-30 11:41:15] [自报][Dev] 领单 T-0263 —— 开工": "T-0263",
		"[10-30 12:00:00] [自报][无头施工席] 收口 T-0257":     "T-0257",
		"[10-30 12:00:00] [总线] MANAGE→Dev directive": "",
	} {
		m := reReport.FindStringSubmatch(ln)
		got := ""
		if m != nil {
			got = m[1]
		}
		if got != want {
			t.Errorf("reReport %q → %q want %q", ln, got, want)
		}
	}
}

// 静默判链：进程实证路径桩化——findProc 真身走 /proc（本机无头票 pid 即实锤，
// 单测不桩外部态）。
func TestEmitThrottle(t *testing.T) {
	dir := t.TempDir()
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("T-6666", "Dev", "")
	tk.lastEmit = time.Now() // 刚发——续帧须走尾帧闸不直发
	p.emitSoon(tk)
	if !tk.emitDirty {
		t.Fatal("节流窗内续帧未挂尾帧")
	}
}

// T-0277 六态四径：排查审计旧方言锚→done、自由票号审计锚→done、
// run rc=0 无收口→exit（正常终态非死）、rc≠0 见证→dead（真异常不稀释）。

func TestAuditDialectExit(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-audit-T-0001.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头审计 T-0001 spawn\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("T-0001", "DM", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 排查审计 T-0001 退出 rc=0\n")
	if tk.state != "done" {
		t.Fatalf("排查审计旧方言锚应 done: %q", tk.state)
	}
}

func TestFreeTicketIDAudit(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-audit-WAKEFIX.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头审计 WAKEFIX spawn\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("WAKEFIX", "DM", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 无头审计 WAKEFIX 退出 rc=0\n")
	if tk.state != "done" {
		t.Fatalf("自由票号审计锚应 done: %q", tk.state)
	}
}

func TestRunExitRc0NoDone(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-headless-BASESWP.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头 BASESWP spawn（devin -p）\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("BASESWP", "Dev", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 无头 BASESWP 进程退出 rc=0\n")
	append(t, p, tk, "[10-30 12:00:01] ⚠ BASESWP 退出但全树无收口待审标记——需人工跟进\n")
	if tk.state != "exit" {
		t.Fatalf("rc=0 见证退出无收口应 exit 非 dead: %q", tk.state)
	}
}

func TestRunExitRc1Dead(t *testing.T) {
	dir := t.TempDir()
	lp := filepath.Join(dir, "devin-headless-T-5555.log")
	os.WriteFile(lp, []byte("[10-30 11:00:00] 无头 T-5555 spawn（devin -p）\n"), 0o644)
	p := newPump(&fakeW{}, dir)
	tk := p.ensure("T-5555", "Dev", lp)
	p.onLog(tk)
	append(t, p, tk, "[10-30 12:00:00] 无头 T-5555 进程退出 rc=1\n")
	if tk.state != "dead" {
		t.Fatalf("rc≠0 见证退出应 dead: %q", tk.state)
	}
}
