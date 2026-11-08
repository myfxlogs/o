// Package seatcard —— 飞书席位活卡消费件（T-0458 业主令件）：
// 一席一卡（业主裁：按席开不按票开——看的=席位活性非票级流水）。
// 状态唯一真源=ops_event（append-only 链账）；卡=ops_event 派生展示面。
// meta internal/feishu/card.go 三件套模式级复用（抄模式不依赖 meta 包）：
// 内容哈希闸 PATCH/懒建/启动对账；卡败落 ops_event seatcard_err 行
// （非 dm 帧白名单事件——不回灌自触发环，meta RecordErr 同判例）。
// 本包零 internal 依赖：SQL 直读（pgx）+出站/班态探针缝注入——装配归
// cmd/seatcard，桩测注入伪缝（notify/pgstore/systemctl 全不进桩测）。
package seatcard

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// 一期席位册（CTO 裁定⑥：无头三席 CTO+施工+CEO——活性盲域最大；
// COS 交互席二期——业主 pane 已直见非刚需）。Aliases=ops_event.seat
// 历史值域归并（席名归一前的旧词照认——末动/班态判定不断史）。
type SeatDef struct {
	Card    string   // 卡名（卡头席名）
	Aliases []string // ops_event.seat 值域
	Units   []string // 班态 unit 探针目标（systemctl --user 单名或 glob）
}

// Phase1Seats 一期三席。决策席=CTO 席旧名、管理席=CEO 席旧名
// （c1d71b08 席名归一词表）；无头施工席=dispatch_worker 拉起的 O_SEAT 缺省值。
var Phase1Seats = []SeatDef{
	{Card: "CTO席", Aliases: []string{"CTO", "决策席"}, Units: []string{"o-cto-seat.service"}},
	{Card: "施工席", Aliases: []string{"施工席", "无头施工席"}, Units: []string{"o-worker-*"}},
	{Card: "CEO席", Aliases: []string{"CEO", "管理席"}, Units: []string{"o-ceo-seat.service"}},
}

// 活性窗——零活性席（近 48h 零行）不补卡不渲染（懒建闸同款；终态/退场席
// 卡面永寂=钦定「终态/零活性席不补」）。
const ActivityWindow = 48 * time.Hour

// EventRow ops_event 行投影（seat/ts/event/ticket/note——fin_0086 列集）。
type EventRow struct {
	Seat   string
	Ts     time.Time
	Event  string
	Ticket string
	Note   string
}

// Worker 施工席并行活件行（批量班不糊律：卡内并行活件列 N 行，不开票级卡）。
type Worker struct {
	Unit  string    // unit 名（o-worker-<TID>.service）
	TID   string    // 票号
	Model string    // DEV_RUN_MODEL（dispatch_worker --setenv 注入）
	Since time.Time // unit 起拍（elapsed=Now-Since 截分显示）
}

// UnitSnap 班态 unit 探针快照。
type UnitSnap struct {
	Active  bool     // 任一目标 unit active|activating
	Workers []Worker // 施工席并行活件（其余席空集）
}

// 在办五事件（CTO 裁定①在办段取值域——最近行推 ticket+事件名）。
var workEvents = []string{"领单", "收口", "移交", "裁决", "派单"}

// 班态字标（CTO 裁定①五标）。
const (
	StatusOnDuty  = "开班在岗" // unit active ∧ 近事件≤10min
	StatusWorking = "作业中"  // active 但事件老；或班未退（无退班行）拍间态
	StatusBlocked = "挂起"   // 挂起行在案（=该席最新行）
	StatusOff     = "退班空闲" // unit inactive 且已退班（退班行在案）
	StatusError   = "异常"   // 异常行在案，或事件>2h 无动（未退班）
)

// DeriveStatus 班态判定（纯函数——班态=unit 态∧事件龄双证+行在案覆盖）：
// 优先级 挂起行>异常行>退班行>unit 态∧事件龄（>2h 无动未退班=异常兜底）。
// 退班行优先于龄兜底——退班后无动是常态非病灶。
func DeriveStatus(latest *EventRow, unitActive bool, now time.Time) string {
	if latest == nil {
		return StatusOff // 零行=零活性（不会到渲染面——懒建闸先挡）
	}
	age := now.Sub(latest.Ts)
	switch latest.Event {
	case "挂起":
		return StatusBlocked
	case "异常":
		return StatusError
	case "退班":
		return StatusOff
	}
	if unitActive {
		if age <= 10*time.Minute {
			return StatusOnDuty
		}
		return StatusWorking
	}
	if age > 2*time.Hour {
		return StatusError
	}
	return StatusWorking
}

// CardState 卡渲染输入（全量重渲——无 diff 计算，哈希闸在外层挡 PATCH）。
type CardState struct {
	Seat          string
	Status        string
	Latest        *EventRow // 末动段（该席最新行）
	Working       *EventRow // 在办段（最近领/收/移/裁/派行；nil=待派）
	Workers       []Worker  // 施工席并行活件（其余席空集）
	QueueReview   int       // 队列段：待裁深
	QueueApproved int       // 队列段：待派深
	Now           time.Time // 渲染时刻（显示截断至分钟——哈希闸不击穿）
}

