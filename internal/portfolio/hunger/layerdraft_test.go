package hunger

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// T-0508 锚①②③桩测：层扫升件行→diagnosis 草案落库 / signature dedup
// （在飞占位+已诊永闭+24h 防灌）/ 畸形行 fail-closed；物料化载荷钉
// 类目/对象/首报时间/计数轨迹（票面钉——不编根因不编修法）。

// seedLayerEsc 层扫升件行素材（T-0507 产出契约形——seat=MECH
// event=升件 note=「层扫定性<类目>:<对象>——<详情>」）。
func seedLayerEsc(t *testing.T, s *Sensor, ctx context.Context, ticket, note string) int64 {
	t.Helper()
	var id int64
	if err := s.pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('MECH','升件',$1,$2) RETURNING id`, ticket, note).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestLayerScanToDraft 锚①：人造层扫升件→fin_ticket_draft pending
// diagnosis 草案（signature=「层扫:<类目>:<对象>」钉死字面）+起草行。
func TestLayerScanToDraft(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedLayerEsc(t, s, ctx, "T-9001",
		"层扫定性dispatched-无拉:T-9001——相位dispatched 龄45m>阈30m 且无在飞o-worker unit")
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if n != 1 {
		t.Fatalf("起草数非 1: %d", n)
	}
	var status, kind, key, sig, payload string
	if err := s.pool.QueryRow(ctx, `SELECT status,kind,lesion_key,signature,payload::text
		FROM fin_ticket_draft WHERE signature='层扫:dispatched-无拉:T-9001'`).Scan(&status, &kind, &key, &sig, &payload); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || kind != "diagnosis" || key != "dispatched-无拉:T-9001" {
		t.Fatalf("草案行形态非期: %s %s %s %s", status, kind, key, sig)
	}
	for _, frag := range []string{"dispatched-无拉", "T-9001", "evidence_ids",
		"evidence_query", "domain_hint", "首报", "签名全期升件 1 行", "近24 h窗 1 行"} {
		if !strings.Contains(payload, frag) {
			t.Fatalf("载荷缺 %q: %s", frag, payload)
		}
	}
	// slug=layerscan- 前缀+签名 crc 尾（materialize 幂等 glob 防跨签名误并）。
	for _, frag := range []string{"layerscan-", "dispatched-t-9001"} {
		if !strings.Contains(payload, frag) {
			t.Fatalf("slug 缺 %q: %s", frag, payload)
		}
	}
	var note string
	if err := s.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='起草' AND ticket='T-0393'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "signature=层扫:dispatched-无拉:T-9001") {
		t.Fatalf("起草行缺签名: %s", note)
	}
}

// TestLayerScanDedupSameSignature 锚①后半：同签名再报（窗内第二行+
// 下拍重扫）零新草案——pending 占位+24h 防灌双闸。
func TestLayerScanDedupSameSignature(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedLayerEsc(t, s, ctx, "", "层扫定性端点失健:dashboard——SPA=502 Ping=502")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("首拍: n=%d err=%v", n, err)
	}
	// 同签名再报一行（绕过 ops_esc 24h 去重的直插面——起草径自防）。
	seedLayerEsc(t, s, ctx, "", "层扫定性端点失健:dashboard——SPA=503 Ping=503 再报")
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("再拍应零产: n=%d err=%v", n, err)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature='层扫:端点失健:dashboard'`).Scan(&c); err != nil || c != 1 {
		t.Fatalf("签名防灌失效: c=%d err=%v", c, err)
	}
	// 异签名不互挡（端点失健 vs origin-落后=两草案）。
	seedLayerEsc(t, s, ctx, "", "层扫定性origin-落后:origin/master——本地master领先 3件")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("异签名应另起: n=%d err=%v", n, err)
	}
}

