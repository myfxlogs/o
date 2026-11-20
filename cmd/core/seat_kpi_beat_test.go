// seat_kpi_beat_test.go —— 各岗 KPI 量表探针（工单#19121）：桩
// ops_event/fin_opp_hypothesis 行喂八口径，断言量表行落账+各段
// 计数/词形对（照 ownermodel_beat_test 先例形）。
package main

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func seatKpiSetup(t *testing.T) (context.Context, coreService) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return ctx, coreService{pool: pool}
}

func oe(t *testing.T, ctx context.Context, svc coreService,
	seat, event, ticket, note string, ts time.Time, scope string) int64 {
	t.Helper()
	var id int64
	if err := svc.pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts,esc_scope)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,'')) RETURNING id`,
		seat, event, ticket, note, ts, scope).Scan(&id); err != nil {
		t.Fatalf("oe %s/%s: %v", event, note, err)
	}
	return id
}

// TestSeatKpiBeat 八口径桩件全谱——每段计数/词形钉死。
func TestSeatKpiBeat(t *testing.T) {
	ctx, svc := seatKpiSetup(t)
	now := time.Now()
	d := func(days, hours float64) time.Time {
		return now.Add(-time.Duration(days*24+hours) * time.Hour)
	}

	// —— ①② 升件+销处置对（升件先插保小 id——①钦定前缀正则碰撞
	// 面在桩件域收敛：引证号只取本件升件 id）。
	esc1 := oe(t, ctx, svc, "探测席", "升件", "", "桩升件A", d(10, 0), "kpa")
	esc2 := oe(t, ctx, svc, "探测席", "升件", "", "桩升件B", d(9, 0), "kpb")
	esc3 := oe(t, ctx, svc, "探测席", "升件", "", "桩升件C", d(8, 0), "") // 无 scope——②分母外
	oe(t, ctx, svc, "探测席", "升件", "", "桩升件复燃", d(10, -6), "kpa")   // esc1 销后 4h 同 scope 再升=复燃
	oe(t, ctx, svc, "处置席", "处置", "", "升件#"+strconv.FormatInt(esc1, 10)+" 销——桩", d(10, -2), "")
	oe(t, ctx, svc, "处置席", "处置", "", "升件#"+strconv.FormatInt(esc2, 10)+" 销——桩", d(9, -6), "")
	oe(t, ctx, svc, "处置席", "处置", "", "升件#"+strconv.FormatInt(esc3, 10)+" 销——误报", d(8, -1), "")
	// ①对集={2h,6h,1h}→中位 2h；②disposed={esc1,esc2}(esc3 无 scope 除名)
	// 复燃=esc1→50.0%；⑥disposed={1,2,3} 误报词仅 esc3→33.3%。

	// —— ③④ 施工完工对+打回。
	oe(t, ctx, svc, "施工席", "认领", "", "工单#91001 认领:开修——桩", d(5, 3), "")
	oe(t, ctx, svc, "施工席", "认领", "", "工单#91002 认领:开修——桩", d(4, 1), "")
	oe(t, ctx, svc, "施工席", "处置", "", "工单#91001 完工回执——桩", d(5, 0), "")
	oe(t, ctx, svc, "施工席", "处置", "", "工单#91002 完工回执——桩", d(4, 0), "")
	oe(t, ctx, svc, "处置席", "处置", "", "工单#91003 销——桩前件", d(2, 0), "")
	oe(t, ctx, svc, "处置席", "处置", "", "工单#91002 打回——桩", d(4, -1), "")
	// ③对集={3h,1h}→中位 2h；④done=2 打回=91002→50.0%。

	// —— ⑤ 审计疑点 P1。
	oe(t, ctx, svc, "审计席", "审计疑点", "", "P1 桩疑点一", d(3, 0), "")
	oe(t, ctx, svc, "审计席", "审计疑点", "", "P1 桩疑点二", d(2, 0), "")
	oe(t, ctx, svc, "审计席", "审计疑点", "", "桩疑点普通级", d(1, 0), "")

	// —— ⑦ 判词引用兑现（钦定正则 工单#([0-9]+)——91003 有前置
	// 处置=兑现，91004 无=未兑现）。
	oe(t, ctx, svc, "自我", "判词", "", "桩判词 引 工单#91003 与 工单#91004", d(1, 0), "")

	// —— ⑧ 研究转化（fin_opp_hypothesis adopted 90d 内）。
	//    共享基线含 md_1095 adopted 种子行——清场回 fin-only 旧径基线
	//    （计数断言=本夹具行唯一口径）。
	if _, err := svc.pool.Exec(ctx, `DELETE FROM fin_opp_hypothesis`); err != nil {
		t.Fatalf("hypo clean: %v", err)
	}
	if _, err := svc.pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis
		(venue,surface,family,state,ts) VALUES('okx','kpitest_surface','trend_follow','adopted',$1)`,
		d(10, 0)); err != nil {
		t.Fatalf("hypo: %v", err)
	}

	svc.seatKpiBeat(ctx)
	var note string
	if err := svc.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi' ORDER BY id DESC LIMIT 1`).
		Scan(&note); err != nil {
		t.Fatalf("seat_kpi 量表行未落: %v", err)
	}
	t.Logf("seat_kpi note: %s", note)
	for _, seg := range []string{
		"v=", "唤知:销中位2.0h·误销50.0%",
		"施工:完工中位2.0h·打回50.0%",
		"审计:P12", "探针:误报33.3%",
		"自我:兑现1/2", "研究:转化1/90d",
	} {
		if !strings.Contains(note, seg) {
			t.Fatalf("缺段 %q: %s", seg, note)
		}
	}

	// 同日二拍=指纹同+未陈→抑（行数不增）。
	svc.seatKpiBeat(ctx)
	var n int
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("同日重拍行数=%d err=%v 期 1（指纹闸抑）", n, err)
	}
}

