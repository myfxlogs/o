package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 升件环：死→恰一升件（24h 同窗去重）；复绿→恢复行携升件#id 自闭。
func TestFeedEscLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='t_dead' OR note LIKE '%t_dead%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	svc.escalateUnresolved(ctx, "断流", "t_dead", "", "面断流 t_dead：零行——超阈。t_fix")
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='断流' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	svc.escalateUnresolved(ctx, "断流", "t_dead", "", "重报") // 未销重报不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='t_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("去重 n=%d err=%v", n, err)
	}
	svc.recoverNote(ctx, "断流", "t_dead", time.Now().Add(-10*time.Minute))
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%t_dead%'`).Scan(&note); err != nil {
		t.Fatalf("恢复行缺: %v", err)
	}
	var id int64
	_ = pool.QueryRow(ctx, `SELECT id FROM ops_event WHERE esc_scope='t_dead'`).Scan(&id)
	if want := "升件#" + strconv.FormatInt(id, 10); !strings.Contains(note, want) {
		t.Fatalf("恢复行未携升件引: %s（望含 %s）", note, want)
	}
}

// escalateUnresolved 值指纹闸（计量类边缘触发）：销案后同值复报不火
// （「销=此量值已阅」——常设态建模根修 2026-11-11）；异值=新现实重火；
// 空指纹=事件类语义不变（销后复报仍火）。
func TestEscValueFingerprint(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='t_fp'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	cnt := func() int {
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_scope='t_fp'`).Scan(&n)
		return n
	}
	svc.escalateUnresolved(ctx, "计量", "t_fp", "28", "利用率28%")
	if n := cnt(); n != 1 {
		t.Fatalf("首火 n=%d", n)
	}
	// 销案（处置行引升件#id）
	var id int64
	_ = pool.QueryRow(ctx, `SELECT id FROM ops_event WHERE esc_scope='t_fp'`).Scan(&id)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('处置席','处置',$1)`, fmt.Sprintf("升件#%d 同判词延续", id)); err != nil {
		t.Fatalf("销案行: %v", err)
	}
	svc.escalateUnresolved(ctx, "计量", "t_fp", "28", "利用率28%再报") // 同值不火
	if n := cnt(); n != 1 {
		t.Fatalf("销后同值复报 n=%d（应仍 1）", n)
	}
	svc.escalateUnresolved(ctx, "计量", "t_fp", "31", "利用率31%") // 异值重火
	if n := cnt(); n != 2 {
		t.Fatalf("异值重火 n=%d（应 2）", n)
	}
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE esc_scope='t_fp' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil ||
		!strings.HasPrefix(note, "v=31 ") {
		t.Fatalf("指纹前缀缺 note=%q err=%v", note, err)
	}
}