// TestLayerScanInflightDiagnosedDedup 锚②③：materialized 票在飞→占位；
// 票翻终态（完成/作废）→已诊永闭——同 signature 再报不重复起
// （复发≠新草案——与探针径「终态释键 24h 后可再起」语义分野钉死）。
func TestLayerScanInflightDiagnosedDedup(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedLayerEsc(t, s, ctx, "T-9002", "层扫定性review-老化:T-9002——相位review 龄3h>阈2h")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("首拍: n=%d err=%v", n, err)
	}
	// 物料化落号 T-9100（桩拟——票在飞事件链：起草/已批/派单非终态）。
	if _, err := s.pool.Exec(ctx, `UPDATE fin_ticket_draft
		SET status='materialized',ticket='T-9100',materialized_at=now()
		WHERE signature='层扫:review-老化:T-9002'`); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []string{"起草", "已批", "派单"} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('决策席',$1,'T-9100','')`, ev); err != nil {
			t.Fatal(err)
		}
	}
	// 草案行陈化出 24h 防灌窗（ts 拨回）——隔离「窗内防灌」与「在飞占位」。
	if _, err := s.pool.Exec(ctx, `UPDATE fin_ticket_draft SET ts=now()-interval '48 hours'
		WHERE signature='层扫:review-老化:T-9002'`); err != nil {
		t.Fatal(err)
	}
	seedLayerEsc(t, s, ctx, "T-9002", "层扫定性review-老化:T-9002——相位review 龄5h 再报")
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("在飞票应挡: n=%d err=%v", n, err)
	}
	var skip string
	if err := s.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='起草' AND note LIKE 'dedup skip signature=层扫:review-老化:T-9002%'
		ORDER BY id DESC LIMIT 1`).Scan(&skip); err != nil {
		t.Fatalf("dedup 跳过行缺: %v", err)
	}
	if !strings.Contains(skip, "在飞票=T-9100") {
		t.Fatalf("跳过行缺在飞票引: %s", skip)
	}
	// 票翻完成→已诊：同 signature 再报仍零新草案（永闭——票面消行语义）。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席','完成','T-9100','')`); err != nil {
		t.Fatal(err)
	}
	// 跳过行自身 24h 前缀去重（drafterSkipRow 同例）——清前记让已诊跳过行可落。
	if _, err := s.pool.Exec(ctx, `DELETE FROM ops_event
		WHERE event='起草' AND note LIKE 'dedup skip signature=层扫:review-老化:T-9002%'`); err != nil {
		t.Fatal(err)
	}
	seedLayerEsc(t, s, ctx, "T-9002", "层扫定性review-老化:T-9002——相位review 龄9h 复发再报")
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("已诊签名复发应零产: n=%d err=%v", n, err)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature='层扫:review-老化:T-9002'`).Scan(&c); err != nil || c != 1 {
		t.Fatalf("已诊签名不重复起失效: c=%d err=%v", c, err)
	}
	var skip2 string
	if err := s.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='起草' AND note LIKE 'dedup skip signature=层扫:review-老化:T-9002%'
		  AND note LIKE '%已诊票=T-9100%' ORDER BY id DESC LIMIT 1`).Scan(&skip2); err != nil {
		t.Fatalf("已诊跳过行缺: %v", err)
	}
}

