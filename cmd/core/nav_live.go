// nav_live.go —— NAV 三户活读件（T-0086 ①④：资金户/Earn REST 保留面——
// 零快照零落地，venue 唯一事实源；自主文件拆分，行数棘轮）。
// T-0096：+MT5 户活读件（AccountSummary 现读——每真实账号一行）。
// T-0352：+venue 现读辅助三件（自 venue_readers.go 拆出——棘轮 300 行顶线：
// okxAvailFromEvents/ctValReader/okxFundingAvailUSDT，纯搬移零语义改）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/trading/okxexec"

	mt5pb "o/internal/gen/mtapi/mt5"

	"github.com/jackc/pgx/v5/pgxpool"
)

// mt5Acct MT5 户行（T-0096——login=真实账号，equity=净值现读 USD 本位；
// balance=可用余额附标。T-0327：orders=在持定单枚举——残仓面 venue 实拍源，
// 读败=ordersErr 显形不静默不假零）。
type mt5Acct struct {
	Login   string
	Equity  float64
	Balance float64
	// T-0327：venue 在持定单（残仓归属判定源——同会话捎读，零额外线）。
	Orders    []mt5Ord
	OrdersErr error
}

// mt5Ord MT5 在持定单行（残仓展示要素——symbol/ticket/开仓时点）。
type mt5Ord struct {
	Symbol string
	Ticket int64
	At     time.Time
}

// mt5 会话单飞+短窗缓存（T-0101 验收修补：mtapi 并发会话读 AccountSummary
// 返 0 实证——卡面/对拍/风控三调用方各自建会话互踩。20s 窗内共享一次
// 会话结果；窗口过期重建。登录号/净值/余额三元组同缓存）。
// mt5SumBy O-39440：单飞缓存按 roster venue token 分键（mt5/xmglobal
// 逐户互不串——并发会话互踩返零坑在案，同 token 仍单飞）。
var mt5SumBy sync.Map // venue token → *mt5SumEnt

// mt5SummaryOnce MT5 户概要单飞读（TTL 20s——闲户净值秒级漂移无意义，
// 并发会话互踩返零才是真实风险）。venueTok=roster venue token
// （"mt5"/"xmglobal"——O-39440 逐户径）。
func mt5SummaryOnce(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client,
	venueTok string, openMt5 func(context.Context) (*mt5.Client, error)) (mt5Acct, error) {
	if venueTok == "" {
		venueTok = "exness"
	}
	entAny, _ := mt5SumBy.LoadOrStore(venueTok, &struct {
		mu   sync.Mutex
		at   time.Time
		acct mt5Acct
		err  error
	}{})
	ent := entAny.(*struct {
		mu   sync.Mutex
		at   time.Time
		acct mt5Acct
		err  error
	})
	ent.mu.Lock()
	defer ent.mu.Unlock()
	if time.Since(ent.at) < 20*time.Second {
		return ent.acct, ent.err
	}
	open := openMt5
	if open == nil {
		open = func(ctx context.Context) (*mt5.Client, error) {
			return mt5.OpenSession(ctx, pool, cfg, venueTok)
		}
	}
	cl, err := open(ctx)
	if err != nil {
		ent.at, ent.err = time.Now(), err
		if IsVenueDead(err) { // 工单#26999：结构死 venue 分类——暂态读败非同类
			ent.err = venueDeadWrap(err)
		}
		return ent.acct, ent.err
	}
	defer cl.Close()
	login, _, _, ferr := cl.AccountFacts(ctx)
	eq, _, bal, aerr := cl.AccountSummary(ctx)
	if ferr != nil || aerr != nil {
		ent.at = time.Now()
		ent.err = fmt.Errorf("mt5 facts/summary: %v / %v", ferr, aerr)
		if IsVenueDead(ferr) || IsVenueDead(aerr) {
			ent.err = venueDeadWrap(ent.err)
		}
		return ent.acct, ent.err
	}
	acct := mt5Acct{Login: fmt.Sprintf("%d", login), Equity: eq, Balance: bal}
	// T-0327：同会话捎读在持定单（残仓面枚举源——并发会话互踩返零坑在案，
	// 故不另开线；单读败=ordersErr 显形不拖 summary 主读）。
	if r, oerr := cl.MT5().OpenedOrders(cl.Authed(ctx),
		&mt5pb.OpenedOrdersRequest{Id: cl.Token()}); oerr != nil {
		acct.OrdersErr = oerr
	} else if e := r.GetError(); e != nil {
		acct.OrdersErr = fmt.Errorf("opened orders: %s", e.GetMessage())
	} else {
		for _, o := range r.GetResult() {
			mo := mt5Ord{Symbol: o.GetSymbol(), Ticket: o.GetTicket()}
			if t := o.GetOpenTime(); t != nil {
				mo.At = t.AsTime()
			}
			acct.Orders = append(acct.Orders, mo)
		}
	}
	ent.acct = acct
	ent.at, ent.err = time.Now(), nil
	return ent.acct, nil
}

// T-0086 ①④：NAV 三户活读件（资金户=逐币余额、Earn=逐行 amt——REST 保留面
// 非快照零落地；凭据缺席=空表 0 不误报）。
func fundingLiveFor(okcli *okxexec.Client) func(context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		if okcli == nil {
			return out, nil
		}
		bals, err := okcli.AssetBalances(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range bals {
			out[b.Ccy] += okxexec.ParseF(b.Bal)
		}
		return out, nil
	}
}

