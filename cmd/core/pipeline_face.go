// pipeline_face.go —— 驾驶舱 v4 工作流管道聚合（三卡合一——1票1行管道流）。
// 数据源=ops_event append-only 流水（ticket+event+ts+seat+note）按票重建席位
// 经过链，禁新表（设计稿钉：ops_ticket_state 是每票最新拍视图——完整流水在
// 基表 ops_event）；进程级灯色联查 ops_headless（alive/stalled/dead/overdue）；
// 完结=部署上线（ops_event event='部署' 行——deploy_image.sh 尾拍落，部署 ts
// ≥票末事件 ts 的票退场）。步链/灯语纯派生层=pipeline_steps.go。
// 流推：ops_event 插行→ev_pipeline 触发器 NOTIFY（fin_0092）→Watch 一跳载荷
// 帧=GetPipeline 全量（≤20 行）——状态变化秒级到屏（锚 c/d ≤10s）。
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"o/internal/gen/dashboardv1"
)

// pipelineRowCap 管道行上限（设计稿上限节：防撑爆瓦片；超出行数走
// total_active 前端「+N more」）。
const pipelineRowCap = 20

// titleCache 票标题缓存（docs/tasks/T-*.md 文件名 slug——best-effort 显示件；
// 生产镜像无 docs 面（.dockerignore）=零命中空表不报错，60s 一刷防每查扫盘）。
var titleCache struct {
	mu sync.Mutex
	m  map[string]string
	at time.Time
}

func ticketTitles(dir string) map[string]string {
	titleCache.mu.Lock()
	defer titleCache.mu.Unlock()
	if titleCache.m != nil && time.Since(titleCache.at) < time.Minute {
		return titleCache.m
	}
	m := map[string]string{}
	ents, _ := filepath.Glob(filepath.Join(dir, "T-*.md"))
	for _, p := range ents {
		base := filepath.Base(p)
		ticket := strings.SplitN(base, "-", 3)
		if len(ticket) >= 3 {
			m[ticket[0]+"-"+ticket[1]] = strings.TrimSuffix(ticket[2], ".md")
		}
	}
	titleCache.m, titleCache.at = m, time.Now()
	return m
}