// reconEscCheck 歧态腿探：腿在案→升件；未销在案→不暴；销后归零→恢复行；
// 新批腿→未销才抑放行重升（同窗 dedup 不误吞新事证）。
func TestReconEscAmbiguousLegs(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_order_leg WHERE cl_ord_id LIKE 't-rec%';
		DELETE FROM fin_execution WHERE inst_id='T-REC-USDT';
		DELETE FROM ops_event WHERE esc_scope='ambiguous_legs'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES(NULL,'T-REC-USDT',10,'open') RETURNING id`).Scan(&eid); err != nil {
		t.Fatalf("exec: %v", err)
	}
	var legID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'t-rec-1','okx','buy','T-REC-USDT',1,'ambiguous') RETURNING id`, eid).Scan(&legID); err != nil {
		t.Fatalf("leg: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}

	svc.reconEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='对账' AND esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	var escID int64
	var note string
	if err := pool.QueryRow(ctx, `SELECT id,note FROM ops_event
		WHERE esc_class='对账' AND esc_scope='ambiguous_legs'`).Scan(&escID, &note); err != nil {
		t.Fatalf("升件行: %v", err)
	}
	if !strings.Contains(note, "歧态腿 1 条") {
		t.Fatalf("note 缺实数: %s", note)
	}
	svc.reconEscCheck(ctx, deadSince) // 未销在案不暴
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("未销抑 n=%d", n)
	}

	// 腿裁掉→恢复行携升件#id 自闭
	if _, err := pool.Exec(ctx, `UPDATE fin_order_leg SET state='failed' WHERE id=$1`, legID); err != nil {
		t.Fatalf("裁腿: %v", err)
	}
	svc.reconEscCheck(ctx, deadSince)
	var rec string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%ambiguous_legs%'`).Scan(&rec); err != nil {
		t.Fatalf("恢复行缺: %v", err)
	}
	if want := "升件#" + strconv.FormatInt(escID, 10); !strings.Contains(rec, want) {
		t.Fatalf("恢复行未携引: %s", rec)
	}

	// 新批腿→已销不抑,重升
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'t-rec-2','okx','sell','T-REC-USDT',1,'ambiguous')`, eid); err != nil {
		t.Fatalf("新腿: %v", err)
	}
	svc.reconEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='ambiguous_legs'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("新批重升 n=%d", n)
	}
}

// cognitionEscCheck：未决升件在案+处置行缺席>24h→唤知面死升件；
// 处置行落地→恢复自闭（alerter nil=人推安全跳过）。
func TestCognitionEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope IN ('t_open','cognition_dead') OR event='处置'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','','t 未决','断流','t_open')`); err != nil {
		t.Fatalf("esc: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='唤知' AND esc_scope='cognition_dead'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("唤知死升件 n=%d err=%v", n, err)
	}
	// 处置行落地→唤知面复绿（t_open 未决仍在,恢复语义只看处置行新鲜度）
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','','t 处置行落地')`); err != nil {
		t.Fatalf("处置: %v", err)
	}
	svc.cognitionEscCheck(ctx, deadSince, nil)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%cognition_dead%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// dataGapEscCheck：unevaluable 同真因≥3 持续→升件 data_gap_*；真因消→恢复。
func TestDataGapEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-GAP%';
		DELETE FROM ops_event WHERE esc_scope='data_gap_t_missing'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
			VALUES($1,'t_fam','gap_carry','unevaluable','[]',
				('{"reason":"t_missing"}')::jsonb)`,
			fmt.Sprintf("T-GAP-%d", i)); err != nil {
			t.Fatalf("eval%d: %v", i, err)
		}
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	svc.dataGapEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='数据缺' AND esc_scope='data_gap_t_missing'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件 n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_eval WHERE inst_id LIKE 'T-GAP%'`); err != nil {
		t.Fatalf("消: %v", err)
	}
	svc.dataGapEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%data_gap_t_missing%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("恢复 n=%d", n)
	}
}

// escEdgeU 未销才抑沿：非死→死恰一报,持续死不重报(未销在案),复绿自闭。
func TestEscEdge(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='t_edge'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	ds := map[string]time.Time{}
	fire := func(d bool) {
		svc.escEdgeU(ctx, ds, "资金闸", "t_edge", d, func() string { return "t_edge note" })
	}
	fire(false)
	fire(true)
	fire(true) // 持续死不重报
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='t_edge'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("沿触发 n=%d err=%v", n, err)
	}
	fire(false) // 复绿自闭
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='恢复' AND note LIKE '%t_edge%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自闭 n=%d err=%v", n, err)
	}
}

