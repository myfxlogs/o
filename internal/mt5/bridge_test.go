// bridge_test.go —— 桥腿桩测（T-0141）：共享会话纯消费腿契约面——
// mt5.session_token 共享行读写/复挂/死会话退避；symbols 键域缺/空=不虚构标的集。
// 凭据径测试随 2026-11-15 共享会话接管退役（腿不再消费 bridge.mt5.* 凭据）。
package mt5

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// stubConnM ConnectionClient 测试桩——CheckConnect 按 alive 回执，
// connectCalls 记 Connect/ConnectEx 被调次数（消费腿禁自开 login 的断言面）。
type stubConnM struct {
	mt5pb.ConnectionClient
	alive        bool
	connectCalls int
}

func (s *stubConnM) CheckConnect(ctx context.Context, r *mt5pb.CheckConnectRequest, opts ...grpc.CallOption) (*mt5pb.CheckConnectReply, error) {
	if s.alive {
		return &mt5pb.CheckConnectReply{Result: "ok"}, nil
	}
	return &mt5pb.CheckConnectReply{Error: &mt5pb.Error{Message: "session dead"}}, nil
}

func (s *stubConnM) Connect(ctx context.Context, r *mt5pb.ConnectRequest, opts ...grpc.CallOption) (*mt5pb.ConnectReply, error) {
	s.connectCalls++
	return &mt5pb.ConnectReply{Error: &mt5pb.Error{Message: "stub 禁真 login"}}, nil
}

func (s *stubConnM) ConnectEx(ctx context.Context, r *mt5pb.ConnectExRequest, opts ...grpc.CallOption) (*mt5pb.ConnectExReply, error) {
	s.connectCalls++
	return &mt5pb.ConnectExReply{Error: &mt5pb.Error{Message: "stub 禁真 login"}}, nil
}

// TestSharedSessionRound 共享行读写锚——owner 落行=消费腿原值读回
// （addr+token+ts 三域齐）。
func TestSharedSessionRound(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if _, err := sharedSessionRead(ctx, pool); err == nil {
		t.Fatal("行缺应 err（消费腿 fail-closed 等 owner）")
	}
	sharedSessionWrite(ctx, pool, "mt5grpc2.mtapi.io:443", "tok-abc")
	ss, err := sharedSessionRead(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if ss.Addr != "mt5grpc2.mtapi.io:443" || ss.Token != "tok-abc" || ss.Ts == "" {
		t.Fatalf("行载荷错: %+v", ss)
	}
	// 幂等覆写（owner 重建换 token——消费腿读新值）。
	sharedSessionWrite(ctx, pool, "mt5grpc3.mtapi.io:443", "tok-xyz")
	ss, _ = sharedSessionRead(ctx, pool)
	if ss.Addr != "mt5grpc3.mtapi.io:443" || ss.Token != "tok-xyz" {
		t.Fatalf("换血覆写未生效: %+v", ss)
	}
}

// TestSharedAttach 消费腿复挂锚——行缺=err；活 token=shared 标置 client 返；
// 死 token=err 且 conn 关闭。全径零 Connect/ConnectEx（owner 独占 login 权）。
func TestSharedAttach(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	stub := &stubConnM{alive: true}
	dial := func(addr string) (*Client, error) {
		if addr != "mt5grpc2.mtapi.io:443" {
			t.Fatalf("应拨共享行 addr got %s", addr)
		}
		return &Client{connM: stub}, nil
	}
	if _, err := sharedAttach(ctx, pool, dial); err == nil ||
		!strings.Contains(err.Error(), "共享会话未立") {
		t.Fatalf("行缺应返等汇市巡 err got %v", err)
	}
	sharedSessionWrite(ctx, pool, "mt5grpc2.mtapi.io:443", "tok-live")
	cl, err := sharedAttach(ctx, pool, dial)
	if err != nil {
		t.Fatal(err)
	}
	if !cl.shared || cl.Token() != "tok-live" {
		t.Fatalf("复挂 client shared=%v token=%q", cl.shared, cl.Token())
	}
	stub.alive = false
	if _, err := sharedAttach(ctx, pool, dial); err == nil ||
		!strings.Contains(err.Error(), "共享会话已死") {
		t.Fatalf("死 token 应返等重建 err got %v", err)
	}
	if stub.connectCalls != 0 {
		t.Fatalf("消费腿触 login %d 次——owner 独占权被破", stub.connectCalls)
	}
}

// TestSharedDisconnectNoop 共享会话登出幂等锚——shared client Disconnect
// 空转（token 不清零不触 connM——连坐杀会话坑防面）；非 shared 对照径
// 保持原语义（空 token 短路）。
func TestSharedDisconnectNoop(t *testing.T) {
	cl := &Client{}
	cl.SetSharedToken("tok-sh")
	cl.Disconnect(context.Background())
	if cl.Token() != "tok-sh" {
		t.Fatal("共享会话 Disconnect 应空转——token 被清=连坐杀会话门开")
	}
	var plain Client
	plain.Disconnect(context.Background()) // 非 shared 空 token 短路不炸
}

// 锚：symbols 键域热启——缺=空集（bridgeSession fail-closed 返错不订阅）。
func TestBridgeSymbolsFailClosed(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "test")}
	if syms := c.symbols(); len(syms) != 0 {
		t.Fatalf("键缺应空集 got %v", syms)
	}
}
