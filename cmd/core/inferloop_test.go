package main

// inferloop_test.go —— 工单#27050 锚：消费环三终态各一断言+
// 慧量表造数断言分母口径（会话/自处置/复火三造例）。

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func inferSvc(t *testing.T) (context.Context, *coreService) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return ctx, &coreService{pool: pool}
}

func seedLesson(t *testing.T, ctx context.Context, c *coreService, sig string, ts time.Time) int64 {
	t.Helper()
	var id int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_eng_lesson(signature,ts)
		VALUES($1,$2) RETURNING id`, sig, ts).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func seedEv(t *testing.T, ctx context.Context, c *coreService, ts time.Time,
	seat, event, ticket, note, cls, scope string) int64 {
	t.Helper()
	var id int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO ops_event(ts,seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		ts, seat, event, ticket, note, cls, scope).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// ①已免疫：签名升件簇→升件#引证行→工单#N→销行 merge:sha→lesson 标
// verdicted+verdict_ref 载合点（终态账=ops_decision 触发器裁决行）。
func TestLessonConsumeImmune(t *testing.T) {
	ctx, c := inferSvc(t)
	now := time.Now()
	lid := seedLesson(t, ctx, c, "升件:读败:mt5dead", now.Add(-time.Hour))
	eid := seedEv(t, ctx, c, now.Add(-2*time.Hour), "探测席", "升件", "", "v=x 读败", "读败", "mt5dead")
	seedEv(t, ctx, c, now.Add(-90*time.Minute), "处置席", "处置", "", "升件#"+itoa(eid)+" 销——根因票=工单#777", "", "")
	seedEv(t, ctx, c, now.Add(-80*time.Minute), "复审席", "处置", "", "工单#777 销——merge:abc1234(簿记对齐)", "", "")

	c.lessonConsume(ctx, map[string]time.Time{})
	var status, ref string
	if err := c.pool.QueryRow(ctx, `SELECT status,verdict_ref FROM fin_eng_lesson WHERE id=$1`, lid).
		Scan(&status, &ref); err != nil {
		t.Fatal(err)
	}
	if status != "verdicted" || !strings.Contains(ref, "已免疫@abc1234") {
		t.Fatalf("免疫终态败: status=%s ref=%s", status, ref)
	}
	// 幂等：复拍不再动（非 pending 不入扫面）。
	c.lessonConsume(ctx, map[string]time.Time{})
}

// ②待canon化：签名簇仍复火（lesson.ts 后同 cls:scope 新升件）→
// learn_canon 升件行落账，lesson 保持 pending（禁盲归纳——只请会话）。
func TestLessonConsumeCanonWait(t *testing.T) {
	ctx, c := inferSvc(t)
	now := time.Now()
	lts := now.Add(-2 * time.Hour)
	lid := seedLesson(t, ctx, c, "升件:断流:蒸馏 testdead", lts)
	seedEv(t, ctx, c, now.Add(-time.Hour), "探测席", "升件", "", "v=y 断流复火", "断流", "蒸馏 testdead")

	dead := map[string]time.Time{}
	c.lessonConsume(ctx, dead)
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='learn_canon'
		AND esc_scope='lesson:升件:断流:蒸馏 testdead'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("待canon化升件应=1 got %d", n)
	}
	var status string
	_ = c.pool.QueryRow(ctx, `SELECT status FROM fin_eng_lesson WHERE id=$1`, lid).Scan(&status)
	if status != "pending" {
		t.Fatalf("复火未免疫行不应终态化: %s", status)
	}
	// 簇熄复拍→learn_canon 边沿复绿（recoverNote 行）——deadSince 同键清。
	// （复火源已过 lts 不可撤——改验 learn_stale 不误伤本例：龄 2h<7d。）
}

// ③learn_stale：pending>7d 零动作→learn_stale 升件；签名非升件形也照点。
func TestLessonConsumeStale(t *testing.T) {
	ctx, c := inferSvc(t)
	now := time.Now()
	lid := seedLesson(t, ctx, c, "手感:oldslug", now.Add(-8*24*time.Hour))

	c.lessonConsume(ctx, map[string]time.Time{})
	var n int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='learn_stale'
		AND esc_scope='lesson:手感:oldslug'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("learn_stale 升件应=1 got %d", n)
	}
	var status string
	_ = c.pool.QueryRow(ctx, `SELECT status FROM fin_eng_lesson WHERE id=$1`, lid).Scan(&status)
	if status != "pending" {
		t.Fatalf("stale 不终态化: %s", status)
	}
	// 新鲜 pending 不误报 stale。
	fresh := seedLesson(t, ctx, c, "手感:freshslug", now.Add(-time.Hour))
	c.lessonConsume(ctx, map[string]time.Time{})
	var n2 int
	_ = c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='learn_stale'
		AND esc_scope='lesson:手感:freshslug'`).Scan(&n2)
	if n2 != 0 {
		t.Fatalf("新鲜行不应 stale got %d", n2)
	}
	_ = fresh
}

// ④慧量表造数：3 升件——e1 会话席销、e2 探测席自处置、e3 与 e2 同
// scope 复火 → 慧=自处置66%/会话33%/复火66%（total3 sess1 self2 refire2）。
func TestWisdomGaugeCounts(t *testing.T) {
	ctx, c := inferSvc(t)
	now := time.Now()
	e1 := seedEv(t, ctx, c, now.Add(-3*time.Hour), "探测席", "升件", "", "v=1", "断流", "faceA")
	e2 := seedEv(t, ctx, c, now.Add(-2*time.Hour), "探测席", "升件", "", "v=2", "读败", "faceB")
	seedEv(t, ctx, c, now.Add(-time.Hour), "探测席", "升件", "", "v=3", "读败", "faceB")
	// e1：处置席销=会话介入；e2：探测席引证=自处置；e3：无引证=自处置。
	seedEv(t, ctx, c, now.Add(-150*time.Minute), "处置席", "处置", "", "升件#"+itoa(e1)+" 销", "", "")
	seedEv(t, ctx, c, now.Add(-100*time.Minute), "探测席", "记事", "", "升件#"+itoa(e2)+" 观察", "", "")

	c.wisdomGaugeBeat(ctx)
	var note string
	if err := c.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='量表' AND ticket='wisdom_gauge' ORDER BY id DESC LIMIT 1`).
		Scan(&note); err != nil {
		t.Fatalf("慧量表行缺: %v", err)
	}
	for _, want := range []string{"慧=自处置66%/会话33%/复火66%", "N=3"} {
		if !strings.Contains(note, want) {
			t.Fatalf("慧量表口径败 want=%q note=%s", want, note)
		}
	}
	// 同纹抑：复拍不重发（gaugeDue 指纹闸）。
	c.wisdomGaugeBeat(ctx)
	var n int
	_ = c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='量表' AND ticket='wisdom_gauge'`).Scan(&n)
	if n != 1 {
		t.Fatalf("同纹应抑 got %d", n)
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
