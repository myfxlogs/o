// opener.go —— live 策略开仓消费面（T-0050 首仓序④）：fin_strategy state='live'
// 且无未平仓行 → Opener 缝开仓（cmd 装配=gate.Check→BuildLegs→Runner.Execute
// 与 openfirst 同码径——投资域不反向依赖交易/风控域，guard.Executor 先例）。
// 幂等=fin_position 行（开成即有行=不再开）；事件驱（ev_strategy/ev_eval NOTIFY，
// T-0048 脊）零轮询。
package lifecycle

import (
	"context"
	"log"
)

// OpenSpec live 开仓规格（funding-hedge 模板：direction=short_perp_long_spot）。
type OpenSpec struct {
	StrategyRef int64
	Venue       string
	InstID      string // SWAP 键（BTC-USDT-SWAP）——现货腿装配层派生
	Direction   string
	NotionalUSD float64
}

// Opener 开仓执行缝（未接线=nil fail-closed 不开——禁无闸直开）。
type Opener interface {
	Open(ctx context.Context, spec OpenSpec) (execID int64, err error)
}

// AttachOpener 挂开仓缝（cmd 装配层接线）。
func (r *Runner) AttachOpener(o Opener) *Runner { r.op = o; return r }

// openLive live 策略开仓拍：无未平仓行的 live 行逐个过缝。败=计数+log 不炸
// 事件主径（下帧事件续试——ev_strategy 行变才触发，无热环）。
func (r *Runner) openLive(ctx context.Context) {
	if r.op == nil {
		return // 未接线=无消费面（cmd 未挂——禁开）
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.direction, s.notional_usd, COALESCE(s.venue,'okx')
		FROM fin_strategy s
		WHERE s.state='live' AND NOT EXISTS (
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)`)
	if err != nil {
		r.rej.Inc("write_err")
		return
	}
	type cand struct {
		spec OpenSpec
	}
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.spec.StrategyRef, &c.spec.InstID, &c.spec.Direction,
			&c.spec.NotionalUSD, &c.spec.Venue); err != nil {
			rows.Close()
			r.rej.Inc("write_err")
			return
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		eid, err := r.op.Open(ctx, c.spec)
		if err != nil {
			r.rej.Inc("open_failed")
			log.Printf("[lifecycle] WARN 策略 %d 开仓败（%v exec_id=%d）——事件续试", c.spec.StrategyRef, err, eid)
			continue
		}
		log.Printf("[lifecycle] 策略 %d %s live 开仓成 exec_id=%d notional=%.2f",
			c.spec.StrategyRef, c.spec.InstID, eid, c.spec.NotionalUSD)
	}
}
