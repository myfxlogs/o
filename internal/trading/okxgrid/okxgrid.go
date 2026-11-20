// Package okxgrid —— OKX 原生网格 bot 适配器（工单#25840：venue 原生
// 策略产品=免费执行面——O 干选参（gridfit 判词卡）/拉起/监控/记账/拔线，
// 不自建网格执行径；OKX 网格引擎被几百万用户跑过实证面）。
// 协议面取证在案（ops_event#26084 字段级全谱）：本件字段形与
// docs-v5 tradingBot/grid 族钉齐。接口面选型理由（票面②）：不走
// execution.Executor（Place/Resolve/Cancel 腿级语义）——venue 原生
// bot 语义是「拉起 algo 单」非「下单」，腿生命周期归 venue 引擎自管；
// 故独立薄包+Doer 传输面（okxexec.Client 满足——h1/tcp4 钉死口径同享）。
package okxgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Doer 签名传输面（*okxexec.Client 满足；mutating 面须常规件——
// ReadOnly 件 Post 结构拒，拔线/拉起径天然闸在传输层）。测试桩替。
type Doer interface {
	Get(ctx context.Context, pathWithQuery string) (json.RawMessage, error)
	Post(ctx context.Context, path string, body []byte) (json.RawMessage, error)
}

// Adapter tradingBot/grid 族端点薄封装（零业务语义——字段透传+信封解）。
type Adapter struct{ c Doer }

// NewAdapter 装配（注入签名传输件——凭据不出本包）。
func NewAdapter(c Doer) *Adapter { return &Adapter{c: c} }

// ── 拉起面 ──

// LaunchReq spot grid 拉起参（algoOrdType 恒 "grid"——本族现货域；
// contract_grid 面取证在案另票再接）。gridfit 判词映射：maxPx=band_hi、
// minPx=band_lo、gridNum=档数、quoteSz=投入额（USDT）。
type LaunchReq struct {
	InstID   string  `json:"instId"`
	MaxPx    float64 `json:"-"`
	MinPx    float64 `json:"-"`
	GridNum  int     `json:"-"`
	QuoteSz  float64 `json:"-"`           // 报价币投入额（与 baseSz 二选一——本族恒 quote）
	RunType  string  `json:"runType"`     // "1"算术/"2"几何——缺省算术
	AlgoClID string  `json:"algoClOrdId"` // O 侧追溯键（英数≤32）
	Tag      string  `json:"tag"`
}

// Launch POST /tradingBot/grid/order-algo → algoId（限速 20/2s UID+instId；
// stopType 随 triggerParams 缺省不挂——拔线归显式 Stop 径非条件触发）。
func (a *Adapter) Launch(ctx context.Context, r LaunchReq) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"instId":      r.InstID,
		"algoOrdType": "grid",
		"maxPx":       fmtNum(r.MaxPx),
		"minPx":       fmtNum(r.MinPx),
		"gridNum":     fmt.Sprintf("%d", r.GridNum),
		"runType":     orDef(r.RunType, "1"),
		"quoteSz":     fmtNum(r.QuoteSz),
		"algoClOrdId": r.AlgoClID,
		"tag":         r.Tag,
	})
	var out []struct {
		AlgoID string `json:"algoId"`
		SCode  string `json:"sCode"`
		SMsg   string `json:"sMsg"`
	}
	if err := a.post(ctx, "/api/v5/tradingBot/grid/order-algo", body, &out); err != nil {
		return "", err
	}
	if len(out) == 0 || out[0].AlgoID == "" {
		return "", fmt.Errorf("order-algo 空回执")
	}
	return out[0].AlgoID, nil
}

// Stop POST /tradingBot/grid/stop-order-algo（批量≤10/请求——本族单调
// 逐停）。stopType：现货 "1"=卖基币回收 quote /"2"=留基币（kill 径
// 钦定回收——拔线语义=钱回 quote 非留币敞口）。
func (a *Adapter) Stop(ctx context.Context, algoID, instID, stopType string) error {
	body, _ := json.Marshal([]map[string]string{{
		"algoId": algoID, "instId": instID,
		"algoOrdType": "grid", "stopType": stopType,
	}})
	var out []struct {
		AlgoID string `json:"algoId"`
		SCode  string `json:"sCode"`
		SMsg   string `json:"sMsg"`
	}
	if err := a.post(ctx, "/api/v5/tradingBot/grid/stop-order-algo", body, &out); err != nil {
		return err
	}
	if len(out) == 0 {
		return fmt.Errorf("stop-order-algo 空回执")
	}
	return nil
}

// WithdrawIncome POST /tradingBot/grid/withdraw-income → 收割额
// （现货族专属——已实现利润拔回可用余额）。
func (a *Adapter) WithdrawIncome(ctx context.Context, algoID string) (float64, error) {
	body, _ := json.Marshal(map[string]string{"algoId": algoID})
	var out []struct {
		Profit string `json:"profit"`
	}
	if err := a.post(ctx, "/api/v5/tradingBot/grid/withdraw-income", body, &out); err != nil {
		return 0, err
	}
	if len(out) == 0 {
		return 0, fmt.Errorf("withdraw-income 空回执")
	}
	return parseF(out[0].Profit), nil
}

