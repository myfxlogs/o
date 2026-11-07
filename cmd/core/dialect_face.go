// dialect_face.go —— T-0432 方言对表面（向导第④步——映射=确认件非推断件）：
// ①候选清单机造（SymbolList/SymbolParams 全表枚举+base/quote/类型相似度
// 排序——只省翻页不做决定，实盘符号逐只挑）；②硬验机（SymbolParams 对拍
// 官名语义——ctVal/计价币/digits/tradeable/session 全过+GetQuote 探活+点差
// 快照；语义不符=拒收非警告零落表）；③落库 fin_symbol_dialect（mt5.
// DialectStore 运行时译缝单源）。官名清单=okx.symbols owner 册实读。
// venue 语义词全取自 mtapi 实返禁自造（skills/mt5-quotes-history 方言坑：
// SymbolParams 须 venue 实报名——XAUUSD err/XAUUSDm ok 实拍）。
package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect/v2"

	dashboardv1 "o/internal/gen/dashboardv1"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

// dialectSession MT5 方言面会话窄缝（*mt5.Client 结构满足；桩测注假 seam：
// fake 喂真形回包，禁桩真 venue——T-0430 mt5VerifySession 同律）。
type dialectSession interface {
	Connect(ctx context.Context, cr mt5.Credentials) error
	SymbolList(ctx context.Context) ([]string, error)
	SymbolSpec(ctx context.Context, symbol string) (*mt5pb.SymbolInfo, *mt5pb.SymGroup, error)
	GetQuoteOne(ctx context.Context, symbol string) (mt5.QuoteSnapshot, error)
	Disconnect(ctx context.Context)
	Close()
}

// dialDialectDefault 生产径拨号（mtapi 网关——T-0430 dialSessionDefault 同 addr 键）。
func dialDialectDefault(addr string) (dialectSession, error) { return mt5.Dial(addr) }

// dialectPace 逐候选 SymbolParams 步进限速（catalogPace 同款限频面守——
// var=桩测覆写锚）。
var dialectPace = 120 * time.Millisecond

// dialectMeta 官名语义源缝（产线=OKX /public/instruments 公开实返；桩测注假
// ——官名源与 venue 源同律禁桩真）。
type dialectMeta func(ctx context.Context, canonical string) (ctVal float64, ctValCcy, state string, err error)

// canonicalMetaDefault 生产径：OKX 公开 instruments 实返（零凭据公开面——
// 官名 ctVal=单位错配病对拍锚，禁自造）。
func canonicalMetaDefault(ctx context.Context, canonical string) (float64, string, string, error) {
	cli := okxexec.NewClient(okxexec.Credentials{}, false)
	instType := "SPOT"
	switch {
	case strings.HasSuffix(canonical, "-SWAP"):
		instType = "SWAP"
	case len(canonical) >= 7 && canonical[len(canonical)-7] == '-':
		instType = "FUTURES" // -YYMMDD 到期形（T-0216 basis 族）
	}
	m, err := cli.InstrumentMeta(ctx, instType, canonical)
	if err != nil {
		return 0, "", "", err
	}
	return m.CtVal, m.CtValCcy, m.State, nil
}

// canonicalBase 官名 base 提取（首段——BTC-USDT-SWAP→BTC；判据禁散落）。
func canonicalBase(canonical string) string {
	return strings.SplitN(canonical, "-", 2)[0]
}

