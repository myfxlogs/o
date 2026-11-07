---
schema_version: 2
id: component-extract-shared-import-grep-all-consumers
kind: pitfall
risk: low
status: verified
keywords: [React, 拆分, 搬移, import, onLive, 消费面, grep, tsc, TS2304]
scope: [web, react, refactor, 组件抽取]
summary: 组件搬移拆文件时按「随件走」撤 import 会漏同名钩的其余消费面——onLive 被 LiveBadge 与页面回放守卫双消费，盲撤即 TS2304
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: doc
    ref: web/src/react/pages/Cockpit.tsx:13
  - type: doc
    ref: web/src/react/components/pipeline.tsx
---

## 事实/症状

T-0421 回退件复建拆分骨架（PipelineTile/LiveBadge 等自 Cockpit.tsx 搬出）：
import 面改道时按「onLive 随 LiveBadge 走」把 `import { onLive }` 一并撤出
Cockpit.tsx——tsc 即红 `TS2304: Cannot find name 'onLive'`（CockpitPage
断流复连回放守卫本就用 onLive，与 LiveBadge 是同钩双消费）。

## 根因或策略解释

搬移组件=搬走「组件体+它的依赖」，但依赖的 import 行是**共享资源**——
同名 import 的消费面=全文件作用域，非随组件一对一。按组件直觉撤行必漏
同文件其他消费点。

## 处置

拆文件改 import 面**先 `grep -n "<符号>" <源文件>` 清点全部消费行**，
消费面>1=import 留源文件（或两边都留），=1 才随件迁；撤后 tsc 兜底
（TS2304 是秒级反馈，别省）。T-0411 首拆时同一处没踩坑是运气序（import
行当时是另写的），回退再拆就露——拆分骨架类操作每次都要过这把尺。

## 回改点

后续任何组件搬移/拆分票，import 改道前 grep 全消费面；回退类票复建
拆分骨架时同尺（T-0421 实案在案）。
