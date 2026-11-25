package ledger

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FundCcy 基金的钱币种集（唯一真源——T-0095 v2 起版，T-0100 账户域作用化）：
// 键={作用域,ccy}，作用域=venue:account（okx:trading/funding/earn）或裸 venue
// （mt5=基金专属券商户全计，USD 本位恒在册）。三分量：
//   - USDT 本位：okx 三账户域全在册；
//   - **open** filled 腿基币：只落 trading 域（ETH-USDT→ETH——活仓腿是基金钱；
//     Earn/资金户同币尘=托管非基金钱，T-0100 ①渗漏根修）；
//   - 停泊溯源币：fin_risk_decision op='residual_park' 成功凭据币种落
//     funding/earn 域（v3 残币活期归口=基金可动钱回集，T-0100 ②）。
//
// 「钱在哪」分区过滤、venue 侧对拍白名单、earn 残币归口判据同源消费。
func FundCcy(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]bool, error) {
	out := map[[2]string]bool{}
	// O-39440：种子子=fin_account active 行——账户域键=account_id 直发
	// （okx:trading/funding/earn/dcd+mt5:<login>——venue 桶折叠径退役,
	// 对拍面账户粒度;读面缺席=空集不炸,下游键面照旧 scope 形）。
	arows, err := pool.Query(ctx,
		`SELECT account_id,ccy FROM fin_account WHERE state='active'`)
	if err != nil {
		return nil, err
	}
	for arows.Next() {
		var aid, ccy string
		if err := arows.Scan(&aid, &ccy); err != nil {
			arows.Close()
			return nil, err
		}
		if ccy != "" {
			out[[2]string{aid, ccy}] = true
		}
	}
	arows.Close()
	// O-39440：腿基币键=账户域——join fin_account 解析归属
	// （sub_type='trading' 直中；mt5 类无 trading 子型→active main 全户
	// 兜底——腿 venue 不载 login,双户同键白名单=宁宽勿漏,持币户自证
	// 差、零币户 eq=net=0 天然平;frozen/无册 venue 不出键=对拍面缺席
	// 同旧 venue 桶无读同果）。
	rows, err := pool.Query(ctx, `SELECT DISTINCT fa.account_id,split_part(l.inst_id,'-',1)
		FROM fin_order_leg l JOIN fin_position p ON p.execution_id=l.execution_id
		JOIN fin_account fa ON fa.venue=l.venue AND fa.state='active'
		  AND fa.sub_type IN ('trading','main')
		WHERE p.state<>'closed' AND l.state='filled'`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var v, c string
		if err := rows.Scan(&v, &c); err != nil {
			rows.Close()
			return nil, err
		}
		out[[2]string{v, c}] = true
	}
	rows.Close()
	prows, err := pool.Query(ctx, `SELECT DISTINCT intent->>'venue',intent->>'ccy'
		FROM fin_risk_decision WHERE intent->>'op'='residual_park'`)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var v, c string
		if err := prows.Scan(&v, &c); err != nil {
			return nil, err
		}
		out[[2]string{v + ":funding", c}] = true
		out[[2]string{v + ":earn", c}] = true
	}
	// T-0101 第三类别：dcd 基线入册币落 venue:dcd 域（已部署产品持仓=基金的钱，
	// 卡面走专属区不经此域——此域只为对拍 venue 键供集）。工单#22919：同
	// ordId 已落 dcd_settle 凭据=基线退役（结算后不再供 dcd 域键）。
	drows, err := pool.Query(ctx, `SELECT DISTINCT intent->>'venue',intent->>'ccy'
		FROM fin_risk_decision r WHERE intent->>'op'='dcd_baseline'
		AND NOT EXISTS (SELECT 1 FROM fin_risk_decision s
			WHERE s.intent->>'op'='dcd_settle' AND s.intent->>'ordId'=r.intent->>'ordId')`)
	if err != nil {
		return nil, err
	}
	defer drows.Close()
	for drows.Next() {
		var v, c string
		if err := drows.Scan(&v, &c); err != nil {
			return nil, err
		}
		out[[2]string{v + ":dcd", c}] = true
	}
	// 工单#20904(b)：dcd 结算回款币落 venue:funding/earn 域（结算回款入资金户
	// =基金的钱——BTC 等非册币回款永久漏计面根修；venue settleCcy 明告不猜）。
	srows, err := pool.Query(ctx, `SELECT DISTINCT intent->>'venue',intent->>'ccy'
		FROM fin_risk_decision WHERE intent->>'op'='dcd_settle' AND intent->>'ccy'<>''`)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var v, c string
		if err := srows.Scan(&v, &c); err != nil {
			return nil, err
		}
		out[[2]string{v + ":funding", c}] = true
		out[[2]string{v + ":earn", c}] = true
	}
	return out, srows.Err()
}

