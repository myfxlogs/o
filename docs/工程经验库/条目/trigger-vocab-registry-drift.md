---
id: trigger-vocab-registry-drift
title: 结案触发词谱×认知集迁移覆盖漂移——新 trigger 词未入册=结案全误标 trigger_unknown
kind: pit
date: 2026-11-28
source: 工单 O-41388（值守晨报 dir_* 结案 trigger_verdict='trigger_unknown' 连环——fin_guard_close 18 张实证）
---

# 词册随码产必同步迁移入认知集——滞后=异常流污染（O-41388）

**坑形**：`guard.postmortem.known_triggers`（app_config 词册）是
postmortem 结案判词的合法词表——`fin_guard_close.trigger_kind`
词不在册即打 `trigger_unknown` 异常码进诊断径。dirtrig.go 四词
（dir_stop_loss/dir_take_profit/dir_ttl/dir_signal_flip）与
emergency_exit 落成后词册未同步——18 张真结案全被误标，诊断额度
为陈废册买单。

**同族病灶**：与 exempt-whitelist-shape-drift（O-41387）同构——
「码面产词面」与「册面认知面」两册分离必漂；能力前进册必须
同 commit 或紧随 commit 补，否则册=假否定源。

**根修三件套**：

1. **补词幂等迁移**（fin_0277——fin_0167 同径 `value || '"词"'::jsonb`
   追加）：新词入册不冲击存量序。
2. **历史判词净账**：`trigger_verdict='trigger_unknown'` 恰单码且
   guard 记 kind ∈ 新词集者撤码复净——`trigger_kind` 原貌审计链在
   fin_guard_close 不动，多码并存行不整撤。
3. **迁移覆盖对拍闸**（`check_trigger_vocab.sh` 入 check_all）：
   码面词谱三源（`closeTrig{kind:` 字面量+`dirTrigKind`/`ddTrigger`
   常量+postmortem 缺省 `manual`）逐词须见于提及 known_triggers 的
   迁移文件——新词忘写迁移=闸红，与 drill 白名单哨同模。

**判别式**：新增 trigger_kind 产点=同步迁移义务；词在码面而不在
认知集册=误标不缺席。