// canonicalQuote 官名 quote 提取（次段——BTC-USDT-SWAP→USDT；无=空）。
func canonicalQuote(canonical string) string {
	parts := strings.SplitN(canonical, "-", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// dollarPegged 美元锚定计价族（计价币对拍判据——CFD 计价方言：broker 自报
// ProfitCurrency USD/USDT/USDC 经济同族（美元锚定），官名 quote 直等禁
// （BTCUSDm profit=USD vs 官名 quote=USDT 字面异经济同）；族外=拒）。
func dollarPegged(ccy string) bool {
	switch strings.ToUpper(strings.TrimSpace(ccy)) {
	case "USD", "USDT", "USDC":
		return true
	}
	return false
}

// spreadCapBPS 点差合理带上限（硬验拒收带——bps；超带=数据病/死市快照，
// 拒收非警告）。保守宽带上限——正常主流 CFD 全谱远低于此。
const spreadCapBPS = 5000

// mt5CredsOf 凭据族读解（app_secrets <venue>.mt5.*|单例 mt5.*——cryptoutil.
// Open 正径；缺族/缺件=err 拒——映射操作必持在册凭据）。
func (c coreService) mt5CredsOf(ctx context.Context, venue string) (login uint64, password, server, host string, err error) {
	mk, err := config.ReadMasterKey()
	if err != nil {
		return 0, "", "", "", err
	}
	rows, err := c.pool.Query(ctx, `SELECT key,ciphertext FROM app_secrets`)
	if err != nil {
		return 0, "", "", "", err
	}
	defer rows.Close()
	fields := map[string]string{}
	for rows.Next() {
		var k string
		var ct []byte
		if err := rows.Scan(&k, &ct); err != nil {
			return 0, "", "", "", err
		}
		v, proto, field, ok := brokerCredKeySplit(k)
		if !ok || proto != "mt5" || v != venue {
			continue
		}
		pt, err := cryptoutil.Open(mk, ct)
		if err != nil {
			return 0, "", "", "", fmt.Errorf("凭据解封 %s: %w", k, err)
		}
		fields[field] = pt
	}
	if err := rows.Err(); err != nil {
		return 0, "", "", "", err
	}
	login, err = strconv.ParseUint(fields["login"], 10, 64)
	if err != nil || fields["password"] == "" || fields["server"] == "" {
		return 0, "", "", "", fmt.Errorf("venue=%s mt5 凭据族缺件（login/password/server 必备）", venue)
	}
	return login, fields["password"], fields["server"], fields["host"], nil
}

// dialectSessionOf 方言会话拨号+connect（凭据在册实测——死会话自然拒=硬验
// session 环构内化）。c.dialDialect 桩测缝（nil=dialDialectDefault）。
func (c coreService) dialectSessionOf(ctx context.Context, venue string) (dialectSession, uint64, error) {
	login, password, server, host, err := c.mt5CredsOf(ctx, venue)
	if err != nil {
		return nil, 0, err
	}
	addr, err := c.cfg.GetString("mt5.gateway_addr")
	if err != nil {
		return nil, 0, fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	dial := c.dialDialect
	if dial == nil {
		dial = dialDialectDefault
	}
	sess, err := dial(addr)
	if err != nil {
		return nil, 0, fmt.Errorf("网关不可达 dial: %w", err)
	}
	vctx, cancel := context.WithTimeout(ctx, verifyBudget)
	defer cancel()
	if err := sess.Connect(vctx, mt5.Credentials{Login: login, Password: password, Server: server, Host: host}); err != nil {
		sess.Close()
		return nil, 0, fmt.Errorf("connect: %w", err)
	}
	return sess, login, nil
}

// ── ①候选清单机造──────────────────────────────────────────────────────

// ListSymbolCandidates 候选清单 RPC（venue SymbolList 全目录枚举→base 命中
// 逐符 SymbolParams 实读步进限速→base+quote+类型相似度排序——只省翻页不做
// 决定；返回者皆 venue 实认存在）。
func (c coreService) ListSymbolCandidates(ctx context.Context,
	req *dashboardv1.ListSymbolCandidatesRequest) (*dashboardv1.ListSymbolCandidatesResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	venue := req.GetVenue()
	if !brokerVenueRE.MatchString(venue) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"venue 形拒——[a-z0-9_-]{1,32} token（同 credscan 形闸）")
	}
	if req.GetCanonical() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "canonical 空——官名必携")
	}
	sess, _, err := c.dialectSessionOf(ctx, venue)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, "会话拨号败: "+err.Error())
	}
	defer sess.Close()
	syms, err := sess.SymbolList(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, "SymbolList: "+err.Error())
	}
	rows, scanned := c.rankCandidates(ctx, sess, syms,
		canonicalBase(req.GetCanonical()), canonicalQuote(req.GetCanonical()), int(req.GetLimit()))
	return &dashboardv1.ListSymbolCandidatesResponse{Rows: rows, Scanned: int32(scanned)}, nil
}