// ParkedAmt 停泊溯源币台账量（T-0100 ③对拍 ledger 侧按币量建键——
// residual_park 凭据行 Σamt 逐 (venue,ccy)；赎回/出清径未开则台账只增，
// venue 侧余量为零时该键天然缺席不比对）。
func ParkedAmt(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]float64, error) {
	out := map[[2]string]float64{}
	rows, err := pool.Query(ctx, `SELECT intent->>'venue',intent->>'ccy',
		sum((intent->>'amt')::numeric) FROM fin_risk_decision
		WHERE intent->>'op'='residual_park' GROUP BY 1,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v, c string
		var a float64
		if err := rows.Scan(&v, &c, &a); err != nil {
			return nil, err
		}
		out[[2]string{v, c}] += a
	}
	return out, rows.Err()
}

// ReconSets 对拍四件一次出（T-0100/T-0101+O-34287——账户域作用集折叠
// venue×ccy 白名单+停泊台账币数+dcd 基线台账币数+原生币基线台账币数）。
func ReconSets(ctx context.Context, pool *pgxpool.Pool) (wlSet map[[2]string]bool,
	parked, dcdBase, nativeBase map[[2]string]float64, err error) {
	set, err := FundCcy(ctx, pool)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	wlSet = map[[2]string]bool{}
	for k := range set {
		// O-39440：账户域作用键直用为对拍键（account_id==scope 键——
		// venue 桶折叠退役,对拍面账户粒度;无账户行的 scope 键在
		// 对拍循环按 fin_account 枚举自然缺席）。
		wlSet[k] = true
	}
	parked, err = ParkedAmt(ctx, pool)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	dcdBase, err = DcdBaselineAmt(ctx, pool)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	nativeBase, err = NativeBaselineAmt(ctx, pool)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// O-34287 验收⑤：原生币基线键并入对拍域（venue 真值首拍幂等建账——
	// 杂币余额开账位=账本域存在，非白名单静默放行）。
	for k := range nativeBase {
		wlSet[k] = true
	}
	// O-39440：台账组件 venue 级键→账户归户白名单键——parked→earn、
	// dcdBase→dcd、nativeBase→trading（compAccount 同规则含单 main
	// 兜底）；venue 无对应 sub_type=不入对拍（与原 venue 桶无键同果）。
	// intent 域键=venue+ccy 原样不动，归户只发生在对拍键面。
	if accts, aerr := AccountRows(ctx, pool, "active"); aerr == nil {
		subOf := func(v, sub string) string {
			if id := compAccount(accts, v, sub); id != "" {
				return id
			}
			for _, a := range accts {
				if a.Venue == v && a.SubType == sub {
					return a.ID
				}
			}
			return ""
		}
		tradingOf := func(v string) string {
			if id := subOf(v, "trading"); id != "" {
				return id
			}
			return compAccount(accts, v, "trading")
		}
		for k := range parked {
			if id := subOf(k[0], "earn"); id != "" {
				wlSet[[2]string{id, k[1]}] = true
			}
		}
		for k := range dcdBase {
			if id := subOf(k[0], "dcd"); id != "" {
				wlSet[[2]string{id, k[1]}] = true
			}
		}
		for k := range nativeBase {
			if id := tradingOf(k[0]); id != "" {
				wlSet[[2]string{id, k[1]}] = true
			}
		}
	}
	return
}

// NativeBaselineAmt 原生币基线台账量（O-34287 验收⑤：op='native_baseline'
// 凭据行 Σamt 逐 (venue,ccy)——存量非 USDT/USD 资产首拍 venue 真值驱动
// 幂等建 opening 账，形同 dcd_baseline 归因凭据语义；BTC 尘/Earn 杂币/
// TRX 托管币等 FundCcy 未登记币的对账域载体）。
func NativeBaselineAmt(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]float64, error) {
	out := map[[2]string]float64{}
	rows, err := pool.Query(ctx, `SELECT intent->>'venue',intent->>'ccy',
		sum((intent->>'amt')::numeric) FROM fin_risk_decision
		WHERE intent->>'op'='native_baseline'
		GROUP BY 1,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v, c string
		var a float64
		if err := rows.Scan(&v, &c, &a); err != nil {
			return nil, err
		}
		out[[2]string{v, c}] += a
	}
	return out, rows.Err()
}

// DcdBaselineAmt dcd 基线台账量（T-0101：op='dcd_baseline' 凭据行 Σamt 逐
// (venue,ccy)——venue 名义币量对拍的 ledger 侧键；新购/结算 venue 侧缺席或
// 超额=真差告警，基线行=部署决策书面归因凭据）。
func DcdBaselineAmt(ctx context.Context, pool *pgxpool.Pool) (map[[2]string]float64, error) {
	out := map[[2]string]float64{}
	rows, err := pool.Query(ctx, `SELECT intent->>'venue',intent->>'ccy',
		sum((intent->>'amt')::numeric) FROM fin_risk_decision r
		WHERE intent->>'op'='dcd_baseline'
		AND NOT EXISTS (SELECT 1 FROM fin_risk_decision s
			WHERE s.intent->>'op'='dcd_settle' AND s.intent->>'ordId'=r.intent->>'ordId')
		GROUP BY 1,2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v, c string
		var a float64
		if err := rows.Scan(&v, &c, &a); err != nil {
			return nil, err
		}
		out[[2]string{v, c}] += a
	}
	return out, rows.Err()
}
