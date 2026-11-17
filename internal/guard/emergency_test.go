package guard

// 末路只减仓出口测锚（工单#22247）：
//   ①对冲仓一腿外结+正径 sizer 败域（attempts 耗尽）→幸存腿 reduce-only
//     只减仓平掉，敞口单调不增（spec 腿量=venue 实读非腿账）；
//   ②venue 实读败=不发单 fail-closed（spec 零腿+仓仍开）；
//   ③受令「止血/仓#N」走同径（attempts 未尽也触发）。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeEmgExec 末路执行缝假件——录 spec 可编程败。
type fakeEmgExec struct {
	specs []EmergencySpec
	fail  error
	execN int64
}

func (f *fakeEmgExec) ExecuteEmergency(_ context.Context, spec EmergencySpec) (int64, error) {
	f.specs = append(f.specs, spec)
	if f.execN == 0 {
		f.execN = 555
	}
	return f.execN, f.fail
}

// emgSizer 可编程 venue 现读假件（per-inst 残量+符号位+lot+errOn）。
type emgSizer struct {
	spot  map[string]float64
	ctr   map[string]float64
	dir   map[string]float64
	lot   float64
	errOn string
}

func (s emgSizer) SpotAvailBal(_ context.Context, instID string) (float64, error) {
	if s.errOn != "" && strings.HasSuffix(instID, s.errOn) {
		return 0, errors.New("venue read down")
	}
	return s.spot[instID], nil
}

func (s emgSizer) ContractPos(_ context.Context, instID string) (float64, error) {
	if s.errOn != "" && strings.HasSuffix(instID, s.errOn) {
		return 0, errors.New("venue read down")
	}
	return s.ctr[instID], nil
}

func (s emgSizer) LotSz(context.Context, string) (float64, error) { return s.lot, nil }

func (s emgSizer) ContractPosDir(_ context.Context, instID string) (float64, error) {
	return s.dir[instID], nil
}

