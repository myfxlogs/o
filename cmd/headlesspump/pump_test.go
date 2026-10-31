// pump_test.go —— 票级态机单测（fake writer 缝——不连库验锚行→五态派生链）。
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// ── 秒级化钉案（2026-10-31）：tick 三门控/退避/防伪醒对抗锚 ──

// failW 首败后愈的写缝（退避锚——败→degraded 退 30s 拍，愈→自愈回 5s）。
type failW struct {
	failLeft int
	ok       fakeW
}

func (f *failW) Exec(ctx context.Context, s string, a ...any) (pgconn.CommandTag, error) {
	if f.failLeft > 0 {
		f.failLeft--
		return pgconn.CommandTag{}, context.DeadlineExceeded
	}
	return f.ok.Exec(ctx, s, a)
}

// spawnAnchorProc 真 /proc 锚进程（cmdline 嵌票锚串——findProc 实证可及；
// 测试毕杀净）。kill 翻转/心跳门控/退避自愈三锚共用。
func spawnAnchorProc(t *testing.T, tid string) {
	t.Helper()
	cmd := exec.Command("bash", "-c", fmt.Sprintf("sleep 30; : headless-%s-prompt", tid)) // 复合命令防 bash exec 优化（单命令注释被吃→cmdline 无锚）
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
}

