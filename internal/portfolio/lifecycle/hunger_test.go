package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/opschain"
	"o/internal/platform/testutil"
	"o/internal/portfolio/hunger"
	"o/internal/store/pgstore"
)

// T-0378 接线桩测三件——锚①Detect 每拍被调（独立节拍+非触发零响应）；
// 锚②触发构造→Respond 四步跑通→饥饿/饥饿响应/参数调整行落账；锚③败径
// 隔离（detect/respond 假败→主拍不炸+升件自报行落+同窗 24h 去重）。

// stubSensor 感知缝桩（计数 Detect/Respond——接线面机械实证）。
type stubSensor struct {
	rep      *hunger.Report
	derr     error
	rerr     error
	perr     error
	merr     error
	mineN    int
	depErr   error
	detects  int
	responds int
	probes   int
	mines    int
	deps     int
}

func (s *stubSensor) Detect(context.Context) (*hunger.Report, error) {
	s.detects++
	if s.derr != nil {
		return nil, s.derr
	}
	return s.rep, nil
}

func (s *stubSensor) Respond(context.Context, *hunger.Report) ([]hunger.StepResult, error) {
	s.responds++
	if s.rerr != nil {
		return nil, s.rerr
	}
	return nil, nil
}

func (s *stubSensor) ProbeFamilies(context.Context) error {
	s.probes++
	return s.perr
}

func (s *stubSensor) ProbeDeps(context.Context) error {
	s.deps++
	return s.depErr
}

func (s *stubSensor) MineLessons(context.Context) (int, error) {
	s.mines++
	return s.mineN, s.merr
}

// stubVerifier 验链缝桩（T-0386——Verify/ReportBreak 计数接线实证）。
type stubVerifier struct {
	rep      *opschain.Report
	verr     error
	verifies int
	reports  int
}

func (s *stubVerifier) Verify(context.Context) (*opschain.Report, error) {
	s.verifies++
	if s.verr != nil {
		return nil, s.verr
	}
	return s.rep, nil
}

func (s *stubVerifier) ReportBreak(context.Context, *opschain.Report) error {
	s.reports++
	return nil
}

// TestHungerTickChainVerifySameSlot 锚（T-0386）：验链拍挂饥饿同槽——
// 每拍全链重算先调；断链检出→ReportBreak 自报+Detect 照常（败情隔离）；
// 验链假败→升件自报行落（写败/LIFECYCLE-HUNGER）+ReportBreak 零误响。
func TestHungerTickChainVerifySameSlot(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	stub := &stubSensor{rep: &hunger.Report{Triggered: false, WindowDays: 3}}
	cv := &stubVerifier{rep: &opschain.Report{Findings: []opschain.Finding{
		{ID: 7, Kind: "伪戳", Detail: "桩断"}}}}
	r := &Runner{pool: pool, sensor: stub, chainV: cv}
	r.hungerTick(ctx)
	if cv.verifies != 1 || cv.reports != 1 {
		t.Fatalf("验链同槽拍不齐: verifies=%d reports=%d", cv.verifies, cv.reports)
	}
	if stub.detects != 1 {
		t.Fatalf("断链检出后 Detect 未照常: detects=%d", stub.detects)
	}
	cv2 := &stubVerifier{verr: errors.New("假败——验链函数缺位")}
	r2 := &Runner{pool: pool, sensor: stub, chainV: cv2}
	r2.hungerTick(ctx)
	if cv2.verifies != 1 || cv2.reports != 0 {
		t.Fatalf("验链败径错位: verifies=%d reports=%d", cv2.verifies, cv2.reports)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='LIFECYCLE-HUNGER' AND note LIKE 'hungertick opschain 败%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("验链败自报行缺: n=%d err=%v", n, err)
	}
}

