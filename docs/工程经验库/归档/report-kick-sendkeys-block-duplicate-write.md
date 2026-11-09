---
schema_version: 2
id: report-kick-sendkeys-block-duplicate-write
kind: pitfall
risk: low
status: retired
keywords: [report.sh, REPORT_KICK, tmux send-keys, O-DM, ops_event 重行,
  无头席, kick 链]
scope: [~/bin/report.sh, ops_event 流水面, dm 班次裁行落账径]
summary: report.sh 继承 REPORT_KICK=O-DM 时 tmux send-keys 可阻塞（pane
  输入未排干）；杀子进程留 bash -c 包装壳继续走完剩余调用链→每事件
  各落一遍=裁决/完成行重行。批写裁行前置 REPORT_KICK=（kick 纯加速，
  ops_event 在账即不丢）。
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: doc
    ref: ~/bin/report.sh KICK 段（send-keys -l + C-m 无超时兜底）
  - type: commit
    ref: T-0355/0357/0358 裁决班实证——单批六连报：首调卡 send-keys
      O-DM pane 数分钟，kill 子进程后包装壳续调剩余五件全落行；
      REPORT_KICK= 重跑再落一遍→三票裁决/完成各双行（通过行
      task_transition 去重径仅一行）
---

## 事实/症状

无头席批量 `report.sh 裁决/完成` 时：首调挂在
`tmux send-keys -t O-DM -l`（REPORT_KICK 自环境继承，O-DM pane 存在
但输入未排干即阻塞——kick 段在 feishu/DB 写**之前**，阻塞=行未落）。
`kill` report.sh 子进程后，`bash -c "...; report.sh A; report.sh B"` 包装壳
不死、继续顺序调用剩余 report.sh——这些调用各自（可能再挂或超时后）
**照常落 DB 行**。随后以 `REPORT_KICK=` 重跑全批再落一遍：

- `裁决` 调用 → ops_event `裁决` 行 ×2（report.sh 直写）；
- `通过` 行 ×1（meta.task_transition 同事务写——函数侧判重，第二遍
  同 (ticket,to) 不再行）；
- `完成` 行 ×2。

## 根因

kick 段无超时/无熔断（send-keys 卡则全卡）；kill 只死当步子进程，
父壳 `;` 链续跑——「杀停了」的直觉与实际续写行为相反。

## 处置

- 无头/批量裁行落账前置 `REPORT_KICK=`（空）——踢径全跳，
  ops_event 行照常在账（kick=纯加速件，链哑不丢票）；
- 要杀批量链须杀 `bash -c` 包装壳 PID 而非当步 report.sh 子 PID；
- 重行在 append-only 流水面=噪音非脏账（dm_queue 最新刻逻辑不受影响，
  不删行不补删）；
- 若再犯面广：kick 段加 timeout 包装或挪至 DB 写之后（修法另票）。