// TestLayerScanFailClosed 锚④外延：畸形/异契约行 fail-closed——
// 前缀残缺无 pair/无冒号/空类目空对象/非 MECH 席/非升件事件全弃，
// 零草案零错（签名不明宁漏不猜——benny 同律）。
func TestLayerScanFailClosed(t *testing.T) {
	s, ctx := drafterFixture(t)
	// 畸形 note（有前缀无 pair 分隔合法性）。
	seedLayerEsc(t, s, ctx, "", "层扫定性")       // 空 pair
	seedLayerEsc(t, s, ctx, "", "层扫定性无冒号类目")  // 无 ':'
	seedLayerEsc(t, s, ctx, "", "层扫定性:空类目前缀") // 空类目
	seedLayerEsc(t, s, ctx, "", "层扫定性空对象尾:")  // 空对象
	// 异契约行：非 MECH 席同前缀（伪造面——seat 钉死滤除）。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席','升件','','层扫定性dispatched-无拉:T-9003——伪席')`); err != nil {
		t.Fatal(err)
	}
	// 非升件事件同前缀（digest 行 event='层扫' 不误入）。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('MECH','层扫','层扫','层扫定性deploy-批龄:deploy面——digest 体不误入')`); err != nil {
		t.Fatal(err)
	}
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("fail-closed 应零错: %v", err)
	}
	if n != 0 {
		t.Fatalf("畸形/异契约误起草: n=%d", n)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft`).Scan(&c); err != nil || c != 0 {
		t.Fatalf("fail-closed 破: c=%d err=%v", c, err)
	}
}

// TestLayerScanProbePathIsolation 双径零串味：探针类册升件不产
// 层扫草案（esc 径不变），层扫行不产探针草案（kind=diagnosis 独占）。
func TestLayerScanProbePathIsolation(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedEsc(t, s, ctx, "观测盲", "xvenue-carry", "族沉默 xvenue-carry")
	seedLayerEsc(t, s, ctx, "T-9004", "层扫定性draft-滞留:T-9004——相位draft 龄9d>阈7d")
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if n != 2 {
		t.Fatalf("双径起草数非 2: %d", n)
	}
	var probeKind, layerKind string
	if err := s.pool.QueryRow(ctx, `SELECT kind FROM fin_ticket_draft
		WHERE signature='观测盲:xvenue-carry'`).Scan(&probeKind); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT kind FROM fin_ticket_draft
		WHERE signature='层扫:draft-滞留:T-9004'`).Scan(&layerKind); err != nil {
		t.Fatal(err)
	}
	if probeKind != "观测盲" || layerKind != "diagnosis" {
		t.Fatalf("双径 kind 串味: probe=%s layer=%s", probeKind, layerKind)
	}
}

// TestLayerScanSelfDisposeSkip T-0672 断源①：归领自销类目
// （done-翻漏/幽灵票件/无籍行/done-无收口链）升件不产草案——
// 机械可证类归领当场自销不起票，进水口零号耗。
func TestLayerScanSelfDisposeSkip(t *testing.T) {
	s, ctx := drafterFixture(t)
	for i, cat := range []string{"done-翻漏", "幽灵票件", "无籍行", "done-无收口链"} {
		seedLayerEsc(t, s, ctx, fmt.Sprintf("T-9%03d", i),
			fmt.Sprintf("层扫定性%s:T-9%03d——机械自销类样件", cat, i))
	}
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("自销类应零起草: n=%d err=%v", n, err)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft`).Scan(&c); err != nil || c != 0 {
		t.Fatalf("自销类草案漏产: c=%d err=%v", c, err)
	}
	// 非自销类照常产（对照——断源不误杀语义类）。
	seedLayerEsc(t, s, ctx, "T-9100", "层扫定性draft-滞留:T-9100——相位draft 龄9d")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("语义类应产: n=%d err=%v", n, err)
	}
}

// TestLayerScanClaimedSkip T-0672 断源②：已认领升件不进起草口——
// 后随 note 引「升件#id」处置行=归领已销；未销对照照常产。
func TestLayerScanClaimedSkip(t *testing.T) {
	s, ctx := drafterFixture(t)
	eid := seedLayerEsc(t, s, ctx, "T-9200", "层扫定性draft-滞留:T-9200——相位draft 龄9d")
	seedLayerEsc(t, s, ctx, "T-9201", "层扫定性draft-滞留:T-9201——相位draft 龄9d")
	// 销 T-9200 的升件（处置 note 引升件#id=归领认领径同契）。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','处置','T-9200',$1)`,
		fmt.Sprintf("升件#%d 销——样件处置", eid)); err != nil {
		t.Fatal(err)
	}
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("只未销件应产: n=%d err=%v", n, err)
	}
	var c9200, c9201 int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature='层扫:draft-滞留:T-9200'`).Scan(&c9200); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature='层扫:draft-滞留:T-9201'`).Scan(&c9201); err != nil {
		t.Fatal(err)
	}
	if c9200 != 0 || c9201 != 1 {
		t.Fatalf("认领豁免失效: T-9200草案=%d T-9201草案=%d", c9200, c9201)
	}
}
