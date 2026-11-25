// venue_exec.go —— O-37449 xvenue 执行面名册驱动装配（设计 SSOT
// venue-multi-account.md §D4）：xvenueFleet 按腿 venue 实时路由——名册
// proto/roles/state 每调重读（disabled/removed 即拍拒单，替户下单=绝对
// 禁线：路由键缺=err 上浮永不回退 'mt5'）。名册读败/execute 授权面零行
// =fail-closed 只装存量静态对+WARN ops_event（部署迁移窗零回归）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/guard"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
	"o/internal/trading/okxexec"
)

// venueExecDeps 单 venue 执行三件（会话件+方言译件+方言仓——xvenueOpen
// 预检面与 Executor 路由面同源同件）。
type venueExecDeps struct {
	venue string
	cli   xvenueSess
	dial  mtexec.Dialect
	ds    *mt5.DialectStore
}

// xvenueFleet 名册驱动执行路由件（execution.Executor 实现——Place/
// Resolve/Cancel 按 leg.Venue 逐调解算；deps 懒建缓存，名册态每调重读
// 不缓存授权面）。
type xvenueFleet struct {
	pool   *pgxpool.Pool
	cfg    *config.Client
	ds     *mt5.DialectStore
	okx    execution.Executor
	mu     sync.Mutex
	deps   map[string]*venueExecDeps
	legacy map[string]execution.Executor
	newDep func(venue string) *venueExecDeps // 桩测缝——nil=产线 mt5LazyCli 径
	warnMu sync.Mutex
	warnAt time.Time
}

// newXvenueFleet 装配（primary='mt5' 存量三件注入——deps['mt5'] 预挂
// 保会话件同一性；legacy 静态对=名册缺席兜底，构造即备非惰性）。
func newXvenueFleet(pool *pgxpool.Pool, cfg *config.Client, creds okxexec.Credentials,
	primary *venueExecDeps) *xvenueFleet {
	f := &xvenueFleet{
		pool: pool, cfg: cfg, ds: mt5.NewDialectStore(pool),
		okx:    &execution.OkxExecutor{C: okxexec.NewClient(creds, false)},
		deps:   map[string]*venueExecDeps{},
		legacy: map[string]execution.Executor{},
	}
	if primary != nil {
		f.deps[primary.venue] = primary
		f.legacy[primary.venue] = mtexec.Executor{C: primary.cli, X: primary.dial}
	}
	f.legacy["okx"] = f.okx
	return f
}

// attachOkx mutating 凭据件就位覆写（ro=零凭据占位不触网；!ro boot 段
// 单线程调用——okx 执事与 legacy 兜底件同源同换不分裂）。
func (f *xvenueFleet) attachOkx(creds okxexec.Credentials) {
	f.okx = &execution.OkxExecutor{C: okxexec.NewClient(creds, false)}
	f.legacy["okx"] = f.okx
}

