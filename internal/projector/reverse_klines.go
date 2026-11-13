// reverse_klines.go —— 工单#13065 切片④日K反向桥件（reverse.go 300 棘轮
// 分拆——同包同 Runner，机制=正本副本全表 diff 照 reverseEarnRate 形）。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// md_klines 反向桥——SG o-core/okxws 日K写件正本（fin 链）→PG-md 消费面
// 副本（评估/假想机读侧）。复合键 (venue,inst_id,bar,ts) 全表 diff；
// confirm 翻转=同键覆写非新行（收线语义随正本）。
const qKlinesRev = `SELECT venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm FROM md_klines`

const uKlinesMirror = `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
	VALUES($1,$2,$3,$4::timestamptz,$5,$6,$7,$8,$9,$10,$11)
	ON CONFLICT (venue,inst_id,bar,ts) DO UPDATE SET o=EXCLUDED.o,h=EXCLUDED.h,
		l=EXCLUDED.l,c=EXCLUDED.c,vol=EXCLUDED.vol,vol_ccy=EXCLUDED.vol_ccy,
		confirm=EXCLUDED.confirm`

// reverseKlines 日K正本→副本全表 diff（eval 消费径：假想机/评估面读 PG-md
// 副本，缺径=md 侧永空集）。复合键行判等=全数据列；形成帧覆写至 confirm
// 翻转随正本收敛（recv_ts=副本本地到达戳不入判）。低频小表——期界对账
// 兜底足够，不造新 NOTIFY 范式。
func (r *Runner) reverseKlines(ctx context.Context) error {
	type key struct {
		venue, inst, bar string
		ts               time.Time
	}
	type row struct {
		o, h, l, c, vol, volC float64
		confirm               bool
	}
	fetch := func(pool *pgxpool.Pool) (map[key]row, error) {
		rows, err := pool.Query(ctx, qKlinesRev)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[key]row{}
		for rows.Next() {
			var k key
			var x row
			if err := rows.Scan(&k.venue, &k.inst, &k.bar, &k.ts,
				&x.o, &x.h, &x.l, &x.c, &x.vol, &x.volC, &x.confirm); err != nil {
				return nil, err
			}
			out[k] = x
		}
		return out, rows.Err()
	}
	want, err := fetch(r.sg)
	if err != nil {
		return fmt.Errorf("klines 正本读: %w", err)
	}
	have, err := fetch(r.md)
	if err != nil {
		return fmt.Errorf("klines 副本读: %w", err)
	}
	var batch [][]any
	for k, x := range want {
		if h, ok := have[k]; !ok || h != x {
			batch = append(batch, []any{k.venue, k.inst, k.bar, k.ts,
				x.o, x.h, x.l, x.c, x.vol, x.volC, x.confirm})
		}
	}
	return batchExec(ctx, r.md, uKlinesMirror, batch)
}
