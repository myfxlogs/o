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

	"github.com/jackc/pgx/v5"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
)

// mt5LazyCli 懒连会话件拆件归 mt5lazycli.go（棘轮 D-588——O-37449）。

// mt5LiveDialect 方言译件懒连实现（mtexec.Dialect——account 键会话实读非
// 凭据直取：同一把会话件开执双用，凭据/户号单一真源）。
type mt5LiveDialect struct {
	S     *mt5.DialectStore
	cli   *mt5LazyCli
	Seat  string
	venue string // O-37449——服务哪户（空/'mt5'=Exness 兼容；方言行 venue 键同会话户）
}

func (d mt5LiveDialect) v() string {
	if d.venue == "" {
		return "exness"
	}
	return d.venue
}

func (d mt5LiveDialect) VenueSymbol(ctx context.Context, canonical string) (string, error) {
	acct, err := d.cli.account(ctx)
	if err != nil {
		return "", fmt.Errorf("mt5 户号未定（会话未建——fail-closed 禁猜）: %w", err)
	}
	r, err := d.S.Lookup(ctx, d.v(), acct, canonical)
	if err != nil {
		return "", err
	}
	return r.VenueSymbol, nil
}
func (d mt5LiveDialect) Escalate(ctx context.Context, canonical, reason string) {
	_ = d.S.Escalate(ctx, d.Seat, "方言缺映射",
		fmt.Sprintf("canonical=%s reason=%s——fail-closed 拒单（禁官名硬发）", canonical, reason), d.v())
}