// Store 卡面读写真源缝（ops_event 推导读+ops_seat_card 卡册写）。
type Store interface {
	// SeatRows 别名集近窗行（末动/在办/班态判定单查多消费）。
	SeatRows(ctx context.Context, seats []string, since time.Time) ([]EventRow, error)
	// QueueDepth 队列深（dm_queue.sh 判据同源复刻——只读推导面不改队列逻辑）。
	QueueDepth(ctx context.Context) (review, approved int, err error)
	// CardRow 卡册行读（无行 ok=false）。
	CardRow(ctx context.Context, seat string) (messageID, chatID, lastHash string, ok bool, err error)
	// SaveCard 懒建成功落行（message_id 持久——重启原地刷同卡）。
	SaveCard(ctx context.Context, seat, messageID, chatID, hash string) error
	// SaveHash PATCH 成功刷哈希+updated_at。
	SaveHash(ctx context.Context, seat, hash string) error
	// RecordErr 卡败落账（ops_event seatcard_err 行——append-only 链账
	// INSERT，非 eventbus 白名单事件不回灌）。
	RecordErr(ctx context.Context, seat, note string) error
}

// ProbeUnits 班态探针缝（缺省=systemctl --user 实件；桩测注入伪缝）。
type ProbeUnits func(ctx context.Context, patterns []string) (UnitSnap, error)

// SystemdProbe systemctl --user 实件：list-units glob 取活集；worker 行
// show 单元补起拍+DEV_RUN_MODEL（批量班不糊律 N 行源）。systemctl 缺席/
// 查询败=Active=false 零降级非错（班态退「未在岗」语义面，不炸渲染环）。
func SystemdProbe(ctx context.Context, patterns []string) (UnitSnap, error) {
	snap := UnitSnap{}
	for _, pat := range patterns {
		out, err := exec.CommandContext(ctx, "systemctl", "--user", "list-units",
			pat, "--state=active,activating", "--no-legend", "--plain").Output()
		if err != nil {
			return snap, nil // tolerate: systemctl 缺席/无 unit=按零活处理（班态语义面非故障面）
		}
		for _, ln := range strings.Split(string(out), "\n") {
			f := strings.Fields(ln)
			if len(f) == 0 || !strings.HasSuffix(f[0], ".service") {
				continue
			}
			snap.Active = true
			u := f[0]
			if strings.HasPrefix(u, "o-worker-") {
				if w := probeWorker(ctx, u); w != nil {
					snap.Workers = append(snap.Workers, *w)
				}
			}
		}
	}
	return snap, nil
}

// probeWorker 单 worker 行补全（ActiveEnterTimestamp+Environment=
// DEV_RUN_MODEL——dispatch_worker.sh --setenv 注入径）。解析败=裸行降级
// （TID 照取——unit 名即票号），零炸。
func probeWorker(ctx context.Context, unit string) *Worker {
	out, err := exec.CommandContext(ctx, "systemctl", "--user", "show", unit,
		"-p", "ActiveEnterTimestamp", "-p", "Environment", "--value").Output()
	if err != nil {
		return &Worker{Unit: unit, TID: workerTID(unit)}
	}
	w := &Worker{Unit: unit, TID: workerTID(unit)}
	for _, ln := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(ln, "DEV_RUN_MODEL="):
			w.Model = strings.TrimPrefix(ln, "DEV_RUN_MODEL=")
		case strings.Contains(ln, ":") && !strings.HasPrefix(ln, "DEV_RUN_MODEL"):
			if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", strings.TrimSpace(ln)); err == nil {
				w.Since = t
			}
		}
	}
	return w
}

// workerTID unit 名→票号（o-worker-T-0451.service→T-0451）。
func workerTID(unit string) string {
	u := strings.TrimSuffix(strings.TrimPrefix(unit, "o-worker-"), ".service")
	return u
}

// Assemble 全席装配（Refresh 单入口——每席一次 Rows 读+一次队列深读+
// 一次探针；无卡/零活性席照装配，消费面懒建闸自挡）。
func Assemble(ctx context.Context, st Store, probe ProbeUnits, seats []SeatDef, now time.Time) ([]CardState, error) {
	since := now.Add(-ActivityWindow)
	out := make([]CardState, 0, len(seats))
	for _, def := range seats {
		rows, err := st.SeatRows(ctx, def.Aliases, since)
		if err != nil {
			return nil, err
		}
		snap, err := probe(ctx, def.Units)
		if err != nil {
			return nil, err
		}
		cs := CardState{Seat: def.Card, Workers: snap.Workers, Now: now}
		for i := range rows {
			r := rows[i]
			if cs.Latest == nil {
				lr := r
				cs.Latest = &lr
			}
			if cs.Working == nil && contains(workEvents, r.Event) {
				wr := r
				cs.Working = &wr
			}
		}
		cs.Status = DeriveStatus(cs.Latest, snap.Active, now)
		out = append(out, cs)
	}
	review, approved, err := st.QueueDepth(ctx)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].QueueReview, out[i].QueueApproved = review, approved
	}
	return out, nil
}

// ActiveSeat 席活性判（懒建闸——近窗零行=零活性席不补卡）。
func ActiveSeat(cs CardState, now time.Time) bool {
	return cs.Latest != nil && now.Sub(cs.Latest.Ts) <= ActivityWindow
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