// capabilityEscCheck 能力债探（T-0523）：启用模板缺凭据/评估→ops_state_row
// 况行（fin_0202 减法B批——升件改态表行写,「不升件不唤醒,变了才更新一行」）；
// 凭据补上→况行自消；模板禁用→键死行除（want 缺席即删）。
// 占用感知回归锚——零占免演模板（excell⑤）不火演习债只火评估债。
func TestCapabilityEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_template;
		DELETE FROM fin_exec_drill WHERE cell_key LIKE 'funding-hedge@%';
		DELETE FROM fin_opp_eval WHERE template IN ('funding-hedge','vol_sell_strangle');
		DELETE FROM fin_strategy WHERE template_ref IN ('funding-hedge','vol_sell_strangle');
		DELETE FROM ops_state_row WHERE scope LIKE 'capability_%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,enabled,state)
		VALUES('funding-hedge','okx',true,'enabled'),
		      ('vol_sell_strangle','okx',true,'enabled');
		INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('BTC-USDT-SWAP','funding-hedge','live',50)`); err != nil {
		t.Fatalf("模板: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	detail := func(scope string) string {
		var d string
		if err := pool.QueryRow(ctx, `SELECT detail FROM ops_state_row
			WHERE kind='能力债' AND scope=$1`, scope).Scan(&d); err != nil {
			t.Fatalf("况行 %s 缺: %v", scope, err)
		}
		return d
	}

	// 占用格模板零凭据零评估→况行（funding-hedge 缺②③两项）；
	// 零占免演模板→只缺③（vol_sell_strangle 演习债豁免）
	svc.capabilityEscCheck(ctx, deadSince)
	if d := detail("capability_funding-hedge"); !strings.Contains(d, "无绿演习凭据") || !strings.Contains(d, "零评估产出") {
		t.Fatalf("况行缺缺项载: %s", d)
	}
	if d := detail("capability_vol_sell_strangle"); strings.Contains(d, "无绿演习凭据") || !strings.Contains(d, "零评估产出") {
		t.Fatalf("免演族况行误载演习债: %s", d)
	}
	// 同值再拍→行在而 updated_at 不动（「变了才更新一行」）
	var u1 time.Time
	if err := pool.QueryRow(ctx, `SELECT updated_at FROM ops_state_row
		WHERE scope='capability_funding-hedge'`).Scan(&u1); err != nil {
		t.Fatalf("updated_at: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	var u2 time.Time
	if err := pool.QueryRow(ctx, `SELECT updated_at FROM ops_state_row
		WHERE scope='capability_funding-hedge'`).Scan(&u2); err != nil {
		t.Fatalf("updated_at2: %v", err)
	}
	if !u1.Equal(u2) {
		t.Fatalf("同值重拍 updated_at 动了: %v→%v", u1, u2)
	}

	// 凭据+评估补上→况行自消（事实消失=行除非销案）
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,cell_key)
		VALUES(-1,'BTC-USDT-SWAP','pass','funding-hedge@okx/short_perp_long_spot#usd=50');
		INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('BTC-USDT-SWAP','funding-hedge','short_perp_long_spot','rejected','[]','{}'),
		      ('BTC-C-150+BTC-P-100','vol_sell_strangle','vol_sell','rejected','[]','{}')`); err != nil {
		t.Fatalf("凭据: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_state_row
		WHERE kind='能力债' AND scope LIKE 'capability_%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("凭据补上况行未消 n=%d err=%v", n, err)
	}
}

