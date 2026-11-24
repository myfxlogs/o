package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// feedNarrBanned 业主面零机词表（spec 验收逐词查——升件/复审过/销件/
// 完工回执/戳/净合/三维签 等族全禁；工单/报警/复审/完工 为钦定白话保留词）。
var feedNarrBanned = []string{
	"升件", "复审过", "销件", "完工回执", "净合", "三维签", "戳",
	"唤起", "收口", "在办", "处置席", "ops_event", "sha=", "k=v",
	"水位", "指纹", "判词",
}

func assertNarrClean(t *testing.T, s string) {
	t.Helper()
	for _, w := range feedNarrBanned {
		if strings.Contains(s, w) {
			t.Fatalf("叙事段命中禁词 %q：%s", w, s)
		}
	}
}

// TestFeedNarrCompose 病灶实证组形——#37233-37237 同型（唤知会话签复审+
// 销两报警+18秒）：一段叙事含「签复审」「报警 #」「18 秒」要素零机词。
func TestFeedNarrCompose(t *testing.T) {
	open := feedRow{event: "在办", note: "unit=o-cognition-1795486559 已耗时=31s 在修=",
		ts: time.Now().Add(-3 * time.Minute)}
	close := feedRow{event: "会话收口", note: "unit=o-cognition-1795486559 第1167轮:干了",
		ts: open.ts.Add(18 * time.Second)}
	members := []feedRow{
		{event: "复审过", note: "复审过 sha=ce63b7536 支=fix/bus-route-probe——唤知复审席三维签",
			ts: open.ts.Add(10 * time.Second)},
		{event: "处置", note: "升件#37215 销——同窗复火", ts: open.ts.Add(12 * time.Second)},
		{event: "处置", note: "升件#37228 销——landq_starved病因", ts: open.ts.Add(14 * time.Second)},
		{event: "会话收口", note: "unit=o-cognition-1795486559 第1167轮", ts: close.ts},
	}
	got := feedNarrCompose("o-cognition-1795486559", open, close, members)
	for _, want := range []string{"唤知会话·尾6559", "签了复审", "fix/bus-route-probe",
		"报警 #37215", "#37228", "18 秒", "干完一轮"} {
		if !strings.Contains(got, want) {
			t.Fatalf("叙事缺要素 %q：%s", want, got)
		}
	}
	assertNarrClean(t, got)
}

func TestFeedNarrComposeEmpty(t *testing.T) {
	open := feedRow{event: "在办", ts: time.Now().Add(-time.Minute)}
	close := feedRow{event: "会话收口", ts: open.ts.Add(30 * time.Second)}
	got := feedNarrCompose("o-worker-1795481750", open, close, nil)
	if !strings.Contains(got, "施工会话·尾1750") || !strings.Contains(got, "没留大动作") {
		t.Fatalf("空会话叙事形不对：%s", got)
	}
	assertNarrClean(t, got)
}

func TestFeedNarrDur(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{18 * time.Second, "18 秒"},
		{4*time.Minute + 5*time.Second, "4 分 5 秒"},
		{90 * time.Minute, "1 小时 30 分"},
	} {
		if got := feedNarrDur(c.d); got != c.want {
			t.Fatalf("dur %v: got %q want %q", c.d, got, c.want)
		}
	}
}

