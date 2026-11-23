// deskorphan.go —— 票桌孤儿池压力卡（工单O-35463②）：workable∧
// tkind='工单'∧dispatch_rank IS NULL∧filed_ts>24h 陈票聚合计数——
// fin_0212 证锚闸把无证票挡在派发池外（设计本意=逼分诊），但分诊
// 无器官承接+自域票双剔+NOT dispatchable NULL 漏计三连环=只进不出
// 死角。探针只述数不分诊：升件 note=计数+最陈 id+死因粗分（无证锚/
// 有证未排），分诊三择（补证复活/销案/挂起留据）归唤知会话。
//
// 池清零自销：count=0 且有未销 desk_orphan 升件在案→处置行引
// 「升件#id」销账（ops_esc_pending 引证同契——池清条件存则自销，
// 条件复燃照火=escalateUnresolved 律同构）。去重=24h 同窗至多一报
// （depblindDedupHours 同钦——持续池每日一压非每拍暴）。
// 只报不修——修票归会话裁断。
package hunger

import (
	"context"
	"fmt"
	"strings"
)

const (
	// deskOrphanEscScope 升件域键（票桌孤儿池单键——与 closeout 的
	// worker_closeout_missing 不撞面）。
	deskOrphanEscScope = "desk_orphan"
	// deskOrphanEscClass 升件类（判类册自由词——票桌死角类）。
	deskOrphanEscClass = "票桌孤儿"
	// deskOrphanDedupHours 同窗去重（depblind/closeout 同钦 24h）。
	deskOrphanDedupHours = 24
	// deskOrphanMinAgeCh 陈票龄界（spec 钦定 24h——新立票给分诊宽限）。
	deskOrphanMinAgeCh   = "24 hours"
	deskOrphanMinAgeSecs = 24 * 3600
)

// ProbeDeskOrphan 票桌孤儿池探针一拍（hungerTick 6h 同槽）：池>0→
// 聚合升件；池=0→未销升件自销。败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeDeskOrphan(ctx context.Context) error {
	var n, nocert, certed int
	var oldestID int64
	var oldestAgeH float64
	if err := s.pool.QueryRow(ctx, `SELECT count(*),
		COALESCE(min(id),0),
		COALESCE(round(extract(epoch from now()-min(filed_ts))/3600,1),0),
		count(*) FILTER (WHERE note !~ '证=')
		FROM ops_ticket_desk
		WHERE workable AND tkind='工单' AND dispatch_rank IS NULL
		  AND filed_ts < now()-make_interval(secs=>$1)`,
		deskOrphanMinAgeSecs).Scan(&n, &oldestID, &oldestAgeH, &nocert); err != nil {
		return fmt.Errorf("孤儿池读败: %w", err)
	}
	certed = n - nocert
	if n == 0 {
		return s.deskOrphanResolve(ctx)
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope=$1
		AND ts>now()-make_interval(hours=>$2)`,
		deskOrphanEscScope, deskOrphanDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("孤儿池去重读败: %w", err)
	}
	if dup > 0 {
		return nil // 同窗已报——计数续压归下窗
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		fmt.Sprintf("票桌孤儿池：workable∧rank NULL∧龄>%s 陈票 %d 张（最陈#%d·%.0fh）——死因粗分 无证锚=%d 有证未排=%d（回执剔/自域死角）；分诊三择=补证复活/销案/挂起留据（工单O-35463）",
			deskOrphanMinAgeCh, n, oldestID, oldestAgeH, nocert, certed),
		deskOrphanEscClass, deskOrphanEscScope); err != nil {
		return fmt.Errorf("孤儿池升件行: %w", err)
	}
	return nil
}

// deskOrphanResolve 池清零自销：未销 desk_orphan 升件全数引「升件#id」
// 处置销账（pending 视图引证同契）；零未销=静默。读败上抛让位。
func (s *Sensor) deskOrphanResolve(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT e.id FROM ops_event e
		WHERE e.event='升件' AND e.esc_scope=$1
		  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))
		ORDER BY e.id`, deskOrphanEscScope)
	if err != nil {
		return fmt.Errorf("孤儿池销账面读败: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("孤儿池销账行扫败: %w", err)
		}
		ids = append(ids, fmt.Sprintf("升件#%d", id))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("孤儿池销账行迭代败: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	// event='记事'非'处置'——处置为 sem_matrix 闸事件（探测席不在白名单，
	// O-33705 虚报洞同族教训）；ops_esc_pending 清账只认 note 引「升件#id」
	// 不挑事件类，记事=探测席自账合法形。
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,'记事',$2,$3)`,
		hungerSeat, hungerTicket,
		strings.Join(ids, "、")+" 销——票桌孤儿池清零（workable∧rank NULL∧陈>24h=0）——探针自销"); err != nil {
		return fmt.Errorf("孤儿池销账行: %w", err)
	}
	return nil
}
