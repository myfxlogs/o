---
id: spec-threshold-needs-prod-max-check
title: spec 钦定阈值先对真库极值核——N=2 变零覆盖死闸
kind: pit
date: 2026-11-24
source: 工单O-38942 施工（dead-signature 抑制闸阈 N 取舍）
---

# spec 建议阈值照抄=死闸（O-38942 实证）

**坑形**：spec/设计档给判据阈值（「尸签≥N 次才抑制」N=2）——施工
直抄落码，桩测绿、check_all 绿、验收全过，但上线后**零触发**：
真库实测 1023 条 verdicted 尸体按签名粒度
(template_ref,inst_id,direction,reason) max=1，N=2 闸永远不
满足——消费件存在但行为面是死代码，且「零触发」与「闸正常无命中」
在账面上不可分（死闸静默）。

**判别**：阈值类判据（抑制/告警/冷却次数）落地前，先对生产分布
跑 `SELECT key, count(*), max(n) FROM ... GROUP BY key` 取真实
max——spec 建议值常凭直觉定，真库粒度可能根本到不了。

**修法**：取 N=min(spec建议, 真库max)，并把分组粒度保留（reason
等维度不并入主键就有余量）；阈值作为包级常量集中一处，矿化成熟
收紧只动常量不动结构。偏差在 commit message 申报+回执注明，不静默。

**推广**：凡「阈值/窗口/上限」类 spec 参数，验收证据必须含「真库
分布下该阈值可达」的实证行；纯桩测绿不构成阈值正确性证据。