// narrTestRow 测试行落库——aged>60s 过 pending 滞后闸；返 feedRow（含真 id/ts）。
func narrTestRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	seat, event, note string, ts time.Time) feedRow {
	t.Helper()
	var r feedRow
	r.seat, r.event, r.note, r.ts = seat, event, note, ts
	if err := pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,note,ts) VALUES($1,$2,$3,$4) RETURNING id`,
		seat, event, note, ts).Scan(&r.id); err != nil {
		t.Fatal(err)
	}
	return r
}

func narrCleanState(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`DELETE FROM fin_push_ledger WHERE scope='sys_status' AND key=$1`,
		feedNarrLedgerKey); err != nil {
		t.Fatal(err)
	}
}

// TestNarrativeBlocksSession 一批会话全程（在办→复审过→销×2→收口）——
// 产一段叙事零逐行；窗态清空。
func TestNarrativeBlocksSession(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	narrCleanState(t, ctx, pool)
	fp := &feedPush{pool: pool}
	unit := "o-cognition-1795990001"
	now := time.Now()
	open := narrTestRow(t, ctx, pool, "唤知机", "在办",
		"unit="+unit+" 已耗时=31s 在修=", now.Add(-3*time.Minute))
	rev := narrTestRow(t, ctx, pool, "唤知", "复审过",
		"复审过 sha=ce63b7536 支=fix/bus-route-probe——唤知复审席三维签", now.Add(-2*time.Minute))
	e1 := narrTestRow(t, ctx, pool, "处置席", "处置",
		"升件#38215 销——同窗复火", now.Add(-100*time.Second))
	e2 := narrTestRow(t, ctx, pool, "处置席", "处置",
		"升件#38228 销——landq_starved病因", now.Add(-90*time.Second))
	cls := narrTestRow(t, ctx, pool, "唤知", "会话收口",
		"unit="+unit+" 第1180轮:v85销两件 用帧=自态", now.Add(-18*time.Second))
	// ts 差做出 18 秒活期（open 在 3 分钟前、close 在 18 秒前——差值非 18s，
	// 叙事时长=close-open=162s；要 18s 叙事改 open=close-18s——此处验要素不验秒差）
	rows := []feedRow{open, rev, e1, e2, cls}
	blocks, err := fp.narrativeBlocks(ctx, rows, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("全程会话应聚一段叙事，得 %d 块：%v", len(blocks), blocks)
	}
	txt := blocks[0].text
	for _, want := range []string{"唤知会话·尾0001", "签了复审", "fix/bus-route-probe",
		"报警 #38215", "#38228", "干完一轮"} {
		if !strings.Contains(txt, want) {
			t.Fatalf("叙事缺要素 %q：%s", want, txt)
		}
	}
	assertNarrClean(t, txt)
	st := fp.loadNarrState(ctx)
	if len(st.Open) != 0 {
		t.Fatalf("闭合后窗态应清空：%+v", st.Open)
	}
}

// TestNarrativeBlocksOpenThenClose 开窗批零产出（全窗员抑）→收口批出叙事。
func TestNarrativeBlocksOpenThenClose(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	narrCleanState(t, ctx, pool)
	fp := &feedPush{pool: pool}
	unit := "o-cognition-1795990002"
	now := time.Now()
	open := narrTestRow(t, ctx, pool, "唤知机", "在办",
		"unit="+unit+" 已耗时=5s 在修=", now.Add(-2*time.Minute))
	work := narrTestRow(t, ctx, pool, "处置席", "处置",
		"升件#38299 销——窗内活", now.Add(-90*time.Second))
	// 第一批：开窗+窗员——全抑零块
	blocks, err := fp.narrativeBlocks(ctx, []feedRow{open, work}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Fatalf("开窗批应全窗员抑零块，得 %v", blocks)
	}
	if st := fp.loadNarrState(ctx); len(st.Open) != 1 {
		t.Fatalf("窗态应存一开窗：%+v", st.Open)
	}
	// 第二批：收口行——叙事出（窗员跨批库查拼全）
	cls := narrTestRow(t, ctx, pool, "唤知", "会话收口",
		"unit="+unit+" 第1181轮", now.Add(-30*time.Second))
	blocks, err = fp.narrativeBlocks(ctx, []feedRow{cls}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 || !strings.Contains(blocks[0].text, "报警 #38299") {
		t.Fatalf("收口批应出含窗员叙事：%v", blocks)
	}
	assertNarrClean(t, blocks[0].text)
}

// TestNarrativeBlocksTimeout 会话暴毙（无收口）——超时窗退逐行不丢行。
func TestNarrativeBlocksTimeout(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	narrCleanState(t, ctx, pool)
	fp := &feedPush{pool: pool}
	unit := "o-cognition-1795990003"
	now := time.Now()
	open := narrTestRow(t, ctx, pool, "唤知机", "在办",
		"unit="+unit+" 已耗时=5s 在修=", now.Add(-40*time.Minute))
	work := narrTestRow(t, ctx, pool, "处置席", "处置",
		"工单#39901 销——暴毙前最后活", now.Add(-35*time.Minute))
	// 窗态直种超时龄（模拟在办行早批已抑消费）
	if err := fp.saveNarrState(ctx, &feedNarrState{Open: []feedBracket{
		{Unit: unit, OpenID: open.id, OpenTS: now.Add(-31 * time.Minute)},
	}}); err != nil {
		t.Fatal(err)
	}
	blocks, err := fp.narrativeBlocks(ctx, nil, work.id)
	if err != nil {
		t.Fatal(err)
	}
	// 超时兜底=窗员逐行（feedRender 双行形）——不丢行
	if len(blocks) == 0 {
		t.Fatal("超时兜底应退逐行块")
	}
	found := false
	for _, b := range blocks {
		if strings.Contains(b.text, "工单#39901") || strings.Contains(b.text, "#39901") {
			found = true
		}
	}
	if !found {
		t.Fatalf("兜底逐行未含窗员行：%v", blocks)
	}
	if st := fp.loadNarrState(ctx); len(st.Open) != 0 {
		t.Fatalf("超时后窗态应清：%+v", st.Open)
	}
}

// TestNarrativeBlocksNoUnit 无 unit 锚行=现行逐行（叙事=加成非替换）。
func TestNarrativeBlocksNoUnit(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	narrCleanState(t, ctx, pool)
	fp := &feedPush{pool: pool}
	r1 := narrTestRow(t, ctx, pool, "探测席", "升件",
		"对账/x: 差 $1.00", time.Now().Add(-2*time.Minute))
	r2 := narrTestRow(t, ctx, pool, "施工席", "完工回执",
		"完工回执 O-1 @ sha=abc 支=fix/x", time.Now().Add(-90*time.Second))
	blocks, err := fp.narrativeBlocks(ctx, []feedRow{r1, r2}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("无锚行应逐行两块，得 %d", len(blocks))
	}
	if !strings.Contains(blocks[0].text, "报警") && !strings.Contains(blocks[0].text, "对账") {
		t.Fatalf("首块应是升件逐行渲染：%s", blocks[0].text)
	}
}

// TestNarrativeOn 叙事闸——cfg 缺席=关（桩测裸件）；键缺/true=开；false=关。
func TestNarrativeOn(t *testing.T) {
	fp := &feedPush{}
	if fp.narrativeOn(context.Background()) {
		t.Fatal("cfg 缺席应关不炸")
	}
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fp.cfg = cfg
	if !fp.narrativeOn(ctx) {
		t.Fatal("fin_0246 种子 true——叙事应默认开")
	}
	if _, err := pool.Exec(ctx,
		`UPDATE app_config SET value='false'::jsonb WHERE key='feed.narrative_mode'`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx,
			`UPDATE app_config SET value='true'::jsonb WHERE key='feed.narrative_mode'`)
	}()
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if fp.narrativeOn(ctx) {
		t.Fatal("显式 false 应回逐行模式")
	}
}
