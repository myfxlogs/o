// roster.go —— fin_venue_roster 账户名册读面+会话环 state 自报（O-37448
// 多账户面根修A——设计 SSOT /opt/o-research/designs/venue-multi-account.md
// D1/D2）。venue token 一身三任：数据面 venue 键+执行路由键+凭据键前缀；
// proto='mt5' 户=CFD 同质面一件适配器服务全 CFD 商（XM/IC 零码同径）。
package mt5

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/roster"
)

// VenueAcct 名册户投影（fin_venue_roster 行——cred_ns=app_secrets 键前缀，
// login=出面户号=fin_symbol_dialect account 粒度键）。
type VenueAcct struct {
	Venue  string
	CredNS string
	Login  string
	Alias  string
	Roles  []string
}

// SessionKey 共享会话行键——<venue>.session_token（venue=账户身份 token
// 一身三任，'mt5'→'mt5.session_token' 存量键不动=Exness 兼容零迁移窗；
// venue 寻址非 cred_ns——消费腿（PG-md 桥）无 fin_venue_roster 读权，
// 会话行归账户不归凭据族；键含 token=credentialRE owner 硬拒域同现制）。
func (a VenueAcct) SessionKey() string { return a.Venue + ".session_token" }

// SymbolsKey 订阅清单显式册键——<cred_ns>.symbols（app_config 热启）。
func (a VenueAcct) SymbolsKey() string { return a.CredNS + ".symbols" }

// VenueAccounts 名册活跃户读——proto 名下 state∉{disabled,removed} 全量
// （会话数=名册动态勿写死；roles 不滤——采集面只读语义，授权面归执行票）。
func VenueAccounts(ctx context.Context, pool *pgxpool.Pool, proto string) ([]VenueAcct, error) {
	rows, err := pool.Query(ctx, `SELECT venue,cred_ns,coalesce(login,''),coalesce(alias,''),roles
		FROM fin_venue_roster WHERE proto=$1 AND state NOT IN ('disabled','removed')
		ORDER BY venue`, proto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VenueAcct
	for rows.Next() {
		var a VenueAcct
		if err := rows.Scan(&a.Venue, &a.CredNS, &a.Login, &a.Alias, &a.Roles); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CredNSFor venue→凭据键前缀（OpenSession/消费腿 venue 参数化缝——
// roster 行是唯一真源；'mt5' 兜底 cred_ns='mt5'——存量单键凭据族语义
// 不变，roster 未播种/表缺迁移窗兼容）。他户缺行=err 显形禁猜键前缀。
func CredNSFor(ctx context.Context, pool *pgxpool.Pool, venue string) (string, error) {
	var ns string
	err := pool.QueryRow(ctx, `SELECT cred_ns FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&ns)
	if err == nil {
		return ns, nil
	}
	if venue == "mt5" {
		return "mt5", nil
	}
	return "", fmt.Errorf("roster 无户 venue=%s: %w", venue, err)
}

// ReportVenueState roster state 自报（UPDATE 限 state/last_error/updated_at
// 域——venue/proto/cred_ns/login 建后不可变换户=新行；disabled/removed
// 外部停用守卫不覆写——会话环自报不得复活人工停用）。prev=迁移前态
// （首入 degraded 判词锚）；户不在册=静默返。
func ReportVenueState(ctx context.Context, pool *pgxpool.Pool, venue, state string, lastErr error) (string, error) {
	var prev string
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&prev); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if prev == "disabled" || prev == "removed" {
		return prev, nil
	}
	var le any
	if lastErr != nil {
		le = lastErr.Error()
	}
	if _, err := pool.Exec(ctx, `UPDATE fin_venue_roster
		SET state=$2,last_error=$3,updated_at=now()
		WHERE venue=$1 AND state NOT IN ('disabled','removed')`,
		venue, state, le); err != nil {
		return prev, err
	}
	return prev, nil
}

// reportVenueFail connect 败径自报——degraded+last_error 落名册；首入
// degraded（prev≠degraded）补 ops_event「venue接入」败径行
// （esc_scope=venue:login——broker_face 拍注面已读此径，esc_class='接入败'
// 同 onboard 先例形）。自报败/写败不炸会话=日志留痕。
func (c *Collector) reportVenueFail(ctx context.Context, a *acctRun, cause error) {
	prev, err := ReportVenueState(ctx, c.pool, a.acct.Venue, "degraded", cause)
	if err != nil {
		log.Printf("[mt5] %s roster 自报败: %v", a.acct.Venue, err)
		return
	}
	if prev == "" || prev == "degraded" {
		return // 户不在册/已 degraded——迁移判词只在首入补（去重靠态机非 note 查重）
	}
	note := fmt.Sprintf("beat=connect key=%s:%s ConnectEx 败: %v",
		a.acct.Venue, a.acct.Login, cause)
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,esc_class,esc_scope)
		VALUES('汇市巡','venue接入',$1,'接入败',$2)`,
		note, a.acct.Venue+":"+a.acct.Login); err != nil {
		log.Printf("[mt5] %s 败径行写败（roster 态已落）: %v", a.acct.Venue, err)
	}
}

// reportVenueActive connect 成自报——active+last_error 清（幂等覆写，
// 重连每次报=名册即真源；迁移判词不在此径——恢复可见性归 state 列）。
func (c *Collector) reportVenueActive(ctx context.Context, a *acctRun) {
	if _, err := ReportVenueState(ctx, c.pool, a.acct.Venue, "active", nil); err != nil {
		log.Printf("[mt5] %s roster active 自报败: %v", a.acct.Venue, err)
	}
}

// rosterPollEvery 名册热读拍距（spec 钦定「每拍重查或 config_changed 触发
// ——新户 disabled 即拍即停，新增户即拍即起」；30s=beat 同拍对齐，var=桩测覆写锚）。
var rosterPollEvery = 30 * time.Second

// Run 上岗：Register+Beat → roster 热读循环（syncRoster diff 起停各户会话
// ——新增户即拍即起/disabled·removed 即拍即停；会话环内层=session 逻辑
// 逐户保留）。AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, c.pool, employeeName, employeeDept, employeeTitle, nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", employeeName, err)
		return
	}
	go roster.BeatLoop(ctx, c.pool, employeeName, beatEvery, errc)
	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	c.syncRoster(ctx, dial)
	t := time.NewTicker(rosterPollEvery) // sched-exempt-ok 名册热读拍（非业务轮询——户增删事件面）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			for _, r := range c.runs {
				r.cancel()
			}
			c.mu.Unlock()
			return
		case <-t.C:
			c.syncRoster(ctx, dial)
		}
	}
}

