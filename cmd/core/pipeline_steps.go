// pipeline_steps.go —— 驾驶舱 v4 管道纯派生层（ops_event 流水→步链/灯语——
// 无 IO 无状态，pipeline_face.go 聚合面同文件族；拆件=行规模棘轮 D-588）。
package main

import (
	"fmt"
	"strings"
	"time"
)

// seatDisplay 席名归一（纯显示映射——生产自报名散形 Dev/施工席/DM/审计席/…
// 归一设计稿席位语；未知名原样透传不吞）。
func seatDisplay(seat string) string {
	switch seat {
	case "Dev", "施工席", "无头施工席", "Dev席":
		return "施工"
	case "DM", "审计席", "DM席":
		return "审计"
	case "决策席", "CTO席":
		return "CTO"
	case "管理席", "席MANAGE", "席claude", "MANAGE", "CEO席":
		return "CEO"
	default:
		return seat
	}
}

// stepVerdict 收步事件→该席步结果（空串=开放事件——步续干）。
// 裁决双义拆分（生产实证 10-31：裁决 529 行=通过 262+进度自报 ~135+驳回散条
// ——note 前缀判：驳回…/通过…=终态，其余=审计进度非终态）。
func stepVerdict(event, note string) string {
	switch event {
	case "收口", "移交":
		return "closed"
	case "挂起":
		return "blocked"
	case "异常":
		return "error"
	case "裁决":
		switch {
		case strings.HasPrefix(note, "驳回"):
			return "rejected"
		case strings.HasPrefix(note, "通过"):
			return "passed"
		}
	}
	return ""
}

// pipelineStep 步累积器（同席连续事件=一步；切席/收步事件断步）。
type pipelineStep struct {
	seat    string // 归一席名（Rn 后缀出件前才挂）
	disp    string // 出件席名（驳回环带 R2/R3）
	enter   time.Time
	exit    time.Time
	hasExit bool
	result  string // working|closed|passed|rejected|blocked|error
	// 出件灯面（在干步径：Active+Spin/State/Breathe——进程活证 spinner，
	// 死/停滞改灯；已收步径由 lampState 出件时派生）。
	active  bool
	spin    bool
	state   int32
	breathe bool
}

// pipelineEventRow 事件流水行（SQL 扫描载体）。
type pipelineEventRow struct {
	ticket string
	event  string
	ts     time.Time
	seat   string
	note   string
}

// buildSteps 事件流水→步链（时间升序）。断步两径：①同席收步事件（收口/移交/
// 挂起/异常/裁决终态）=显式结果收步；②切席=前席隐式移交（无收口自报径——
// 结果记 closed）。尾步无收步=在干（result=working，exit 空）。
func buildSteps(evs []pipelineEventRow) []*pipelineStep {
	var steps []*pipelineStep
	var cur *pipelineStep
	for _, e := range evs {
		disp := seatDisplay(e.seat)
		if cur != nil && cur.seat != disp {
			cur.exit, cur.hasExit = e.ts, true
			cur.result = "closed" // 隐式移交（cur 只可能处于在干态——显式收步即断链）
			steps = append(steps, cur)
			cur = nil
		}
		if cur == nil {
			cur = &pipelineStep{seat: disp, enter: e.ts, result: "working"}
		}
		if v := stepVerdict(e.event, e.note); v != "" {
			cur.exit, cur.hasExit = e.ts, true
			cur.result = v
			steps = append(steps, cur)
			cur = nil
		}
	}
	if cur != nil {
		steps = append(steps, cur)
	}
	// 驳回环标注：同席 ≥2 次出现，后续标 R2/R3（首次不标——设计稿钉）。
	seen := map[string]int{}
	for _, s := range steps {
		seen[s.seat]++
		if seen[s.seat] > 1 {
			s.disp = fmt.Sprintf("%sR%d", s.seat, seen[s.seat])
		} else {
			s.disp = s.seat
		}
	}
	return steps
}

// lampState 步结果→灯态 [state, breathe]（步级徽章灯语：绿呼吸=完成/通过
// ——该席完成本职；黄=驳回/挂起；红=异常；在干步=spinner 径不走此表）。
func lampState(result string) (int32, bool) {
	switch result {
	case "passed", "closed":
		return 0, true
	case "rejected", "blocked":
		return 1, true
	case "error":
		return 2, false
	}
	return 3, false
}

// rowLamp 行级灯=票当前滞留态（设计稿灯语「黄=驳回/挂起/等审计」——票面
// 滞在等审计/驳回/挂起皆黄呼吸；通过待部署=完成绿呼吸；异常红）。
// 与 lampState 分表：步级 closed=该席完成绿，行级 closed=等审计黄。
func rowLamp(result string) (int32, bool) {
	switch result {
	case "passed":
		return 0, true
	case "closed", "rejected", "blocked":
		return 1, true
	case "error":
		return 2, false
	}
	return 3, false
}

// headlessLamp 进程五态→灯态 [state, breathe]（活证语义同 Cockpit HL_DOT：
// alive 绿呼吸/stalled 黄呼吸/dead·overdue 红静止/done 灰——事件滞后于进程
// 完工时以进程面为准不装活）。
func headlessLamp(state string) (int32, bool) {
	switch state {
	case "alive":
		return 0, true
	case "stalled":
		return 1, true
	case "dead", "overdue":
		return 2, false
	}
	return 3, false
}

// hhmm 服务端格式化进/出时刻（设计稿钉 HH:MM——同日语义）。
func hhmm(t time.Time) string { return t.Format("15:04") }
