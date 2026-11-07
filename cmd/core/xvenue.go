// xvenue.go —— T-0145 跨所 carry 真执行装配面（DM 裁定钦定面）：
//
//	xvenue-carry 主行（venue='okx'）开仓缝 xvenueOpen——「mt5 cfd 腿先行→
//	okx perp 腿」双腿原子序（执行原子件=execution.Runner.Execute LEGGED→
//	rollback 在案径）；副腿行永不独立开仓（openLive 路由闸在 lifecycle）。
//	mt5LazyCli 懒连会话件——首用 OpenSession；RPC 败→Suspect() 标失效下拍
//	重连自愈（常驻会话跨重启/断链存活性——T-0145 装配钦定）。
//	mt5LiveDialect/Mt5Sizer——方言缝/平仓量现读面 mt5 侧实现（官名→venue
//	符号唯一天=fin_symbol_dialect；平仓量=OpenedOrders 实读手数）。
//	xvenueUntradeable 双腿退役处置——终局码入册+双腿+提案退役三件（trigger
//	续扫余量；显式迁移先于入册序位律同 T-0147）。
package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
	"o/internal/trading/okxexec"
)

// mt5LazyCli 懒连会话件（mtexec.Cli 产线实现+Suspecter——T-0145 装配钦定：
// 常驻会话跨重启/断链自愈——首用 OpenSession（凭据缺=取用点 err 显形非
// boot 炸）；RPC 败→Suspect() 标失效下次 ensure 重连）。
type mt5LazyCli struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	mu   sync.Mutex
	c    *mt5.Client
	dead bool
	acct string
}

func (l *mt5LazyCli) ensure(ctx context.Context) (*mt5.Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.c != nil && !l.dead {
		return l.c, nil
	}
	if l.c != nil {
		l.c.Disconnect(ctx)
		l.c.Close()
		l.c, l.acct = nil, ""
	}
	c, err := mt5.OpenSession(ctx, l.pool, l.cfg)
	if err != nil {
		l.dead = false // 未建连=非失效标（下拍续试不连坐既有会话）
		return nil, err
	}
	l.c, l.dead = c, false
	return c, nil
}

// Suspect 会话失效标记（mtexec.Suspecter——RPC 传输败径调用；下拍重连自愈）。
func (l *mt5LazyCli) Suspect() {
	l.mu.Lock()
	l.dead = true
	l.mu.Unlock()
}

// account 户号实读（方言 account 键——AccountFacts 会话内现读一次后钉；
// 会话未建=ensure 先连，凭据缺=err fail-closed 禁猜户号）。
func (l *mt5LazyCli) account(ctx context.Context) (string, error) {
	c, err := l.ensure(ctx)
	if err != nil {
		return "", err
	}
	l.mu.Lock()
	acct := l.acct
	l.mu.Unlock()
	if acct != "" {
		return acct, nil
	}
	login, _, _, err := c.AccountFacts(ctx)
	if err != nil {
		l.Suspect()
		return "", fmt.Errorf("mt5 AccountFacts 户号实读: %w", err)
	}
	acct = strconv.FormatInt(login, 10)
	l.mu.Lock()
	l.acct = acct
	l.mu.Unlock()
	return acct, nil
}

func (l *mt5LazyCli) Authed(ctx context.Context) context.Context {
	if c, err := l.ensure(ctx); err == nil {
		return c.Authed(ctx)
	}
	return ctx // 会话未建=无 token——RPC 败上浮 executor 显形（err 路径 Suspect 自愈）
}
func (l *mt5LazyCli) Conn() *grpc.ClientConn {
	if c, err := l.ensure(context.Background()); err == nil {
		return c.Conn()
	}
	return nil // 懒连败=nil conn——RPC 调用点 err 非 panic（grpc nil conn 调用=错误上浮）
}
func (l *mt5LazyCli) MT5() mt5pb.MT5Client {
	if c, err := l.ensure(context.Background()); err == nil {
		return c.MT5()
	}
	return mt5pb.NewMT5Client(nil)
}

