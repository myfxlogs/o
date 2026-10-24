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
type InstMeta struct {
	CtVal float64
	LotSz float64
	MinSz float64
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

// LatestPx 最新价（md_ticks 就近取——事实行流径非 REST 轮询）。
func LatestPx(ctx context.Context, pool *pgxpool.Pool, instID string) (float64, error) {
	var px float64
	var ts time.Time
	if err := pool.QueryRow(ctx,
		"SELECT last,ts FROM md_ticks WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1",
		instID).Scan(&px, &ts); err != nil {
		return 0, fmt.Errorf("md_ticks 无 %s 行: %w", instID, err)
	}
	if time.Since(ts) > 60*time.Second {
		return 0, fmt.Errorf("md_ticks %s 陈（%v>60s）", instID, time.Since(ts))
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
	half := notional / 2
	// perp: 张数=floor(notional/ctVal)；spot: 数量=notional/px——均过 minSz 闸。
	perpSz := math.Floor(half / perpM.CtVal)
	if perpSz < perpM.MinSz {
		return [2]Leg{}, spotInst, fmt.Errorf("perp 最小张起不来: sz=%.0f<minSz=%.4f", perpSz, perpM.MinSz)
	}
	spotSz := half / px
	if spotSz < spotM.MinSz {
		return [2]Leg{}, spotInst, fmt.Errorf("spot 最小张起不来: sz=%.6f<minSz=%.6f", spotSz, spotM.MinSz)
	}
	if perpM.LotSz > 0 {
		perpSz = math.Floor(perpSz/perpM.LotSz) * perpM.LotSz
	}
	if spotM.LotSz > 0 {
		spotSz = math.Floor(spotSz/spotM.LotSz) * spotM.LotSz
	}
	return [2]Leg{
		{Venue: venue, InstID: instID, Side: "sell", Sz: strconv.FormatFloat(perpSz, 'f', -1, 64), Notional: half},
		{Venue: venue, InstID: spotInst, Side: "buy", Sz: strconv.FormatFloat(spotSz, 'f', -1, 64), Notional: half},
	}, spotInst, nil
}
