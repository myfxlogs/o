package ledger

// diff_px.go —— O-38064 非本位键折价+币量豁免判差（run.go 300 行棘轮
// 拆件）。病灶实证（spec 取证节）：无价键走 tol+exempt+1 强制式——
// 豁免加多少告警涨多少的自举怪圈+隐含价十倍摆像；有价键取价无鲜度界
// （BTC 吃到陈 7 日的行）；币量差被冠 $ 印出。修法三件：
// S1 diffPx 两级鲜价源（md_tick_scan 本键 venue→md_ticks∪md_mark_index，
// 皆须 ts>now()-10min——ScanPoller 60s 拍×10）；S2 qty:<units> 币量豁免
// （币量空间先扣不经价）；S3 币量残量 rq 折价判差，无价 fail-closed 照报
// 不产出美元数（无价≠静默）。USDT/USD 键路径逐字节不动。

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// diffPxMaxAge 折价鲜度界——md_tick_scan 由 ScanPoller 60s 拍驱动，
// 10min=10 拍容差（陈行=无价按 fail-closed 照报不估）。
const diffPxMaxAge = 10 * time.Minute

// isBaseCCY 本位币键（USDT/USD——判差/文案逐字节守恒域，spec 硬闸）。
func isBaseCCY(ccy string) bool { return ccy == "USDT" || ccy == "USD" }

// qtyFmt 币量格式——≤8 位小数去尾零（BTC 0.0217 与 0.02 可辨；
// 51758586.877993 级币量保真；币量永不冠 $）。
func qtyFmt(v float64) string {
	s := strconv.FormatFloat(v, 'f', 8, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

// diffOut 非本位键判差产出（S3 三分流载体——币量残量+鲜价折算；
// alertLedgerDiff/ledgerDiffHuman 消费用，nil=本位键旧径）。
type diffOut struct {
	qty    float64 // 币量残差 rq（币量豁免后）
	priced bool    // 有鲜行折价成功
	usd    float64 // rq*px（priced 时——无价键不产出美元数）
}

// diffPx 非本位币→USDT 折价（S1 取数序）：
// ① md_tick_scan（venue=本键 venue，inst_id=ccy||'-USDT'，last>0——
//
//	venue /market/tickers 采集面库内现成，当前未被本查询读取）；
//
// ② md_ticks（venue<>'sim'）∪ md_mark_index（现行源）。
// 两级皆须 ts 鲜行（>now()-10min）；无鲜行 ok=false。
// 查询 err 禁吞（现行 `_ =` 判例修——#10063③ 同形）：rej recon_probe_failed
// +WARN+ok=false。不新增 REST 调用（库内现成面）。
func (r *Runner) diffPx(ctx context.Context, venue, ccy string) (float64, bool) {
	inst := ccy + "-USDT"
	var px float64
	err := r.pool.QueryRow(ctx, `SELECT last FROM md_tick_scan
		WHERE venue=$1 AND inst_id=$2 AND last>0
		  AND ts > now()-$3::interval`, venue, inst, diffPxMaxAge.String()).Scan(&px)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 折价读败 md_tick_scan %s/%s——按无价照报: %v", venue, ccy, err)
		return 0, false
	}
	if px > 0 {
		return px, true
	}
	err = r.pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks
		WHERE inst_id=$1||'-USDT' AND venue<>'sim' UNION ALL
		SELECT px,ts FROM md_mark_index WHERE inst_id=$1||'-USDT') q
		WHERE ts > now()-$2::interval ORDER BY ts DESC LIMIT 1`, ccy, diffPxMaxAge.String()).Scan(&px)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		r.rej.Inc("recon_probe_failed")
		log.Printf("WARN [ledger] 折价读败 md_ticks/mark_index %s/%s——按无价照报: %v", venue, ccy, err)
		return 0, false
	}
	if px > 0 {
		return px, true
	}
	return 0, false
}