// syncRoster 名册 diff——在册户起会话 goroutine、出册/停用户收会话。
// roster 读败=本拍让位不拆既有会话；读败且存量 mt5 凭据在=单户兜底
// （迁移未落/瞬断窗 Exness 行为守恒——名册不可读不等于户死）。
func (c *Collector) syncRoster(ctx context.Context, dial func(string) (*Client, error)) {
	accts, err := VenueAccounts(ctx, c.pool, "mt5")
	if err != nil {
		log.Printf("[mt5] roster 读败本拍让位（%v）", err)
		if _, _, cerr := readCreds(ctx, c.pool, c.cfg, "mt5"); cerr != nil {
			return
		}
		accts = []VenueAcct{{Venue: "mt5", CredNS: "mt5"}}
	}
	want := map[string]VenueAcct{}
	for _, a := range accts {
		want[a.Venue] = a
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for v, r := range c.runs {
		if _, ok := want[v]; !ok {
			r.cancel()
			delete(c.runs, v)
			log.Printf("[mt5] 户 %s 出册/停用——会话收", v)
		}
	}
	for v, a := range want {
		if _, ok := c.runs[v]; ok {
			continue
		}
		rctx, cancel := context.WithCancel(ctx)
		r := &acctRun{acct: a, cancel: cancel}
		c.runs[v] = r
		go c.acctLoop(rctx, r, dial)
		log.Printf("[mt5] 户 %s（cred_ns=%s login=%s）会话起", v, a.CredNS, a.Login)
	}
}

// acctLoop 单户会话环（原 Run 内层逐户保留——Connect→boot 读→订阅→
// OnQuote 流消费；流断=退避重连同重订阅；一户死只退避本户）。
func (c *Collector) acctLoop(ctx context.Context, a *acctRun, dial func(string) (*Client, error)) {
	if _, err := ReportVenueState(ctx, c.pool, a.acct.Venue, "connecting", nil); err != nil {
		log.Printf("[mt5] %s roster connecting 自报败: %v", a.acct.Venue, err)
	}
	backoff := reconMin
	for {
		wait := backoff
		if err := c.session(ctx, dial, a); err != nil && ctx.Err() == nil {
			c.rej.Inc("stream_dead")
			if d, ok := SessionBackoff(err); ok && d > wait {
				wait = d // 会话冲突指纹——照 mtapi 冷却睡，勿 2s 节奏硬撞
			}
			log.Printf("[mt5] %s 会话终止（%.0fs 后重连）: %v", a.acct.Venue, wait.Seconds(), err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait): // sched-exempt-ok 断线退避重连（期界等待非轮询——含 mtapi 冷却自白量）
		}
		backoff = min(backoff*2, reconMax)
	}
}
