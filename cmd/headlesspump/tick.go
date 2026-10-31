// tick.go —— 5s 周期拍（状态面秒级化钉案 2026-10-31，自 pump.go 拆出应行规模
// 棘轮 D-077/D-588）：对账/在飞 proc 实证/心跳三门控+写败退避自愈。态机本体
// 仍 pump.go——本件只含周期拍路径。
package main

import (
	"path/filepath"
	"strings"
	"time"
)

// tick 5s 拍（状态面秒级化钉案：驾驶舱 spawn/kill 可见性 ≤10s）。三事全门控：
// ①日志目录对账——fsnotify 漏事件自愈（新日志建档+已知日志尾扫；onLog 无新字节
// 零副作用）；②在飞票 /proc 实证——procSeen 票（曾实证活）殁即翻死 emit，不等
// 30min 静默判（kill 突殁无 exit 行=anchor b 病根）；未实证票不翻（spawn 起跑窗
// 防伪死——静默判兜底）；/proc 读败=本轮跳过不翻（瞬时抖动防假死）。③30s 心跳
// 拍——在飞行 updated_at 保鲜（终态行零写=非全量刷，anchor c）。写败退避期内
// 整拍跳过（emit 成功自愈回 5s——fail-loud 见 emit）。
func (p *pump) tick() {
	if p.degraded && time.Since(p.degradedAt) < backoffFor {
		return
	}
	for _, pat := range []string{prefixRun + "*" + logSuffix, prefixAudit + "*" + logSuffix} {
		fs, _ := filepath.Glob(filepath.Join(p.dir, pat))
		for _, f := range fs {
			base := filepath.Base(f)
			seat, pre := "Dev", prefixRun
			if strings.HasPrefix(base, prefixAudit) {
				seat, pre = "DM", prefixAudit
			}
			tid := strings.TrimSuffix(strings.TrimPrefix(base, pre), logSuffix)
			t := p.ensure(tid, "", "")
			if t.logPath == "" {
				p.ensure(tid, seat, f) // 首见建档（fsnotify 漏 Create 自愈）
				p.onLog(t)
			} else if t.logPath == f {
				p.onLog(t) // 现役件尾扫
			}
			// 非现役件跳过——双文件票（run/audit 同 id）offset 单锚随 logPath，
			// 交叉读即 size<offset 误判截短→全量重读→emitSoon 每拍重发（实测
			// 16 张 done 票每 5s 重写）。双件并发活动仍走 fsnotify 事件径。
		}
	}
	for _, t := range p.tks {
		if !t.procAlive || t.doneF {
			continue
		}
		_, m, alive := findProc(t.tid)
		if alive {
			t.procSeen = true
			if m != "" && t.model == "" {
				t.model = m
			}
			continue
		}
		if !t.procSeen || !procReadable() {
			continue // 未实证活（起跑窗）或 /proc 读败——不翻死
		}
		t.procAlive = false
		p.setState(t) // 变化门控——翻转才 emit
	}
	if time.Since(p.lastHb) >= heartbeatEvery {
		for _, t := range p.tks {
			if t.procAlive && !t.doneF {
				p.emit(t)
			}
		}
		if !p.degraded {
			p.lastHb = time.Now() // 拍净才推窗——带伤拍不吞窗（退避开即重试心跳）
		}
	}
}
