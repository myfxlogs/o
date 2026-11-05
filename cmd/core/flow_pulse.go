// flow_pulse.go —— 活水段读面（T-0307：运行时机器流段链——ops_event 票=组织
// 流语义不混，段=机器流；各段 fail-soft 表缺/查询败=该段 ok=false 不炸面）。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// stageThresh 段灯阈（预期周期秒——按各臂真实节拍钉：返修② collect/bridge
// 改 recv_ts（镜像到达戳——周期边界 ts 负滞后+停摆失明病灶修）。collect=funding 采集
// ~5m、bridge=投影帧 ~10m、eval=评估拍 ~30m、proposal=提案拍 ~30m、exec=钱流
// 事件（分钟级）、report=日报 24h+余、scout=日拍 25h、roster=心跳 60s）。
var stageThresh = map[string]int64{
	"collect": 300, "bridge": 600, "eval": 1800, "proposal": 1800,
	"exec": 3600, "report": 27 * 3600, "scout": 25 * 3600,
}

// stageDef 段定义（key/中文标/查询式——last_ts 与 count24h 两参补位；
// neutral=事件驱动段无周期臂——灯恒中性灰（返修①②：lag 文本照显，节拍阈
// 只对周期臂有意义，proposal/exec 误报红=假警病灶修）。
type stageDef struct {
	key, label string
	sql        string
	neutral    bool
}

var flowStages = []stageDef{
	{"collect", "采集", `SELECT max(recv_ts), count(*) FILTER (WHERE recv_ts>now()-interval '24 hours') FROM md_funding`, false},
	{"bridge", "投影", `SELECT max(recv_ts), count(*) FILTER (WHERE recv_ts>now()-interval '24 hours') FROM md_ticks`, false},
	{"eval", "评估", `SELECT max(ts), count(*) FILTER (WHERE ts>now()-interval '24 hours') FROM fin_opp_eval`, false},
	{"proposal", "提案", `SELECT max(created_at), count(*) FILTER (WHERE created_at>now()-interval '24 hours') FROM fin_proposal`, true},
	{"exec", "执行", `SELECT max(occurred_at), count(*) FILTER (WHERE occurred_at>now()-interval '24 hours') FROM fin_cashflow`, true},
	{"report", "报表", `SELECT max(generated_at), count(*) FILTER (WHERE generated_at>now()-interval '24 hours') FROM fin_report`, false},
	{"scout", "侦察", `SELECT max((metrics->>'measured_at')::timestamptz), count(*) FILTER (WHERE (metrics->>'measured_at')::timestamptz>now()-interval '24 hours') FROM fin_venue_candidate WHERE metrics ? 'measured_at'`, false},
}

