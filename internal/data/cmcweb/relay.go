// relay.go —— 本地 WS 中继：冒充 connectapi CMC WS 端点，把浏览器旁录的真 app
// 帧喂给 o-core 现有 WsClient（auth/subscribe/heartbeat/续期四种控制帧本地代答，
// 数据帧原样透传）。上游 app WS 死 → 掐客户端连（如实反映上游死态，客户端
// 退避重连+REST seed 兜底按原路径工作）。
package cmcweb

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	relayWriteTimeout = 10 * time.Second
	relayFrameBuf     = 1024 // 旁录帧环形缓冲容量（满丢最旧——客户端重连种子吃最新帧即可）
	relayQueueBuf     = 256  // 单客户端出站队列（慢即掐——宁断不堆积）
)

// Relay 本地 WS 中继。frames 由浏览器捕获件投递；客户端连接=下游消费者。
type Relay struct {
	mu        sync.Mutex
	clients   map[*relayClient]struct{}
	buf       [][]byte  // 环形缓冲（新连客户端回灌用）
	upAlive   bool      // 上游 app WS 活态——死则掐全客户端
	upDeadAt  time.Time // 上游死态起点（healthz 死窗判——转移日志面同源）
	watchBeat time.Time // watchdog 拍心跳（假活判——拍龄超阈=看门狗楔死）
	bootAt    time.Time // 进程起点（boot 宽限判）
	addr      string
	srv       *http.Server
	browser   *Browser // REST 代理页内 fetch 出口（BindBrowser 接线）
}

// BindBrowser REST 代理的浏览器出口（构造序:NewRelay→NewBrowser→Bind）。
func (r *Relay) BindBrowser(b *Browser) { r.browser = b }

type relayClient struct {
	conn *websocket.Conn
	out  chan []byte
	done chan struct{}
}

var relayUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true }, // 仅本机/docker 内网消费
}

func NewRelay(addr string) *Relay {
	return &Relay{addr: addr, clients: make(map[*relayClient]struct{}), bootAt: time.Now()}
}

// Feed 投递一帧旁录数据（浏览器捕获侧调；满则丢最旧）。
func (r *Relay) Feed(payload string) {
	b := []byte(payload)
	r.mu.Lock()
	if len(r.buf) >= relayFrameBuf {
		r.buf = r.buf[1:]
	}
	r.buf = append(r.buf, b)
	for c := range r.clients {
		select {
		case c.out <- b:
		default: // 慢客户端——掐
			r.dropLocked(c)
		}
	}
	r.mu.Unlock()
}

// SetUpstreamAlive 上游 app WS 活态转换——活→死掐全客户端（如实反映，触发客户端
// 退避重连；新 app WS 起后客户端回灌环形缓冲拿种子帧）。
func (r *Relay) SetUpstreamAlive(alive bool) {
	r.mu.Lock()
	if r.upAlive == alive {
		r.mu.Unlock()
		return
	}
	r.upAlive = alive
	if alive {
		r.upDeadAt = time.Time{}
	} else {
		r.upDeadAt = time.Now()
	}
	log.Printf("[cmcweb-relay] upstream alive=%v clients=%d", alive, len(r.clients))
	if !alive {
		for c := range r.clients {
			r.dropLocked(c)
		}
	}
	r.mu.Unlock()
}

// SetWatchdogBeat watchdog 拍心跳（Service.watchdog 每拍喂——healthz 假活判源：
// 拍沉默=看门狗 goroutine 楔死,11-18 产线 8h 零日志实证）。
func (r *Relay) SetWatchdogBeat() {
	r.mu.Lock()
	r.watchBeat = time.Now()
	r.mu.Unlock()
}

func (r *Relay) dropLocked(c *relayClient) {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	delete(r.clients, c)
	c.conn.Close()
}

// ListenAndServe 起 HTTP+WS 服务（阻塞，ctx 取消即退）。
func (r *Relay) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", r.serveHealthz)
	mux.HandleFunc("/api/", r.apiProxy(r.browser)) // REST 页内 fetch 代理——单会话制 Go 零会话
	mux.HandleFunc("/", r.serveWS)
	r.srv = &http.Server{Addr: r.addr, Handler: mux}
	ln, err := net.Listen("tcp", r.addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		_ = r.srv.Close()
	}()
	log.Printf("[cmcweb-relay] listen %s", r.addr)
	return r.srv.Serve(ln)
}

