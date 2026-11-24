package hunger

// idleverdict_test.go —— 有理闲置判词质量闸锚（工单O-36962）：
// ①判词「有理闲置」缺停泊+回引 live_drought/idle_capital 升件→聚合升件；
// ②判词含「停泊」→静默；③回引非闲置类升件→静默；④闸起点前不溯；
// ⑤24h 同窗 dedup；⑥停泊基线段入归因卡（fin_park_baseline 视图实读）。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedIdleEsc 升件桩：esc_class/scope 定死闲置两类之一或自定义。
func seedIdleEsc(t *testing.T, pool *pgxpool.Pool, class, scope string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		 VALUES('探测席','升件','','esc-test',$1,$2) RETURNING id`,
		class, scope).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// seedVerdict 判词桩：event='处置' note 自定，tsSQL 空=now()（闸起点案覆写）。
func seedVerdict(t *testing.T, pool *pgxpool.Pool, note string, tsSQL string) int64 {
	t.Helper()
	var ts any
	if tsSQL != "" {
		ts = tsSQL
	}
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ops_event(ts,seat,event,ticket,note)
		 VALUES(COALESCE($2::timestamptz,now()),'处置席','处置','','工单/x '||$1)
		 RETURNING id`, note, ts).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func idleGateCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event
		 WHERE event='升件' AND esc_class='idle_verdict_gate'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIdleVerdictGateFires(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	escID := seedIdleEsc(t, pool, "live_drought", "live_drought")
	verdictID := seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：判据过线无候选", escID), "")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 1 {
		t.Fatalf("判词质量闸升件未落 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event
		 WHERE event='升件' AND esc_class='idle_verdict_gate'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "停泊基线") || !strings.Contains(note, "#"+fmt.Sprint(verdictID)) {
		t.Fatalf("升件 note 缺必备要素: %s", note)
	}
	// 同窗 dedup——二拍零新行
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 1 {
		t.Fatalf("同窗复火 n=%d", n)
	}
}

func TestIdleVerdictGateSilentWithPark(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	escID := seedIdleEsc(t, pool, "资金闸", "idle_capital")
	seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：停泊基线=Earn 9385.66@1.74%% 无候选过线 到期=11-30", escID), "")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 0 {
		t.Fatalf("含停泊判词误报 n=%d", n)
	}
}

func TestIdleVerdictGateSilentNonIdleEsc(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	escID := seedIdleEsc(t, pool, "断流", "feed_stale")
	seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：无候选", escID), "")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 0 {
		t.Fatalf("非闲置类回引误报 n=%d", n)
	}
}

func TestIdleVerdictGateSilentPreGate(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	escID := seedIdleEsc(t, pool, "live_drought", "live_drought")
	// 闸起点 2026-11-24 前的旧判词不溯
	seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：旧判词无停泊", escID),
		"2026-11-20T00:00:00+00")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 0 {
		t.Fatalf("闸起点前判词误溯 n=%d", n)
	}
}

func TestIdleVerdictGateThreeFields(t *testing.T) {
	// spec 验收：必备三字段缺任一=不合格——分测缺「过线」「到期」两臂
	s, pool := liveDroughtFixture(t)
	escID := seedIdleEsc(t, pool, "live_drought", "live_drought")
	// 停泊有、过线因有、缺到期日 → 不合格
	seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：停泊=Earn@1.7%% 无候选过线", escID), "")
	// 停泊有、到期有、缺过线因 → 不合格
	seedVerdict(t, pool,
		fmt.Sprintf("有理闲置——升件#%d 销：停泊=Earn@1.7%% 到期=11-30", escID), "")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 1 {
		t.Fatalf("三字段缺二应聚合一件 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event
		 WHERE event='升件' AND esc_class='idle_verdict_gate'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "2 件判词") {
		t.Fatalf("两不合格件应聚合计数=2: %s", note)
	}
}

func TestIdleVerdictGateSilentNoRef(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	// 有理闲置判词但无 #id 回引——不属本闸面（散文判词归处置席自律）
	seedVerdict(t, pool, "有理闲置：判据过线，钱留 Earn 停泊", "")
	if err := s.ProbeIdleVerdictGate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := idleGateCount(t, pool); n != 0 {
		t.Fatalf("无回引判词误报 n=%d", n)
	}
}

func TestParkBaselineCardSegment(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	// md_account_events 不在 SeedRestore 清表列——测后自清防跨测污染
	// （earn_parking 快照残留会让后续卡测读到非 NULL 本金）。
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM md_account_events WHERE venue='okx'
			 AND channel IN ('earn_parking','dcd_live')`)
	})
	// 停泊基线种子：earn_parking 快照+30d 息流+DCD 在仓
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_account_events(ts,venue,channel,ccy,inst_id,data)
		 VALUES(now(),'okx','earn_parking','USDT','',
		   '{"earn_bal":9385.66}'::jsonb)
		 ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=now(),data=EXCLUDED.data`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_cashflow(ts,occurred_at,kind,venue,ccy,amount,source_ref,source_table)
		 VALUES(now(),now()-interval '10 days','interest','okx','USDT',4.50,990001,'md_account_events'),
		       (now(),now()-interval '5 days','interest','okx','USDT',9.14,990002,'md_account_events')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO md_account_events(ts,venue,channel,ccy,inst_id,data)
		 VALUES(now(),'okx','dcd_live','','',
		   '{"orders":[{"notionalSz":"30","notionalCcy":"USDT"},{"notionalSz":"20","notionalCcy":"USDT"}]}'::jsonb)
		 ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=now(),data=EXCLUDED.data`); err != nil {
		t.Fatal(err)
	}
	// 触发归因卡：live 在册=0 且史无 live（liveDroughtAge 回退策略宇宙诞生——
	// 种一枚老 paper 策略即超龄）
	seedStrategy(t, pool, "paper", 9)
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND esc_class='live_drought'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"停泊=Earn 9385.66USDT@", "%/yr(实收30d息13.64)",
		"+DCD 2笔/50USDT", "机会成本=候选net_apr≥停泊", "%+溢价4.00%"} {
		if !strings.Contains(note, want) {
			t.Fatalf("停泊基线段缺 %q —— note=%s", want, note)
		}
	}
}
