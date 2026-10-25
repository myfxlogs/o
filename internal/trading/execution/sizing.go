// sizing 双腿乐器派生+数量换算（T-0033 返修）：leg1=strategy.inst_id(SWAP 卖)、
// leg2=派生现货（去 "-SWAP" 尾——派生败 fail-closed 拒构）。sz 换算=
// perp floor(notional/ctVal)、spot notional/px（价源 md_ticks 最新行就近取），
// sz 换算必须 ≥minSz——不足=fail-closed（最小张都起不来=拒）。
package execution

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InstMeta 乐器元数据（/public/instruments 公开只读——钦定保留面）。
// CtValCcy=张面值币种（T-0037 根修）：BTC-USDT-SWAP ctVal=0.01 是 **BTC 本位**
// 面值非 USD——币本位面值须乘 perp 价换 USD；ctValCcy=USD/USDT 才可直除。
type InstMeta struct {
	CtVal    float64
	LotSz    float64
	MinSz    float64
	CtValCcy string
}

// perpContracts perp 张数换算（T-0037 根修）：ctValCcy=USD/USDT → 张=usd/ctVal；
// 币本位（base coin，如 BTC）→ 张=usd/(ctVal×px)——px 缺（≤0）=拒构不虚构。
func perpContracts(usd, ctVal float64, ctValCcy string, px float64) (float64, error) {
	if ctVal <= 0 {
		return 0, fmt.Errorf("ctVal 非法: %v", ctVal)
	}
	if ctValCcy == "USD" || ctValCcy == "USDT" {
		return usd / ctVal, nil
	}
	if px <= 0 {
		return 0, fmt.Errorf("币本位 ctValCcy=%s 须 px 换算（px 缺拒构）", ctValCcy)
	}
	return usd / (ctVal * px), nil
}

// MetaLoader 元数据拉取口（cmd/openfirst 装配注入 okxexec——公开端点零凭据径）。
type MetaLoader func(ctx context.Context, instID string) (InstMeta, error)

// DeriveSpotInst 派生现货乐器（去 "-SWAP" 尾——派生败 fail-closed 拒构）。
func DeriveSpotInst(instID string) (string, error) {
	if !strings.HasSuffix(instID, "-SWAP") {
		return "", fmt.Errorf("非 SWAP 乐器不可派生现货: %s", instID)
	}
	spot := strings.TrimSuffix(instID, "-SWAP")
	if spot == "" || spot == instID {
		return "", fmt.Errorf("派生现货败: %s", instID)
	}
	return spot, nil
}

// LatestPx 最新价（md_ticks∪md_mark_index 末帧就近取——T-0132 分级后非活跃
// 集无 tick 行，mark/index 价续供；事实行流径非 REST 轮询）。
func LatestPx(ctx context.Context, pool *pgxpool.Pool, instID string) (float64, error) {
	var px float64
	var ts time.Time
	if err := pool.QueryRow(ctx,
		`SELECT v,ts FROM (SELECT last v,ts FROM md_ticks WHERE inst_id=$1
			UNION ALL SELECT px,ts FROM md_mark_index WHERE inst_id=$1) q
			ORDER BY ts DESC LIMIT 1`,
		instID).Scan(&px, &ts); err != nil {
		return 0, fmt.Errorf("md_ticks/md_mark_index 无 %s 行: %w", instID, err)
	}
	if time.Since(ts) > 60*time.Second {
		return 0, fmt.Errorf("md_ticks/md_mark_index %s 陈（%v>60s）", instID, time.Since(ts))
	}
	return px, nil
}

// BuildLegs 双腿构（perp=floor(notional/ctVal)、spot=notional/px；minSz 闸+乐器异断言）。
func BuildLegs(ctx context.Context, pool *pgxpool.Pool, loadMeta MetaLoader,
	venue, instID string, notional float64) ([2]Leg, string, error) {
	spotInst, err := DeriveSpotInst(instID)
	if err != nil {
		return [2]Leg{}, "", err
	}
	perpM, err := loadMeta(ctx, instID)
	if err != nil {
		return [2]Leg{}, spotInst, fmt.Errorf("perp 元数据: %w", err)
	}
	spotM, err := loadMeta(ctx, spotInst)
	if err != nil {
		return [2]Leg{}, spotInst, fmt.Errorf("spot 元数据: %w", err)
	}
	px, err := LatestPx(ctx, pool, spotInst)
	if err != nil {
		return [2]Leg{}, spotInst, err
	}
	// px_perp=SWAP 腿自身最新价（ctVal 币本位换算用——T-0037 根修主径）。
	pxPerp, err := LatestPx(ctx, pool, instID)
	if err != nil {
		return [2]Leg{}, spotInst, err
	}
	half := notional / 2
	// perp: 张数=usd/(ctVal×px_perp)（币本位）或 usd/ctVal（USD 面值）；spot: 数量=usd/px。
	perpRaw, err := perpContracts(half, perpM.CtVal, perpM.CtValCcy, pxPerp)
	if err != nil {
		return [2]Leg{}, spotInst, err
	}
	// 先落 lot 网格再过 minSz 闸（次病灶：raw<1 先 floor 整数=0 再闸必拒——
	// 币本位张数普遍 <1 张，网格取整才是合法最小可下单量形）。
	perpSz := floorLot(perpRaw, perpM.LotSz)
	if perpSz < perpM.MinSz {
		return [2]Leg{}, spotInst, fmt.Errorf("perp 最小张起不来: sz=%.4f<minSz=%.4f", perpSz, perpM.MinSz)
	}
	// T-0062 根修①：perp 落网格后的实名义额反推 spot 腿（lot 砍残失衡=
	// delta_drift 绞肉环根因——出生即对齐带内）。USD 面值张=ctVal 直乘；
	// 币本位=ctVal×pxPerp 换 USD。
	perpUSD := perpSz * perpM.CtVal
	if perpM.CtValCcy != "USD" && perpM.CtValCcy != "USDT" {
		perpUSD *= pxPerp
	}
	// T-0064 追加决策（业主问案采纳）：spot 腿改 tgtCcy=quote_ccy 市价径——
	// sz=USDT 额（=perpUSD）venue 内部换算，免 base 侧 lotSz floor（失衡残差
	// 从粒度级→舍入级）；官方限域=币币市价单（executor 只对 cash+market+buy
	// 带 TgtCcy，限价/卖/合约径不沾——docs-v5 原文钉）。
	if perpUSD < spotM.MinSz*px {
		return [2]Leg{}, spotInst, fmt.Errorf("spot 最小额起不来: usd=%.4f<minSz=%.6f×px", perpUSD, spotM.MinSz)
	}
	return [2]Leg{
		{Venue: venue, InstID: instID, Side: "sell", Sz: strconv.FormatFloat(perpSz, 'f', -1, 64), Notional: half},
		{Venue: venue, InstID: spotInst, Side: "buy", Sz: strconv.FormatFloat(perpUSD, 'f', -1, 64), Notional: half, TgtQuote: true},
	}, spotInst, nil
}

