// optchain_store.go —— O-32540 期权链采集面·装配臂（optchain.go
// 目录供给件的 Collector 侧挂点：订阅帧装配+enum 快照落表+周期
// 快照臂——棘轮拆分件，同族语义见 optchain.go 头注）。
package okxws

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

// optSubscribe 期权订阅装配（dialLoop 内挂点——目录新鲜化+enum 快照
// 落表+订阅帧；读败/订阅败=跳过期权面不炸主连接——加法件 fail-open
// 语义：期权证据缺席不拖既有宇宙面）。
func (c *Collector) optSubscribe(ctx context.Context, conn *websocket.Conn) {
	if c.Opt == nil {
		return
	}
	fresh, err := c.Opt.Refresh(ctx)
	if err != nil {
		c.rej.Inc("univ_expand_err")
		log.Printf("[okxws] opt 目录读败跳过期权订阅（主面不受累）: %v", err)
		return
	}
	if fresh {
		c.optStoreInstruments(ctx)
	}
	args := c.Opt.SubscribeArgs()
	if len(args) == 0 {
		return
	}
	if err := subscribeAll(conn, args); err != nil {
		log.Printf("[okxws] opt subscribe 败（期权面跳过重连自愈）: %v", err)
		return
	}
	log.Printf("[okxws] opt 订阅 args=%d 活跃集=%d", len(args), len(c.Opt.Active()))
}

// optStoreInstruments enum 快照 upsert（fresh 轮才写——全态行
// ~4k/轮，幂等冲突更新；写败计 write_err 不炸订阅径）。
func (c *Collector) optStoreInstruments(ctx context.Context) {
	insts := c.Opt.AllInstruments()
	if len(insts) == 0 {
		return
	}
	b := &pgx.Batch{}
	n := 0
	for _, in := range insts {
		b.Queue(`INSERT INTO md_opt_instruments(venue,inst_id,inst_family,uly,opt_type,
			exp_time,strike,ct_type,settle_ccy,state,tick_sz,lot_sz,min_sz,ct_val,ct_val_ccy,list_time)
			VALUES('okx',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			ON CONFLICT (venue,inst_id) DO UPDATE SET inst_family=EXCLUDED.inst_family,
			uly=EXCLUDED.uly,opt_type=EXCLUDED.opt_type,exp_time=EXCLUDED.exp_time,
			strike=EXCLUDED.strike,ct_type=EXCLUDED.ct_type,settle_ccy=EXCLUDED.settle_ccy,
			state=EXCLUDED.state,tick_sz=EXCLUDED.tick_sz,lot_sz=EXCLUDED.lot_sz,
			min_sz=EXCLUDED.min_sz,ct_val=EXCLUDED.ct_val,ct_val_ccy=EXCLUDED.ct_val_ccy,
			list_time=EXCLUDED.list_time,recv_ts=now()`,
			in.InstID, in.InstFamily, in.Uly, in.OptType,
			nullTime(in.ExpTime), in.Strike, in.CtType, in.SettleCcy, in.State,
			in.TickSz, in.LotSz, in.MinSz, in.CtVal, in.CtValCcy, nullTime(in.ListTime))
		n++
	}
	if n == 0 {
		return
	}
	if err := c.pool.SendBatch(ctx, b).Close(); err != nil {
		c.rej.Inc("write_err")
		log.Printf("[okxws] opt instruments 落表败：行=%d: %v", n, err)
	}
}

// nullTime 零值→NULL（pgx 不认零值 time.Time 语义歧义——显式 nil 更诚实）。
func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// optSnapshotLoop 周期快照臂（optSnapEvery 拍——活跃子集 md_opt_summary
// 行全量追加 md_opt_snapshot；读写同 pool 直走不占帧队列——低频面）。
func (c *Collector) optSnapshotLoop(ctx context.Context) {
	if c.Opt == nil {
		return // 期权面未装=臂停（装配层负全责）
	}
	t := time.NewTicker(optSnapEvery) // sched-exempt-ok 低频证据窗定节拍
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.optSnapshot(ctx)
		}
	}
}

// optSnapshot 单拍：活跃子集 summary 行→snapshot 追加（拍点同值
// snapshot_ts；子集空/读败=跳过不炸——证据窗缺拍优于臂死）。
func (c *Collector) optSnapshot(ctx context.Context) {
	active := c.Opt.Active()
	if len(active) == 0 {
		return
	}
	rows, err := c.pool.Query(ctx,
		`SELECT inst_id,ts,delta,gamma,theta,vega,mark_vol,bid_vol,ask_vol,
			vol_lv,fwd_px,distance,buy_apr,sell_apr,lever
		 FROM md_opt_summary WHERE venue='okx' AND inst_id = ANY($1)`, active)
	if err != nil {
		c.rej.Inc("write_err")
		log.Printf("[okxws] opt snapshot 读败跳过: %v", err)
		return
	}
	defer rows.Close()
	// 拍点截到 optSnapEvery 桶界——同桶重入（环叠拍/手动补拍）冲突
	// DO NOTHING 幂等不膨胀；桶=快照归属时段非调用瞬时。
	snap := time.Now().UTC().Truncate(optSnapEvery)
	b := &pgx.Batch{}
	n := 0
	for rows.Next() {
		var instID string
		var ts time.Time
		var delta, gamma, theta, vega, mv, bv, av, vl, fp, dist, ba, sa, lv *float64
		if err := rows.Scan(&instID, &ts, &delta, &gamma, &theta, &vega, &mv,
			&bv, &av, &vl, &fp, &dist, &ba, &sa, &lv); err != nil {
			c.rej.Inc("write_err")
			return
		}
		b.Queue(`INSERT INTO md_opt_snapshot(snapshot_ts,venue,inst_id,src_ts,
			delta,gamma,theta,vega,mark_vol,bid_vol,ask_vol,vol_lv,fwd_px,
			distance,buy_apr,sell_apr,lever)
			VALUES($1,'okx',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
			ON CONFLICT (snapshot_ts,venue,inst_id) DO NOTHING`,
			snap, instID, ts, delta, gamma, theta, vega, mv, bv, av, vl, fp,
			dist, ba, sa, lv)
		n++
	}
	if err := rows.Err(); err != nil {
		c.rej.Inc("write_err")
		return
	}
	if n == 0 {
		return
	}
	if err := c.pool.SendBatch(ctx, b).Close(); err != nil {
		c.rej.Inc("write_err")
		log.Printf("[okxws] opt snapshot 落表败：行=%d: %v", n, err)
	}
}