// ── 读面 ──

// Details GET orders-algo-details 全字段谱（实盘实测键集——权益/网格利/
// 浮盈/费/档数/状态全在）。state：starting|running|stopping|stopped|
// no_close_position|pending_signal；stopResult 停后残态码。
type Details struct {
	AlgoID              string          `json:"algoId"`
	AlgoClOrdID         string          `json:"algoClOrdId"`
	InstID              string          `json:"instId"`
	InstType            string          `json:"instType"`
	State               string          `json:"state"`
	StopResult          string          `json:"stopResult"`
	Investment          string          `json:"investment"`
	GridProfit          string          `json:"gridProfit"`  // 已实现网格利（收割源）
	FloatProfit         string          `json:"floatProfit"` // 浮盈
	TotalPnl            string          `json:"totalPnl"`
	PnlRatio            string          `json:"pnlRatio"`
	ArbitrageNum        string          `json:"arbitrageNum"` // 套利成交次数
	TradeNum            string          `json:"tradeNum"`
	AnnualizedRate      string          `json:"annualizedRate"`
	TotalAnnualizedRate string          `json:"totalAnnualizedRate"`
	PerMaxProfitRate    string          `json:"perMaxProfitRate"`
	PerMinProfitRate    string          `json:"perMinProfitRate"`
	RunPx               string          `json:"runPx"`
	GridNum             string          `json:"gridNum"`
	MaxPx               string          `json:"maxPx"`
	MinPx               string          `json:"minPx"`
	QuoteSz             string          `json:"quoteSz"`
	BaseSz              string          `json:"baseSz"`
	CurBaseSz           string          `json:"curBaseSz"` // 当前基币仓量
	CurQuoteSz          string          `json:"curQuoteSz"`
	SingleAmt           string          `json:"singleAmt"`
	Fee                 string          `json:"fee"`
	FeeCcy              string          `json:"feeCcy"`
	FundingFee          string          `json:"fundingFee"`
	CTime               string          `json:"cTime"`
	UTime               string          `json:"uTime"`
	Raw                 json.RawMessage `json:"-"` // 原响应行（快照账存全量）
}

func (a *Adapter) Details(ctx context.Context, algoID string) (*Details, error) {
	var out []Details
	q := url.Values{"algoOrdType": {"grid"}, "algoId": {algoID}}.Encode()
	if err := a.get(ctx, "/api/v5/tradingBot/grid/orders-algo-details?"+q, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("algoId=%s details 空——venue 侧单已消（stale）", algoID)
	}
	d := out[0]
	return &d, nil
}

// Pending 在飞 bot 单全列（orders-algo-pending——监控面对账源）。
func (a *Adapter) Pending(ctx context.Context) ([]Details, error) {
	var out []Details
	q := url.Values{"algoOrdType": {"grid"}}.Encode()
	if err := a.get(ctx, "/api/v5/tradingBot/grid/orders-algo-pending?"+q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubOrder 子单（live|filled——网格引擎自管腿，记账不逐腿只审计用）。
type SubOrder struct {
	OrdID   string `json:"ordId"`
	InstID  string `json:"instId"`
	State   string `json:"state"`
	Px      string `json:"px"`
	Sz      string `json:"sz"`
	AvgPx   string `json:"avgPx"`
	AccFill string `json:"accFillSz"`
	Fee     string `json:"fee"`
	FeeCcy  string `json:"feeCcy"`
	Pnl     string `json:"pnl"`
	Side    string `json:"side"`
	CTime   string `json:"cTime"`
}

func (a *Adapter) SubOrders(ctx context.Context, algoID, typ string) ([]SubOrder, error) {
	var out []SubOrder
	q := url.Values{"algoOrdType": {"grid"}, "algoId": {algoID}, "type": {typ}}.Encode()
	if err := a.get(ctx, "/api/v5/tradingBot/grid/sub-orders?"+q, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ── 公共面（无鉴权——AI 参/格数界/最小投） ──

// AIParam AI 建议参数（公共端点——取证实测 BTC-USDT 7D 返 gridNum=67）。
type AIParam struct {
	InstID         string `json:"instId"`
	Duration       string `json:"duration"`
	GridNum        string `json:"gridNum"`
	MaxPx          string `json:"maxPx"`
	MinPx          string `json:"minPx"`
	AnnualizedRate string `json:"annualizedRate"`
	MinInvestment  string `json:"minInvestment"`
	Ccy            string `json:"ccy"`
	RunType        string `json:"runType"`
}

func (a *Adapter) AIParam(ctx context.Context, instID, duration string) (*AIParam, error) {
	var out []AIParam
	q := url.Values{"algoOrdType": {"grid"}, "instId": {instID}, "duration": {orDef(duration, "7D")}}.Encode()
	if err := a.get(ctx, "/api/v5/tradingBot/grid/ai-param?"+q, &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ai-param 空")
	}
	p := out[0]
	return &p, nil
}

// get/post 信封解（code=0 已由传输层断言——data 段直解）。
func (a *Adapter) get(ctx context.Context, path string, out any) error {
	raw, err := a.c.Get(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (a *Adapter) post(ctx context.Context, path string, body []byte, out any) error {
	raw, err := a.c.Post(ctx, path, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func orDef(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
