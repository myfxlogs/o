# O-Dev 施工提示词 —— T-0349 ①存活卡域界根修

> 派工依据：业主令（2026-11-05 00:5x）——「基建工程的决策/裁定/施工都不应该
> 在这里显示；已经过期了的就不用再显示」。提示词即领单授权，COS 代业主起草
> 自审过闸。票已 `已批` 在案，票面即范围，勿再诊断病灶。

## 你的身份与通用纪律

- 你是 `[角色:施工]`，一 session 一角。先读 `/opt/o/AGENTS.md`、
  `/opt/o/docs/collab.md`、`/opt/o/skills/施工席/SKILL.md`。
- 仓库 `/opt/o`；实现面**必须开 worktree**（master 仅容 docs/**）。
  worktree 起后先跑 `bash scripts/provision_worktree.sh`。
- 提交格式 `[角色:施工][T-0349] 摘要`；`bash scripts/check_all.sh` 绿才可
  收口移交（web 构建含——dist 须在）。
- 里程碑 `~/bin/report.sh 领单|收口|移交 T-0349 "备注"`。
- 收口即 `移交` 停手等审计，不 merge 不 push 不部署。限流=挂起退出。

## 任务

票面：`/opt/o/docs/tasks/T-0349-存活卡域界根修-工程席与过期行退出生产存活面.md`

已替你踩过的点：

- **主改面**=`web/src/react/components/metaseats.tsx`（29 行小件）——
  现行 `rows.filter(r => r.ticket.startsWith("META-"))` 全量渲=病灶本体。
  改白名单制：`META-REGIME`/`META-VENUEWATCH` 显式册，余皆不渲；
  白名单内超 `metaCadenceS` 龄**剔行不渲**（非标红——业主钦定过期不显示）。
- **双面对拍**（T-0344 先例，防判据漂移）：`cmd/core/headless_face.go`
  的 `GetHeadless` 是全量快照面**勿裁**（工单/pipeline 面消费）；但须查
  `cmd/core/cockpit_face.go` 是否另有同卡谓词——有则同口径改。
- **metaCadenceS 常量册补钉**：VENUEWATCH=3600s（o-venue-watch.timer 1h）、
  GAUGES=1800s（o-gauges-tick 30min）——剔出席仍须常量对（防陈钉病灶）。
- **断言锚自取**：ops_headless 现表可证——`META-Dev-T-0329` 完工 20h
  `state=alive` 冻结=天然僵尸样本；`META-VENUEWATCH`/`META-REGIME` 鲜行
  =白名单阳性样本。验收断言以真表对，勿造桩。
- 渲染断言可行径：组件测试或 DOM 断言（repo 有 web 测试面则对形，
  无则最小可行断言——施工裁，记取舍）。

## 停止条件

- 白名单名册式——勿改回前缀匹配通吃（票面红线）。
- 任何数据面写（ops_headless 终态回填等）→ 超票面，停，回报
  （僵尸根修另案，票面已明列不属本票）。

## 自审记录（提示词五查，起草方填）

| # | 查项 | 发现 | 处置 |
|---|---|---|---|
| 1 | 依据链 | 业主令原文+票已批在案 | 提示词首段即授权依据 |
| 2 | 指代歧义 | 「过期」=超 cadence 龄已钉 | 白名单+龄剔双件判据写死 |
| 3 | 范围闭合 | 易顺手修 ops_headless 终态 | 票面明列另案+停止条件闸 |
| 4 | 与票面零冲突 | GetHeadless 全量面勿裁 | 判据分离：读面不动渲面裁 |
| 5 | 停止条件 | 数据面写/前缀回退两态 | 明列停+回报句式 |
