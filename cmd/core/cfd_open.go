// cfd_open.go —— O-41368 mt5_dir_mom CFD 方向族单腿真执行装配面：
//
//	mt5_dir_mom 策略 venue='mt5'=域标签非名册户——本缝把域标签解成
//	名册 cfd 户实键（fin_symbol_dialect 覆盖首中户，exness 先序=
//	eval 供腿 venue 钉死同源；无覆盖户=fail-closed 升不猜）。
//	腿构=lot 网格 floor（方言行 CtVal/MinSz/LotStep+SymbolParams
//	TradeMode 活读 FullAccess 断言——xvenueOpen 同判据链）；
//	执行=ExecuteDir（令牌/live 闸/赎回内建——腿 venue=名册户经
//	xvenueFleet executorFor 路由 mtexec 懒连三件）。
//	终局分类=mtexec.TerminalInstErr→untradeable 三件（inst 级终局
//	入册+策略退+armed 提案退——xvenue 终局同律）。
package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
)

// cfdOpen CFD 方向族单腿开仓缝（O-41368——dirOpen 同构差异件：腿
// venue=名册 cfd 户非 okx；手数=lot 网格非合约张数；价=标的 LatestPx）。
func (h hedgeOpener) cfdOpen(ctx context.Context, spec lifecycle.OpenSpec) (int64, error) {
	// venue 解析+执行三件（'mt5' 域标签→名册 cfd 户方言覆盖首中户；
	// spec.Venue 已是名册户键=直取单候选）。
	cfdVenue, deps, err := h.cfdVenueFor(ctx, spec)
	if err != nil {
		return 0, err
	}
	notional := spec.NotionalUSD
	cap1, e1 := h.cfg.GetFloat("exec.probe_notional_usd")
	cap2, e2 := h.cfg.GetFloat("risk.max_notional_usd")
	if e1 != nil || e2 != nil {
		return 0, fmt.Errorf("cap 键缺 fail-closed: %v/%v", e1, e2)
	}
	notional = min(notional, cap1, cap2)
	if notional <= 0 {
		return 0, fmt.Errorf("notional≤0 拒（strategy=%.2f caps=%.2f/%.2f）",
			spec.NotionalUSD, cap1, cap2)
	}
	blocked, err := reopenCooldownBlocked(ctx, h.cfg, h.pool, spec.StrategyRef, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("冷却窗探测 fail-closed: %w", err)
	}
	if blocked {
		return 0, fmt.Errorf("reopen_cooldown: guard 平仓后冷却窗内拒再开（%s）", spec.InstID)
	}
	// 双 inst 黑名单预检（T-0142 同律——终局已判=退役零 venue 调用）。
	var banned string
	err = h.pool.QueryRow(ctx, `SELECT inst_id FROM fin_venue_untradeable
		WHERE inst_id=$1 LIMIT 1`, spec.InstID).Scan(&banned)
	if err == nil {
		if herr := h.untradeable(ctx, spec, fmt.Errorf("blacklist_precheck")); herr != nil {
			return 0, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return 0, fmt.Errorf("venue_untradeable: %s 黑名单预检命中——策略退役", banned)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("黑名单预检读败 fail-closed: %w", err)
	}
	// 方言译名+方言行（官名→venue 符号唯一天——查无=升件 fail-closed 禁硬发）。
	sym, err := deps.dial.VenueSymbol(ctx, spec.InstID)
	if err != nil {
		deps.dial.Escalate(ctx, spec.InstID, err.Error())
		return 0, fmt.Errorf("cfd 方言: %w", err)
	}
	acct, err := deps.cli.account(ctx)
	if err != nil {
		return 0, fmt.Errorf("mt5 户号: %w", err)
	}
	drow, err := deps.ds.Lookup(ctx, cfdVenue, acct, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("cfd 方言行: %w", err)
	}
	// SymbolParams 活读 TradeMode（xvenueOpen 同判据——台账快照禁当放行证；
	// MT5()=nil=会话未建守 err 上浮非 panic）。
	mc := deps.cli.MT5()
	if mc == nil {
		return 0, fmt.Errorf("mt5 会话未建（懒连败）fail-closed——SymbolParams 活读不可得")
	}
	sp, err := mc.SymbolParams(deps.cli.Authed(ctx), &mt5pb.SymbolParamsRequest{Symbol: sym})
	if err != nil {
		deps.cli.Suspect()
		return 0, fmt.Errorf("SymbolParams 活读: %w", err)
	}
	if ge := sp.GetError(); ge != nil {
		return 0, mtexec.PbAPIError(ge)
	}
	tradeMode := int32(sp.GetResult().GetSymbolGroup().GetTradeMode())
	if tradeMode != int32(mt5pb.TradeMode_TradeMode_FullAccess) {
		return 0, fmt.Errorf("cfd %s TradeMode=%d 非 FullAccess（可开性预检拒——broker 动态限）", spec.InstID, tradeMode)
	}
	// 标的价+lot 网格手数（notional→lots=floor(n/px/ctVal/lotStep)*lotStep
	// ——xvenue cfd 腿同换算律；min_lot 起不来=fail-closed 报实数不硬压）。
	px, err := execution.LatestPx(ctx, h.pool, spec.InstID)
	if err != nil {
		return 0, fmt.Errorf("cfd 标的价: %w", err)
	}
	sz, err := cfdLots(notional, px, drow.CtVal, drow.LotStep, drow.MinSz)
	if err != nil {
		return 0, err
	}
	side, err := execution.DirSide(spec.Direction)
	if err != nil {
		return 0, err
	}
	legLabel := "cfd_long"
	if side == "sell" {
		legLabel = "cfd_short"
	}
	dec := h.gate.Check(ctx, risk.Intent{StrategyRef: spec.StrategyRef, Venue: cfdVenue,
		InstID: spec.InstID, Legs: []string{legLabel},
		NotionalUSD: notional, Leverage: 1, Kind: "open"})
	if !dec.Allow {
		return 0, fmt.Errorf("闸拒: %s", dec.Reason)
	}
	eid, xerr := h.r.ExecuteDir(ctx, execution.Intent{DecisionID: dec.DecisionID,
		StrategyRef: spec.StrategyRef, InstID: spec.InstID, NotionalUSD: notional,
		Legs: [2]execution.Leg{{Venue: cfdVenue, InstID: spec.InstID, Side: side,
			Sz: strconv.FormatFloat(sz, 'f', -1, 64), Notional: notional, CtVal: drow.CtVal}}})
	if mtexec.TerminalInstErr(xerr) {
		if herr := h.untradeable(ctx, spec, xerr); herr != nil {
			return eid, fmt.Errorf("终局处置败（须人查）: %w", herr)
		}
		return eid, fmt.Errorf("venue_untradeable: %w", xerr)
	}
	return eid, xerr
}

// cfdLots 名义→CFD 手数（lot 网格 floor+min_lot 闸——xvenue 腿构同换算
// 律提取纯件；px=标的资产价 LatestPx 读面，ctVal=方言行合约规模）。
func cfdLots(notional, px, ctVal, lotStep, minLot float64) (float64, error) {
	if px <= 0 || ctVal <= 0 || lotStep <= 0 {
		return 0, fmt.Errorf("cfd 手数换算参数非法（notional=%.4f px=%.8g ctVal=%.8g lotStep=%.8g）",
			notional, px, ctVal, lotStep)
	}
	sz := math.Floor(notional/px/ctVal/lotStep+1e-9) * lotStep
	if sz < minLot {
		return 0, fmt.Errorf("cfd 最小手起不来: lots=%.4f<minLot=%.4f（notional=%.2f px=%.8g ctVal=%.8g——名义帽与 broker 最小手冲突报实数不硬压）",
			sz, minLot, notional, px, ctVal)
	}
	return sz, nil
}

// cfdVenueFor 'mt5' 域标签→名册 cfd 户实键解析（fin_symbol_dialect 覆盖
// 首中户——exness 先序=eval 供腿 venue 钉死同源；候选=名册 proto='cfd'
// 活跃户∪spec.Venue 已是名册户直取；全灭=fail-closed 列候选不猜户）。
func (h hedgeOpener) cfdVenueFor(ctx context.Context, spec lifecycle.OpenSpec) (string, *venueExecDeps, error) {
	if h.xf == nil {
		// 旧三件兼容径（xvenueOpen 同律——fleet 未装配仅 'exness' 存量户）。
		if spec.Venue != "" && spec.Venue != "mt5" && spec.Venue != "exness" {
			return "", nil, fmt.Errorf("xvenue fleet 未装配——cfd venue=%s 无执行三件 fail-closed", spec.Venue)
		}
		return "exness", &venueExecDeps{venue: "exness", cli: h.xcli, dial: h.xdial, ds: h.xds}, nil
	}
	var candidates []string
	if spec.Venue != "" && spec.Venue != "mt5" {
		candidates = []string{spec.Venue} // 已是名册户键=直取
	} else {
		accts, err := mt5.VenueAccounts(ctx, h.pool, "cfd")
		if err != nil {
			return "", nil, fmt.Errorf("名册 cfd 户枚举败 fail-closed: %w", err)
		}
		for _, a := range accts {
			if a.Platform == "" || a.Platform == "mt5" {
				candidates = append(candidates, a.Venue)
			}
		}
		sort.Strings(candidates) // exness<xmglobal 字典序=eval 供腿 venue 先序自然落位
	}
	var tried []string
	for _, v := range candidates {
		d, err := h.xf.depsFor(ctx, v)
		if err != nil {
			tried = append(tried, v+"(deps:"+err.Error()+")")
			continue
		}
		acct, err := d.cli.account(ctx)
		if err != nil {
			tried = append(tried, v+"(acct:"+err.Error()+")")
			continue
		}
		if _, err := d.ds.Lookup(ctx, v, acct, spec.InstID); err != nil {
			tried = append(tried, v+"(dialect:"+err.Error()+")")
			continue
		}
		return v, d, nil
	}
	return "", nil, fmt.Errorf("cfd venue 无方言覆盖户（inst=%s 候选=%v）fail-closed——禁猜户下单", spec.InstID, tried)
}
