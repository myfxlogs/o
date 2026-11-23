package hunger

// deskorphan_test.go —— 票桌孤儿池压力卡锚（工单O-35463②）：
// ①无证陈票→聚合升件（计数+最陈 id+无证/有证死因粗分）；
// ②24h 同窗 dedup 二拍零新行；③fresh(<24h)票不入池；
// ④有证未排（回执剔）票照计；⑤池清零→未销升件处置自销。

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// closeAllOrphans 测前清场：存量孤儿票全销（ops_event append-only——
// 处置销行是唯一合法清径；跨测持久面每测自重铺）。
func closeAllOrphans(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id FROM ops_ticket_desk
		WHERE workable AND tkind='工单' AND dispatch_rank IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := pool.Exec(context.Background(),
			`INSERT INTO ops_event(seat,event,ticket,note)
			 VALUES('处置席','处置','工单',$1)`,
			"工单#"+itoa(id)+" 销——deskorphan 测前清场"); err != nil {
			t.Fatal(err)
		}
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

// seedDeskTicket 票桌票桩：ageHours 前落工单行（note 形钉 tkind='工单'；
// cert=false=无证锚天然 rank NULL；cert=true 需配 receipt act 另裁）。
func seedDeskTicket(t *testing.T, pool *pgxpool.Pool, ageHours float64, note string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ops_event(ts,seat,event,ticket,note)
		 VALUES(now()-make_interval(secs=>$1),'唤知机','工单','',$2)
		 RETURNING id`,
		int(ageHours*3600), "工单/deskorphan-test: "+note).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func deskOrphanEscRows(t *testing.T, pool *pgxpool.Pool) []struct {
	id   int64
	note string
} {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id,note FROM ops_event
		WHERE event='升件' AND esc_scope=$1 ORDER BY id`, deskOrphanEscScope)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []struct {
		id   int64
		note string
	}
	for rows.Next() {
		var r struct {
			id   int64
			note string
		}
		if err := rows.Scan(&r.id, &r.note); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func TestDeskOrphanFires(t *testing.T) {
	s, pool := droughtFixture(t)
	closeAllOrphans(t, pool)
	old1 := seedDeskTicket(t, pool, 50, "陈票一 无证锚")
	old2 := seedDeskTicket(t, pool, 30, "陈票二 无证锚")
	seedDeskTicket(t, pool, 1, "fresh 票不入池") // <24h 不计
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	esc := deskOrphanEscRows(t, pool)
	if len(esc) != 1 {
		t.Fatalf("孤儿池升件未落/多落 n=%d", len(esc))
	}
	for _, want := range []string{"陈票 2 张", "最陈#" + itoa(old1), "无证锚=2", "有证未排=0"} {
		if !strings.Contains(esc[0].note, want) {
			t.Fatalf("归因段缺 %q —— note=%s", want, esc[0].note)
		}
	}
	_ = old2 // 次陈票在计数内——最陈锚=old1
	// dedup 二拍零新行
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(deskOrphanEscRows(t, pool)); n != 1 {
		t.Fatalf("二拍重复升件 n=%d", n)
	}
}

func TestDeskOrphanCertOrphanCounted(t *testing.T) {
	s, pool := droughtFixture(t)
	closeAllOrphans(t, pool)
	// 有证票被回执剔出排位集=孤儿（workable 仍真、rank NULL）。
	id := seedDeskTicket(t, pool, 50, "有证陈票 证=spec@/tmp/x.md")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('施工席','处置','工单',$1)`,
		"工单#"+itoa(id)+" 完工回执——已交付待销"); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	esc := deskOrphanEscRows(t, pool)
	if len(esc) != 1 {
		t.Fatalf("有证孤儿升件未落 n=%d", len(esc))
	}
	if !strings.Contains(esc[0].note, "无证锚=0") || !strings.Contains(esc[0].note, "有证未排=1") {
		t.Fatalf("死因粗分错——note=%s", esc[0].note)
	}
}

func TestDeskOrphanSelfResolve(t *testing.T) {
	s, pool := droughtFixture(t)
	closeAllOrphans(t, pool)
	id := seedDeskTicket(t, pool, 50, "陈票一")
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	esc := deskOrphanEscRows(t, pool)
	if len(esc) != 1 {
		t.Fatalf("前置升件未落 n=%d", len(esc))
	}
	// 池清零（销陈票）→自销处置行引「升件#id」。
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('处置席','处置','工单',$1)`,
		"工单#"+itoa(id)+" 销——测试清场"); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	var resolved bool
	if err := pool.QueryRow(context.Background(), `SELECT EXISTS(
		SELECT 1 FROM ops_event WHERE note ~ ('升件#'||$1::text||'([^0-9]|$)'))`,
		itoa(esc[0].id)).Scan(&resolved); err != nil {
		t.Fatal(err)
	}
	if !resolved {
		t.Fatal("池清零未产自销处置行")
	}
	// 复燃案：新孤儿再入池→24h 窗内 dedup 抑（同窗已报含自销后——
	// 自销行非升件不占 dedup 位，二升在 dedup 窗内仍被抑=设计内）。
	seedDeskTicket(t, pool, 50, "陈票二")
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(deskOrphanEscRows(t, pool)); n != 1 {
		t.Fatalf("dedup 窗内复燃误升 n=%d", n)
	}
}

func TestDeskOrphanSilentClean(t *testing.T) {
	s, pool := droughtFixture(t)
	closeAllOrphans(t, pool)
	if err := s.ProbeDeskOrphan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(deskOrphanEscRows(t, pool)); n != 0 {
		t.Fatalf("净池误升 n=%d", n)
	}
}