// rankCandidates 候选排序核（纯机造——base 命中才进面：万符噪声不入下拉；
// 分=base 精中+quote 族+perp 型；返回者皆 params 可读=venue 实认存在）。
func (c coreService) rankCandidates(ctx context.Context, sess dialectSession,
	syms []string, base, quote string, limit int) ([]*dashboardv1.SymbolCandidate, int) {
	if limit <= 0 {
		limit = 20
	}
	var rows []*dashboardv1.SymbolCandidate
	for i, s := range syms {
		vb, vs := splitDialect(s)
		if !strings.EqualFold(vb, base) {
			continue // base 不中不入面（下拉省翻页——真身 seat 判）
		}
		if i > 0 && dialectPace > 0 {
			select {
			case <-ctx.Done():
				return rows, len(syms)
			case <-time.After(dialectPace): // sched-exempt-ok：逐符限速步进（限频面守——catalogPace 同款拍内步进非调度原语）
			}
		}
		si, sg, err := sess.SymbolSpec(ctx, s)
		if err != nil || si == nil || sg == nil {
			continue // venue 未认（params 缺）——禁猜名判据同源弃
		}
		score := 100                          // base 精中
		family := strings.TrimSuffix(vs, "m") // m 尾=mini 手数方言标（Exness 实拍）——族判剥
		if quote != "" && family != "" && dollarPegged(family) {
			score += 50
			if strings.EqualFold(family, quote) {
				score += 10 // quote 字面同（族内直等高档——USD≈USDT 同族低档）
			}
		}
		if vs == "" && isPerpSuffix(s) {
			score += 20 // PERP/.P 型后缀=perp 面
		}
		rows = append(rows, &dashboardv1.SymbolCandidate{
			VenueSymbol: s, Base: vb, Suffix: vs,
			ContractSize: si.GetContractSize(), TickSz: dialectTickSz(si),
			MinSz: sg.GetMinLots(), LotStep: sg.GetLotsStep(),
			Digits: si.GetDigits(), TradeMode: int32(sg.GetTradeMode()),
			Spread: si.GetSpread(), Score: int32(score),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].GetScore() > rows[j].GetScore() })
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, len(syms)
}

// dialectMarkers 档位/型后缀（非报价——Exness .pro 实拍在案/期货 .P·PERP；
// 剥后续剥 quote——XAUUSD.pro=base XAUUSD+quote USD+标 .pro 两层方言）。
var dialectMarkers = []string{".PRO", ".P", "PERP"}

// splitDialect 方言名两层剥后缀→(base, quote 后缀)（marker 层先剥档位标，
// quote 层长先剥——mt5.QuoteSuffixes 单源复用禁平行第二套；USDM m 尾方言
// 全含）。base 长度=ASCII 剥余（venue 符 ASCII token 面）。
func splitDialect(sym string) (base, suffix string) {
	rest := strings.ToUpper(sym)
	for _, m := range dialectMarkers {
		if strings.HasSuffix(rest, m) {
			rest = rest[:len(rest)-len(m)]
			break
		}
	}
	for _, q := range mt5.QuoteSuffixes() {
		if q == "PERP" || q == ".P" {
			continue // 型后缀归 marker 层（本件两遍剥）
		}
		if strings.HasSuffix(rest, q) {
			return sym[:len(rest)-len(q)], strings.ToLower(q)
		}
	}
	return sym[:len(rest)], ""
}

// isPerpSuffix perp 型后缀判（PERP/.P——期货永续方言面）。
func isPerpSuffix(sym string) bool {
	u := strings.ToUpper(sym)
	return strings.HasSuffix(u, "PERP") || strings.HasSuffix(u, ".P")
}

// dialectTickSz 点距（TickSize>0 实值否则 Points——T-0143 Exness 实拍：
// tick 双字段 0 不填 points 实填——venue 方言判据全 mtapi 实返）。
func dialectTickSz(si *mt5pb.SymbolInfo) float64 {
	if si.GetTickSize() > 0 {
		return si.GetTickSize()
	}
	return si.GetPoints()
}
