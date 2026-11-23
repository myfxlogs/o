// session_deadstream_test.go —— O-34264 死流看门狗桩测：流开着零数据帧
// →quoteConsume 在 deadThresh 内退出返 deadstream 判词（Recv 阻塞不返
// err 的死流态 A/B 收尸径）；心跳帧持续但零数据帧同断言（f.Result==nil
// 不续命）；实报价帧续命后静默仍按末帧龄判死。
package mt5

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fakeQuoteStream OnQuote 流桩——mode: "block"=Recv 阻塞至 ctx 取消返
// ctx err（死流态 A：后端会话死零帧，TCP/流活性探不出）；"heartbeat"=
// 持续返 Result==nil 空帧（死流态 B：空心跳照到数据帧已死）；
// "data-then-silent"=首帧真报价后转阻塞（活性计时=末数据帧龄锚）。
// ctx 由 runConsume 注入 sessCtx——看门狗 cancel 同径解锁 Recv。
type fakeQuoteStream struct {
	ctx      context.Context
	mode     string
	dataSent bool
}

func (f *fakeQuoteStream) Recv() (*mt5pb.OnQuoteReply, error) {
	if f.mode == "heartbeat" {
		select { // 帧间隔模拟 venue 节拍+尊 ctx——gRPC Recv 语义：
		// sessCtx 取消即返 ctx err（看门狗收尸径同构）
		case <-f.ctx.Done():
			return nil, f.ctx.Err()
		case <-time.After(5 * time.Millisecond): // sched-exempt-ok 测试桩 venue 帧节拍模拟（非生产调度原语）
		}
		return &mt5pb.OnQuoteReply{}, nil
	}
	if f.mode == "data-then-silent" && !f.dataSent {
		f.dataSent = true
		return &mt5pb.OnQuoteReply{Result: &mt5pb.Quote{
			Symbol: "USDJPY", Bid: 150.1, Ask: 150.2, Last: 150.15}}, nil
	}
	<-f.ctx.Done()
	return nil, f.ctx.Err()
}

func (f *fakeQuoteStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeQuoteStream) Trailer() metadata.MD         { return nil }
func (f *fakeQuoteStream) CloseSend() error             { return nil }
func (f *fakeQuoteStream) Context() context.Context     { return f.ctx }
func (f *fakeQuoteStream) SendMsg(any) error            { return nil }
func (f *fakeQuoteStream) RecvMsg(any) error            { return nil }

func newDeadwatchCollector(t *testing.T) *Collector {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "md"); err != nil {
		t.Fatal(err)
	}
	return &Collector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5", "test")}
}

// runConsume quoteConsume 直驱——sessCtx 独立取消面（看门狗 cancel 同径，
// profitErr 空席=无 profit 流死亡干扰归因）。
func runConsume(c *Collector, fs *fakeQuoteStream, dead, every time.Duration) (time.Duration, error) {
	ctx := context.Background()
	sessCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	fs.ctx = sessCtx // 桩 Recv 绑同一取消面——看门狗 cancel 即解锁
	profitErr := make(chan error, 1)
	start := time.Now()
	err := c.quoteConsume(ctx, sessCtx, cancel, fs, profitErr, dead, every)
	return time.Since(start), err
}

// 案①死流态 A：流开着 Recv 永阻塞零帧——deadThresh 内退出+deadstream 判词。
func TestDeadstreamZeroFrames(t *testing.T) {
	c := newDeadwatchCollector(t)
	fs := &fakeQuoteStream{mode: "block"}
	el, err := runConsume(c, fs, 120*time.Millisecond, 20*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "deadstream") {
		t.Fatalf("零帧死流应 deadstream 判词退出 got err=%v", err)
	}
	if el > 2*time.Second {
		t.Fatalf("退出超 deadThresh 量级: %.0fms", float64(el.Milliseconds()))
	}
}

// 案②死流态 B：心跳帧持续到达但零数据帧——心跳不续命同断言。
func TestDeadstreamHeartbeatNoData(t *testing.T) {
	c := newDeadwatchCollector(t)
	fs := &fakeQuoteStream{mode: "heartbeat"}
	el, err := runConsume(c, fs, 120*time.Millisecond, 20*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "deadstream") {
		t.Fatalf("心跳假活应 deadstream 判词退出 got err=%v", err)
	}
	if el > 2*time.Second {
		t.Fatalf("心跳不续命断言败——退出耗时 %.0fms", float64(el.Milliseconds()))
	}
}

// 案③活性锚：首帧真报价后转静默——看门狗按末数据帧龄判死（lastData
// 刷新语义实证：不是按会话起点死，是按末帧死）。
func TestDeadstreamAfterDataSilence(t *testing.T) {
	c := newDeadwatchCollector(t)
	fs := &fakeQuoteStream{mode: "data-then-silent"}
	el, err := runConsume(c, fs, 120*time.Millisecond, 20*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "deadstream") {
		t.Fatalf("数据帧后静默应 deadstream 判词退出 got err=%v", err)
	}
	if el < 100*time.Millisecond {
		t.Fatalf("活性帧未计入——过早判死 %.0fms", float64(el.Milliseconds()))
	}
}
