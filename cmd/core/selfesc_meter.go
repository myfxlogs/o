// selfesc_meter.go —— 自观类探针降级件（工单#22578①）。
// 判据=「不处置不损钱不断数据」的机器自观面（会话/票/学习环/自观器官）
// 信号不落升件,死回合沿只日计一火;实证=7d 升件约 2/3 自观类噪声掩
// 真钱事故 3 条。收窄点=escEdgeUV 顶部（cmd/core 全探针共漏斗）;
// hunger 包直写升件面（face 册/覆盖/依赖对）不在本册——另案降级。
// O-32295 分级复升：唤知/learn_gap 移出本册——降级全禁言实证滞票
// 双盲（#26142滞44h+/32020/32021滞4h+只日计无人知）。现 learnGapEscCheck
// 自分级：计数带(票龄2-12h∧<3张)仍直 bump 本计数件同键日计（降级
// 语义保留盖噪声窗），热带(>12h∨≥3张)走 escEdgeUV 真升件径；册内
// 其余探针保日计不动（spec 禁线）。
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// 自观类降级册——键形=class∨class/scope 闭集显式册,新探针类默认不降级
// （升级须明册防钱径信号静默误吞）。钱径/数据/出带兜底三类永不降:
// 断流·对账·资金闸·数据缺·商业产出·死刑·唤知/cognition_dead（死人
// 开关——环死了升件无人读,但 alerter 人端须通）·动机 hunger_gap/
// paper_dry/breed_dry（钱径漏斗+研究唤触发）。
var selfEscDemote = map[string]bool{
	"唤知/audit_stall":          true, // 审计臂滞=自观器官,唤知面另有 cognition_dead 兜底
	"唤知/lesson_backlog":       true, // 教训矿积压=自观队列
	"唤知/spawn_orphan":         true, // 唤起孤儿会话=自观会话面
	"唤知/cognition_spawn_fail": true, // 唤知环唤起败簇=自观会话面
	"唤知/audit_spawn_fail":     true, // 审计臂唤起败簇=自观会话面
	"动机/goal_idle":            true, // 目标发起滞=自观动机器官
	"反射":                      true, // 反射编译器滞=自观器官
	"懂你":                      true, // reask_denied=自观器官拒回面
	"预期产出":                    true, // 会话产出期望差=自观会话面(expect_<unit> 动态 scope 同册)
	"孤儿账":                     true, // 注册表 pending 积压=自观册卫生(钱径册=断流面另探)
}

// 自观降级计数件——包级单例(coreService 值接收器不可携锁;escLoop 单实例
// 常驻语义=单写者,计数竞态面=测试并发调用须自同步)。
type selfEscMeterT struct {
	mu       sync.Mutex
	counts   map[string]int // "class/scope" → 本日计次
	flushDay string         // 上次日结日键(UTC YYYY-MM-DD——日界=自然计量窗)
}

var selfEscMeter = &selfEscMeterT{counts: map[string]int{}}

// bump 降级信号日计（量表落点=日结行非逐火行——日计数量表同语义）。
func (m *selfEscMeterT) bump(key string) {
	m.mu.Lock()
	m.counts[key]++
	m.mu.Unlock()
}

// flushDue 日结——日界翻转即落上日计数量表行(一日一键一行,append-only
// 语义下量表=日快照非累加器;键无计=无行如实)。返回计数属日（=昨日）。
func (m *selfEscMeterT) flushDue() (string, map[string]int) {
	today := time.Now().UTC().Format("2006-01-02")
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flushDay == "" {
		m.flushDay = today // 初态只立日键——重启后已计行不吞,留待首个日界结
		return "", nil
	}
	if m.flushDay == today {
		return "", nil
	}
	day := m.flushDay
	out := m.counts
	m.counts = map[string]int{}
	m.flushDay = today
	return day, out
}

// selfEscMeterFlush 日结量表写——escLoop 每拍查日界,翻日落一行/class-scope。
func (c coreService) selfEscMeterFlush(ctx context.Context) {
	day, counts := selfEscMeter.flushDue()
	if day == "" || len(counts) == 0 {
		return
	}
	for k, n := range counts {
		if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('探测席','量表','',$1,$2,$3)`,
			fmt.Sprintf("自观探日计 %s @%s——%d 火（降级件#22578①:自观面只计不升,钱径面照旧升件）", k, day, n),
			"自观计", k); err != nil {
			log.Printf("[esc] 自观计量表写败 %s: %v", k, err)
		}
	}
}
