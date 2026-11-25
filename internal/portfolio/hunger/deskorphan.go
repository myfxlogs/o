// deskorphan.go —— 票桌孤儿池压力卡（工单O-35463②；O-39410 判重
// 升级——同 anchor_ref=同病判重硬键，非空锚逐组一报
// esc_scope='desk_orphan:<ref>'，无锚票聚合一桶走语义兜底标「无锚」
// scope='desk_orphan'）：workable∧tkind='工单'∧dispatch_rank IS NULL
// ∧filed_ts>24h 陈票聚合——fin_0212 证锚闸把无证票挡在派发池外
// （设计本意=逼分诊），但分诊无器官承接+自域票双剔+NOT dispatchable
// NULL 漏计三连环=只进不出死角。探针只述数不分诊：升件 note=计数+
// 最陈 id+锚分组；分诊三择（补证复活/销案/挂起留据）归唤知会话。
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
// 锚分组升件（O-39410——同 anchor_ref=同病判重硬键：非空锚逐组一报
// esc_scope='desk_orphan:<ref>'；无锚票聚合一桶走语义兜底标「无锚」
// scope='desk_orphan' 原键）；池=0→未销升件自销（锚组 scope 前缀
// 同收）。败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeDeskOrphan(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT COALESCE(anchor_ref,''), count(*),
		COALESCE(min(id),0),
		COALESCE(round(extract(epoch from now()-min(filed_ts))/3600,1),0),
		count(*) FILTER (WHERE note !~ '证=')
		FROM ops_ticket_desk
		WHERE workable AND tkind='工单' AND dispatch_rank IS NULL
		  AND filed_ts < now()-make_interval(secs=>$1)
		GROUP BY 1 ORDER BY 1`, deskOrphanMinAgeSecs)
	if err != nil {
		return fmt.Errorf("孤儿池读败: %w", err)
	}
	type grp struct {
		aref      string
		n         int
		oldestID  int64
		oldestAge float64
		nocert    int
	}
	var groups []grp
	var total int
	for rows.Next() {
		var g grp
		if err := rows.Scan(&g.aref, &g.n, &g.oldestID, &g.oldestAge, &g.nocert); err != nil {
			rows.Close()
			return fmt.Errorf("孤儿池扫败: %w", err)
		}
		groups = append(groups, g)
		total += g.n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("孤儿池迭代败: %w", err)
	}
	if total == 0 {
		return s.deskOrphanResolve(ctx)
	}
	for _, g := range groups {
		if g.aref == "" {
			continue // 无锚桶最后聚合报——下方统一件
		}
		if err := s.deskEscOnce(ctx, deskOrphanEscClass,
			deskOrphanEscScope+":"+g.aref,
			fmt.Sprintf("票桌孤儿池·同锚判重：anchor=%s 陈票 %d 张（最陈#%d·%.0fh）——同 anchor_ref=同病判重硬键（O-39410）；分诊三择=补证复活/销案/挂起留据",
				g.aref, g.n, g.oldestID, g.oldestAge), deskOrphanDedupHours); err != nil {
			return err
		}
	}
	// 无锚票聚合兜底——原语义兜底径，note 标「无锚」。
	var unanch *grp
	for i := range groups {
		if groups[i].aref == "" {
			unanch = &groups[i]
		}
	}
	if unanch != nil && unanch.n > 0 {
		var dup int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_scope=$1
			AND ts>now()-make_interval(hours=>$2)`,
			deskOrphanEscScope, deskOrphanDedupHours).Scan(&dup); err != nil {
			return fmt.Errorf("孤儿池去重读败: %w", err)
		}
		if dup == 0 {
			if _, err := s.pool.Exec(ctx,
				`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
				VALUES($1,'升件',$2,$3,$4,$5)`,
				hungerSeat, hungerTicket,
				fmt.Sprintf("票桌孤儿池·无锚兜底：workable∧rank NULL∧龄>%s 陈票 %d 张（最陈#%d·%.0fh）——死因粗分 无证锚=%d 有证未排=%d；无锚票语义兜底非判重键；分诊三择=补证复活/销案/挂起留据（工单O-35463/O-39410）",
					deskOrphanMinAgeCh, unanch.n, unanch.oldestID, unanch.oldestAge,
					unanch.nocert, unanch.n-unanch.nocert),
				deskOrphanEscClass, deskOrphanEscScope); err != nil {
				return fmt.Errorf("孤儿池升件行: %w", err)
			}
		}
	}
	return nil
}

// deskOrphanResolve 池清零自销：未销 desk_orphan 升件全数引「升件#id」
// 处置销账（pending 视图引证同契）；零未销=静默。读败上抛让位。
func (s *Sensor) deskOrphanResolve(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT e.id FROM ops_event e
		WHERE e.event='升件' AND (e.esc_scope=$1 OR e.esc_scope LIKE $1||':%')
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
