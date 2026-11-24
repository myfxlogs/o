package okxpriv

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/netmeter"
	"o/internal/platform/roster"
	"o/internal/platform/signutil"
)

// roster.go —— fin_venue_roster 驱动的 CEX 采集面（O-37450——设计 SSOT
// /opt/o-research/designs/venue-multi-account.md D5：okxpriv 同改 roster
// 迭代，CEX 多户同所零码）。本包=OKX 适配器 collect 面实现——CexVenue
// 接口缝见 cexvenue.go（verify 面 SubmitBrokerCred 消费；exec 面
// O-37449 落地）。名册户归族判据=venue token 'okx' 前缀（凭据向导命名
// 约定——他所 venue token 不得 okx 前缀，形闸注释同 cexVenueSet）。

const (
	// billsVenue bills 记账钩只挂此 venue 户（ingest 径凭据=主交易户
	// okcli——只读/副户事件不驱动他户账；venue token 约定同 roster 种子）。
	billsVenue = "okx"
)

// rosterPollEvery 名册热读拍距（mt5 同律 30s——新增户即拍即起/
// disabled·removed 即拍即停；var=桩测覆写锚）。
var rosterPollEvery = 30 * time.Second

// cexAcct 名册 CEX 户投影（fin_venue_roster proto='cex' 行——cred_ns=
// app_secrets 键前缀，venue=数据面写键+账户身份 token 一身三任）。
type cexAcct struct {
	Venue  string
	CredNS string
	Alias  string
	Roles  []string
}

// venueMatchOKX 本 adapter 归族判定——OKX 族=venue token 'okx' 前缀
// （okx/okx_read/okx2…同所多户零码；他所 token 前缀不同构造性不命中）。
func venueMatchOKX(venue string) bool { return len(venue) >= 3 && venue[:3] == "okx" }

// cexAccounts 名册活跃 CEX 户读——proto='cex' ∧ state∉{disabled,removed}
// ∧ 'collect'∈roles ∧ 本 adapter 归族（roles 不滤授权 execute——采集面
// 只读语义，授权面归执行票）。
func cexAccounts(ctx context.Context, pool *pgxpool.Pool) ([]cexAcct, error) {
	rows, err := pool.Query(ctx, `SELECT venue,cred_ns,coalesce(alias,''),roles
		FROM fin_venue_roster
		WHERE proto='cex' AND state NOT IN ('disabled','removed')
		  AND 'collect'=ANY(roles) ORDER BY venue`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cexAcct
	for rows.Next() {
		var a cexAcct
		if err := rows.Scan(&a.Venue, &a.CredNS, &a.Alias, &a.Roles); err != nil {
			return nil, err
		}
		if venueMatchOKX(a.Venue) {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}

// credNSFor venue→凭据键前缀（消费腿 venue 参数化缝——roster 行唯一真源；
// 'okx' 兜底 cred_ns='okx' 存量三键兼容——roster 未播种/表缺迁移窗守恒；
// 他户缺行=err 显形禁猜键前缀，同 mt5.CredNSFor 律）。
func credNSFor(ctx context.Context, pool *pgxpool.Pool, venue string) (string, error) {
	var ns string
	err := pool.QueryRow(ctx, `SELECT cred_ns FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&ns)
	if err == nil {
		return ns, nil
	}
	if venue == "okx" {
		return "okx", nil
	}
	return "", fmt.Errorf("roster 无户 venue=%s: %w", venue, err)
}

// reportVenueState roster state 自报（UPDATE 限 state/last_error/updated_at
// 域——venue/proto/cred_ns/login 建后不可变；disabled/removed 外部停用守卫
// 不覆写——采集环自报不得复活人工停用；同 mt5.ReportVenueState 律）。
func reportVenueState(ctx context.Context, pool *pgxpool.Pool, venue, state string, lastErr error) (string, error) {
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

// reportVenueFail connect/login 败径自报——degraded+last_error 落名册；
// 首入 degraded（prev≠degraded）补 ops_event「venue接入」败径行（同 mt5
// 先例形——broker_face 拍注面已读此径）。
func reportVenueFail(ctx context.Context, pool *pgxpool.Pool, venue string, cause error) {
	prev, err := reportVenueState(ctx, pool, venue, "degraded", cause)
	if err != nil {
		log.Printf("[okxpriv] %s roster 自报败: %v", venue, err)
		return
	}
	if prev == "" || prev == "degraded" {
		return
	}
	note := fmt.Sprintf("beat=ws-login venue=%s 败: %v", venue, cause)
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note,esc_class,esc_scope)
		VALUES('汇市巡','venue接入',$1,'接入败',$2)`,
		note, venue+":ws"); err != nil {
		log.Printf("[okxpriv] %s 败径行写败（roster 态已落）: %v", venue, err)
	}
}

// Supervisor 名册驱动总管——每拍重查 roster，新增户即拍即起会话、
// disabled/removed 即拍即停（mt5.Collector syncRoster 同构）。共享件
// （meter/hooks）挂总管，spawn 时透传各户 Collector。
type Supervisor struct {
	pool   *pgxpool.Pool
	sim    bool
	meter  *netmeter.Meter
	bootFn func(ctx context.Context, creds signutil.Credentials) (map[string]float64, error)
	bills  func(ctx context.Context)
	mu     sync.Mutex
	kids   map[string]context.CancelFunc
}

// NewSupervisor 装配（零凭据读——名册空=零子会话不报错；凭据缺件=子会话
// spawn 时 per-account 读败→degraded 自报，名册不可读不等于户死）。
func NewSupervisor(pool *pgxpool.Pool, sim bool) *Supervisor {
	return &Supervisor{pool: pool, sim: sim, kids: map[string]context.CancelFunc{}}
}

// SetBootFn 注入 boot 首读播种器（签名带凭据——per-account 各户自解密
// 自用；装配层 cmd/core 接 okxexec 件）。
func (s *Supervisor) SetBootFn(fn func(ctx context.Context, creds signutil.Credentials) (map[string]float64, error)) {
	s.bootFn = fn
}

// SetBillsHook 注入 bills 记账钩（仅 billsVenue 户挂——ingest 径凭据=
// 主交易户 okcli）。
func (s *Supervisor) SetBillsHook(fn func(ctx context.Context)) { s.bills = fn }

// AttachMeter 挂带宽计量件（nil=不计量——透传各子会话）。
func (s *Supervisor) AttachMeter(m *netmeter.Meter) { s.meter = m }

// Credentials 主交易户凭据（装配层 okcli/exec/closeSizer 消费——venue='okx'
// roster cred_ns 解析，roster 未播种兜底 'okx' 存量三键兼容；缺件 err
// 上抛=私流守整域缺席同旧律）。
func (s *Supervisor) Credentials(ctx context.Context) (signutil.Credentials, error) {
	ns, err := credNSFor(ctx, s.pool, billsVenue)
	if err != nil {
		return signutil.Credentials{}, err
	}
	return readCreds(ctx, s.pool, ns)
}

// Run 上岗：Register+Beat → roster 热读循环（syncRoster diff 起停各户
// 会话——会话环内层=dialLoop 逐户保留）。
func (s *Supervisor) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, s.pool, employeeName, employeeDept, employeeTitle, nil); err != nil {
		if ctx.Err() == nil {
			errc <- fmt.Errorf("%s register: %w", employeeName, err)
		}
		return
	}
	go roster.BeatLoop(ctx, s.pool, employeeName, 30*time.Second, errc)
	s.syncRoster(ctx)
	t := time.NewTicker(rosterPollEvery) // sched-exempt-ok 名册热读拍（非业务轮询——户增删事件面）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			for _, cancel := range s.kids {
				cancel()
			}
			s.mu.Unlock()
			return
		case <-t.C:
			s.syncRoster(ctx)
		}
	}
}