// GetPipeline 管道聚合读面（GetCockpit.pipeline 同源复用+unary 快照恢复面+
// Watch ev_pipeline 载荷径——一实现三消费）。读败=err 上抛（GetCockpit 侧
// missing 名段显缺；Watch 侧帧照发 log 显形——同族纪律）。
func (c coreService) GetPipeline(ctx context.Context,
	req *dashboardv1.GetPipelineRequest) (*dashboardv1.GetPipelineResponse, error) {
	r := &dashboardv1.GetPipelineResponse{}

	// ① 活跃票界定：有事件簿的票，末事件 ts>末次部署 ts（部署=完结线——
	// 部署 ts 后再无该票事件，则票已上线退场）。部署事件缺席（首部署前）=
	// 无完结线全量在飞。最新票排最前（末事件 ts DESC），上限 20 行，
	// total_active=窗口计数（截断径前端 +N more）。
	type activeTicket struct {
		ticket  string
		lastTs  time.Time
		firstTs time.Time
	}
	var actives []activeTicket
	ar, err := c.pool.Query(ctx, `
		WITH dep AS (SELECT max(ts) AS d FROM ops_event WHERE event='部署')
		SELECT f.ticket, f.last_ts, f.first_ts, count(*) OVER () AS total
		FROM (
			SELECT ticket, max(ts) AS last_ts, min(ts) AS first_ts
			FROM ops_event WHERE ticket<>'' GROUP BY ticket
		) f, dep
		WHERE dep.d IS NULL OR f.last_ts > dep.d
		ORDER BY f.last_ts DESC, f.ticket DESC
		LIMIT $1`, pipelineRowCap)
	if err != nil {
		return nil, err
	}
	total := int32(0)
	for ar.Next() {
		var t activeTicket
		if err := ar.Scan(&t.ticket, &t.lastTs, &t.firstTs, &total); err != nil {
			ar.Close()
			return nil, err
		}
		actives = append(actives, t)
	}
	ar.Close()
	if ar.Err() != nil {
		return nil, ar.Err()
	}
	r.TotalActive = total
	r.AsOf = tsProto(time.Now())

	if len(actives) == 0 {
		// 零活跃也出 seq 水印（部署收尾帧——行退场靠 seq 推进可辨）。
		var seq int64
		if err := c.pool.QueryRow(ctx,
			`SELECT COALESCE(max(id),0) FROM ops_event`).Scan(&seq); err == nil {
			r.Seq = seq
		}
		return r, nil
	}

	// ② 事件流水拉取（≤20 票——IN 下推）+ 席位进程态联查。
	tickets := make([]string, len(actives))
	for i, t := range actives {
		tickets[i] = t.ticket
	}
	er, err := c.pool.Query(ctx, `
		SELECT ticket,event,ts,seat,note FROM ops_event
		WHERE ticket=ANY($1) ORDER BY ts ASC, id ASC`, tickets)
	if err != nil {
		return nil, err
	}
	defer er.Close()
	byTicket := map[string][]pipelineEventRow{}
	for er.Next() {
		var e pipelineEventRow
		if err := er.Scan(&e.ticket, &e.event, &e.ts, &e.seat, &e.note); err != nil {
			return nil, err
		}
		byTicket[e.ticket] = append(byTicket[e.ticket], e)
	}
	if er.Err() != nil {
		return nil, er.Err()
	}
	hl := map[string]string{} // ticket→进程五态机键（无行=交互席自报驱动）
	hr, err := c.pool.Query(ctx,
		`SELECT ticket,state FROM ops_headless WHERE ticket=ANY($1)`, tickets)
	if err == nil {
		defer hr.Close()
		for hr.Next() {
			var tk, st string
			if hr.Scan(&tk, &st) == nil {
				hl[tk] = st
			}
		}
	}
	// 读败不炸面——headless 缺席按无行径（灯=自报流推演）。

	// ③ 步链→出件行（标题 best-effort；灯色=当前活动面推演）。
	titles := ticketTitles(c.docsDir())
	now := time.Now()
	for _, t := range actives {
		steps := buildSteps(byTicket[t.ticket])
		row := &dashboardv1.CockpitPipelineRow{
			Ticket:     t.ticket,
			StartedAt:  tsProto(t.firstTs),
			ElapsedMin: int64(now.Sub(t.firstTs).Minutes()),
		}
		if ti, ok := titles[t.ticket]; ok {
			row.Title = ti
		}
		if n := len(steps); n > 0 {
			last := steps[n-1]
			row.CurrentSeat = last.disp
			row.CurrentState = last.result
			active := !last.hasExit
			procState := hl[t.ticket]
			// 行级灯=当前活动面状态色（设计稿钉）：在干步联查进程五态
			// （活证优先——死/停滞不装活）；无在干步=末席结果色（等审计/
			// 驳回/挂起黄、通过绿、异常红）。
			if active {
				st, br := headlessLamp(procState)
				if procState == "" || procState == "alive" || procState == "done" {
					last.spin = true // 进程活证（或交互席无进程面）=spinner 径
					st, br = 0, true // 行级灯=绿呼吸（在干=活证）
				}
				last.active = true
				last.state, last.breathe = st, br
				row.State, row.Breathe = st, br
			} else {
				st, br := rowLamp(last.result)
				row.State, row.Breathe = st, br
			}
		}
		for _, s := range steps {
			st := &dashboardv1.CockpitPipelineStep{
				Seat: s.disp, Enter: hhmm(s.enter), Result: s.result,
				Active: s.active, Spin: s.spin, State: s.state, Breathe: s.breathe,
			}
			if !s.active {
				// 已收步：灯色=该席最终结果（绿=完成/通过，黄=驳回/挂起，
				// 红=异常——设计稿灯语）；在干步灯面已在行级灯段派生。
				st.State, st.Breathe = lampState(s.result)
			}
			if s.hasExit {
				st.Exit = hhmm(s.exit)
				st.Ms = s.exit.Sub(s.enter).Milliseconds()
			} else {
				st.Ms = now.Sub(s.enter).Milliseconds()
			}
			row.Steps = append(row.Steps, st)
		}
		r.Rows = append(r.Rows, row)
	}
	// ④ seq 水印=max(ops_event.id)（部署行 ticket='' 亦推进——行退场帧可辨）。
	var seq int64
	if err := c.pool.QueryRow(ctx,
		`SELECT COALESCE(max(id),0) FROM ops_event`).Scan(&seq); err == nil {
		r.Seq = seq
	}
	return r, nil
}

// docsDir 票标题扫面目录（coreService 注入缝——缺省 repo 相对径；生产容器
// 无 docs 面=空表优雅降级；测试注入绝对径）。
func (c coreService) docsDir() string {
	if c.docsRoot != "" {
		return c.docsRoot
	}
	if wd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "docs", "tasks")); err == nil {
			return filepath.Join(wd, "docs", "tasks")
		}
	}
	return "docs/tasks"
}