// TestSeatKpiNoteShape 纯函数词形——nil 段呈 ? 不虚构。
func TestSeatKpiNoteShape(t *testing.T) {
	h := 2.5
	p1, res, k, nn := 3, 7, 1, 4
	note := seatKpiNote("2026-11-16", seatKpiM{
		escH: &h, refirePct: &h, doneH: &h, rejectPct: &h,
		p1: &p1, falsePct: &h, selfK: &k, selfN: &nn, research: &res,
		moneyK: &k, moneyN: &nn,
	})
	for _, seg := range []string{"v=2026-11-16", "销中位2.5h", "误销2.5%",
		"完工中位2.5h", "打回2.5%", "P13", "误报2.5%", "兑现1/4", "转化7/90d", "施工钱环:1/4"} {
		if !strings.Contains(note, seg) {
			t.Fatalf("缺段 %q: %s", seg, note)
		}
	}
	empty := seatKpiNote("2026-11-16", seatKpiM{})
	for _, seg := range []string{"销中位?h", "误销?%", "完工中位?h", "打回?%",
		"P1?", "误报?%", "兑现?/?", "转化?/90d", "施工钱环:?/?"} {
		if !strings.Contains(empty, seg) {
			t.Fatalf("空读面缺 ? 段 %q: %s", seg, empty)
		}
	}
}

// TestSeatKpiStrictDaily 严格日拍（业主令#20930）：面值一抖窗内不重发；
// 首行 ts 推到档期外→再拍重发。
func TestSeatKpiStrictDaily(t *testing.T) {
	ctx, svc := seatKpiSetup(t)
	now := time.Now()
	d := func(days, hours float64) time.Time {
		return now.Add(-time.Duration(days*24+hours) * time.Hour)
	}
	e1 := oe(t, ctx, svc, "探测席", "升件", "", "桩升件", d(10, 0), "")
	oe(t, ctx, svc, "处置席", "处置", "", "升件#"+strconv.FormatInt(e1, 10)+" 销——桩", d(10, -1), "")

	svc.seatKpiBeat(ctx)
	var n int
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("首拍行数=%d err=%v 期 1", n, err)
	}
	// 面值变化（新增一对升件+处置改误报率）→窗内仍抑（不看指纹）
	e2 := oe(t, ctx, svc, "探测席", "升件", "", "桩升件二", d(9, 0), "")
	oe(t, ctx, svc, "处置席", "处置", "", "升件#"+strconv.FormatInt(e2, 10)+" 销误报——桩", d(9, -1), "")
	svc.seatKpiBeat(ctx)
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("面值抖动窗内重发: n=%d err=%v 期 1", n, err)
	}
	// 首行推到档期外→再拍重发
	if _, err := svc.pool.Exec(ctx, `UPDATE ops_event SET ts=ts-interval '21 hours'
		WHERE event='量表' AND ticket='seat_kpi'`); err != nil {
		t.Fatalf("回拨首行: %v", err)
	}
	svc.seatKpiBeat(ctx)
	if err := svc.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='量表' AND ticket='seat_kpi'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("档期外未重发: n=%d err=%v 期 2", n, err)
	}
}