// capabilityEscCheck 启用窗起算（工单#17938）：新启用族(<7d)演习债
// 宽限——凭据窗起点=enabled_at（hypothesis adopted 日∨参数调整
// 启行最新ts）；无据=-infinity 回落旧 7d 回看。评估债臂不修。
// fin_0202 起断言面=ops_state_row 况行存/消（原升件/恢复行语义退役）。
func TestCapabilityEscCheckEnableGrace(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_template WHERE template='funding-hedge';
		DELETE FROM fin_exec_drill WHERE cell_key LIKE 'funding-hedge@%';
		DELETE FROM fin_opp_eval WHERE template='funding-hedge';
		DELETE FROM fin_strategy WHERE template_ref='funding-hedge';
		DELETE FROM fin_opp_hypothesis WHERE venue='okx' AND surface='hedge_s';
		DELETE FROM ops_state_row WHERE scope='capability_funding-hedge';
		DELETE FROM ops_event WHERE event='参数调整' AND note LIKE 'target=fin_opp_template key=funding-hedge@%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,surface,enabled,state)
		VALUES('funding-hedge','okx','hedge_s',true,'enabled');
		INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('BTC-USDT-SWAP','funding-hedge','live',50)`); err != nil {
		t.Fatalf("模板: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	row := func() string {
		var d string
		if err := pool.QueryRow(ctx, `SELECT detail FROM ops_state_row
			WHERE scope='capability_funding-hedge'`).Scan(&d); err != nil {
			return "" // 无行=况消
		}
		return d
	}

	// 相位A：hypothesis adopted=now()（启用<7d）→演习债宽限,
	// 况行只载评估债不载演习债。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_hypothesis(venue,surface,family,state)
		VALUES('okx','hedge_s','directional','adopted')`); err != nil {
		t.Fatalf("hypo: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	if d := row(); d == "" {
		t.Fatalf("相位A 况行缺（评估债应在）")
	} else if strings.Contains(d, "无绿演习凭据") {
		t.Fatalf("新启用族演习债未宽限: %s", d)
	} else if !strings.Contains(d, "零评估产出") {
		t.Fatalf("评估债臂应照载: %s", d)
	}
	// 相位A2：补评估行→全腿 gap=false→况行自消。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('BTC-USDT-SWAP','funding-hedge','short_perp_long_spot','rejected','[]','{}')`); err != nil {
		t.Fatalf("评估行: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	if d := row(); d != "" {
		t.Fatalf("相位A2 况行应自消: %s", d)
	}

	// 相位B：adopted=8d 前（启用≥7d）→演习债宽限退场照载（评估债已清,
	// 况行应只载演习债）。
	if _, err := pool.Exec(ctx, `UPDATE fin_opp_hypothesis SET ts=now()-interval '8 days'
		WHERE venue='okx' AND surface='hedge_s'`); err != nil {
		t.Fatalf("hypo 老化: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	if d := row(); !strings.Contains(d, "无绿演习凭据") {
		t.Fatalf("启用≥7d 零凭据应照载: %s", d)
	} else if strings.Contains(d, "零评估产出") {
		t.Fatalf("评估债已清不应再载: %s", d)
	}

	// 相位C：hypothesis 除名,参数调整启行=now()（hunger 复活径）
	// →演习债再宽限;evalN>0 全腿 gap=false 况行再消。
	if _, err := pool.Exec(ctx, `DELETE FROM fin_opp_hypothesis WHERE venue='okx' AND surface='hedge_s';
		INSERT INTO ops_event(seat,event,note) VALUES('探测席','参数调整',
		'target=fin_opp_template key=funding-hedge@okx 旧=enabled=false/state=retired 新=enabled=true/state=enabled 证据=proposed_eval×3@窗7天')`); err != nil {
		t.Fatalf("启行: %v", err)
	}
	svc.capabilityEscCheck(ctx, deadSince)
	if d := row(); d != "" {
		t.Fatalf("相位C 况行应自消: %s", d)
	}
}

// TestCognitionSpawnFailEsc 唤起败连败探——2h 窗 ≥2 败=死态（降级契约
// 工单#22578①：自观会话面不落升件,死回合沿=日计一火）;单败=瞬态
// 不计;同回合死态持续不重计;复绿再死=新回合再计。
func TestCognitionSpawnFailEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='cognition_spawn_fail' OR note LIKE '%t-spawn%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	var n int
	resetSelfMeter()

	// 单败=瞬态——不计
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知机','唤起败','','unit=t-spawn-1 reason=session_discovery')`); err != nil {
		t.Fatalf("败行1: %v", err)
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 0 {
		t.Fatalf("单败误计: %v", selfEscMeter.counts)
	}

	// 连败→死态——无升件,日计一火
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('唤知机','唤起败','','unit=t-spawn-2 reason=session_discovery')`); err != nil {
		t.Fatalf("败行2: %v", err)
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='cognition_spawn_fail'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 1 {
		t.Fatalf("死回合未入日计: %v", selfEscMeter.counts)
	}

	// 死态持续（再拍同死）→同回合不重计
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 1 {
		t.Fatalf("同回合重复计: %v", selfEscMeter.counts)
	}

	// 败行全销净（条件翻假）→复绿清态零恢复行;再连败=新回合再计一火
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE seat='唤知机' AND event='唤起败' AND note LIKE '%t-spawn%'`); err != nil {
		t.Fatalf("清败: %v", err)
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%cognition_spawn_fail%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件不应出恢复行 n=%d err=%v", n, err)
	}
	for i := 3; i <= 4; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','unit=t-spawn-3 reason=session_discovery')`); err != nil {
			t.Fatalf("新败行%d: %v", i, err)
		}
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 2 {
		t.Fatalf("复绿再死未计新回合: %v", selfEscMeter.counts)
	}
}