// mkExhausted 仓 fixture+attempts 打满（maxAtt=3 测面键）——返 posID。
func mkExhausted(t *testing.T, ctx context.Context, w *Warden) int64 {
	t.Helper()
	pid := mkPosition(t, ctx, w, "live", 10, time.Now().Add(-96*time.Hour))
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,attempts,last_try_at)
		VALUES($1,'close_failed','delta_drift',3,now())`, pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

// 锚①：对冲仓 spot 腿外结（avail=0）+SWAP 幸存裸空——attempts 耗尽后
// auto 末路：幸存腿按 venue 实读定量 reduce-only 平掉，仓闭合。
func TestEmergencyAutoNakedSurvivor(t *testing.T) {
	ctx, w, _ := setup(t)
	emg := &fakeEmgExec{}
	w.AttachEmergencyExecutor(emg)
	w.AttachSizer(emgSizer{spot: map[string]float64{"BTC-USDT": 0},
		ctr: map[string]float64{"BTC-USDT-SWAP": 10}, dir: map[string]float64{"BTC-USDT-SWAP": -10}, lot: 0.01})
	pid := mkExhausted(t, ctx, w)

	w.emergencyScan(ctx, 3)

	if len(emg.specs) != 1 {
		t.Fatalf("末路 spec 数=%d want 1", len(emg.specs))
	}
	sp := emg.specs[0]
	if sp.PositionID != pid || sp.Trigger != "auto" {
		t.Fatalf("spec 载位错: pos=%d trig=%s", sp.PositionID, sp.Trigger)
	}
	if len(sp.Legs) != 1 {
		t.Fatalf("末路腿数=%d want 1（spot 外结不构腿）", len(sp.Legs))
	}
	lg := sp.Legs[0]
	if lg.InstID != "BTC-USDT-SWAP" || lg.Side != "buy" || lg.Sz != 10 {
		t.Fatalf("幸存腿错: %+v（dir=-10 空→buy 平，量=venue 实读 10）", lg)
	}
	closed, state, trig := posClosed(t, ctx, w, pid)
	if !closed || state != "closed" || !strings.Contains(trig, "emergency_exit") {
		t.Fatalf("仓态错: closed=%v state=%s trig=%s", closed, state, trig)
	}
}

// 锚②：venue 实读败=不发单 fail-closed——spec 零调用+仓仍开（升件面
// 走 Alert 通道，测试面断言「无单+不闭」即契约）。
func TestEmergencyReadFailClosed(t *testing.T) {
	ctx, w, _ := setup(t)
	emg := &fakeEmgExec{}
	w.AttachEmergencyExecutor(emg)
	w.AttachSizer(emgSizer{spot: map[string]float64{}, ctr: map[string]float64{},
		lot: 0.01, errOn: "-USDT"}) // 全腿读败
	pid := mkExhausted(t, ctx, w)

	w.emergencyScan(ctx, 3)

	if len(emg.specs) != 0 {
		t.Fatalf("读败发单=%d want 0（fail-closed）", len(emg.specs))
	}
	closed, _, _ := posClosed(t, ctx, w, pid)
	if closed {
		t.Fatal("读败仓被闭——fail-closed 破")
	}
}

// 锚③：受令「止血/仓#N」同径——attempts 未尽（正径仍在重试域）也触发，
// 双腿俱在=对冲态受令不受 auto 闸。
func TestEmergencyOrderedTrigger(t *testing.T) {
	ctx, w, _ := setup(t)
	emg := &fakeEmgExec{}
	w.AttachEmergencyExecutor(emg)
	w.AttachSizer(emgSizer{spot: map[string]float64{"BTC-USDT": 1},
		ctr: map[string]float64{"BTC-USDT-SWAP": 10}, dir: map[string]float64{"BTC-USDT-SWAP": -10}, lot: 0.01})
	pid := mkPosition(t, ctx, w, "live", 10, time.Now().Add(-96*time.Hour))
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_guard_close(position_id,state,trigger_kind,attempts)
		VALUES($1,'close_failed','delta_drift',1)`, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('自我','处置','',$1)`,
		fmt.Sprintf("止血/仓#%d 正径失效人工令——末路只减仓", pid)); err != nil {
		t.Fatal(err)
	}

	w.emergencyScan(ctx, 3)

	if len(emg.specs) != 1 {
		t.Fatalf("受令 spec 数=%d want 1", len(emg.specs))
	}
	sp := emg.specs[0]
	if sp.Trigger != "order" || len(sp.Legs) != 2 {
		t.Fatalf("受令 spec 错: trig=%s legs=%d（对冲态受令双腿全平）", sp.Trigger, len(sp.Legs))
	}
	for _, lg := range sp.Legs {
		if lg.Sz <= 0 {
			t.Fatalf("末路腿零量: %+v", lg)
		}
	}
	closed, state, _ := posClosed(t, ctx, w, pid)
	if !closed || state != "closed" {
		t.Fatalf("受令仓态错: closed=%v state=%s", closed, state)
	}
	// 幂等：同令行重拾不双发（emgSeen 去重+venue 残量≈0 语义）。
	w.emergencyScan(ctx, 3)
	if len(emg.specs) != 1 {
		t.Fatalf("重扫 spec 数=%d want 1（去重/已闭不重发）", len(emg.specs))
	}
}

// 对照锚：双腿俱在对冲态+attempts 耗尽——auto 不动（受令才可达）。
func TestEmergencyHedgedNoAuto(t *testing.T) {
	ctx, w, _ := setup(t)
	emg := &fakeEmgExec{}
	w.AttachEmergencyExecutor(emg)
	w.AttachSizer(emgSizer{spot: map[string]float64{"BTC-USDT": 1},
		ctr: map[string]float64{"BTC-USDT-SWAP": 10}, dir: map[string]float64{"BTC-USDT-SWAP": -10}, lot: 0.01})
	pid := mkExhausted(t, ctx, w)

	w.emergencyScan(ctx, 3)

	if len(emg.specs) != 0 {
		t.Fatalf("对冲态 auto 发单=%d want 0", len(emg.specs))
	}
	closed, _, _ := posClosed(t, ctx, w, pid)
	if closed {
		t.Fatal("对冲态仓被 auto 末路闭——闸破")
	}
}

// 全腿 venue 已清=簿记收殓（无单可发 markClosed exec=0——锚④产线面：
// attempts 耗尽仓不再滞留「平不掉」终态）。
func TestEmergencyAllSettledCloseout(t *testing.T) {
	ctx, w, _ := setup(t)
	emg := &fakeEmgExec{}
	w.AttachEmergencyExecutor(emg)
	w.AttachSizer(emgSizer{spot: map[string]float64{"BTC-USDT": 0},
		ctr: map[string]float64{"BTC-USDT-SWAP": 0}, lot: 0.01})
	pid := mkExhausted(t, ctx, w)

	w.emergencyScan(ctx, 3)

	if len(emg.specs) != 0 {
		t.Fatalf("全清发单=%d want 0（无单可发）", len(emg.specs))
	}
	closed, state, trig := posClosed(t, ctx, w, pid)
	if !closed || state != "closed" || !strings.Contains(trig, "emergency_exit") {
		t.Fatalf("收殓态错: closed=%v state=%s trig=%s", closed, state, trig)
	}
}