// floorLot lot 网格向下取整（lot≤0=整数 floor）。
func floorLot(v, lot float64) float64 {
	if lot <= 0 {
		return math.Floor(v)
	}
	return math.Floor(v/lot) * lot
}

// RouterExecutor 跨 venue 路由（map[venue]Executor——腿按 Venue 分派，互不串）。
type RouterExecutor map[string]Executor

func (r RouterExecutor) Place(ctx context.Context, leg Leg, id string) (OrderResult, error) {
	ex, ok := r[leg.Venue]
	if !ok {
		return OrderResult{ClOrdID: id, State: "unknown"}, fmt.Errorf("venue 无执行件: %s", leg.Venue)
	}
	return ex.Place(ctx, leg, id)
}
func (r RouterExecutor) Resolve(ctx context.Context, leg Leg, id string) (OrderResult, error) {
	ex, ok := r[leg.Venue]
	if !ok {
		return OrderResult{ClOrdID: id, State: "unknown"}, fmt.Errorf("venue 无执行件: %s", leg.Venue)
	}
	return ex.Resolve(ctx, leg, id)
}

// CFD 腿构（mt5 侧 lot 换算——SymbolParams contract_size 系；lot<min_lot fail-closed；
// T-0037 根修：perp 张数同 BuildLegs 径=usd/(perpCtVal×px)——px 即 BTC 价双腿共用）。
// TradeMode=SymbolGroup.TradeMode（T-0039：Exness 平台级 CloseOnly 动态限实勘——
// 下单时点断言 FullAccess 非此拒构，禁读台账快照当放行证；mt5pb 数值枚举 0-4）。
type CfdMeta struct {
	ContractSize, MinLot, LotStep float64
	TradeMode                     int32
}

// BuildCfdLegs 跨 venue 双腿：leg1=okx SWAP 卖（perp 张数 ctVal 系——
// okx 腿 sizing 仍走 BuildLegs 正径的工具侧已在用；本件产出 cfd 腿+拼腿）。
// cfd 腿 inst=cfdInst（exec.cfd_map 键映射 BTC-USDT-SWAP→BTCUSDm——工具侧读）。
func BuildCfdLegs(instID, cfdInst string, notional float64, cfdM CfdMeta, px float64, perpCtVal float64) ([2]Leg, string, error) {
	if cfdInst == "" {
		return [2]Leg{}, "", fmt.Errorf("exec.cfd_map 无 %s 映射——fail-closed", instID)
	}
	// T-0039 可开性预检：TradeMode==FullAccess(4) 断言——CloseOnly(3)/Disabled(0)/
	// 单向限(1/2) 皆拒构（文案带实测值；Exness 动态限=下单时点查非快照）。
	if cfdM.TradeMode != 4 {
		return [2]Leg{}, "", fmt.Errorf("cfd %s TradeMode=%d 非 FullAccess（可开性预检拒——broker 动态限，单腿裸露防线）", cfdInst, cfdM.TradeMode)
	}
	if perpCtVal <= 0 {
		return [2]Leg{}, "", fmt.Errorf("perp ctVal 缺")
	}
	half := notional / 2
	if px <= 0 {
		return [2]Leg{}, "", fmt.Errorf("px 缺")
	}
	lots := math.Floor(half/px/cfdM.ContractSize/cfdM.LotStep) * cfdM.LotStep
	if lots < cfdM.MinLot {
		return [2]Leg{}, "", fmt.Errorf("cfd 最小手数起不来: lots=%.4f<min=%.4f", lots, cfdM.MinLot)
	}
	perpRaw, err := perpContracts(half, perpCtVal, "", px) // CFD 腿无 ctValCcy 面——按币本位径（×px）
	if err != nil {
		return [2]Leg{}, "", err
	}
	perpSz := floorLot(perpRaw, 0)
	if perpSz <= 0 {
		return [2]Leg{}, "", fmt.Errorf("perp 张数=0 拒构")
	}
	return [2]Leg{
		{Venue: "okx", InstID: instID, Side: "sell", Sz: strconv.FormatFloat(perpSz, 'f', -1, 64), Notional: half},
		{Venue: "mt5", InstID: cfdInst, Side: "buy", Sz: strconv.FormatFloat(lots, 'f', -1, 64), Notional: half},
	}, cfdInst, nil
}
