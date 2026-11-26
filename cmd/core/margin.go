package main

// margin.go —— O-40707 margin-short 接线残件（采集+环主件；在借仓拍三件
// 在 margin_loan.go 姊妹件）：
// ① marginCollectOnce 采集细胞——负 funding 可借集+锚币 InterestRates/
//    MaxLoanCcy(ccy-only 形——instId∩ccy 互斥 50024 实证)→md_margin_borrow
//    upsert，评估闸三闸（新鲜度/息尖/额度下限）有粮；
// ② marginBeat 环主件——采集拍+在借仓拍同槽（collect_ivl_s 拍距，
//    okcli=nil/键缺=细胞让位 fail-closed）。
// 全部读面 fail-closed：venue 读败/键缺=跳拍/拒动不虚构。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/trading/okxexec"
)

// marginVenue 借贷/下单/权益/负债读面窄接口（桩测注伪缝——生产=*okxexec.Client）。
type marginVenue interface {
	InterestRates(ctx context.Context, ccys []string) ([]okxexec.MarginRate, error)
	MaxLoanCcy(ctx context.Context, ccy, mgnMode, mgnCcy string) (float64, error)
	SpotManualBorrowRepay(ctx context.Context, ccy string, amt float64, side string) (json.RawMessage, error)
	SpotBorrowRepayHistory(ctx context.Context, ccy, typ string, limit int) ([]okxexec.BorrowRepayHistRow, error)
	PlaceOrder(ctx context.Context, p okxexec.OrderParams) (json.RawMessage, error)
	ResolveOrderDetail(ctx context.Context, instID, clOrdID string) (okxexec.OrderState, float64, float64, float64, string, error)
	AccountAdjEquity(ctx context.Context) (float64, error)
	SpotBorrowLiab(ctx context.Context, ccy string) (float64, bool, error)
}

// marginLoan 在借仓行（fin_margin_loan open/closing/flatten_failed 态——
// flatten_failed=平仓序败留账仍计息负债，枚举窗扩后息腿/健康闸/重试
// 三面全续，O-41350）。
type marginLoan struct {
	ID               int64
	Ccy              string
	InstID           string
	Amt              float64
	APRSnap          float64
	Status           string
	StrategyRef      *int64
	PositionRef      *int64
	ExecutionID      *int64
	LastAccrualAt    time.Time
	Venue            string
	AccruedInterest  float64
	FlattenFails     int
	LastFlattenTryAt *time.Time
	FlattenEscalated bool
}

// marginCollectOnce 单拍采集：宇宙=近 24h 负 funding 基币∪锚币
// （BTC/ETH/USDT——spec 钉）；率批量一次拉，额度逐币 ccy-only 形。
// 单币读败=跳该币记日志不拖批（采集面部分产出优于全拒——评估闸按
// 新鲜度自行拒陈币）。
func marginCollectOnce(ctx context.Context, pool *pgxpool.Pool, venue marginVenue) (int, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT split_part(inst_id,'-',1) AS ccy
		FROM md_funding WHERE funding_rate<0 AND ts>now()-interval '1 day'`)
	if err != nil {
		return 0, err
	}
	ccySet := map[string]bool{"BTC": true, "ETH": true, "USDT": true}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err == nil && c != "" {
			ccySet[c] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	ccys := make([]string, 0, len(ccySet))
	for c := range ccySet {
		ccys = append(ccys, c)
	}
	// O-41363：无参全量拉取→本地交集——venue 不吃逗号多币（ccy=A,B 恒
	// 51000 毒批实证）；宇宙币不在响应=venue 无借贷市跳过不产证据行。
	rates, err := venue.InterestRates(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("interest-rate: %w", err)
	}
	rateOf := map[string]float64{}
	for _, r := range rates {
		rateOf[r.Ccy] = r.InterestRate // 每小时小数利率
	}
	n := 0
	for _, ccy := range ccys {
		rate, ok := rateOf[ccy]
		if !ok {
			continue // venue 无该币借贷市——不产证据行（闸按 missing 拒）
		}
		maxLoan, lerr := venue.MaxLoanCcy(ctx, ccy, "cross", "USDT")
		if lerr != nil {
			log.Printf("[margin] max-loan %s 读败跳拍: %v", ccy, lerr)
			continue
		}
		if _, err := pool.Exec(ctx, `INSERT INTO md_margin_borrow(ccy,inst_id,mgn_mode,borrow_apr,max_loan,ts)
			VALUES($1,$2,'cross',$3,$4,now())
			ON CONFLICT (ccy,inst_id,mgn_mode) DO UPDATE SET
			  borrow_apr=EXCLUDED.borrow_apr,max_loan=EXCLUDED.max_loan,ts=EXCLUDED.ts`,
			ccy, ccy+"-USDT", rate*8760, maxLoan); err != nil {
			log.Printf("[margin] md_margin_borrow %s 写败: %v", ccy, err)
			continue
		}
		n++
	}
	return n, nil
}

// marginSpotPx 现货对最新价（diff_px 同两级读面——md_ticks.last∪
// md_mark_index.px，10min 鲜度窗；读败/无鲜行=0 由调用面按不可证
// fail-closed。注：md_ticks 列名=last 非 px——borrowquote.spotPx 同
// UNION 写法对 md_ticks 段结构性读败恒 0=额度恒不可证 fail-closed 兜底，
// 本件按真列名落地）。
func marginSpotPx(ctx context.Context, pool *pgxpool.Pool, instID string) float64 {
	var px float64
	_ = pool.QueryRow(ctx, `SELECT v FROM (
		SELECT last v, ts FROM md_ticks WHERE inst_id=$1 AND venue<>'sim'
		UNION ALL SELECT px, ts FROM md_mark_index WHERE inst_id=$1
	) q WHERE ts > now()-interval '10 minutes' ORDER BY ts DESC LIMIT 1`, instID).Scan(&px)
	return px
}

// marginBeat 环主件——首拍即跑（在借仓回填息腿）此后 cfg 拍距。
// okcli=nil=凭据缺席整环让位（venue_pos_beat 同约）；collect_ivl_s 键缺
// =细胞让位不猜（fail-closed——拍距无权威值不裸跑）。
func marginBeat(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, okcli *okxexec.Client) {
	if okcli == nil {
		return
	}
	ivl, err := cfg.GetFloat("margin.collect_ivl_s")
	if err != nil || ivl <= 0 {
		log.Printf("[margin] collect_ivl_s 键缺/非正——细胞让位: %v", err)
		return
	}
	for {
		if n, err := marginCollectOnce(ctx, pool, okcli); err != nil {
			log.Printf("[margin] 采集拍败（下拍重试）: %v", err)
		} else if n > 0 {
			log.Printf("[margin] 采集拍毕：%d 币证据行", n)
		}
		if err := marginLoanTick(ctx, pool, cfg, okcli); err != nil && ctx.Err() == nil {
			log.Printf("[margin] 在借仓拍败（下拍重试）: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(ivl * float64(time.Second))): // sched-exempt-ok 会话内拍
		}
	}
}
