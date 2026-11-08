package main

// T-0458 锚 e：dm 帧→Watch 收到→重渲调用断言——进程内 connect 服务端
// （httptest+伪 EventBusHandler 喂帧：ctx 帧不触发/心跳不触发/dm 帧触发），
// 真协议面过 watchOnce→onDM 合流→cards.Refresh 真环（伪 Store 断言
// SeatRows 被调=重渲跑通）。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"o/internal/gen/eventv1"
	"o/internal/gen/eventv1/eventv1connect"
	"o/internal/ops/seatcard"
)

// ---------- 伪服务端（帧序喂完挂起——测试 cancel 收敛） ----------

type fakeSvc struct{ frames []*eventv1.DataAdvanced }

func (s *fakeSvc) Watch(ctx context.Context, _ *eventv1.WatchRequest,
	st eventv1connect.EventBusWatchServerStream) error {
	for _, fr := range s.frames {
		if err := st.Send(fr); err != nil {
			return err
		}
	}
	<-ctx.Done() // 帧尽挂起——Receive 侧语义同 real bus（帧尽等待非断流）
	return ctx.Err()
}

// ---------- 伪 Store（重渲调用断言面——全接口桩） ----------

type countingStore struct {
	mu            sync.Mutex
	seatRowsCalls int
}

func (c *countingStore) SeatRows(context.Context, []string, time.Time) ([]seatcard.EventRow, error) {
	c.mu.Lock()
	c.seatRowsCalls++
	c.mu.Unlock()
	return nil, nil
}

func (c *countingStore) QueueDepth(context.Context) (int, int, error) { return 0, 0, nil }

func (c *countingStore) CardRow(context.Context, string) (string, string, string, bool, error) {
	return "", "", "", false, nil
}

func (c *countingStore) SaveCard(context.Context, string, string, string, string) error { return nil }
func (c *countingStore) SaveHash(context.Context, string, string) error                 { return nil }
func (c *countingStore) RecordErr(context.Context, string, string) error                { return nil }

// ---------- 锚 e ----------

func TestWatchDMFrameTriggersRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 进程内 eventbus（真 connect 协议面——客户端径同产线 watchBus）。
	server := connect.NewServer()
	eventv1connect.RegisterEventBusHandler(server, &fakeSvc{frames: []*eventv1.DataAdvanced{
		{Source: "d", Kind: "ctx", MaxId: 1},   // 非 dm 帧——不触发
		{Source: "d", Kind: "dm", MaxId: 2},    // dm 帧——触发重渲
		{Source: "__hb__", Kind: "", MaxId: 0}, // 心跳——不触发
	}})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cli := eventv1connect.NewEventBusClient(connect.NewClient(
		connecthttp.NewTransport(http.DefaultClient, srv.URL)))

	st := &countingStore{}
	cards := seatcard.NewCards(seatcard.Config{
		Seats: seatcard.Phase1Seats,
		Live:  func() bool { return false }, // 桩测 dry-run——渲染不出站
		Probe: func(context.Context, []string) (seatcard.UnitSnap, error) {
			return seatcard.UnitSnap{}, nil // 伪探针——systemctl 不进桩测
		},
		Now: func() time.Time { return time.Date(2026, 11, 8, 10, 30, 15, 0, time.Local) },
	}, st)

	trig := make(chan struct{}, 1)
	go func() { // 刷新环（cmd main 同构——帧触发→Refresh 单线程串行）
		for {
			select {
			case <-trig:
				cards.Refresh(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
	onDM := func() {
		select {
		case trig <- struct{}{}:
		default:
		}
	}

	done := make(chan error, 1)
	go func() { done <- watchOnce(ctx, cli, map[string]int64{}, onDM) }()

	// dm 帧到→重渲调用断言（SeatRows 被调——Assemble 每席一行读，
	// 一拍=Phase1Seats 三席三调用）。
	deadline := time.Now().Add(2 * time.Second)
	for {
		st.mu.Lock()
		n := st.seatRowsCalls
		st.mu.Unlock()
		if n >= len(seatcard.Phase1Seats) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("dm 帧未触发重渲（锚e 破）")
		}
		time.Sleep(5 * time.Millisecond) // sched-exempt-ok 桩测期界等待（伪帧到位轮询）
	}

	// 帧尽挂起位——cancel 收敛（watchOnce 解阻塞返回）。
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测收敛上界（期界等待）
		t.Fatal("watchOnce 未随 ctx 收敛")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.seatRowsCalls != len(seatcard.Phase1Seats) {
		t.Fatalf("重渲调用数=%d 期 %d（恰一拍）——非 dm 帧（ctx/心跳）不得触发",
			st.seatRowsCalls, len(seatcard.Phase1Seats))
	}
}
