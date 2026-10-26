---
schema_version: 2
id: skills-index-drift-via-cross-face-commit
kind: pitfall
risk: low
status: verified
keywords: [kbgen, skills, 索引, 漂移, pre-commit, 闸, 坑引, 经验库]
scope: [scripts/hooks, tools/kbgen, skills/, 工程经验库]
summary: 生成物索引的派生列可跨面漂移——skills/索引.md 坑引列取自经验库技能清单.md，而 pre-commit 8b 门只在 staged 触 skills/ 时跑 --check，经验库-only commit（坑条目+kbgen gen）照常过闸、索引漂移静默上 master（T-0052 验收实案：五枚新坑入档后 --check 红）；派生依赖面 ⊆ 触发面才闭环，跨面派生须把依赖面也纳入触发面
first_seen: 2026-10-26
last_verified: 2026-10-26
skills: [策略评估与仓位]
evidence_refs:
  - type: code_fact
    ref: tools/kbgen/skills.go——坑引列=解析 docs/工程经验库/技能清单.md；scripts/hooks/pre-commit 闸 8b——修复前触发面仅 `grep -qE '^skills/'`，master 实测 `kbgen skills --check` exit 1（dual-listenloop-serial-starvation 等 5 枚新坑在坑引列缺）
  - type: commit
    ref: e7dffc2
  - type: doc
    ref: docs/tasks/T-0052（验收毕段）+T-0158（闸触发面修复单——原编号 T-0154 撞号让号重编）
---

# 生成物索引跨面派生列——触发面不含依赖面=闸静默漏漂移

## 病灶形

`kbgen skills` 产 `skills/索引.md`：坑引列自 `docs/工程经验库/技能清单.md`
反向解析条目 id。pre-commit 8b 门（T-0052 立）：**staged 触 `skills/`**
→ `kbgen skills --check`。于是：

1. 施工 commit 只动 `docs/工程经验库/`（坑条目+`kbgen gen` 重建经验库索引）
2. 8b 门触发面 `^skills/` 不命中——**不查**
3. 坑引列已实质变化（新坑 id 该入列），索引陈旧
4. 漂移静默上 master，直到下次有人跑 `--check` 才红

T-0052 验收（2026-10-26）实案：当日 T-0143/0144/0147 五枚坑条目入档后，
master 上 `kbgen skills --check` exit 1。

## 根因

**派生依赖面 ⊋ 触发面**。闸设计时只想到「改 skills/ 该重建索引」，
漏了「改坑引的源（经验库）也改变索引的正确值」。生成物的输入有两面，
闸的触发面只盖了一面。

## 处置形

- 触发面 ⊇ 依赖面：闸 grep 扩为 `^skills/|^docs/工程经验库/`（T-0158 修复单）
- 通律：凡「A 生成的产物内容取自 B 面」——A 面闸的触发面必须含 B 面；
  派生链每加一个输入源，触发面同步扩，否则闸=装饰
