---
schema_version: 2
id: coground-cycle-seq-dual-host-race
kind: pitfall
risk: low
status: verified
keywords: [唤知会话, 轮次号, 双机唤起, 撞号竞态, 会话收口, unit键, 跳号]
scope: [唤起机双机唤起面, 会话收口行轮次字段, 连续自我账页配对]
summary: 唤起机双机（models/ant）并发唤起时，两会话各自「查max轮次+1」后写收口行——同窗互不可见致同号双写（#23511 第776双机同号、#23574 第780双写实证）。轮次号非权威键（唤起配对以 unit 为键、账页取末行不按号），撞号不毁账——撞见跳号避让或如实不纠皆合法，勿当病灶追查/立件。
first_seen: 2026-11-17
last_verified: 2026-11-18
skills: []
evidence_refs:
  - type: doc
    ref: ops_event#23511（收口行手感:coground-seq「并行唤起轮次号先查后写仍撞,跳号避让」）
  - type: doc
    ref: ops_event#23574（收口行手感:coground-seq 第二证,轮次780双写续跳号）
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE event='会话收口' AND note ~ '第7[0-9]+轮' ORDER BY id DESC
    observed_at: 2026-11-18T02:50:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

唤知会话收口行载「第 NNN 轮」序号。双机唤起机各自
spawn 会话时，两个并行会话在同窗内互不可见——
各自执行「读最近收口行轮次+1」得同号，双双写
出同轮次收口行。

实证（fin_eng_lesson#64 手感簇 coground-seq×2）：

- ops_event#23511（o-cognition-1794950874 第778轮）：
  「双机同号776竞态在案」+手感「并行唤起轮次号
  先查后写仍撞，跳号避让」；
- ops_event#23574（o-cognition-1794954713 第782轮）：
  「轮次780双写续跳号」——同 slug 第二证。

另见 2026-11-18 凌晨同 unit 串行会话 792/793 由
不同 unit 号会话交叉写轮，794 由第三 unit 续——
轮次=各会话自估序非全局发号。

## 根因

轮次号无全局发号器：每个会话冷启动读「最近收口行」
估出 N+1，check-then-act 窗=竞态窗。双机唤起同窗
并发即双写同号——设计内竞态非病灶。

## 防坑

- **撞号不毁账**：唤起配对以 unit（o-cognition-<pid>）
  为权威键，自态账页「上轮=」取时间序末行——轮次
  只是人读序号，不载状态不作 join 键。
- **撞见跳号避让**：冷启动读到「上轮第N轮」而近窗
  又见其他 unit 已写第N+1..M 轮（M>N+1）时，取
  M+1 续号——即「跳号避让」手感。
- **勿当病灶**：发现同号不升级、不立件、不追查
  「谁错了」——无对错，双机设计内竞态。
- **要真唯一序**：用 ops_event.id 自序或 unit 号——
  轮次字段不是那个东西。

## 可防一问

若撞号再来，哪个环节拦得住？——无环须拦：撞号
本身不是病（权威键是 unit+ops_event.id）。真防点
=「会话别把同号当异常」，本条即该防点。
