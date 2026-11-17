// residual_halt.go —— 残币处置停摆键（工单#22248①「会自主动钱的常驻件
// 各有停摆键，置位即零动钱，置位超时有显形」）：
//   - earn.residual_clear_halted 置位=残币处置面全段零动钱（扫袋/停泊/
//     清仓同止——「该件决策零发」为验收锚）；
//   - 键读败/缺键=非停照旧跑（risk.kill_switch 同形置位语义——停摆须
//     显式宣告非默态）；
//   - 置位超 24h 升件显形一行（同键节流幂等——防忘键永久暗停）。
package earn

import (
	"context"
	"fmt"
)

// keyResidualHalt 停摆键（app_config bool——生产缺键=false 非停）。
const keyResidualHalt = "earn.residual_clear_halted"

// residualHalted 停摆判据读：cfg 快照热读（Watch 通配置更代次——读败/
// 缺键=非停照旧跑，不默停）。stale=置位龄超阈（app_config.updated_at
// 现读——bump_config_generation 触发器保证 updated_at 恒=置位时刻；
// 龄读败=停已显式宣告仍停，不升件）。阈=w.haltStaleAfter（测缝——
// 生产 24h，updated_at 受触发器保护测试不可回拨故开时长注入位）。
func (w *Worker) residualHalted(ctx context.Context) (halted, stale bool) {
	on, err := w.cfg.GetBool(keyResidualHalt)
	if err != nil || !on {
		return false, false
	}
	var ageS float64
	if err := w.pool.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM now()-updated_at) FROM app_config WHERE key=$1`, keyResidualHalt).Scan(&ageS); err != nil {
		return true, false
	}
	return true, ageS > w.haltStaleAfter.Seconds()
}

// escalateHaltStale 置位超时显形升件（同键节流幂等——复扫不连鸣，
// 行在案即显形；销行=处置面恢复在案的凭据）。
func (w *Worker) escalateHaltStale(ctx context.Context) {
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		SELECT '配管','杂币清仓停摆','T-22248',$1
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='杂币清仓停摆' AND note LIKE 'residual_clear_halted %')`,
		fmt.Sprintf("residual_clear_halted 置位超%dh仍在置——停摆键忘置回=杂币处置面永久暗停（恢复=置false或删键；工单#22248）",
			int(w.haltStaleAfter.Hours()))); err != nil {
		w.rej.Inc("write_err")
	}
}