// serveHealthz 健康面——双死判非恒活（11-18 假活窗 8h 实证：upstream=false
// 照报 200=docker healthy 假绿）：
//
//	① watchdog 拍龄>90s（boot 后从未拍过也算）=看门狗 goroutine 楔死——503
//	② 上游死窗>15min（deadRestartMin+余量——看门狗该自愈而未愈）=503
//
// boot 宽限 2min——会话建立期 upstream 本就死、watchdog 未开拍。
func (r *Relay) serveHealthz(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	up, n := r.upAlive, len(r.clients)
	deadAt, beat, boot := r.upDeadAt, r.watchBeat, r.bootAt
	r.mu.Unlock()
	age := time.Since(boot)
	beatAge := time.Since(beat)
	var deadWin time.Duration
	if !deadAt.IsZero() {
		deadWin = time.Since(deadAt)
	}
	sick := ""
	switch {
	case age > 2*time.Minute && (beat.IsZero() || beatAge > 90*time.Second):
		sick = fmt.Sprintf("watchdog 拍龄 %.0fs 超阈（goroutine 楔死疑）", beatAge.Seconds())
	case age > 2*time.Minute && deadWin > 15*time.Minute:
		sick = fmt.Sprintf("upstream 死窗 %.0fm 超阈（看门狗自愈未达）", deadWin.Minutes())
	}
	status := http.StatusOK
	word := "ok"
	if sick != "" {
		status = http.StatusServiceUnavailable
		word = "sick"
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf("%s upstream=%v dead=%.0fs beat=%.0fs clients=%d %s\n",
		word, up, deadWin.Seconds(), beatAge.Seconds(), n, sick)))
}

// serveWS 升级 + 控制帧应答循环 + 数据帧泵。上游死=503 拒升（下放陈旧
// 回放帧会假报活——宁拒连客户端退避等新上游,不喂假数据）。
func (r *Relay) serveWS(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	up := r.upAlive
	r.mu.Unlock()
	if !up {
		http.Error(w, "上游 app WS 未建立", http.StatusServiceUnavailable)
		return
	}
	conn, err := relayUpgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	c := &relayClient{conn: conn, out: make(chan []byte, relayQueueBuf), done: make(chan struct{})}
	r.mu.Lock()
	r.clients[c] = struct{}{}
	backlog := append([][]byte(nil), r.buf...) // 新连接回灌最近帧（type7 种子就在这）
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.dropLocked(c)
		r.mu.Unlock()
	}()

	// 写泵：控制帧代答 + 数据帧透传共用同一写径（conn 写串行化）。
	// pumpGate：客户端首帧直读 auth ack——ack 未发前数据帧禁出（否则首帧不是
	// ack 客户端判败无限重连）。
	writes := make(chan []byte, relayQueueBuf+64)
	pumpGate := make(chan struct{})
	go func() {
		<-pumpGate
		for _, b := range backlog {
			select {
			case writes <- b:
			case <-c.done:
				return
			}
		}
		for {
			select {
			case <-c.done:
				return
			case b := <-c.out:
				select {
				case writes <- b:
				case <-c.done:
					return
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-c.done:
				return
			case b := <-writes:
				_ = conn.SetWriteDeadline(time.Now().Add(relayWriteTimeout))
				if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
					r.mu.Lock()
					r.dropLocked(c)
					r.mu.Unlock()
					return
				}
			}
		}
	}()

	// 读径：auth/subscribe/heartbeat → 本地代答（形状照 connectapi 信封）
	for {
		if _, data, err := conn.ReadMessage(); err != nil {
			return
		} else {
			var env map[string]json.RawMessage
			if json.Unmarshal(data, &env) != nil {
				continue
			}
			var ack []byte
			switch {
			case env["auth"] != nil:
				ack = []byte(`{"auth":{"status":"ok"}}`)
				select {
				case <-pumpGate:
				default:
					close(pumpGate) // 首个 auth 应答发出即开数据闸
				}
			case env["subscribe"] != nil:
				ack = []byte(`{"subscribe":{"status":"ok"}}`)
			case env["heartbeat"] != nil:
				ack = []byte(`{"heartbeat":{"status":"ok"}}`)
			default:
				continue
			}
			select {
			case writes <- ack:
			case <-c.done:
				return
			}
		}
	}
}
