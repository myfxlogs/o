// Package ctx —— 外部事实采集库（T-0310 ark-intelligent vendored 吸收——
// 采集客户端适配 O Fetcher/store 模式：cot/fred/calendar/news 四源→
// md_ctx_fact；mql5 High 事件→md_macro_event upsert 替种子不删。
// 出处：github.com/myfxlogs/ark-intelligent（Apache-2.0——LICENSE-ark 随件；
// Telegram/cmd bot/BadgerDB/信号 confluence/AI SDK 不入——票面注记⑤红线）。
package ctx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
)

// Fetcher HTTP 取件缝（scout 同款——公开零凭据端点）。
type Fetcher func(ctx context.Context, url string) ([]byte, error)

// ctxClient 30s 总超时+边节点轮换拨号——FRED 边池对 Go TLS 指纹按节点异态
// 踢流（stream INTERNAL_ERROR 实击）：默认解析粘首个 A 记录=重试全砸同
// 节点白重；RR 轮换=每次新连换边（T-0311 返修⑤）。
var rrDial atomic.Uint64

var ctxClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: func() *http.Transport {
		t := http.DefaultTransport.(*http.Transport).Clone() // Clone 保 ForceAttemptHTTP2——手搓 Transport 缺省 h1 在 FRED 边池挂死（返修⑤坑）
		d := &net.Dialer{Timeout: 10 * time.Second}
		t.DisableKeepAlives = true
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil || len(ips) == 0 {
				return d.DialContext(ctx, network, addr)
			}
			n := rrDial.Add(1)
			return d.DialContext(ctx, network,
				net.JoinHostPort(ips[n%uint64(len(ips))].String(), port))
		}
		return t
	}(),
}

var errTransport = errors.New("transport err") // 可重试类标记（边节点异态踢流换连即换节点）

func httpGet(ctx context.Context, url string) ([]byte, error) {
	var last error
	for i := 0; i < 3; i++ { // 传输败三试（T-0311 返修⑤）
		if i > 0 {
			time.Sleep(time.Duration(i) * time.Second) // sched-exempt-ok 传输重试退避
		}
		b, err := httpTryGet(ctx, url)
		if err == nil {
			return b, nil
		}
		last = err
		if !errors.Is(err, errTransport) {
			break // 非传输层败（status 拒/建请败）不重试
		}
	}
	if errors.Is(last, errTransport) {
		if b, err := curlGet(ctx, url); err == nil { // curl TLS 指纹过 Akamai bot 检——Go JA3 被拦径实证备用径（T-0311 返修⑤）
			return b, nil
		}
	}
	return nil, last
}

// curlGet Go 传输尽后 curl 备用径（单 IP 边节点对 Go TLS 指纹间歇踢流——
// 同盒 curl 同端点实证通；stdout 出件 stderr 吞防 api_key 落日志）。
func curlGet(ctx context.Context, url string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "curl", "-sfS", "--http1.1", "--max-time", "25",
		url).Output()
	if err != nil {
		return nil, fmt.Errorf("http get: %w: curl", errTransport)
	}
	return out, nil
}

func httpTryGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// 禁自造 UA——o-ctx/o-research 自命名 UA 被 Akamai 边池拉黑 tarpit/踢流
	// （四 UA 隔离实证：curl/Go 默认 UA=200，自造=挂死；返修⑤根凶）。
	resp, err := ctxClient.Do(req)
	if err != nil {
		var ue *neturl.Error
		if errors.As(err, &ue) {
			return nil, fmt.Errorf("http get: %w: %w", errTransport, ue.Err) // 脱 URL——api_key 不落日志
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

// Runner 采集臂（五源一拍——各源独立 fail-soft）。
type Runner struct {
	pool    *pgxpool.Pool
	escPool *pgxpool.Pool // ops_event 升件写径（fin 域——md 池无 ops_event 面）
	rej     *rejects.Door
	fetch   Fetcher
	fredKey string
	fredFn  func(context.Context) (string, error)
	now     func() time.Time
}

func NewRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool,
		rej:   rejects.New("ctx", "fetch_err", "write_err", "parse_err", "probe_err"),
		fetch: httpGet, now: time.Now}
}

// AttachEscPool 断供探针升件写径挂接（ops_event 驻 fin 域——研究节点 FIN_PG_DSN
// 小池；未挂=探针守位不探，采集照常）。
func (r *Runner) AttachEscPool(p *pgxpool.Pool) *Runner { r.escPool = p; return r }

// AttachFredKey FRED 键注入口径（键缺=整源跳过——凭据归 pg：装配件
// cfg.Secret("fred_api_key") 按次直读注入，env 直读门禁钉）。
func (r *Runner) AttachFredKey(k string) *Runner { r.fredKey = k; return r }

// AttachFredFn FRED 键每拍自取缝（T-0311 返修⑤——boot 一次性读键瞬败=
// 整源哑到下次重启；按次直读=钦定口径本尊，瞬败下拍自愈）。
func (r *Runner) AttachFredFn(fn func(context.Context) (string, error)) *Runner {
	r.fredFn = fn
	return r
}

// Run 6h 节拍（宏观日历/OI 密度低——日四拍足；analyst 同拍错峰无竞争）。
func (r *Runner) Run(c context.Context, errc chan<- error) {
	for {
		if err := r.TickOnce(c); err != nil {
			errc <- fmt.Errorf("ctx 拍败: %w", err)
		}
		select {
		case <-c.Done():
			return
		case <-time.After(6 * time.Hour): // sched-exempt-ok 6h 钦定节拍
		}
	}
}

// TickOnce 五源各拍+断供探针尾槽（OI 补回=五源+探针六件）——单源败=该源
// 缺席不拖全局；探针败=log 自报（errc=岗终径，观测臂不拖崩采集臂）。
func (r *Runner) TickOnce(c context.Context) error {
	for _, fn := range []func(context.Context) error{r.collectCalendar, r.collectFred, r.collectFx, r.collectCot, r.collectOI} {
		if err := fn(c); err != nil {
			r.rej.Inc("fetch_err")
		}
	}
	if err := r.ProbeStale(c); err != nil {
		log.Printf("[ctx] 断供探针拍败（下拍续试）: %v", err)
		r.rej.Inc("probe_err")
	}
	return nil
}

func (r *Runner) ins(c context.Context, source, kind, key string, payload any) error {
	b, _ := json.Marshal(payload)
	var dk any
	if key != "" {
		dk = key // ''→NULL 不入唯一键（NOT NULL 撞同键防）
	}
	if _, err := r.pool.Exec(c, `INSERT INTO md_ctx_fact(source,kind,payload,dedupe_key)
		VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, source, kind, b, dk); err != nil {
		r.rej.Inc("write_err")
		return err
	}
	return nil
}