// mt5LiveDialect 方言译件懒连实现（mtexec.Dialect——account 键会话实读非
// 凭据直取：同一把会话件开执双用，凭据/户号单一真源）。
type mt5LiveDialect struct {
	S    *mt5.DialectStore
	cli  *mt5LazyCli
	Seat string
}

func (d mt5LiveDialect) VenueSymbol(ctx context.Context, canonical string) (string, error) {
	acct, err := d.cli.account(ctx)
	if err != nil {
		return "", fmt.Errorf("mt5 户号未定（会话未建——fail-closed 禁猜）: %w", err)
	}
	r, err := d.S.Lookup(ctx, "mt5", acct, canonical)
	if err != nil {
		return "", err
	}
	return r.VenueSymbol, nil
}
func (d mt5LiveDialect) Escalate(ctx context.Context, canonical, reason string) {
	_ = d.S.Escalate(ctx, d.Seat, "方言缺映射",
		fmt.Sprintf("canonical=%s reason=%s——fail-closed 拒单（禁官名硬发）", canonical, reason), "mt5")
}

// xvenueOpen 跨所 carry 开仓缝（T-0145 DM 裁定钦定面——主行 venue='okx'
// 驱动双腿原子序；副腿行 venue≠okx=永不独立开仓）。序：暗发闸→双帽→冷却→
// pair 对位→黑名单双检→方言/SymbolParams 活读可开性→价/元数据→BuildXvenueLegs→
// 风控闸→Execute 原子件→终局分类（okx TerminalInstErr/mtexec TerminalInstErr
// →双腿退役+入册）。
func (h hedgeOpener) xvenueOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	if spec.Venue != "okx" {
		return 0, fmt.Errorf("xvenue-carry 副腿行（venue=%s）永不独立开仓——主行驱动双腿原子序", spec.Venue)
	}
	if h.xcli == nil || h.xdial == nil || h.xds == nil {
		return 0, fmt.Errorf("xvenue 装配缺（mt5 会话/方言/仓件未挂）fail-closed")
	}
	on, err := h.cfg.GetBool("exec.xvenue_open")
	if err != nil || !on {
		return 0, fmt.Errorf("exec.xvenue_open 未开 fail-closed（venue 语义彩排前置——彩排过后 dmconfig 翻）")
	}
	notional := spec.NotionalUSD
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2)
	}
	notional = min(notional, cap1, cap2)
	if notional <= 0 {
		return 0, fmt.Errorf("notional≤0 拒（strategy=%.2f caps=%.2f/%.2f）", spec.NotionalUSD, cap1, cap2)
	}
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("冷却窗探测 fail-closed: %w", err)
	}
	if blocked {
		return 0, fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID)
	}
	// cfd 官名+副腿行对位（提案双行→bridge.xvenue.map 兜底——pair 行未落期后备径）
	cfd, mateID, err := h.xvenuePair(ctx, spec)
	if err != nil {
		return 0, err
	}
	// 双 inst 黑名单预检（终局已判=退双腿+提案零 venue 调用——T-0145③钦定）
	var bannedInst string
	err = h.pool.QueryRow(ctx, `SELECT inst_id FROM fin_venue_untradeable
		WHERE inst_id IN ($1,$2) LIMIT 1`, spec.InstID, cfd).Scan(&bannedInst)
	if err == nil {
		if herr := h.xvenuePairRetire(ctx, spec, mateID, "blacklist_precheck:"+bannedInst); herr != nil {
			return 0, fmt.Errorf("预检退役败（须人查）: %w", herr)
		}
		return 0, fmt.Errorf("venue_untradeable: %s 黑名单预检命中——双腿+提案退役", bannedInst)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("黑名单预检读败 fail-closed: %w", err)
	}
	// 方言译名（官名→venue 符号唯一天——查无=升件 fail-closed 禁硬发）
	sym, err := h.xdial.VenueSymbol(ctx, cfd)
	if err != nil {
		h.xdial.Escalate(ctx, cfd, err.Error())
		return 0, fmt.Errorf("cfd 方言: %w", err)
	}
	acct, err := h.xcli.account(ctx)
	if err != nil {
		return 0, fmt.Errorf("mt5 户号: %w", err)
	}
	drow, err := h.xds.Lookup(ctx, "mt5", acct, cfd)
	if err != nil {
		return 0, fmt.Errorf("cfd 方言行: %w", err)
	}
	// SymbolParams 活读 TradeMode（T-0039 下单时点断言——台账快照禁当放行证）
	sp, err := h.xcli.MT5().SymbolParams(h.xcli.Authed(ctx), &mt5pb.SymbolParamsRequest{Symbol: sym})
	if err != nil {
		h.xcli.Suspect()
		return 0, fmt.Errorf("SymbolParams 活读: %w", err)
	}
	tradeMode := int32(sp.GetResult().GetSymbolGroup().GetTradeMode())
	cfdM := execution.CfdMeta{ContractSize: drow.CtVal, MinLot: drow.MinSz,
		LotStep: drow.LotStep, TradeMode: tradeMode}
	spotInst, err := execution.DeriveSpotInst(spec.InstID)
	if err != nil {
		return 0, err
	}
	pxCfd, err := execution.LatestPx(ctx, h.pool, spotInst)
	if err != nil {
		return 0, fmt.Errorf("cfd 标的价: %w", err)
	}
	pxPerp, err := execution.LatestPx(ctx, h.pool, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("perp 价: %w", err)
	}
	perpM, err := h.loadMeta(ctx, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("perp 元数据: %w", err)
	}
	legs, lerr := execution.BuildXvenueLegs(spec.Direction, spec.InstID, cfd,
		notional, perpM, cfdM, pxCfd, pxPerp)
	if lerr != nil {
		return 0, fmt.Errorf("xvenue 腿构: %w", lerr)
	}
	var legLabels []string
	for _, l := range legs {
		tag, dir := "perp", "long"
		if l.Venue == "mt5" {
			tag = "cfd"
		}
		if l.Side == "sell" {
			dir = "short"
		}
		legLabels = append(legLabels, tag+"_"+dir)
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: spec.Venue,
		InstID: spec.InstID, Legs: legLabels, NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	eid, xerr := h.r.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional, Legs: legs})
	// T-0145③ 终局分类双腿处置：okx 腿终局→okx inst 入册；mt5 腿终局→cfd
	// canonical 入册（trigger 自动退役 mt5 副行+扫余量）——双腿+提案退役三件。
	if okxexec.TerminalInstErr(xerr) {
		if herr := h.xvenueUntradeable(ctx, spec, mateID, spec.InstID, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	if mtexec.TerminalInstErr(xerr) {
		if herr := h.xvenueUntradeable(ctx, spec, mateID, cfd, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}

// newXvenueDeps xvenue 三面同件构造（懒连会话+方言仓+译件——凭据缺=取用点
// err 显形非 boot 炸；ro 件亦注——mt5 凭据独立面）。
func newXvenueDeps(pool *pgxpool.Pool, cfg *config.Client) (*mt5LazyCli, *mt5.DialectStore, mtexec.Dialect) {
	xcli := &mt5LazyCli{pool: pool, cfg: cfg}
	xds := mt5.NewDialectStore(pool)
	return xcli, xds, mt5LiveDialect{S: xds, cli: xcli, Seat: "o-core"}
}

// newExecRunner 双腿路由执事装配（T-0145——mt5 腿走懒连会话件+方言译件；
// okx 腿原径不变；凭据缺=取用点 err 显形非挂死）。
func newExecRunner(pool *pgxpool.Pool, cfg *config.Client, creds okxexec.Credentials,
	xcli *mt5LazyCli, xdial mtexec.Dialect) *execution.Runner {
	return execution.NewRunner(pool, cfg, execution.RouterExecutor{
		"okx": &execution.OkxExecutor{C: okxexec.NewClient(creds, false)},
		"mt5": mtexec.Executor{C: xcli, X: xdial},
	})
}