// GetFlowPulse 活水段快照（unary——各段独立 fail-soft：单段查询败/表缺=
// ok=false 显缺，不拖整面；roster 同径）。
func (c coreService) GetFlowPulse(ctx context.Context,
	_ *dashboardv1.GetFlowPulseRequest) (*dashboardv1.GetFlowPulseResponse, error) {
	resp := &dashboardv1.GetFlowPulseResponse{AsOf: timestamppb.Now()}
	now := time.Now()
	for _, d := range flowStages {
		st := &dashboardv1.FlowStage{Key: d.key, Label: d.label, ExpectS: stageThresh[d.key]}
		var ts *time.Time
		var n int64
		if err := c.pool.QueryRow(ctx, d.sql).Scan(&ts, &n); err != nil {
			log.Printf("[flow_pulse] %s 段查询败（显缺不炸）: %v", d.key, err)
			st.Ok = false
		} else {
			st.Ok = true
			st.Count_24H = n
			if ts != nil {
				st.LastTs = timestamppb.New(*ts)
				st.LagS = int64(now.Sub(*ts).Seconds())
			}
			st.State = StageLamp(d.key, st.LagS, ts != nil)
			if d.neutral {
				st.State = 3 // 恒中性灰——事件驱动段无节拍阈
				switch d.key {
				case "proposal":
					st.Status = c.proposalStatus(ctx) // T-0350 态标非龄
				case "exec":
					st.Status = c.execStatus(ctx)
				}
			}
		}
		resp.Stages = append(resp.Stages, st)
	}
	resp.WorkItems = c.workItems(ctx)

	// roster 心跳段（fin_employee_registry——23 岗含盒上臂经隧道写；读败=0 显缺）。
	// T-0344：pure_interactive 席形同 GetCockpit 口径豁免（勿双写防两端漂移）。
	var rc int64
	var minBeat *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(last_beat) FROM fin_employee_registry WHERE status='active' AND meta->>'form' IS DISTINCT FROM 'pure_interactive'`).
		Scan(&rc, &minBeat); err != nil {
		log.Printf("[flow_pulse] roster 查询败（显缺不炸）: %v", err)
	} else {
		resp.RosterCount = int32(rc)
		if minBeat != nil {
			resp.RosterWorstLagS = int64(now.Sub(*minBeat).Seconds())
		}
	}
	return resp, nil
}

// StageLamp 段灯级（0绿<2×阈 1黄<4× 2红超/缺——前端同阈表复用防两端漂移；
// 导出口径测试消费）。
func StageLamp(key string, lagS int64, ok bool) int32 {
	th, has := stageThresh[key]
	if !ok || !has {
		return 2
	}
	switch {
	case lagS < 2*th:
		return 0
	case lagS < 4*th:
		return 1
	default:
		return 2
	}
}

// workItems 钱径工件三类 union（T-0309——proposal→position→venue 序 cap 10；
// 各查询独立 fail-soft：单类败=该类缺席不炸面；proposal 活态词='fin_proposed'
// +fin_armed 码面取值域勿猜——fin_0006 CHECK 钉）。
func (c coreService) workItems(ctx context.Context) []*dashboardv1.WorkItem {
	out := []*dashboardv1.WorkItem{}
	// ①proposal 活态（fin_proposed=待开仓 armed；fin_armed=执行中——码面词集）。
	if rs, err := c.pool.Query(ctx, `SELECT id,inst_id,state,created_at FROM fin_proposal
		WHERE state IN ('fin_proposed','fin_armed') ORDER BY created_at DESC`); err == nil {
		for rs.Next() {
			var id int64
			var inst, st string
			var ts time.Time
			if rs.Scan(&id, &inst, &st, &ts) != nil {
				break
			}
			stage := "执行中"
			if st == "fin_proposed" {
				stage = "待开仓"
			}
			out = append(out, &dashboardv1.WorkItem{Kind: "proposal",
				Ref: fmt.Sprintf("#%d %s", id, inst), Stage: stage,
				SinceTs: timestamppb.New(ts)})
		}
		rs.Close()
	}
	// ②position 未平+残仓臂（T-0327——venue 在持无活策略归属类级入集：策略
	// 死/产品活幽灵态在工件面显形；读件缺=显缺行不静默不假零）。
	if rs, err := c.pool.Query(ctx, `SELECT inst_id, ts FROM fin_position_view
		WHERE kind='exec' AND closed_at IS NULL ORDER BY ts DESC`); err == nil {
		for rs.Next() {
			var inst string
			var ts *time.Time
			if rs.Scan(&inst, &ts) != nil {
				break
			}
			w := &dashboardv1.WorkItem{Kind: "position", Ref: inst, Stage: "持仓中"}
			if ts != nil {
				w.SinceTs = timestamppb.New(*ts)
			}
			out = append(out, w)
		}
		rs.Close()
	}
	if rems, rmiss := c.venueRemnants(ctx); len(rems)+len(rmiss) > 0 {
		for _, it := range rems {
			w := &dashboardv1.WorkItem{Kind: "position", Ref: it.ref, Stage: "残仓"}
			if it.since != nil {
				w.SinceTs = timestamppb.New(*it.since)
			}
			out = append(out, w)
		}
		for _, m := range rmiss {
			out = append(out, &dashboardv1.WorkItem{Kind: "position",
				Ref: "残仓面 " + m, Stage: "读件缺"})
		}
	}
	// ③venue measuring（返修①：md_venue_funding 归 PG-md 不在 SG——原联查
	// 必败致整类缺席；progress 改读 metrics.boards persist 显「持续 XX%」——
	// SG 镜像列内数据零外联；since=首入度量缺——metrics measured_at 兜底）。
	if rs, err := c.pool.Query(ctx, `SELECT venue, metrics->>'measured_at',
		(metrics->'boards'->0->>'persist')::float8 FROM fin_venue_candidate v
		WHERE state='measuring' ORDER BY venue`); err == nil {
		for rs.Next() {
			var venue, at *string
			var persist *float64
			if rs.Scan(&venue, &at, &persist) != nil {
				break
			}
			w := &dashboardv1.WorkItem{Kind: "venue", Ref: *venue, Stage: "度量中"}
			if persist != nil {
				w.Progress = fmt.Sprintf("持续 %.0f%%", *persist*100)
			}
			if at != nil {
				if t, err := time.Parse(time.RFC3339, *at); err == nil {
					w.SinceTs = timestamppb.New(t)
				}
			}
			out = append(out, w)
		}
		rs.Close()
	}
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}