// TestHungerLoopDetectEveryBeat 锚①：独立节拍逐拍调 Detect；非触发拍
// 至多零响应（窗口判据在 Detect——Respond 不被绕开）；ctx 退即净退。
func TestHungerLoopDetectEveryBeat(t *testing.T) {
	stub := &stubSensor{rep: &hunger.Report{Triggered: false, WindowDays: 3}}
	r := &Runner{sensor: stub} // 非触发径零池触——无池 Runner 合法
	old := hungerBeatEvery
	hungerBeatEvery = 5 * time.Millisecond // 测试面覆写（生产值钦定不动）
	defer func() { hungerBeatEvery = old }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { r.hungerLoop(ctx); close(done) }()
	deadline := time.After(2 * time.Second) // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
	for stub.detects < 3 {
		select {
		case <-deadline:
			t.Fatalf("Detect 每拍未连调: n=%d", stub.detects)
		case <-time.After(10 * time.Millisecond): // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second): // sched-exempt-ok 桩测期界等待（净退界非生产调度）
		t.Fatal("hungerLoop 未随 ctx 净退")
	}
	if stub.responds != 0 {
		t.Fatalf("非触发拍 respond 误响: %d", stub.responds)
	}
}

// TestHungerLoopFirstDetectOnStart 锚（T-0381）：Run 拉起即首侦——Detect
// 不等首拍 ticker 立调一次；ticker 未到拍前不再响（首侦≠改节拍）。
func TestHungerLoopFirstDetectOnStart(t *testing.T) {
	stub := &stubSensor{rep: &hunger.Report{Triggered: false, WindowDays: 3}}
	r := &Runner{sensor: stub}
	old := hungerBeatEvery
	hungerBeatEvery = time.Hour // 长拍隔——首侦若等 ticker 必超时
	defer func() { hungerBeatEvery = old }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { r.hungerLoop(ctx); close(done) }()
	deadline := time.After(2 * time.Second) // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
	for stub.detects < 1 {
		select {
		case <-deadline:
			t.Fatalf("拉起未即首侦: n=%d", stub.detects)
		case <-time.After(10 * time.Millisecond): // sched-exempt-ok 桩测期界等待（测试面拍锚非生产调度）
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second): // sched-exempt-ok 桩测期界等待（净退界非生产调度）
		t.Fatal("hungerLoop 未随 ctx 净退")
	}
	if stub.detects != 1 {
		t.Fatalf("ticker 未到拍误响: detects=%d", stub.detects)
	}
	if stub.responds != 0 {
		t.Fatalf("非触发首侦 respond 误响: %d", stub.responds)
	}
}

// hungerWireFixture 接线桩测库景（machine_test 同款——独立实例+fin 迁移）。
func hungerWireFixture(t *testing.T) (*Runner, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return NewRunner(pool, cfg), pool
}

// TestHungerTickTriggeredRunsRespondWithRows 锚②：净库零提案（窗键 fin_0116
// 种 3）→触发；休眠族带 proposed 证据（hunger 包自测同款桩景）→Respond 四步
// 跑通：饥饿行+四步行落账，复活径实迹（族翻位+参数调整行）。
func TestHungerTickTriggeredRunsRespondWithRows(t *testing.T) {
	r, pool := hungerWireFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,net_apr,legs,evidence)
		VALUES('okx:perp:BTC-USDT-SWAP','sleep_family','long_perp_short_spot','proposed',0.1,'[]','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,enabled,state)
		VALUES('sleep_family','okx',false,'retired')`); err != nil {
		t.Fatal(err)
	}
	r.hungerTick(ctx)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='饥饿' AND seat='决策席' AND ticket='T-0376'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("饥饿行缺: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='饥饿响应' AND ticket='T-0376' AND note LIKE 'step=%'`).Scan(&n); err != nil || n != 4 {
		t.Fatalf("四步响应行: n=%d err=%v", n, err)
	}
	var en bool
	var st string
	if err := pool.QueryRow(ctx,
		`SELECT enabled,state FROM fin_opp_template WHERE template='sleep_family'`,
	).Scan(&en, &st); err != nil || !en || st != "enabled" {
		t.Fatalf("休眠族未复活: en=%v st=%s err=%v", en, st, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='参数调整' AND ticket='T-0376' AND note LIKE 'target=fin_opp_template%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("参数调整行缺: n=%d err=%v", n, err)
	}
}

// TestHungerTickDetectFailIsolatedSelfReported 锚③：Detect 假败→主拍不炸
// （tick 正常返）+升件自报行落（写败/LIFECYCLE-HUNGER）+respond 零误响+
// 同窗 24h 去重（T-0369 自报率限）。
func TestHungerTickDetectFailIsolatedSelfReported(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	stub := &stubSensor{derr: errors.New("假败——窗键不可读")}
	r := &Runner{pool: pool, sensor: stub}
	r.hungerTick(ctx)
	if stub.responds != 0 {
		t.Fatalf("detect 败后 respond 误响: %d", stub.responds)
	}
	const esc = `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='LIFECYCLE-HUNGER' AND note LIKE 'hungertick detect 败%'`
	var n int
	if err := pool.QueryRow(ctx, esc).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自报行缺: n=%d err=%v", n, err)
	}
	r.hungerTick(ctx) // 同窗再败
	if err := pool.QueryRow(ctx, esc).Scan(&n); err != nil || n != 1 {
		t.Fatalf("同窗去重失效: n=%d err=%v", n, err)
	}
}

// TestHungerTickProbeEveryBeat 锚（T-0384）：族沉默探针与 Detect 同槽——
// 每拍先探（饥饿与否无关）；探针假败→升件自报行落+Detect 照常（败情隔离）。
func TestHungerTickProbeEveryBeat(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	stub := &stubSensor{rep: &hunger.Report{Triggered: false, WindowDays: 3},
		perr: errors.New("假败——kind 谱读败")}
	r := &Runner{pool: pool, sensor: stub}
	r.hungerTick(ctx)
	if stub.probes != 1 || stub.detects != 1 {
		t.Fatalf("探针/Detect 同槽拍不齐: probes=%d detects=%d", stub.probes, stub.detects)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='LIFECYCLE-HUNGER' AND note LIKE 'hungertick silence_probe 败%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("探针败情自报行缺: n=%d err=%v", n, err)
	}
}

// TestHungerTickRespondFailSelfReported 锚③补面：Respond 假败→环级自报行
// 落（步级败情由 hunger 包 logStep 自落——环级行补 logStep 自身败情盲区）。
func TestHungerTickRespondFailSelfReported(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	stub := &stubSensor{rep: &hunger.Report{Triggered: true, WindowDays: 3,
		Reasons: []string{"zero_proposal:桩景"}}, rerr: errors.New("假败——步序列中断")}
	r := &Runner{pool: pool, sensor: stub}
	r.hungerTick(ctx)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='LIFECYCLE-HUNGER' AND note LIKE 'hungertick respond 败%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("respond 自报行缺: n=%d err=%v", n, err)
	}
}
