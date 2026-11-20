package hunger

// closeoutprobe_test.go —— 施工认领配对探针锚（工单#26306 谓词放宽——
// 原#25790「回执→收口」配对改「认领→回执」配对）：
// ①认领超72h无回执→升件行落（note 载票号/认领号）；
// ②同票完工回执在案→零行（回执=收口同义认，放宽锚）；
// ③挂起还票/销票→零行；复办撤还票→照火；
// ④同票未销升件在案→二拍零新行；同 scope 开工单→静默。

import (
	"context"
	"strings"
	"testing"

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

// ①认领>72h∧无配对→升件；同认领二拍 dedup。
func TestCloseoutProbeFires(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25758 认领：开修——unit=o-worker-test")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("认领未配对升件未落 n=%d", n)
	}
	var note, cls string
	if err := pool.QueryRow(context.Background(),
		`SELECT note,esc_class FROM ops_event WHERE event='升件' AND esc_scope=$1`,
		closeoutEscScope).Scan(&note, &cls); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"施工认领未配对", "工单#25758", "认领#"} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因段缺 %q —— note=%s", want, note)
		}
	}
	if cls != closeoutEscClass {
		t.Fatalf("esc_class=%q want %q", cls, closeoutEscClass)
	}
	// 二拍 dedup——同票未销升件在案即抑。
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("二拍重复升件 n=%d", n)
	}
}

// ②放宽锚：认领后同票完工回执在案→配对零行（回执=收口同义认）。
func TestCloseoutProbePairedByReceipt(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25798 认领：开修——unit=o-worker-a")
	seedOpsEvent(t, pool, 30*60, "施工席", "处置",
		"工单#25798 完工回执——fix/x@abc123：全链绿戳在案")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("回执配对仍升件 n=%d", n)
	}
}

// ②b 陈旧回执不算数——回执 ts 早于认领 ts→不配对照火（重认领票）。
func TestCloseoutProbeStaleReceiptNoPair(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 120*60, "施工席", "处置",
		"工单#25798 完工回执——fix/y@def456")
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25798 认领：再办——unit=o-worker-b")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("陈旧回执误配对 n=%d", n)
	}
}

// ③a 挂起还票双形豁免：收口字段「挂起还票<表>」∨处置邻近「挂起条件=」。
func TestCloseoutProbeReturnExempt(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25801 认领：开修——unit=o-worker-c")
	seedOpsEvent(t, pool, 50*60, "施工席", "记事",
		"会话收口——完工25790；挂起还票25801；在案2")
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25805 认领：开修——unit=o-worker-d")
	seedOpsEvent(t, pool, 50*60, "施工席", "处置",
		"工单#25805 挂起条件=裁定——业主未授权域")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("还票态误升件 n=%d", n)
	}
}

// ③b 复办撤还票——还票后「挂起解=」再滞留照火。
func TestCloseoutProbeResumeCancelsReturn(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25807 认领：开修——unit=o-worker-e")
	seedOpsEvent(t, pool, 80*60, "施工席", "处置",
		"工单#25807 挂起条件=认领窗——预算耗尽")
	seedOpsEvent(t, pool, 60*60, "施工席", "处置",
		"工单#25807 挂起解=复办")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 1 {
		t.Fatalf("复办后滞留未升件 n=%d", n)
	}
}

// ③c 票销豁免——处置销票行→ops_event_act act='close'（触发器径）。
func TestCloseoutProbeCloseExempt(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25840 认领：开修——unit=o-worker-f")
	seedOpsEvent(t, pool, 50*60, "施工席", "处置",
		"工单#25840 销——裁定作废票")
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("销票仍升件 n=%d", n)
	}
	var act string
	if err := pool.QueryRow(context.Background(),
		`SELECT act FROM ops_event_act WHERE rel=25840 ORDER BY ts DESC LIMIT 1`).Scan(&act); err != nil {
		t.Fatalf("act 行未落（触发器径断）: %v", err)
	}
	if act != "close" {
		t.Fatalf("act=%q want close", act)
	}
}

// ④静默面：未超龄认领/外席认领/无认领→零行。
func TestCloseoutProbeSilent(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 10*60, "施工席", "认领",
		"工单#25843 认领：开修——unit=o-worker-g") // 10h<72h 未超龄
	seedOpsEvent(t, pool, 100*60, "外席", "认领",
		"工单#25844 认领：开修——unit=o-ext-h") // 外席域外
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("静默面误升件 n=%d", n)
	}
}

// ⑤票抑同契：同 scope 开工单在案→票管催，探针静默。
func TestCloseoutProbeTicketSuppress(t *testing.T) {
	s, pool := droughtFixture(t)
	seedOpsEvent(t, pool, 100*60, "施工席", "认领",
		"工单#25880 认领：开修——unit=o-worker-i")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('唤知席','工单','',
		 '滞留催办 scope=worker_closeout_missing 期限=48h')`); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeWorkerCloseout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := closeoutEscCount(t, pool); n != 0 {
		t.Fatalf("开工单在案仍升件 n=%d", n)
	}
}
