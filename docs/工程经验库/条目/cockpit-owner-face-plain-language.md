---
schema_version: 2
id: cockpit-owner-face-plain-language
kind: invariant
risk: medium
status: verified
keywords: [cockpit 人话面, 机器原文禁入前端, net_apr 拒因, hover 悬停, 态标映射, 业主面]
scope: [web, cockpit 管道瓦片, FlowStage status, React 渲染层]
summary: cockpit 可见面=人话面——机器原文（net_apr/阈/armed/snake_case 态词）禁入前端渲染树任何层含 title/aria；态标一律过静态人话映射表（可见短句+悬停详述带数值），未识词落人话兜底宁失细节不露原文；原文只留 DB/日志面
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: doc
    ref: web/src/react/components/pipeStatus.ts
  - type: doc
    ref: web/src/react/components/flowpulse.tsx
  - type: doc
    ref: docs/tasks/T-0366-管道瓦片拒因hover化-可见面人话.md
---

## 事实/症状

- 业主令 2026-11-05（T-0366 票面直录）：「括号黑话看不懂还拉错位，鼠标放上去才显示具体原因」「前端是人在看不是机器在看」。
- 病灶形态：服务端态标串（如 `闲·评估全拒（末因 net_apr<阈）`、`活跃·3 armed`）原样进可见面与悬停——业主面读不懂机器词，长括号串还把七段瓦片撑错位。

## 根因或策略解释

- cockpit 是业主面不是调试面——机器原文（字段名/枚举态/判据符）属于 DB/日志层事实，渲染层最后一公里必须翻译成人话；原文进前端任何层（含 title/aria/DOM 文本）即破面。
- 拒因类词随评估判据演化不可枚举——静态映射表只能覆盖常见形态，必须有哨兵兜底：映射后仍含机读特征（snake_case/armed/阈）则整句落人话兜底，宁失细节不露原文。

## 处置

- 渲染层置 `humanStatus` 单一翻译闸（pipeStatus.ts）：态标串→{可见短句, 悬停详述} 二态输出，映射表静态钉死随代码走不接 LLM。
- 悬停详述带数值：可读句+同源现字段实数（如评估段 count24h→「近 24 小时评估 N 个机会全部未过放行线」），数据源/proto/查询零动。
- 长串不撑版：瓦片 flex 等分+态标 ellipsis 截断，全文走悬停详述——人话化与等宽互为表里。
- 机检锚：DOM 断言机器原文不出现在渲染树任何层（含 title），未识拒因兜底断言在案。

## 证据

- check: `cd web && node scripts/flowpulse-domtest.mjs`
- result: 23/23 PASS（⑥人话面 7 锚：可见短句/原文禁入渲染树/悬停详述带数值/执行段态词/活态/未识兜底/等宽截断）
- ref: T-0366

## 反证条件

- check: 生产新拒因形态进库后渲染面仍露 snake_case/机器词
- falsifies_when: machineLeft 哨兵未覆盖的新机读特征出现（如 camelCase 态词、数字编码态）——哨兵词表须随评估判据词集同步扩

## 适用边界

- 面界=cockpit/复盘等业主可见渲染层；DB 列值、日志、CLI 诊断面、proto 字段名不受此约束（原文是那层的正本）。
- 映射表只译服务端态标串，不译业务实体名（venue/inst/ref 属业主识读域）。

## 伴生量表

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
- 豁免由：渲染层文本不变式——机检由 flowpulse-domtest.mjs ⑥断言锚承（无 gauges 适配表）。
