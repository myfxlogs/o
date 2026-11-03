# 施工返修①提示词：T-0314——gate 键 seed+skillgen frontmatter+阈读键

**角色**：`[角色:施工]`　**票**：T-0314　**分支**：`t/0314-econ-lesson-loop` 续提
**被驳回 commit**：`8483d125`（骨架全对，三处落地即死/偏差）

## R1：`gate.*` 键零 seed——机械消费腿物理死锁

gatefx 钦定「key 不在插行只 UPDATE 在册键」——而 `fin_0102` 一个 `gate.*`
键都没插，app_config 生产面也零 gate 键。**任何 Apply 永远返「键不在册」**。

修：fin_0102 追加 seed 块（迁移未部署可直改文件）：

```sql
INSERT INTO app_config(key,value) VALUES
  ('gate.kill.funding_carry','false'::jsonb),
  ('gate.lesson.cost_diverge_pct','20'::jsonb)
ON CONFLICT (key) DO NOTHING;
```

`gate.kill.funding_carry`=funding runner 实读的键（run.go 已接）；
`gate.lesson.cost_diverge_pct`=R3 同坑键。将来新族 kill 键由裁定票追加 seed
（「key 不在插行」防线保留——键先 seed 后 Apply，两拍分权是钦定语义）。

## R2：skillgen 生成件无 frontmatter——技能系统不认

`tools/skillgen/main.go` 的 doc 模板缺 `---name/description/domain---` 头——
`skills/ctx-sources/SKILL.md` 逐字对形。修模板：

```
---
name: econ-<slug>
description: <kind>:<ref> 经济案 canon（fin_econ_lesson:<id>）——裁定回改点见案由
domain: 经济
---
```

## R3：costDiverge 阈硬编 0.2

`internal/review/run.go` `costDiverge` 里 `>0.2` 改读
`cfg.GetFloat("gate.lesson.cost_diverge_pct")`（默认 20——Runner 若无 cfg
注入面，照 scout.AttachCfg 先例补 Attach 径）。

## 验收

- fin_0102 全量跑：`gate.kill.funding_carry`/`gate.lesson.cost_diverge_pct`
  两键在册；
- `skillgen` 产出件含 frontmatter 三行；
- `go test ./internal/review/ ./internal/research/funding/` 绿；
- gatefx.Apply("kill_trigger:funding_carry") 对 seed 键真 UPDATE 成（测试
  桩或 SQL 实迹）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 「key 不在插行」防线与 seed 缺一以为是施工漏 | 判责误 | 提示词原文没钉 seed 键——规格缺口在决策侧，R1 定性=双方各半 |
| 2 | 新族 kill 键将来靠谁 seed | 流程残留 | R1 注明=裁定票追加 seed（两拍分权语义写清） |