// syncRoster 名册 diff——在册户起会话 goroutine、出册/停用户收会话。
// roster 读败=本拍让位不拆既有会话；读败且存量 okx 凭据在=单户兜底
// （迁移未落/瞬断窗存量行为守恒——名册不可读不等于户死，同 mt5 律）。
func (s *Supervisor) syncRoster(ctx context.Context) {
	accts, err := cexAccounts(ctx, s.pool)
	if err != nil {
		log.Printf("[okxpriv] roster 读败本拍让位（%v）", err)
		if _, cerr := readCreds(ctx, s.pool, "okx"); cerr != nil {
			return
		}
		accts = []cexAcct{{Venue: billsVenue, CredNS: "okx"}}
	}
	want := map[string]cexAcct{}
	for _, a := range accts {
		want[a.Venue] = a
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for v, cancel := range s.kids {
		if _, ok := want[v]; !ok {
			cancel()
			delete(s.kids, v)
			log.Printf("[okxpriv] 户 %s 出册/停用——会话收", v)
		}
	}
	for v, a := range want {
		if _, ok := s.kids[v]; ok {
			continue
		}
		kctx, cancel := context.WithCancel(ctx)
		s.kids[v] = cancel
		go s.acctLoop(kctx, a)
		log.Printf("[okxpriv] 户 %s（cred_ns=%s）私有会话起", v, a.CredNS)
	}
}

// acctLoop 单户私有会话环——凭据按 cred_ns 解（缺件=degraded 自报让位，
// 名册行在=凭据面病灶显形不静默）；会话=Collector.Run 逐户保留。
func (s *Supervisor) acctLoop(ctx context.Context, a cexAcct) {
	c, err := newAccountCollector(ctx, s.pool, s.sim, a.Venue, a.CredNS)
	if err != nil {
		reportVenueFail(ctx, s.pool, a.Venue, err)
		return // 凭据缺件户——roster degraded 在册，凭据补位后下拍重 spawn
	}
	if s.meter != nil {
		c.AttachMeter(s.meter)
	}
	if s.bootFn != nil {
		c.SetBootFn(func(bctx context.Context) (map[string]float64, error) {
			return s.bootFn(bctx, c.Credentials())
		})
	}
	if s.bills != nil && a.Venue == billsVenue {
		c.SetBillsHook(s.bills)
	}
	c.Run(ctx, make(chan error, 1)) // 子会话 err 不外抛——会话环内层自带退避
}
