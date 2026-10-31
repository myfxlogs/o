package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 步链重建单测（纯函数——事件流水→步链/R 环/结果派生，设计稿交互节逐例）。
func TestBuildSteps(t *testing.T) {
	base := time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)
	ev := func(min int, seat, event, note string) pipelineEventRow {
		return pipelineEventRow{seat: seat, event: event, note: note, ts: base.Add(time.Duration(min) * time.Minute)}
	}
	// ① 设计稿原型行：CEO发起→CTO派→施工收→审计驳回→施工R2在干（驳回环）。
	steps := buildSteps([]pipelineEventRow{
		ev(0, "席MANAGE", "领单", ""),
		ev(1, "决策席", "领单", "派单"),
		ev(5, "Dev", "领单", ""),
		ev(32, "Dev", "收口", ""),
		ev(33, "DM", "裁决", "驳回:锚①未跑"),
		ev(34, "Dev", "领单", "返修"),
	})
	if len(steps) != 5 {
		t.Fatalf("步数错: %d (%+v)", len(steps), steps)
	}
	want := []struct{ seat, result string }{
		{"CEO", "closed"}, {"CTO", "closed"}, {"施工", "closed"}, {"审计", "rejected"}, {"施工R2", "working"},
	}
	for i, w := range want {
		if steps[i].disp != w.seat || steps[i].result != w.result {
			t.Fatalf("步%d错: got(%s,%s) want(%s,%s)", i, steps[i].disp, steps[i].result, w.seat, w.result)
		}
	}
	// 在干末步：无出口；收口步有出口且 ms 对拍。
	if steps[4].hasExit {
		t.Fatal("末步在干不该有出口")
	}
	if !steps[2].hasExit || steps[2].exit.Sub(steps[2].enter) != 27*time.Minute {
		t.Fatalf("收口步时长错: %+v", steps[2])
	}
	// ② 隐式移交（无收口自报径——切席即收步 closed）+审计进度自报非终态。
	steps = buildSteps([]pipelineEventRow{
		ev(0, "Dev", "领单", ""),
		ev(10, "DM", "裁决", "审计开工"),
	})
	if len(steps) != 2 || steps[0].result != "closed" || steps[1].result != "working" || steps[1].disp != "审计" {
		t.Fatalf("隐式移交径错: %+v", steps)
	}
	// ③ 挂起/异常终态。
	steps = buildSteps([]pipelineEventRow{ev(0, "Dev", "领单", ""), ev(3, "Dev", "挂起", "等锚")})
	if len(steps) != 1 || steps[0].result != "blocked" {
		t.Fatalf("挂起径错: %+v", steps)
	}
	steps = buildSteps([]pipelineEventRow{ev(0, "Dev", "领单", ""), ev(3, "Dev", "异常", "熔断")})
	if len(steps) != 1 || steps[0].result != "error" {
		t.Fatalf("异常径错: %+v", steps)
	}
	// ④ R3 三环（同席三次开闭——首现不标，二现 R2 三现 R3；开闭开闭开=3 步）。
	steps = buildSteps([]pipelineEventRow{
		ev(0, "Dev", "领单", ""), ev(1, "Dev", "挂起", ""),
		ev(2, "Dev", "领单", ""), ev(3, "Dev", "挂起", ""),
		ev(4, "Dev", "领单", ""),
	})
	if len(steps) != 3 || steps[1].disp != "施工R2" || steps[2].disp != "施工R3" || steps[0].disp != "施工" {
		t.Fatalf("R3 环错: %d %+v %+v %+v", len(steps), steps[0], steps[1], steps[2])
	}
	// ⑤ 灯语对拍：passed/closed 绿呼吸、rejected/blocked 黄呼吸、error 红静止。
	for _, c := range []struct {
		result  string
		state   int32
		breathe bool
	}{{"passed", 0, true}, {"closed", 0, true}, {"rejected", 1, true}, {"blocked", 1, true}, {"error", 2, false}} {
		st, br := lampState(c.result)
		if st != c.state || br != c.breathe {
			t.Fatalf("灯语错 %s: (%d,%v)", c.result, st, br)
		}
	}
	// ⑥ 进程五态灯：alive 绿/stalled 黄/dead·overdue 红。
	for _, c := range []struct {
		s  string
		st int32
	}{{"alive", 0}, {"stalled", 1}, {"dead", 2}, {"overdue", 2}} {
		if st, _ := headlessLamp(c.s); st != c.st {
			t.Fatalf("进程灯错 %s: %d", c.s, st)
		}
	}
}