// rosterRow 单户名册行（proto/state/execute 授权三列——读败 err 上浮
// 由调用侧分兜底/拒单两径）。
func (f *xvenueFleet) rosterRow(ctx context.Context, venue string) (proto, state string, canExec bool, found bool, err error) {
	err = f.pool.QueryRow(ctx, `SELECT proto,state,'execute'=ANY(roles)
		FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&proto, &state, &canExec)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, false, nil
	}
	if err != nil {
		return "", "", false, false, err
	}
	return proto, state, canExec, true, nil
}

// rosterLive 名册 execute 授权面活性（零行=名册未播种/全停——调用侧
// 回退静态对守迁移窗；>0=名册权威在册，缺席 venue 不得兜底）。
func (f *xvenueFleet) rosterLive(ctx context.Context) (bool, error) {
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_roster
		WHERE 'execute'=ANY(roles) AND state NOT IN ('disabled','removed')`).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// warnRoster 名册读败 WARN 落 ops_event（5min 节流——「记事」矩阵外
// ungated；账本只述事实不处置）。
func (f *xvenueFleet) warnRoster(ctx context.Context, venue string, err error) {
	f.warnMu.Lock()
	if time.Since(f.warnAt) < 5*time.Minute {
		f.warnMu.Unlock()
		return
	}
	f.warnAt = time.Now()
	f.warnMu.Unlock()
	if _, werr := f.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('o-core','记事',$1)`,
		fmt.Sprintf("WARN xvenueFleet 名册读败 venue=%s: %v——execute 面按静态对兜底（名册缺席窗兼容径）", venue, err)); werr != nil {
		log.Printf("WARN [xvenue] roster 读败 ops_event 写败: %v", werr)
	}
}

// legacyExec 静态对兜底（名册读败/授权面零行——存量 okx+mt5 旧行为
// 守恒；无静态件 venue=err 上浮不兜底造假）。
func (f *xvenueFleet) legacyExec(venue string) (execution.Executor, error) {
	if ex, ok := f.legacy[venue]; ok {
		return ex, nil
	}
	return nil, fmt.Errorf("venue %s 无执行件（名册缺席兜底面亦无）fail-closed", venue)
}

// checkVenue 名册授权闸（executorFor/depsFor 共用判据——行在册：
// disabled/removed/缺 execute 角色=拒单即拍；不在册：名册活→拒、
// 名册死/空→调用侧定兜底）。返 proto 供分发。
func (f *xvenueFleet) checkVenue(ctx context.Context, venue string) (proto string, legacyOK bool, err error) {
	p, state, canExec, found, rerr := f.rosterRow(ctx, venue)
	if rerr != nil {
		f.warnRoster(ctx, venue, rerr)
		return "", true, nil // 读败=名册缺席窗——兜底判定归调用侧
	}
	if !found {
		live, lerr := f.rosterLive(ctx)
		if lerr != nil {
			f.warnRoster(ctx, venue, lerr)
			return "", true, nil
		}
		if !live {
			return "", true, nil // 授权面零行=静态对窗
		}
		return "", false, fmt.Errorf("venue %s 不在册且名册 execute 面存活——无名册执行授权 fail-closed（替户下单禁线）", venue)
	}
	if state == "disabled" || state == "removed" || !canExec {
		return "", false, fmt.Errorf("venue %s 未授权执行（state=%s can_exec=%v）——disabled 即拍拒单", venue, state, canExec)
	}
	return p, false, nil
}

// depsFor venue→执行三件（cfd 类户懒建缓存——会话件 venue 参数化
// 复挂本户共享行；非 mt5-proto=err）。
func (f *xvenueFleet) depsFor(ctx context.Context, venue string) (*venueExecDeps, error) {
	proto, legacyOK, err := f.checkVenue(ctx, venue)
	if err != nil {
		return nil, err
	}
	if legacyOK {
		if d, ok := f.deps[venue]; ok {
			return d, nil
		}
		return nil, fmt.Errorf("venue %s 无执行三件（名册缺席兜底面）fail-closed", venue)
	}
	if proto != "cfd" {
		return nil, fmt.Errorf("venue %s proto=%s 非 cfd 类——无 CFD 执行三件", venue, proto)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if d, ok := f.deps[venue]; ok {
		return d, nil
	}
	var d *venueExecDeps
	if f.newDep != nil {
		d = f.newDep(venue)
	} else {
		cli := &mt5LazyCli{pool: f.pool, cfg: f.cfg, venue: venue}
		d = &venueExecDeps{venue: venue, cli: cli,
			dial: mt5LiveDialect{S: f.ds, cli: cli, Seat: "o-core", venue: venue}, ds: f.ds}
	}
	f.deps[venue] = d
	return d, nil
}

// executorFor venue→Executor（名册每调重读授权面；proto 分发——cex 仅
// okx 在册实现，cfd→mtexec 懒连三件（MT5 协议平台；mt4 平台另件），余 proto=无执行件 err）。
func (f *xvenueFleet) executorFor(ctx context.Context, venue string) (execution.Executor, error) {
	proto, legacyOK, err := f.checkVenue(ctx, venue)
	if err != nil {
		return nil, err
	}
	if legacyOK {
		return f.legacyExec(venue)
	}
	switch proto {
	case "cex":
		if venue == "okx" {
			return f.okx, nil
		}
		return nil, fmt.Errorf("cex proto venue=%s 无执行件（仅 okx 在册实现）fail-closed", venue)
	case "cfd":
		d, derr := f.depsFor(ctx, venue)
		if derr != nil {
			return nil, derr
		}
		return mtexec.Executor{C: d.cli, X: d.dial}, nil
	default:
		return nil, fmt.Errorf("proto=%s venue=%s 无执行件 fail-closed", proto, venue)
	}
}

// Place 腿级路由执行（RouterExecutor 语义逐腿 venue 分派——名册授权面
// 每调重读=disabled 即拍生效非缓存窗）。
func (f *xvenueFleet) Place(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	ex, err := f.executorFor(ctx, leg.Venue)
	if err != nil {
		return execution.OrderResult{}, err
	}
	return ex.Place(ctx, leg, clOrdID)
}

// Resolve 歧态势查单同径路由。
func (f *xvenueFleet) Resolve(ctx context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	ex, err := f.executorFor(ctx, leg.Venue)
	if err != nil {
		return execution.OrderResult{}, err
	}
	return ex.Resolve(ctx, leg, clOrdID)
}

// Cancel 撤挂事务同径路由（mt5 面=maker 不支持 err 无害——降级径不问撤果）。
func (f *xvenueFleet) Cancel(ctx context.Context, leg execution.Leg, clOrdID string) error {
	ex, err := f.executorFor(ctx, leg.Venue)
	if err != nil {
		return err
	}
	return ex.Cancel(ctx, leg, clOrdID)
}

// wireXvenueFleet 装配总件（primary='mt5' 存量三件预挂进 fleet——
// hedgeOpener 预检面/执事路由面/sizer 面同件同源不分裂；返回 primary
// deps 供调用侧取 cli/ds/dial 三面引用——懒连会话+方言仓+译件凭据缺
// =取用点 err 显形非 boot 炸）。
func wireXvenueFleet(pool *pgxpool.Pool, cfg *config.Client, creds okxexec.Credentials) (*xvenueFleet, *venueExecDeps) {
	cli := &mt5LazyCli{pool: pool, cfg: cfg, venue: "exness"}
	ds := mt5.NewDialectStore(pool)
	prim := &venueExecDeps{venue: "exness", cli: cli,
		dial: mt5LiveDialect{S: ds, cli: cli, Seat: "o-core"}, ds: ds}
	return newXvenueFleet(pool, cfg, creds, prim), prim
}

// newExecRunner 名册驱动双腿路由执事装配（O-37449——xvenueFleet 腿
// venue=名册路由键，proto 分发，disabled 即拍拒单；名册读败/授权面
// 零行=存量静态对兜底+WARN）。fleet 由调用侧构造注入——hedgeOpener
// 预检面与执事路由面同件同源。
func newExecRunner(pool *pgxpool.Pool, cfg *config.Client, fleet *xvenueFleet) *execution.Runner {
	return execution.NewRunner(pool, cfg, fleet)
}

// AttachWardenSizers warden venue sizer 按户挂接（O-37449——'mt5' 字面量+名册 mt5-proto 全户；名册枚举败/户 depsFor 败=该户 sizer 缺席 fail-closed 让位显形）。
func (f *xvenueFleet) AttachWardenSizers(ctx context.Context, wd *guard.Warden) {
	if p, ok := f.deps["exness"]; ok { // 'exness' 存量户=primary 预挂件（O-39440① venue 改名）
		wd.AttachVenueSizer("exness", Mt5Sizer{C: p.cli, X: p.dial, D: p.ds, Venue: "exness"})
	}
	xvs, err := f.Mt5ProtoVenues(ctx)
	if err != nil {
		log.Printf("WARN [core] 名册 mt5-proto 户枚举败（%v）——仅 'exness' sizer 在挂", err)
		return
	}
	for _, xv := range xvs {
		if xv == "exness" {
			continue
		} // primary 已预挂跳过
		xd, derr := f.depsFor(ctx, xv)
		if derr != nil {
			log.Printf("WARN [core] venue=%s sizer 装配跳（%v）——该户平仓量现读面缺席 fail-closed", xv, derr)
			continue
		}
		wd.AttachVenueSizer(xv, Mt5Sizer{C: xd.cli, X: xd.dial, D: xd.ds, Venue: xv})
	}
}

// Mt5ProtoVenues 名册 mt5-proto 活跃户全量（sizer 路由面装配源——
// 'mt5' 存量户由调用侧字面量兼容锚并入，防名册未播种窗漏挂）。
func (f *xvenueFleet) Mt5ProtoVenues(ctx context.Context) ([]string, error) {
	accts, err := mt5.VenueAccounts(ctx, f.pool, "cfd")
	if err != nil {
		return nil, err
	}
	out := []string{"exness"} // 存量兼容锚（O-39440① venue 改名）
	for _, a := range accts {
		if a.Venue != "exness" && (a.Platform == "" || a.Platform == "mt5") {
			out = append(out, a.Venue) // 仅 mt5 平台户——mtexec=MT5 协议执行件
		}
	}
	return out, nil
}
