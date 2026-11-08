---
schema_version: 2
id: systemd-workingdir-chdir-cold-start-deadlock
kind: pitfall
risk: medium
status: verified
keywords: ["systemd", "oneshot", "WorkingDirectory", "200/CHDIR", "ExecStartPre", "冷启动", "mkdir", "timer 装机"]
scope: [deploy/*.service, systemd user units, timer 装机, meta 拍族]
summary: oneshot unit 的 WorkingDirectory 指向 ExecStartPre 才 mkdir 的目录=冷启动死锁：systemd chdir 先于 ExecStartPre，目录缺席即 200/CHDIR 永败且永不自愈（dir 永远建不出来）。幸存实例全靠手工建目录侥幸活。正形=unit 零 WorkingDirectory（脚本自 mkdir+全绝对径）。
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: [施工席]
evidence_refs:
  - type: doc
    ref: deploy/o-venue-watch.service（修正本——去 WorkingDirectory 注释在案）
  - type: server_fact
    ref: 2026-11-05 00:32 首拍 journal：ExecStartPre status=200/CHDIR（/var/tmp/o-venuewatch 缺席）；修后同径 0/SUCCESS
---

# WorkingDirectory 指 ExecStartPre 才建的目录=冷启动 200/CHDIR 死锁

## 事实/症状

T-0348 首拍（o-venue-watch.timer enable --now 后首触发）：
`ExecStartPre=/usr/bin/env mkdir -p /var/tmp/o-venuewatch (code=exited, status=200/CHDIR)`
——服务永败。且**永不自愈**：ExecStartPre 要跑 mkdir 就得先过 chdir，chdir 又要目录已在——鸡生蛋死锁，每次触发同败。

对形件 o-gauges-tick.service 同形（WorkingDirectory+ExecStartPre mkdir 双写）却活着——
非因正形，因 /var/tmp/o-gauges 早被手工/此前运行建好。侥幸存活冒充正形，冷装机即翻车。

## 根因或策略解释

systemd 启动顺序：chdir(WorkingDirectory) → ExecStartPre → ExecStart。
chdir 对缺席目录直接 200/CHDIR 退出，ExecStartPre（mkdir 的载体）根本没机会跑。
unit 内「自举目录」的意图与 systemd 的顺序假设天然相抵触。

## 处置

正形二选一：
1. **unit 零 WorkingDirectory**（钦定）——meta 拍族脚本本就全绝对径且自 `mkdir -p $ST`，cwd 无关紧要；删行即冷启动安全。
2. 装 machine 时先手工建目录（治标——忘建即复发）。

## 证据

- check: `systemctl --user start o-venue-watch.service && systemctl --user status o-venue-watch.service`
- result: 修前 200/CHDIR；修后 ExecStartPre 0/SUCCESS + ExecStart 0/SUCCESS + hb alive 同拍落
- ref: deploy/o-venue-watch.service 修正注释；T-0348 票面结果段

## 反证条件

- check: 全新主机（/var/tmp/o-venuewatch 缺席）enable --now o-venue-watch.timer
- falsifies_when: 首拍 SUCCESS（说明顺序假设有误或 systemd 版本行为不同）

## 适用边界

systemd oneshot/timer unit 族；WorkingDirectory 指向运行期才存在的路径。
脚本若真需 cwd（相对径消费者），改为 ExecStartPre 建目录+ExecStart 内 cd，或装期预建。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: heartbeat_dead（既有——unit 永败=META 行 stale，>7d 幽灵闸兜底显形）
- class: 幽灵
- query: SELECT count(*) FROM (SELECT ticket,seat,max(last_activity) m,regexp_replace(ticket,'^META-Dev-','') tno FROM ops_headless GROUP BY ticket,seat) t WHERE m<now()-interval '7 days' AND ((ticket LIKE 'META-%' AND ticket NOT LIKE 'META-Dev-%') OR ticket LIKE 'o-%' OR ticket=seat OR (tno<>'' AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=t.tno AND e.event IN ('裁决','完成','驳回')) AND EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=t.tno AND e.ts>now()-interval '7 days')))
- threshold: 1