// 端到端（测试实例）：活跃/完结退场/排序/截断/水印/无头联查灯色全链对拍。
func TestGetPipelineEndToEnd(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool, docsRoot: "docs/tasks"}
	// 桩（now() 相对分——测试实例时钟无关）：T-9001 驳回环在飞（带 ops_headless
	// alive 行）、T-9002 收口等审计、T-9000 已完结（末事件<部署 ts→退场）。
	ins := func(agoMin int, seat, event, ticket, note string) {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(ts,seat,event,ticket,note)
			VALUES(now()-$1*interval '1 minute',$2,$3,$4,$5)`, agoMin, seat, event, ticket, note); err != nil {
			t.Fatal("事件桩:", err)
		}
	}
	ins(90, "席MANAGE", "领单", "T-9001", "")
	ins(89, "决策席", "领单", "T-9001", "派单")
	ins(85, "Dev", "领单", "T-9001", "")
	ins(60, "Dev", "收口", "T-9001", "")
	ins(59, "DM", "裁决", "T-9001", "驳回:桩")
	ins(58, "Dev", "领单", "T-9001", "返修")
	ins(30, "Dev", "领单", "T-9002", "")
	ins(20, "Dev", "收口", "T-9002", "")
	ins(150, "Dev", "领单", "T-9000", "")
	ins(130, "DM", "裁决", "T-9000", "通过")
	ins(120, "DEPLOY", "部署", "", "deploy 桩")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_headless(ticket,seat,model,state,last_activity)
		VALUES('T-9001','Dev','桩模','alive',now())`); err != nil {
		t.Fatal("无头桩:", err)
	}
	r, err := c.GetPipeline(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	// ① 完结退场：T-9000（部署前末事件）不在 rows；活跃=2。
	if r.TotalActive != 2 || len(r.Rows) != 2 {
		t.Fatalf("活跃对拍错: total=%d rows=%d", r.TotalActive, len(r.Rows))
	}
	for _, row := range r.Rows {
		if row.Ticket == "T-9000" {
			t.Fatal("已部署票未退场")
		}
	}
	// ② 最新票排最前：T-9002（15:10）> T-9001（14:32）。
	if r.Rows[0].Ticket != "T-9002" || r.Rows[1].Ticket != "T-9001" {
		t.Fatalf("序错: %s, %s", r.Rows[0].Ticket, r.Rows[1].Ticket)
	}
	// ③ 驳回环行：T-9001 步链=CEO closed→CTO closed→施工closed→审计rejected
	// →施工R2 working(spinner)+行级灯=绿呼吸（ops_headless alive 活证）。
	row := r.Rows[1]
	if len(row.Steps) != 5 {
		t.Fatalf("T-9001 步数错: %d", len(row.Steps))
	}
	s0, s1, s2 := row.Steps[2], row.Steps[3], row.Steps[4]
	if s0.Seat != "施工" || s0.Result != "closed" || s0.Exit == "" ||
		s1.Seat != "审计" || s1.Result != "rejected" || s1.State != 1 ||
		s2.Seat != "施工R2" || s2.Result != "working" || !s2.Active || !s2.Spin {
		t.Fatalf("驳回环步链错: %+v | %+v | %+v", s0, s1, s2)
	}
	if row.CurrentSeat != "施工R2" || row.CurrentState != "working" || row.State != 0 || !row.Breathe {
		t.Fatalf("行级灯/当前席错: %+v", row)
	}
	// ④ 等审计行：T-9002 末步 closed→行灯黄呼吸（等审计=黄 设计稿灯语）。
	row2 := r.Rows[0]
	if row2.CurrentState != "closed" || row2.State != 1 || !row2.Breathe {
		t.Fatalf("等审计行灯错: %+v", row2)
	}
	// ⑤ 水印=max(ops_event.id)>0 且随部署行推进。
	if r.Seq <= 0 {
		t.Fatalf("水印错: %d", r.Seq)
	}
	var maxID int64
	if err := pool.QueryRow(ctx, `SELECT max(id) FROM ops_event`).Scan(&maxID); err != nil || r.Seq != maxID {
		t.Fatalf("水印对拍错: seq=%d max=%d err=%v", r.Seq, maxID, err)
	}
}

// 截断锚 f：活跃>20 行——20 行封顶+total_active 全量（前端 +N more 径）。
func TestGetPipelineTruncation(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}
	for i := 0; i < 22; i++ {
		if _, err := pool.Exec(ctx,
			`INSERT INTO ops_event(seat,event,ticket,note,ts) VALUES('Dev','领单',$1,'桩',now()-$2*interval '1 minute')`,
			fmt.Sprintf("T-%04d", 9100+i), 30-i); err != nil {
			t.Fatal("桩:", err)
		}
	}
	r, err := c.GetPipeline(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rows) != 20 || r.TotalActive != 22 {
		t.Fatalf("截断对拍错: rows=%d total=%d", len(r.Rows), r.TotalActive)
	}
	// 最前=最新（T-9121，30 分钟前）。
	if r.Rows[0].Ticket != "T-9121" {
		t.Fatalf("顶行错: %s", r.Rows[0].Ticket)
	}
}

// GetCockpit 并入管道（一往返）+拔径显缺（ops_event 缺→missing "pipeline"，
// 其余块不受染）。拔径桩钉本 schema 拆表+search_path 去 public 兜底——共享
// 测试实例公共 schema 有残表（2026-10-31 实证 o_test public.ops_event 10-29
// 残数据），不隔离则 unqualified 解析兜到残表=假活。
func TestGetCockpitPipelineMerged(t *testing.T) {
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := coreService{pool: pool}
	r, err := c.GetCockpit(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Pipeline == nil {
		t.Fatal("管道未并入首帧")
	}
	// 本 schema 拆表（qualified——不碰公共 schema）
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DROP TABLE "+schema+".ops_event CASCADE"); err != nil {
		t.Fatal("拆表:", err)
	}
	// 新池：search_path 钉本 schema（无 public 兜底——查询必败显缺径）
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("O_TEST_PG_DSN 未设")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.AfterConnect = func(ctx context.Context, cc *pgx.Conn) error {
		_, err := cc.Exec(ctx, "SET search_path TO "+schema)
		return err
	}
	pool2, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool2.Close()
	c2 := coreService{pool: pool2}
	r2, err := c2.GetCockpit(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Pipeline != nil || !containsStr(r2.Missing, "pipeline") {
		t.Fatalf("拔径未显缺: pipeline=%v missing=%v", r2.Pipeline, r2.Missing)
	}
	if r2.Money == nil {
		t.Fatal("拔径污染其余块")
	}
}
