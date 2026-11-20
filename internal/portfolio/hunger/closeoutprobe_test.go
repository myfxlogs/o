package hunger

// closeoutprobe_test.go —— 施工收口探针四锚（工单#25790）：
// ①回执超窗无收口→升件行落（note 载票号/回执号）；②同席收口载完工→
// 零行（新契字段形+老形双锚）；③未超窗/外席回执→零行；④同票未销
// 升件在案→二拍零新行（dedup 同契）。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedOpsEvent 事件行落桩（aged=分钟前；ts 显式落桩——探针窗判据锚）。
func seedOpsEvent(t *testing.T, pool *pgxpool.Pool, ageMin int, seat, event, note string) {
	t.Helper()
	var cls, scope any
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_event(ts,seat,event,ticket,note,esc_class,esc_scope)
		 VALUES(now()-make_interval(mins=>$1),$2,$3,'',$4,$5,$6)`,
		ageMin, seat, event, note, cls, scope); err != nil {
		t.Fatal(err)
	}
}

func closeoutEscCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_scope=$1`,
		closeoutEscScope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCloseoutProbeFires(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 40, "施工席", "处置",
		"工单#25758 完工回执——fix/x@abc123：全链绿戳在案")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("收口缺失升件未落 n=%d", n)
	}
	var note, cls string
	if err := pool.QueryRow(context.Background(),
		`SELECT note,esc_class FROM ops_event WHERE event='升件' AND esc_scope=$1`,
		closeoutEscScope).Scan(&note, &cls); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"施工收口缺失", "工单#25758", "回执#"} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因段缺 %q —— note=%s", want, note)
		}
	}
	if cls != closeoutEscClass {
		t.Fatalf("升件类=%q want=%q", cls, closeoutEscClass)
	}
	// 同票未销升件在案——二拍零新行
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("dedup 复火 n=%d", n)
	}
}

func TestCloseoutProbeCoveredBothForms(t *testing.T) {
	cases := []struct{ name, note string }{
		{"新契字段形", "unit=o-worker——干活;认领25758;完工25758;挂起还票无"},
		{"老形", "unit=o-worker——工单#25758完工：修毕"},
		{"票表形", "unit=o-worker——干活;认领25758,25760;完工25758,25760;挂起还票无"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, pool := droughtFixture(t)
			seedOpsEvent(t, pool, 40, "施工席", "处置",
				"工单#25758 完工回执——fix/x@abc123：全链绿戳在案")
			seedOpsEvent(t, pool, 30, "施工席", "会话收口", c.note)
			if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
				t.Fatal(err)
			}
			if n := closeoutEscCount(t, pool); n != 0 {
				t.Fatalf("已收口误报 n=%d", n)
			}
		})
	}
}

func TestCloseoutProbeSilent(t *testing.T) {
	t.Run("未超窗", func(t *testing.T) {
		s, pool := droughtFixture(t)
		seedOpsEvent(t, pool, 10, "施工席", "处置",
			"工单#25758 完工回执——fix/x@abc123：全链绿戳在案")
		if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
			t.Fatal(err)
		}
		if n := closeoutEscCount(t, pool); n != 0 {
			t.Fatalf("窗内误报 n=%d", n)
		}
	})
	t.Run("外席域外", func(t *testing.T) {
		s, pool := droughtFixture(t)
		seedOpsEvent(t, pool, 40, "外席", "记事",
			"工单#25758 完工回执——fix/x@abc123：全链绿戳在案")
		if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
			t.Fatal(err)
		}
		if n := closeoutEscCount(t, pool); n != 0 {
			t.Fatalf("外席回执误报 n=%d", n)
		}
	})
	t.Run("零回执", func(t *testing.T) {
		s, pool := droughtFixture(t)
		if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
			t.Fatal(err)
		}
		if n := closeoutEscCount(t, pool); n != 0 {
			t.Fatalf("零回执误报 n=%d", n)
		}
	})
}

func TestCloseoutProbeTicketSuppress(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 40, "施工席", "处置",
		"工单#25758 完工回执——fix/x@abc123：全链绿戳在案")
	// 同 scope 开工单在案（无处置销）——票抑静默
	seedOpsEvent(t, pool, 20, "唤知席", "工单",
		"收口缺失核处——scope=worker_closeout_missing")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("票抑失效 n=%d", n)
	}
}

func TestClaimedDoneTickets(t *testing.T) {
	cases := []struct {
		note string
		want []string
	}{
		{"unit=o-worker——干活;认领25790;完工25790;挂起还票无", []string{"25790"}},
		{"完工25790,25798,25801,25805", []string{"25790", "25798", "25801", "25805"}},
		{"完工无", nil},
		{"完工回执在案", nil},
		{"unit=x;认领25801;完工无;挂起还票25801", nil},
		{"工单#24376完工：md_klines 修毕", []string{"24376"}},
		{"工单#200完工回执在案", nil},
		{"完工#16226（补证）", []string{"16226"}},
	}
	for i, c := range cases {
		got := claimedDoneTickets(c.note)
		if len(got) != len(c.want) {
			t.Fatalf("case%d: got=%v want=%v", i, got, c.want)
		}
		for _, w := range c.want {
			if !got[w] {
				t.Fatalf("case%d: 缺 %s got=%v", i, w, got)
			}
		}
	}
	_ = time.Now
}
