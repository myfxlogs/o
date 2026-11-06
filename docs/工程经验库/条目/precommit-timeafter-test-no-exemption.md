---
schema_version: 2
id: precommit-timeafter-test-no-exemption
kind: pitfall
risk: low
status: verified
keywords: [pre-commit, D-0008-B, time.After, sched-exempt-ok, 桩测, 期界等待]
scope: [工程纪律, 机检门禁, 测试面]
summary: pre-commit 拦 time.After 无测试文件豁免——桩测期界等待也须行内 sched-exempt-ok 锚
first_seen: 2026-11-06
last_seen: 2026-11-06
evidence_refs:
  - type: provenance
    ref: T-0378 施工（lifecycle/hunger_test.go 桩测两轮被拦）
---

## pre-commit 周期原语闸无测试豁免——桩测 time.After 一样挨拦

- **实证**：T-0378 hunger_test.go 桩测用 `time.After` 作拍锚超时守卫（轮询
  Detect 计数+净退等待）——pre-commit 钩（D-0008-B）两轮拦下：
  `BLOCK(D-0008-B): 新增周期调度原语`。闸扫 `git diff --cached` 新增行正则
  NewTicker/Tick/After/Sleep，**无 `*_test.go` 豁免分支**（不同于 check-lines
  棘轮闸明豁免测试文件）。
- **教训**：写桩测涉及节拍/超时等待时，`time.After` 行内直接带
  `// sched-exempt-ok 桩测期界等待（…非生产调度）` 一次到位——省一轮
  commit 拒。生产面 ticker 同理（行内锚，勿写块尾）。
- **收口**：hunger_test.go 三处 time.After 行内锚注后过闸（T-0378
  ba4c567d）；check_all 全链同闸（check_no_forbidden_patterns.sh 面）
  无红。
