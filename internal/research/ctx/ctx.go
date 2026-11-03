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
	"net/http"
	neturl "net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
)

// Fetcher HTTP 取件缝（scout 同款——公开零凭据端点）。
type Fetcher func(ctx context.Context, url string) ([]byte, error)

// ctxClient 30s 总超时——h2 边节点间歇踢流（stream INTERNAL_ERROR 实击：
// FRED 边池对 Go 指纹按节点异态——重试换节点=最稳解；h1 实试反挂死已弃）。
var ctxClient = &http.Client{Timeout: 30 * time.Second}

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
	return nil, last
}

func httpTryGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "o-ctx/1.0")
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

// Runner 采集臂（四源一拍——各源独立 fail-soft）。
type Runner struct {
	pool    *pgxpool.Pool
	rej     *rejects.Door
	fetch   Fetcher
	fredKey string
	fredFn  func(context.Context) (string, error)
	now     func() time.Time
}

func NewRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool,
		rej:   rejects.New("ctx", "fetch_err", "write_err", "parse_err"),
		fetch: httpGet, now: time.Now}
}

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

// TickOnce 四源各拍（OI 补回=四源+OI 五件）——单源败=该源缺席不拖全局。
func (r *Runner) TickOnce(c context.Context) error {
	for _, fn := range []func(context.Context) error{r.collectCalendar, r.collectFred, r.collectCot, r.collectOI} {
		if err := fn(c); err != nil {
			r.rej.Inc("fetch_err")
		}
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
