---
schema_version: 2
id: sweep-verdict-parser-format-drift
kind: pitfall
risk: medium
status: verified
keywords: [verdict_resolve, sweep, 判词解析, 状态字段, 票号字母尾, 审计回放, extract_status]
scope: [eval-sweep, tooling, 机读契约]
summary: "判词解析器 sed 谱漏 '## 状态'节题独占行/'### 状态：X'/'state: X'/判决节四形+票号字母尾截断撞号——15 票已判被误计判词缺位、T-0220b/c/d 静默漏审、'在飞单全灭'误伤开放态排除"
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: []
evidence_refs:
  - type: provenance
    ref: T-0249 清债排查 session（devin-headless）
  - type: doc
    ref: /opt/meta/eval/bin/verdict_resolve.sh
  - type: doc
    ref: /opt/meta/eval/bin/sweep_collect.sh
---

## 判词机读面三坑——解析谱/标识域/排除谓词（T-0249 实证）

- **解析器格式谱漏四形**：`verdict_resolve.sh` 原 `extract_status` sed 只覆
  行式 `状态：X`、表式 `| 状态 | X |`、节式 `## 态：X` 三形；审计席实际落点
  还有 ①`## 状态` 节题独占行+次非空行载值（T-0186/0191/0190/0189/0188/0084）
  ②`### 状态：X`（`状`字挡在 `态` 前，原 `^#+\s*态：` 正则失配——T-0138/0173/
  0179）③英文 `state: X`（T-0103）④`## 无头审计判决` 节+次行载判词（T-0074）。
  15 票判词已落 worktree 副本被误报「判词缺位」。另判词候选取**末位**载判决
  键值者——票面先存态行（如 `状态：✅验收合入部署`）会压后 append 的审计
  判词行（T-0138 实证）。
- **票号字母尾截断撞号**：`sweep_collect.sh` `grep -oE 'T-[0-9]+'` 把
  T-0220a/b/c/d 全截成 T-0220——队列四行同 tid，driver `spawn_one` 只取
  首行 file（T-0220a）审计，reap `qset` 按 tid 全行同写 state——**b/c/d 三票
  零审计被标 fail**。修=`T-[0-9]+[a-z]?` 保尾。
- **状态值不截 `|` 致开放态误伤**：`> 状态：**完成** | 领取：… | 优先级：…
  在飞单全灭）` 行式字段值吞整行尾，`is_open` 的 `*在飞*` 谓词命中 `在飞单
  全灭` 子串——T-0257 完工票被误排枚举面。修=值面先截 `[[:space:]]+\|`。

## 处置

`verdict_resolve.sh` extract_status/extract_verdict 重写为 awk 全谱枚举
（行/表/节三族+节题独占行+英文键+判决节，末位判词胜）；`sweep_collect.sh`
source 共用解析器+tid 正则可保字母尾。机读契约教训：**票面字段是机读 API
非散文**——解析器必须按「写者实际落点谱」而非「模板钦定三形」设计，枚举面
变更须对拍存量写者全集。
