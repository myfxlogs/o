// markpx.go —— venue+inst 最新可代价单源读件（O-41385——4+1 处语义拷贝
// 收编：dirpaper markQ/dirtrig lastPx/dirline mark 子查询/notionalNow/
// sizing LatestPx 同族平行件禁再造）。
//
// 价源语义=O-41382/41384 钦定形逐字：md_ticks 中价式
// COALESCE(NULLIF(last,0),(NULLIF(bid,0)+NULLIF(ask,0))/2)——last>0 用
// last，单边缺=NULL 拒；SWAP 符并集 md_mark_index channel='mark-price'
// 行按 ts DESC LIMIT 1。**鲜度界判权归调用点**——本件只管价源，
// 不判龄（dirFloating 无龄界等各点语义原样保留）。
package legset

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// markPxUnionQ venue="" 形——ticks inst_id 直配+venue<>'sim'（dirtrig/
// notionalNow/dirpaper/LatestPx 现行匹配面）；mark_index 双源同加
// channel='mark-price'+venue<>'sim'（O-41385 收编面——index-tickers
// 通道 px 非 mark 价，dirtrig/notionalNow 旧件无 channel 过滤=潜在
// 串源病灶根修；sim 夹具行显拒同 ticks 径）。
const markPxUnionQ = `SELECT v,src,ts FROM (
		SELECT COALESCE(NULLIF(last,0),(NULLIF(bid,0)+NULLIF(ask,0))/2) v,'md_ticks' src,ts
			FROM md_ticks WHERE inst_id=$1 AND venue<>'sim'
		UNION ALL
		SELECT px,'md_mark_index',ts FROM md_mark_index
			WHERE inst_id=$1 AND channel='mark-price' AND venue<>'sim') q
		ORDER BY ts DESC LIMIT 1`

// markPxUnionVenQ venue!="" 形——双源同加 venue=$1（dirline 现行匹配面
// 逐字保留：venue 域收窄语义不动，含 venue='sim' 演习腿照原样可达）。
const markPxUnionVenQ = `SELECT v,src,ts FROM (
		SELECT COALESCE(NULLIF(last,0),(NULLIF(bid,0)+NULLIF(ask,0))/2) v,'md_ticks' src,ts
			FROM md_ticks WHERE venue=$1 AND inst_id=$2
		UNION ALL
		SELECT px,'md_mark_index',ts FROM md_mark_index
			WHERE venue=$1 AND inst_id=$2 AND channel='mark-price') q
		ORDER BY ts DESC LIMIT 1`

// MarkPx venue+inst 最新可代价——ticks 中价 ∪ mark_index 双源按 ts 就近。
// venue=""=无 venue 谓词（各读点现行匹配形）；venue≠""=双源同加
// venue=$1（dirline 现行形）。返回 px(<=0/NULL/无行=ok=false)+markTS+
// src 胜出源表名（'md_ticks'/'md_mark_index'——dirpaper detail 审计位
// 沿用）——**鲜度界判权归调用点**，本件只管价源。
func MarkPx(ctx context.Context, pool *pgxpool.Pool, venue, instID string) (px float64, markTS time.Time, src string, ok bool) {
	var err error
	if venue == "" {
		err = pool.QueryRow(ctx, markPxUnionQ, instID).Scan(&px, &src, &markTS)
	} else {
		err = pool.QueryRow(ctx, markPxUnionVenQ, venue, instID).Scan(&px, &src, &markTS)
	}
	if err != nil || px <= 0 {
		return 0, time.Time{}, "", false
	}
	return px, markTS, src, true
}
