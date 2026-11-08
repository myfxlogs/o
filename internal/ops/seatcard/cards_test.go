package seatcard

// T-0458 桩测五锚（CTO 裁定⑤——check_all 内机判；伪缝注入：notify/pgstore/
// systemctl 全不进桩测）。本文件锚 b/c/d：事件→重渲→哈希变→PATCH、懒建闸、
// 卡败径。锚 a=render_test.go（哈希同态）；锚 e=cmd/seatcard（dm 帧→Watch）。

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------- 伪缝 ----------

type cardRec struct{ mid, chat, hash string }

type memStore struct {
	rows    map[string][]EventRow // alias→rows
	revApp  [2]int                // [review, approved]
	cards   map[string]cardRec    // card 席名→行
	errs    []string              // seatcard_err 落账面
	rowsErr error
}

func (m *memStore) SeatRows(_ context.Context, seats []string, since time.Time) ([]EventRow, error) {
	if m.rowsErr != nil {
		return nil, m.rowsErr
	}
	var out []EventRow
	for _, a := range seats {
		for _, r := range m.rows[a] {
			if !r.Ts.Before(since) {
				out = append(out, r)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	return out, nil
}

func (m *memStore) QueueDepth(context.Context) (int, int, error) {
	return m.revApp[0], m.revApp[1], nil
}

func (m *memStore) CardRow(_ context.Context, seat string) (string, string, string, bool, error) {
	c, ok := m.cards[seat]
	return c.mid, c.chat, c.hash, ok, nil
}

func (m *memStore) SaveCard(_ context.Context, seat, mid, chat, hash string) error {
	m.cards[seat] = cardRec{mid: mid, chat: chat, hash: hash}
	return nil
}

func (m *memStore) SaveHash(_ context.Context, seat, hash string) error {
	c := m.cards[seat]
	c.hash = hash
	m.cards[seat] = c
	return nil
}

func (m *memStore) RecordErr(_ context.Context, seat, note string) error {
	m.errs = append(m.errs, seat+"|"+note)
	return nil
}

// base 定针（卡面截分显示——错开整分防分钟窗翻转抖动）。
var base = time.Date(2026, 11, 8, 10, 30, 15, 0, time.Local)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func testProbe(active bool) ProbeUnits {
	return func(_ context.Context, patterns []string) (UnitSnap, error) {
		snap := UnitSnap{Active: active}
		for _, p := range patterns {
			if p == "o-worker-*" && active {
				snap.Workers = []Worker{{Unit: "o-worker-T-0001.service", TID: "T-0001",
					Model: "swe-2-high", Since: base.Add(-12 * time.Minute)}}
			}
		}
		return snap, nil
	}
}

// oneSeat 只留 CTO席 活（其余席零行——懒建闸挡出渲染面）。
func oneSeatStore() *memStore {
	return &memStore{
		rows: map[string][]EventRow{
			"CTO": {{Seat: "CTO", Ts: base.Add(-5 * time.Minute), Event: "领单", Ticket: "T-0458", Note: "开工"}},
		},
		cards: map[string]cardRec{},
	}
}

func newTestCards(st *memStore, live bool, probe ProbeUnits) (*Cards, *[]string) {
	var mu sync.Mutex
	patches := []string{}
	cfg := Config{
		Seats: Phase1Seats,
		Live:  func() bool { return live },
		Create: func(_ context.Context, chatID, content string) (string, error) {
			mu.Lock()
			patches = append(patches, "create|"+chatID+"|"+content)
			mu.Unlock()
			return "new-mid", nil
		},
		Patch: func(_ context.Context, messageID, content string) error {
			mu.Lock()
			patches = append(patches, "patch|"+messageID+"|"+content)
			mu.Unlock()
			return nil
		},
		Probe: probe, ChatID: "chat-x",
		MinPatchGap: time.Millisecond, // 桩测定针—— gap 专测另配
		Now:         fixedClock(base),
	}
	return NewCards(cfg, st), &patches
}

// ---------- 锚 b：事件→重渲→哈希变→PATCH 断言（fake lark 注伪缝） ----------

func TestRefreshHashGatePatch(t *testing.T) {
	st := oneSeatStore()
	st.cards["CTO席"] = cardRec{mid: "m1", chat: "c1"}
	clock := base
	c, patches := newTestCards(st, true, testProbe(true))
	c.cfg.Now = func() time.Time { return clock }

	clock = base.Add(time.Second)
	c.Refresh(context.Background()) // 首拍：内容变→PATCH
	if n := len(*patches); n != 1 {
		t.Fatalf("首拍 PATCH 数=%d 期 1（锚b：事件→重渲→哈希变→PATCH）", n)
	}
	if !strings.Contains((*patches)[0], "T-0458") {
		t.Fatalf("PATCH 内容缺在办票号: %s", (*patches)[0])
	}

	clock = base.Add(2 * time.Second)
	c.Refresh(context.Background()) // 同态重渲：哈希闸挡——零 PATCH
	if n := len(*patches); n != 1 {
		t.Fatalf("同态拍 PATCH 数=%d 期 1（锚b：哈希同态零 PATCH）", n)
	}

	// 事件落账（新末动行）→重渲内容变→PATCH 再发。
	st.rows["CTO"] = append(st.rows["CTO"],
		EventRow{Seat: "CTO", Ts: base.Add(time.Minute), Event: "收口", Ticket: "T-0458", Note: "收口移交"})
	clock = base.Add(3 * time.Second)
	c.Refresh(context.Background())
	if n := len(*patches); n != 2 {
		t.Fatalf("事件后 PATCH 数=%d 期 2（锚b：哈希变→PATCH）", n)
	}
	if !strings.Contains((*patches)[1], "收口") {
		t.Fatalf("二次 PATCH 内容缺新末动: %s", (*patches)[1])
	}
}

// 锚 b 补面：PATCH 下限闸——窗内事件塌帧（内容变但不 PATCH），窗后一拍
// 重渲最新内容再发（自然合并刷帧）。
func TestRefreshMinPatchGap(t *testing.T) {
	st := oneSeatStore()
	st.cards["CTO席"] = cardRec{mid: "m1", chat: "c1"}
	clock := base
	cfg := Config{
		Seats: Phase1Seats, Live: func() bool { return true },
		Patch: func(_ context.Context, _ string, _ string) error { return nil },
		Create: func(context.Context, string, string) (string, error) {
			return "mid", nil
		},
		Probe: testProbe(true), ChatID: "chat-x",
		MinPatchGap: 5 * time.Second, Now: func() time.Time { return clock },
	}
	c := NewCards(cfg, st)
	ctx := context.Background()
	c.Refresh(ctx) // t0 PATCH#1
	c.mu.Lock()
	h1 := c.lastHash["CTO席"]
	c.mu.Unlock()
	if h1 == "" {
		t.Fatal("首拍未落哈希")
	}

	st.rows["CTO"] = append(st.rows["CTO"],
		EventRow{Seat: "CTO", Ts: base.Add(time.Second), Event: "裁决", Ticket: "T-0458", Note: "通过"})
	clock = base.Add(time.Second) // t0+1s 窗内
	c.Refresh(ctx)                // 内容变但窗内——塌帧不 PATCH 不落哈希
	c.mu.Lock()
	h2 := c.lastHash["CTO席"]
	c.mu.Unlock()
	if h2 != h1 {
		t.Fatal("窗内塌帧哈希不应更新——应留待窗后重试")
	}

	clock = base.Add(6 * time.Second) // t0+6s 窗外
	c.Refresh(ctx)
	c.mu.Lock()
	h3 := c.lastHash["CTO席"]
	c.mu.Unlock()
	if h3 == "" || h3 == h1 {
		t.Fatalf("窗外拍应 PATCH 最新内容并落哈希: before=%q after=%q", h1, h3)
	}
}

// ---------- 锚 c：懒建闸——零活性席不补卡；活性席无卡补建 ----------

func TestLazyCreateGate(t *testing.T) {
	st := oneSeatStore() // CTO 活性（5min 前），施工/CEO 零行
	c, patches := newTestCards(st, true, testProbe(false))
	c.Refresh(context.Background())

	creates := 0
	for _, p := range *patches {
		if strings.HasPrefix(p, "create|") {
			creates++
		}
	}
	if creates != 1 {
		t.Fatalf("懒建 create 数=%d 期 1（活性席补建+零活性席不补）", creates)
	}
	if rec, ok := st.cards["CTO席"]; !ok || rec.mid != "new-mid" {
		t.Fatalf("卡册行未落 message_id: %+v ok=%v", rec, ok)
	}

	// 零活性席（最新行 49h 前）——不补卡（懒建闸：终态/零活性席不补）。
	stOld := oneSeatStore()
	stOld.rows["CTO"] = []EventRow{{Seat: "CTO", Ts: base.Add(-49 * time.Hour), Event: "退班"}}
	c2, patches2 := newTestCards(stOld, true, testProbe(false))
	c2.Refresh(context.Background())
	if n := len(*patches2); n != 0 {
		t.Fatalf("零活性席出站数=%d 期 0（懒建闸）", n)
	}

	// 补建幂等：卡行已落→次拍零 create（create 总数仍 1）。
	c.Refresh(context.Background())
	creates = 0
	for _, p := range *patches {
		if strings.HasPrefix(p, "create|") {
			creates++
		}
	}
	if creates != 1 {
		t.Fatalf("次拍 create 总数=%d 期 1（幂等破）", creates)
	}
}

// ---------- 锚 d：卡败径——lark 败→seatcard_err 行落账+订阅环不死 ----------

func TestCardFailPath(t *testing.T) {
	st := oneSeatStore()
	st.cards["CTO席"] = cardRec{mid: "m1", chat: "c1"}
	fail := true
	var mu sync.Mutex
	cfg := Config{
		Seats: Phase1Seats, Live: func() bool { return true },
		Patch: func(_ context.Context, _, _ string) error {
			mu.Lock()
			defer mu.Unlock()
			if fail {
				return errors.New("lark down")
			}
			return nil
		},
		Create: func(context.Context, string, string) (string, error) {
			return "mid", nil
		},
		Probe: testProbe(true), ChatID: "chat-x",
		MinPatchGap: time.Millisecond, Now: fixedClock(base),
	}
	c := NewCards(cfg, st)
	ctx := context.Background()

	c.Refresh(ctx) // PATCH 败→seatcard_err 落账
	if len(st.errs) != 1 || !strings.Contains(st.errs[0], "卡 PATCH 败") {
		t.Fatalf("卡败未落账: %v", st.errs)
	}

	c.Refresh(ctx) // 环不死：次拍照渲（ErrGap 节流——落账不重复）
	if len(st.errs) != 1 {
		t.Fatalf("seatcard_err 未节流: %v", st.errs)
	}

	fail = false
	c.Refresh(ctx) // 恢复→重试成功（败不落哈希=下拍重试）
	c.mu.Lock()
	hash := c.lastHash["CTO席"]
	c.mu.Unlock()
	if hash == "" {
		t.Fatal("恢复后未 PATCH 成功（环死或哈希误落）")
	}
}

// 创建败径：活性席无卡+create 败→落账+不落行（下拍重试）；他席照渲
// （环不死）。
func TestCreateFailPath(t *testing.T) {
	st := oneSeatStore()
	st.rows["CEO"] = []EventRow{{Seat: "CEO", Ts: base.Add(-time.Minute), Event: "派单", Ticket: "T-0461"}}
	calls := 0
	cfg := Config{
		Seats: Phase1Seats, Live: func() bool { return true },
		Create: func(context.Context, string, string) (string, error) {
			calls++
			return "", errors.New("lark create down")
		},
		Probe: testProbe(true), ChatID: "chat-x",
		MinPatchGap: time.Millisecond, Now: fixedClock(base),
	}
	c := NewCards(cfg, st)
	c.Refresh(context.Background())
	if calls < 2 {
		t.Fatalf("create 调用数=%d——两活性席都应尝试（环不死）", calls)
	}
	if len(st.cards) != 0 {
		t.Fatalf("create 败不应落卡行: %+v", st.cards)
	}
	if len(st.errs) < 2 {
		t.Fatalf("create 败应逐席落账: %v", st.errs)
	}
}

// 装配败径：SeatRows 查败→seatcard_err 落账+Refresh 不 panic（环不死）。
func TestAssembleFailPath(t *testing.T) {
	st := oneSeatStore()
	st.rowsErr = errors.New("pg down")
	c, _ := newTestCards(st, true, testProbe(true))
	c.Refresh(context.Background()) // 不 panic 即过——败面落账
	if len(st.errs) != 1 || !strings.Contains(st.errs[0], "装配败") {
		t.Fatalf("装配败未落账: %v", st.errs)
	}
}