func earnLiveFor(okcli *okxexec.Client) func(context.Context) (map[string]float64, error) {
	return func(ctx context.Context) (map[string]float64, error) {
		out := map[string]float64{}
		if okcli == nil {
			return out, nil
		}
		sav, err := okcli.SavingsBalance(ctx)
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Ccy string `json:"ccy"`
			Amt string `json:"amt"`
		}
		if json.Unmarshal(sav, &rows) != nil {
			return nil, fmt.Errorf("savings parse")
		}
		for _, r2 := range rows {
			out[r2.Ccy] += okxexec.ParseF(r2.Amt)
		}
		return out, nil
	}
}

// posLiveFor OKX 账户持仓活读件（T-0327：/account/positions 全枚举——残仓
// 面 venue 实拍源；凭据缺席/读败=err 上抛，消费方显「读件缺」不假零。
// 无缓存——端点非限流敏感（dcd 429 案独属产品族），直读随卡面刷新频度）。
func posLiveFor(okcli *okxexec.Client) func(context.Context) ([]okxexec.Position, error) {
	return func(ctx context.Context) ([]okxexec.Position, error) {
		if okcli == nil {
			return nil, fmt.Errorf("okx positions 读件缺席（凭据缺）")
		}
		return okcli.Positions(ctx)
	}
}

// dcdLiveFor dcd 产品持仓活读件（T-0101：state=live 枚举——已部署资本第
// 三类别；读件缺席/读败=err 上抛，卡面显「读件缺」不假零。
// T-0101 修补：60s TTL 单飞——卡面+对拍双消费方裸打限流端点曾撞 429；
// 持仓单结算粒度日级，60s 陈旧无风险）。
func dcdLiveFor(okcli *okxexec.Client) func(context.Context) ([]okxexec.DcdOrder, error) {
	var mu sync.Mutex
	var at time.Time
	var ords []okxexec.DcdOrder
	var lastErr error
	return func(ctx context.Context) ([]okxexec.DcdOrder, error) {
		if okcli == nil {
			return nil, fmt.Errorf("dcd 读件缺席（凭据缺）")
		}
		mu.Lock()
		defer mu.Unlock()
		if time.Since(at) < 60*time.Second {
			return ords, lastErr
		}
		o2, err := okcli.DcdLiveOrders(ctx)
		ords, lastErr, at = o2, err, time.Now()
		return ords, lastErr
	}
}

// msStamp 毫秒时戳→RFC3339（dcd cTime 开仓时点展示位——空/解析败=now）。
func msStamp(ms string) string {
	v, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || v <= 0 {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return time.UnixMilli(v).UTC().Format(time.RFC3339)
}

// msDay 毫秒时戳→YYYY-MM-DD（dcd settleTime 展示位——空/解析败=原文返回）。
func msDay(ms string) string {
	v, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return ms
	}
	return time.UnixMilli(v).UTC().Format("2006-01-02")
}

// okxAvailFromEvents 交易户可用额=md_account_events 最新帧 details availBal
// （T-0048 返修③：depositIdle 自 AcctTrading 摆钱——资金户=错户已纠；
// push-on-change 帧龄不判陈，陈度归 WS 活性件管）。
func okxAvailFromEvents(ctx context.Context, pool *pgxpool.Pool, okcli *okxexec.Client) (float64, error) {
	var data json.RawMessage
	if err := pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel='account' ORDER BY ts DESC LIMIT 1`).Scan(&data); err != nil {
		if okcli == nil {
			return 0, err
		}
		return okcli.TradingAvailBal(ctx, "USDT") // 帧缺→REST 兜底（T-0058）
	}
	var top struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return 0, err
	}
	for _, d := range top.Details {
		if d.Ccy == "USDT" {
			return parseNum(d.AvailBal), nil
		}
	}
	if okcli != nil {
		return okcli.TradingAvailBal(ctx, "USDT") // USDT 行缺=REST 兜底
	}
	return 0, fmt.Errorf("USDT 行缺席")
}

// ctValReader 合约面值现读件（T-0046①——T-0048 三令后 facts 表废，改 okx
// instruments 决策时点现读；静态乐器属性 10min TTL 缓存：guard 逐 tick 事件评
// 估，无缓存=限速自杀。读败/零值=false fail-closed 漂移判据跳过不误判）。
func ctValReader(okcli *okxexec.Client) func(ctx context.Context, venue, instID string) (float64, bool) {
	var mu sync.Mutex
	type ent struct {
		v  float64
		at time.Time
	}
	cache := map[string]ent{}
	return func(ctx context.Context, venue, instID string) (float64, bool) {
		if okcli == nil || venue != "okx" || !strings.HasSuffix(instID, "-SWAP") {
			return 0, false
		}
		mu.Lock()
		e, ok := cache[instID]
		mu.Unlock()
		if ok && time.Since(e.at) < 10*time.Minute {
			return e.v, e.v > 0
		}
		m, err := okcli.InstrumentMeta(ctx, "SWAP", instID)
		if err != nil || m.CtVal <= 0 {
			return 0, false
		}
		mu.Lock()
		cache[instID] = ent{m.CtVal, time.Now()}
		mu.Unlock()
		return m.CtVal, true
	}
}

// okxFundingAvailUSDT funding 户 USDT 闲余（T-0149 ②——/asset/balances REST
// 低频件；入金天然落 funding 面钦定补漏；无行=0 不误报）。
func okxFundingAvailUSDT(ctx context.Context, okcli *okxexec.Client) (float64, error) {
	rows, err := okcli.AssetBalances(ctx)
	if err != nil {
		return 0, err
	}
	for _, r := range rows {
		if r.Ccy == "USDT" {
			var v float64
			fmt.Sscanf(r.AvailBal, "%f", &v)
			return v, nil
		}
	}
	return 0, nil
}