// xvenueOpen 跨所 carry 开仓缝（T-0145 DM 裁定钦定面——主行 venue='okx'
// 驱动双腿原子序；副腿行 venue≠okx=永不独立开仓）。序：暗发闸→双帽→冷却→
// pair 对位→黑名单双检→方言/SymbolParams 活读可开性→价/元数据→BuildXvenueLegs→
// 风控闸→Execute 原子件→终局分类（okx TerminalInstErr/mtexec TerminalInstErr
// →双腿退役+入册）。
func (h hedgeOpener) xvenueOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	// T-0145 返修①：全 err 出口终局分类——venue 终局拒任一出口皆入册联动
	// 非仅 Execute xerr（loadMeta/腿构等 Execute 前 okx 面 err 直返=同型漏面）。
	// cfd/mateID 对位前=零值——终局锚缺席按原 err 上浮不妄退。
	var cfd, cfdVenue string
	var mateID int64
	cerr := func(err error) error { return h.xvenueErr(ctx, spec, mateID, cfd, err) }
	if spec.Venue != "okx" {
		return 0, cerr(fmt.Errorf("xvenue-carry 副腿行（venue=%s）永不独立开仓——主行驱动双腿原子序", spec.Venue))
	}
	if h.xf == nil && (h.xcli == nil || h.xdial == nil || h.xds == nil) {
		return 0, cerr(fmt.Errorf("xvenue 装配缺（mt5 会话/方言/仓件未挂）fail-closed"))
	}
	on, err := h.cfg.GetBool("exec.xvenue_open")
	if err != nil || !on {
		return 0, cerr(fmt.Errorf("exec.xvenue_open 未开 fail-closed（venue 语义彩排前置——彩排过后 dmconfig 翻）"))
	}
	notional := spec.NotionalUSD
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, cerr(fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2))
	}
	notional = min(notional, cap1, cap2)
	if notional <= 0 {
		return 0, cerr(fmt.Errorf("notional≤0 拒（strategy=%.2f caps=%.2f/%.2f）", spec.NotionalUSD, cap1, cap2))
	}
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, cerr(fmt.Errorf("冷却窗探测 fail-closed: %w", err))
	}
	if blocked {
		return 0, cerr(fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID))
	}
	// cfd 官名+副腿行对位（提案双行→bridge.xvenue.map 兜底——pair 行未落期后备径）
	cfd, cfdVenue, mateID, err = h.xvenuePair(ctx, spec)
	if err != nil {
		return 0, cerr(err)
	}
	// CFD 面三件按副腿 venue 解析（O-37449——名册驱动 fleet 在=按户
	// 懒建会话/方言/仓；fleet 缺席+'mt5'=旧三件兼容径，余 venue=fail-closed）
	var cli xvenueSess
	var dial mtexec.Dialect
	var ds *mt5.DialectStore
	if h.xf != nil {
		d, derr := h.xf.depsFor(ctx, cfdVenue)
		if derr != nil {
			return 0, cerr(derr)
		}
		cli, dial, ds = d.cli, d.dial, d.ds
	} else {
		if cfdVenue != "exness" {
			return 0, cerr(fmt.Errorf("xvenue fleet 未装配——cfd venue=%s 无执行三件 fail-closed", cfdVenue))
		}
		cli, dial, ds = h.xcli, h.xdial, h.xds
	}
	// 双 inst 黑名单预检（终局已判=退双腿+提案零 venue 调用——T-0145③钦定）
	var bannedInst string
	err = h.pool.QueryRow(ctx, `SELECT inst_id FROM fin_venue_untradeable
		WHERE inst_id IN ($1,$2) LIMIT 1`, spec.InstID, cfd).Scan(&bannedInst)
	if err == nil {
		if herr := h.xvenuePairRetire(ctx, spec, mateID, "blacklist_precheck:"+bannedInst); herr != nil {
			return 0, cerr(fmt.Errorf("预检退役败（须人查）: %w", herr))
		}
		return 0, cerr(fmt.Errorf("venue_untradeable: %s 黑名单预检命中——双腿+提案退役", bannedInst))
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, cerr(fmt.Errorf("黑名单预检读败 fail-closed: %w", err))
	}
	// 方言译名（官名→venue 符号唯一天——查无=升件 fail-closed 禁硬发）
	sym, err := dial.VenueSymbol(ctx, cfd)
	if err != nil {
		dial.Escalate(ctx, cfd, err.Error())
		return 0, cerr(fmt.Errorf("cfd 方言: %w", err))
	}
	acct, err := cli.account(ctx)
	if err != nil {
		return 0, cerr(fmt.Errorf("mt5 户号: %w", err))
	}
	drow, err := ds.Lookup(ctx, cfdVenue, acct, cfd)
	if err != nil {
		return 0, cerr(fmt.Errorf("cfd 方言行: %w", err))
	}
	// SymbolParams 活读 TradeMode（T-0039 下单时点断言——台账快照禁当放行证）
	// T-0145 返修②：MT5()=nil=会话未建——nil 守 err 上浮非 panic。
	mc := cli.MT5()
	if mc == nil {
		return 0, cerr(fmt.Errorf("mt5 会话未建（懒连败）fail-closed——SymbolParams 活读不可得"))
	}
	sp, err := mc.SymbolParams(cli.Authed(ctx), &mt5pb.SymbolParamsRequest{Symbol: sym})
	if err != nil {
		cli.Suspect()
		return 0, cerr(fmt.Errorf("SymbolParams 活读: %w", err))
	}
	// T-0145 返修①：payload 内 venue 拒 err 化——活读面终局码同走分类
	// （SYMBOL_NOT_FOUND 入册锚；err==nil 吞 payload=同型漏面）。
	if ge := sp.GetError(); ge != nil {
		return 0, cerr(mtexec.PbAPIError(ge))
	}
	tradeMode := int32(sp.GetResult().GetSymbolGroup().GetTradeMode())
	cfdM := execution.CfdMeta{ContractSize: drow.CtVal, MinLot: drow.MinSz,
		LotStep: drow.LotStep, TradeMode: tradeMode}
	spotInst, err := execution.DeriveSpotInst(spec.InstID)
	if err != nil {
		return 0, cerr(err)
	}
	pxCfd, err := execution.LatestPx(ctx, h.pool, spotInst)
	if err != nil {
		return 0, cerr(fmt.Errorf("cfd 标的价: %w", err))
	}
	pxPerp, err := execution.LatestPx(ctx, h.pool, spec.InstID)
	if err != nil {
		return 0, cerr(fmt.Errorf("perp 价: %w", err))
	}
	loadM := h.loadMeta
	if h.xmeta != nil {
		loadM = h.xmeta // 桩测缝（返修① e2e 锚——产线 nil=h.loadMeta）
	}
	perpM, err := loadM(ctx, spec.InstID)
	if err != nil {
		return 0, cerr(fmt.Errorf("perp 元数据: %w", err))
	}
	legs, lerr := execution.BuildXvenueLegs(spec.Direction, spec.InstID, cfd, cfdVenue,
		notional, perpM, cfdM, pxCfd, pxPerp)
	if lerr != nil {
		return 0, cerr(fmt.Errorf("xvenue 腿构: %w", lerr))
	}
	var legLabels []string
	for _, l := range legs {
		tag, dir := "perp", "long"
		if l.Venue == cfdVenue {
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
		return 0, cerr(fmt.Errorf("闸拒: %s", dec.Reason))
	}
	eid, xerr := h.r.Execute(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional, Legs: legs})
	return eid, cerr(xerr)
}

// newXvenueDeps/newExecRunner/newXvenueFleet 装配件归 venue_exec.go
// （O-37449——名册驱动执行路由单域文件，棘轮拆件）。