// TestCognitionSpawnFailEscClusterRefire 降级契约下的异簇契——降级后
// 升件面全空,计数语义=死回合数:复绿后的新簇=新回合照计（#22578①
// 收编工单#20690 纹吞病灶——簇间界由复绿沿定非指纹值）。
func TestCognitionSpawnFailEscClusterRefire(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='cognition_spawn_fail' OR note LIKE '%t-spawn2%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	resetSelfMeter()

	// 新簇连败→死态计一;败行全销→复绿;同计数新簇→再计一（异簇异回合）
	for i := 1; i <= 2; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','unit=t-spawn2 reason=session_discovery')`); err != nil {
			t.Fatalf("败行%d: %v", i, err)
		}
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 1 {
		t.Fatalf("簇一未计: %v", selfEscMeter.counts)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE seat='唤知机' AND event='唤起败' AND note LIKE '%t-spawn2%'`); err != nil {
		t.Fatalf("清簇: %v", err)
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	for i := 1; i <= 2; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('唤知机','唤起败','','unit=t-spawn2-b reason=session_discovery')`); err != nil {
			t.Fatalf("新败行%d: %v", i, err)
		}
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 2 {
		t.Fatalf("异簇未再计: %v", selfEscMeter.counts)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='cognition_spawn_fail'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
}

// TestAuditSpawnFailEsc 审计臂唤起败连败探——seat=审计机 同构两败计
// audit_spawn_fail（降级面,无升件）；seat 隔离=唤知机败迹不擡本面。
func TestAuditSpawnFailEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope IN ('audit_spawn_fail','cognition_spawn_fail') OR note LIKE '%t-audit%'`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	resetSelfMeter()

	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('审计机','唤起败','','unit=t-audit-1 reason=session_discovery'),
		      ('审计机','唤起败','','unit=t-audit-2 reason=session_discovery')`); err != nil {
		t.Fatalf("败行: %v", err)
	}
	svc.cognitionSpawnFailEscCheck(ctx, deadSince)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='唤知' AND esc_scope='audit_spawn_fail'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
	if selfEscMeter.counts["唤知/audit_spawn_fail"] != 1 {
		t.Fatalf("审计两败未入日计: %v", selfEscMeter.counts)
	}
	if selfEscMeter.counts["唤知/cognition_spawn_fail"] != 0 {
		t.Fatalf("seat 隔离破: %v", selfEscMeter.counts)
	}
}

// TestAuditStallEsc 审计臂停摆探（降级契约#22578①）——产出锚 NULL=语义
// 空态不死;末审计行陈 48h=死态日计一火无升件;新行落地=复绿清态零恢复行。
func TestAuditStallEsc(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event
		WHERE esc_scope='audit_stall' OR (seat='审计席' AND note LIKE '%t-stall%')
		OR (seat='探测席' AND event='恢复' AND note LIKE '%audit_stall%')`); err != nil {
		t.Fatalf("clean: %v", err)
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	var n int
	resetSelfMeter()

	// 零行=臂未装合法态——不死不计
	svc.auditStallEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_scope='audit_stall'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("空态误火 n=%d err=%v", n, err)
	}
	if selfEscMeter.counts["唤知/audit_stall"] != 0 {
		t.Fatalf("空态误计: %v", selfEscMeter.counts)
	}

	// 末审计行陈 3 天→死态——无升件,日计一火
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
		VALUES('审计席','审计','','t-stall 陈件',now()-interval '3 days')`); err != nil {
		t.Fatalf("陈行: %v", err)
	}
	svc.auditStallEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE esc_class='唤知' AND esc_scope='audit_stall'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟升件 n=%d err=%v", n, err)
	}
	if selfEscMeter.counts["唤知/audit_stall"] != 1 {
		t.Fatalf("死态未入日计: %v", selfEscMeter.counts)
	}

	// 新审计行落地→复绿清态——无恢复行（未开之件无可闭）
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('审计席','审计','','t-stall 新件')`); err != nil {
		t.Fatalf("新行: %v", err)
	}
	svc.auditStallEscCheck(ctx, deadSince)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE seat='探测席' AND event='恢复' AND note LIKE '%audit_stall%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("降级件竟写恢复 n=%d err=%v", n, err)
	}
}