// TestTickKillFlipProcSeen anchor b 钉案：曾实证活的在飞票（真 /proc 锚），
// 进程杀→tick 翻死 emit（kill 突殁无 exit 行 ≤10s 上板）；复拍零新写（变化门控）。
func TestTickKillFlipProcSeen(t *testing.T) {
	spawnAnchorProc(t, "T-KILL1")
	w := &fakeW{}
	p := newPump(w, t.TempDir())
	tk := p.ensure("T-KILL1", "Dev", filepath.Join(p.dir, "devin-headless-T-KILL1.log"))
	tk.procAlive, tk.procSeen, tk.state = true, true, "alive"
	p.tick() // 锚进程在——不翻死
	if tk.derive() != "alive" {
		t.Fatalf("锚进程在飞不应翻死 got %s", tk.derive())
	}
	// 杀锚进程（模拟 kill -9——无 exit 行）→ 拍实证殁→翻死
	if err := exec.Command("pkill", "-f", "headless-T-KILL1-prompt").Run(); err != nil {
		t.Fatalf("杀锚进程: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for tk.derive() == "alive" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok 桩测期界等待（kill 翻转轮询非业务环）
		p.tick()
	}
	if tk.derive() != "dead" {
		t.Fatal("kill 后 tick 应翻死（anchor b）")
	}
	n1 := w.n
	if n1 == 0 {
		t.Fatal("翻死应 emit")
	}
	p.tick() // 复拍门控
	if w.n != n1 {
		t.Fatalf("复拍应零写 got %d→%d", n1, w.n)
	}
}

// TestTickNeverSeenNoFlip spawn 起跑窗防伪死：未实证活（procSeen=false）的在飞
// 票，/proc 无锚不翻死（进程尚在起——静默判 30min 兜底）；心跳照写（在飞行
// updated_at 保鲜面）。
func TestTickNeverSeenNoFlip(t *testing.T) {
	w := &fakeW{}
	p := newPump(w, t.TempDir())
	tk := p.ensure("T-GAP1", "Dev", filepath.Join(p.dir, "devin-headless-T-GAP1.log"))
	tk.procAlive, tk.procSeen, tk.state = true, false, "alive"
	p.tick()
	if tk.derive() != "alive" {
		t.Fatalf("未实证票不应翻死 got %s", tk.derive())
	}
	if w.n != 1 {
		t.Fatalf("应恰心跳 1 写 got %d", w.n)
	}
}

// TestTickHeartbeatGate anchor c 钉案：30s 心跳拍——在飞行一拍一写，窗内复拍
// 零写（非全量刷；终态行零写）。
func TestTickHeartbeatGate(t *testing.T) {
	spawnAnchorProc(t, "T-HB1")
	w := &fakeW{}
	p := newPump(w, t.TempDir())
	a := p.ensure("T-HB1", "Dev", filepath.Join(p.dir, "devin-headless-T-HB1.log"))
	a.procAlive, a.procSeen, a.state = true, true, "alive"
	d := p.ensure("T-HB0", "Dev", filepath.Join(p.dir, "devin-headless-T-HB0.log"))
	d.procAlive, d.state = false, "dead" // 终态行——零写面
	p.tick()                             // 首拍：心跳开窗——在飞 1 写，终态 0 写
	if w.n != 1 {
		t.Fatalf("首拍应心跳 1 写（在飞行）got %d", w.n)
	}
	p.tick() // 窗内复拍：零写
	if w.n != 1 {
		t.Fatalf("窗内复拍应零写 got %d", w.n)
	}
	p.lastHb = time.Now().Add(-31 * time.Second) // 窗过
	p.tick()
	if w.n != 2 {
		t.Fatalf("窗过复拍应心跳 1 写 got %d", w.n)
	}
}

// TestTickBackoff 钉案④：写败入退避（30s 内整拍跳过零写），退避窗过+写愈→
// 自愈回常态（在飞锚进程持续供心跳重试面）。
func TestTickBackoff(t *testing.T) {
	spawnAnchorProc(t, "T-BO1")
	fw := &failW{failLeft: 1}
	p := newPump(fw, t.TempDir())
	a := p.ensure("T-BO1", "Dev", filepath.Join(p.dir, "devin-headless-T-BO1.log"))
	a.procAlive, a.procSeen, a.state = true, true, "alive"
	p.tick() // 心跳写→败→degraded
	if !p.degraded {
		t.Fatal("写败应入退避")
	}
	n := fw.ok.n
	p.tick() // 退避期内：整拍跳过
	if fw.ok.n != n {
		t.Fatalf("退避期内应零写 got %d→%d", n, fw.ok.n)
	}
	p.degradedAt = time.Now().Add(-31 * time.Second) // 退避窗过
	p.tick()                                         // 心跳重试→写愈→自愈
	if p.degraded {
		t.Fatal("写愈应自愈回常态")
	}
	if fw.ok.n == n {
		t.Fatal("退避过应重试心跳写")
	}
}

// TestTickDualFileNoThrash 双文件票防抖锚（实测回归：run/audit 同 id 时 offset
// 交叉污染→每拍全量重读→done 票每 5s 重写）。tick 只服现役件：静态双文件+
// done 态→多拍零写。
func TestTickDualFileNoThrash(t *testing.T) {
	dir := t.TempDir()
	runLog := filepath.Join(dir, "devin-headless-T-0601.log")
	auditLog := filepath.Join(dir, "devin-audit-T-0601.log")
	os.WriteFile(runLog, []byte("无头 T-0601 spawn\n无头 T-0601 进程退出 rc=0\nT-0601 收口待审命中\n"), 0o644)
	os.WriteFile(auditLog, []byte("无头审计 T-0601 spawn\n无头审计 T-0601 退出 rc=0\n"), 0o644)
	w := &fakeW{}
	p := newPump(w, dir)
	p.ensure("T-0601", "Dev", runLog)
	// 模拟事件径：先读 run 件（offset=run 尾，done 锚落定）
	tk := p.tks["T-0601"]
	p.onLog(tk)
	if tk.state != "done" {
		t.Fatalf("done 锚应落定 got %s", tk.state)
	}
	n0 := w.n
	time.Sleep(1100 * time.Millisecond) // sched-exempt-ok 桩测期界等待（emitSoon 1s 尾帧窗过非业务环）
	for i := 0; i < 3; i++ {
		p.tick() // 三拍：现役件无新字节早退，audit 件跳过——零重发零心跳（终态行）
	}
	if w.n != n0 {
		t.Fatalf("静态双文件票 tick 应零写 got %d→%d", n0, w.n)
	}
}
