// dialect_submit.go —— T-0432 ②硬验机+③落库面（映射=确认件非推断件：
// SubmitSymbolMapping verify-then-write 同构——席位挑定→硬验拒收非警告
// （ctVal/计价币/digits/tradeable/探活全过才落行）→fin_symbol_dialect 行
// +ops_event 审计行单事务。判据锚：ctVal=单位错配病判据原点（OKX
// instruments 实返对拍）；计价币=美元锚定族（CFD 计价方言——字面异经济同）；
// 探活=死价/死会话自然拒。候选机造/会话缝面见 dialect_face.go。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/encoding/protojson"

	dashboardv1 "o/internal/gen/dashboardv1"
	"o/internal/mt5"
)

// ── ②硬验机+③落库──────────────────────────────────────────────────────

// SubmitSymbolMapping 映射落行 RPC（verify-then-write 同构：席位挑定→硬验
// 拒收非警告——任一环不过=CodeInvalidArgument 零落表；全过→dialect 行
// +ops_event 审计行单事务落库）。
func (c coreService) SubmitSymbolMapping(ctx context.Context,
	req *dashboardv1.SubmitSymbolMappingRequest) (*dashboardv1.SubmitSymbolMappingResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	venue := req.GetVenue()
	if !brokerVenueRE.MatchString(venue) {
		return nil, connect.NewError(connect.CodeInvalidArgument, "venue 形拒（同 credscan 形闸）")
	}
	if req.GetCanonical() == "" || req.GetVenueSymbol() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "canonical/venue_symbol 必携（席位挑定件）")
	}
	row, ev, login, err := c.verifySymbolMapping(ctx, venue, req.GetCanonical(), req.GetVenueSymbol())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, "硬验拒收（零落表）: "+err.Error())
	}
	evJSON, err := protojson.Marshal(ev)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "evidence 编码: "+err.Error())
	}
	note := fmt.Sprintf("venue=%s login=%d %s→%s ctVal=%g spread=%.0fbps %s",
		venue, login, req.GetCanonical(), req.GetVenueSymbol(), ev.GetCtVal(), ev.GetSpreadBps(), req.GetNote())
	store := mt5.NewDialectStore(c.pool)
	if err := store.UpsertVerifiedWithAudit(ctx, row, evJSON,
		brokerCredActor, "venue符号映射", note, venue+":"+fmt.Sprint(login)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "落行: "+err.Error())
	}
	return &dashboardv1.SubmitSymbolMappingResponse{Verified: true, Evidence: ev}, nil
}

// verifySymbolMapping 硬验机核（拒收非警告——①官名语义 OKX 实返 state=live
// ctVal 正；②venue SymbolParams 实返裸面对拍：ctVal 全等/计价币美元锚定族/
// digits 正/tradeable FullAccess；③GetQuote 探活 bid/ask>0+点差合理带。
// 全过=（行,evidence,login）在案——行面全谱自同一次 SymbolParams 实读补全）。
func (c coreService) verifySymbolMapping(ctx context.Context, venue, canonical, venueSymbol string) (mt5.DialectRow, *dashboardv1.SymbolVerifyEvidence, uint64, error) {
	meta := c.dialectMeta
	if meta == nil {
		meta = canonicalMetaDefault
	}
	ctVal, ctValCcy, state, err := meta(ctx, canonical)
	if err != nil {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("官名语义源（OKX instruments）: %w", err)
	}
	if state != "live" {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("官名 state=%s 非 live——官名不可交易面", state)
	}
	if ctVal <= 0 {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("官名 ctVal=%g 非正——语义源缺", ctVal)
	}
	sess, login, err := c.dialectSessionOf(ctx, venue)
	if err != nil {
		return mt5.DialectRow{}, nil, 0, err
	}
	defer sess.Close()
	si, sg, err := sess.SymbolSpec(ctx, venueSymbol)
	if err != nil {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("venue 语义（SymbolParams）: %w——符号方言禁猜（venue 自报实读）", err)
	}
	vcs := si.GetContractSize()
	if !approxEq(vcs, ctVal) {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("ctVal 对拍拒：venue ContractSize=%g ≠ 官名 ctVal=%g（单位错配病防线）", vcs, ctVal)
	}
	if !dollarPegged(si.GetProfitCurrency()) {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("计价币拒：ProfitCurrency=%q 出美元锚定族 {USD,USDT,USDC}（官名 ctValCcy=%s）",
			si.GetProfitCurrency(), ctValCcy)
	}
	if si.GetDigits() <= 0 {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("digits=%d 非正——venue 语义缺", si.GetDigits())
	}
	if int32(sg.GetTradeMode()) != 4 {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("tradeable 拒：TradeMode=%d 非 FullAccess(4)（broker 动态限实拍）", sg.GetTradeMode())
	}
	q, err := sess.GetQuoteOne(ctx, venueSymbol)
	if err != nil {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("GetQuote 探活: %w", err)
	}
	mid := q.Mid()
	if mid <= 0 {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("探活拒：bid=%g ask=%g 死价快照", q.Bid, q.Ask)
	}
	sbps := q.SpreadBPS()
	if sbps > spreadCapBPS {
		return mt5.DialectRow{}, nil, 0, fmt.Errorf("点差拒：%.0f bps > %d 带上（数据病快照拒收非警告）", sbps, spreadCapBPS)
	}
	ev := &dashboardv1.SymbolVerifyEvidence{
		Bid: q.Bid, Ask: q.Ask, Mid: mid, SpreadBps: sbps,
		CtVal: vcs, CanonicalCtVal: ctVal, QuoteCcy: si.GetProfitCurrency(),
		Digits: si.GetDigits(), TradeMode: int32(sg.GetTradeMode()),
		SpecHash: mt5.SpecHash(si, sg),
	}
	row := mt5.DialectRow{
		Venue: venue, Account: strconv.FormatUint(login, 10),
		Canonical: canonical, VenueSymbol: venueSymbol,
		CtVal: si.GetContractSize(), TickSz: dialectTickSz(si),
		MinSz: sg.GetMinLots(), LotStep: sg.GetLotsStep(), LotUnit: "lots",
		Digits: si.GetDigits(), TradeMode: int32(sg.GetTradeMode()), SpecHash: ev.GetSpecHash(),
	}
	return row, ev, login, nil
}

// approxEq 相对容差等（ctVal 对拍——浮点直等禁，1e-9 相对容差）。
func approxEq(a, b float64) bool {
	if a == b {
		return true
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	m := a
	if b > m {
		m = b
	}
	return d <= m*1e-9
}

// ListCanonicalUniverse 官名清单读面（okx.symbols owner 册实读——升序服务端
// 排序；缺=空集如实——映射操作前置 okx.symbols 在册）。
func (c coreService) ListCanonicalUniverse(ctx context.Context,
	_ *dashboardv1.ListCanonicalUniverseRequest) (*dashboardv1.ListCanonicalUniverseResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	raw, err := c.cfg.Get("okx.symbols")
	if err != nil {
		return &dashboardv1.ListCanonicalUniverseResponse{}, nil // 缺=空集如实（不虚构）
	}
	var insts []string
	if err := json.Unmarshal(raw, &insts); err != nil {
		return &dashboardv1.ListCanonicalUniverseResponse{}, nil
	}
	sort.Strings(insts)
	return &dashboardv1.ListCanonicalUniverseResponse{Insts: insts}, nil
}
