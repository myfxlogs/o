// venuepos.go —— O-33207 手动仓事实面写件（tradebook.go 拆出——棘轮闸）：
// OpenedOrders 在仓快照→fin_venue_position state='open' upsert+消失
// →'stale' 清出 mtm；OrderHistory 平仓行→state='closed'+realized。
// O 单排除=comment→ownOrder 双面判别（禁入册=双簿防线）。
package mt5

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// venuePosClose 手动仓平仓事实→fin_venue_position（非 O 单专属——
// O 仓在 fin_position 域）。upsert 幂等：OrderHistory 重扫重写同值；
// closed 行不被在仓快照复活（WHERE state<>'closed' 闸——票号不复用
// 防御分支）。open_* 字段取自行级聚合（posTicket 域全周期事实）。
func (c *Collector) venuePosClose(ctx context.Context, a *acctRun, o *mt5pb.Order,
	amt float64, occurred time.Time, acctID string) {
	side := "buy"
	if o.GetDealType() == mt5pb.DealType_DealType_DealSell {
		side = "sell"
	}
	var openTs *time.Time
	if ms := o.GetOpenTimestampUTC(); ms > 0 {
		t := time.UnixMilli(ms).UTC()
		openTs = &t
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_venue_position
		(account_id,venue,ticket,symbol,side,lots,contract_size,open_px,open_ts,ccy,state,close_px,close_ts,realized,last_seen)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'USD','closed',$10,$11,$12,now())
		ON CONFLICT (account_id,ticket) DO UPDATE SET
		  state='closed',close_px=EXCLUDED.close_px,close_ts=EXCLUDED.close_ts,
		  realized=EXCLUDED.realized,last_seen=now()
		WHERE fin_venue_position.state<>'closed'`,
		acctID, a.acct.Venue, strconv.FormatInt(o.GetTicket(), 10), o.GetSymbol(),
		side, o.GetLots(), o.GetContractSize(), o.GetOpenPrice(), openTs,
		o.GetClosePrice(), occurred, amt); err != nil {
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 手动仓平仓事实写败 票=%d: %v", a.acct.Venue, o.GetTicket(), err)
	}
}

// openPollOnce OpenedOrders 在仓快照→fin_venue_position（非 O 单）：
// upsert state='open'（last_seen=now 在仓权威）+快照未见行→'stale'
// 清出 mtm（venue 快照=当前在仓全集——消失=已平，平仓实证由
// OrderHistory 后续翻 'closed';读败整拍跳过不清场=fail-closed）。
// O 单 comment=clOrdID 命中排除——其仓走 fin_position+mtm 既有径。
func (c *Collector) openPollOnce(ctx context.Context, cl *Client, a *acctRun) error {
	rep, err := cl.MT5().OpenedOrders(cl.Authed(ctx),
		&mt5pb.OpenedOrdersRequest{Id: cl.Token()})
	if err != nil {
		return err
	}
	if rep.GetError() != nil {
		return fmt.Errorf("OpenedOrders venue err: %v", rep.GetError())
	}
	acctID := c.acctIDFor(ctx, a)
	if acctID == "" {
		return nil // 账户册缺=注册表契约——跳过不猜（cred_gap 升件径另管）
	}
	seen := make([]string, 0, len(rep.GetResult()))
	for _, o := range rep.GetResult() {
		if c.ownOrder(ctx, o.GetComment()) {
			continue // O 自执行仓——fin_position+mtm 径，不入手动仓册
		}
		side := ""
		switch o.GetOrderType() {
		case mt5pb.OrderType_OrderType_Buy:
			side = "buy"
		case mt5pb.OrderType_OrderType_Sell:
			side = "sell"
		default:
			continue // 挂单/他型非仓——不入册
		}
		if o.GetLots() <= 0 {
			continue
		}
		ticket := strconv.FormatInt(o.GetTicket(), 10)
		seen = append(seen, ticket)
		var openTs *time.Time
		if ms := o.GetOpenTimestampUTC(); ms > 0 {
			t := time.UnixMilli(ms).UTC()
			openTs = &t
		}
		if _, err := c.pool.Exec(ctx, `INSERT INTO fin_venue_position
			(account_id,venue,ticket,symbol,side,lots,contract_size,open_px,open_ts,ccy,state,last_seen)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'USD','open',now())
			ON CONFLICT (account_id,ticket) DO UPDATE SET
			  symbol=EXCLUDED.symbol,side=EXCLUDED.side,lots=EXCLUDED.lots,
			  contract_size=EXCLUDED.contract_size,open_px=EXCLUDED.open_px,
			  open_ts=EXCLUDED.open_ts,state='open',last_seen=now()
			WHERE fin_venue_position.state<>'closed'`,
			acctID, a.acct.Venue, ticket, o.GetSymbol(), side,
			o.GetLots(), o.GetContractSize(), o.GetOpenPrice(), openTs); err != nil {
			c.rej.Inc("write_err")
			log.Printf("[mt5] %s 在仓快照写败 票=%s: %v", a.acct.Venue, ticket, err)
		}
	}
	// stale 清场——快照成功拍的权威差集（空 seen=全平）。
	if _, err := c.pool.Exec(ctx, `UPDATE fin_venue_position SET state='stale'
		WHERE account_id=$1 AND state='open' AND NOT (ticket = ANY($2))`,
		acctID, seen); err != nil {
		c.rej.Inc("write_err")
		log.Printf("[mt5] %s 在仓 stale 清场败: %v", a.acct.Venue, err)
	}
	return nil
}
