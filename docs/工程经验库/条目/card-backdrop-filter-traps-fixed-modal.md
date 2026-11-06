---
schema_version: 2
id: card-backdrop-filter-traps-fixed-modal
kind: pitfall
risk: medium
status: verified
keywords: [backdrop-filter, position:fixed, 包含块, stacking context, modal, 弹窗遮挡, 保存键不可点, 玻璃卡, z-index 失效]
scope: [web, CSS, Card, Modal]
summary: .card 的 backdrop-filter 使卡成 fixed 包含块+叠层上下文阱——卡内渲 .modal-overlay(fixed) 只覆卡界、z-index 被困卡内，后续平铺卡把弹窗尾部盖住（保存键不可点）；修径=浮层作卡外兄弟节点渲
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0406-驾驶舱UI两修-弹窗遮挡与UTC时标.md
  - type: doc
    ref: web/src/react/styles/20-cards.css:182
  - type: doc
    ref: web/src/react/components/QuoteBoard.tsx
---

## 事实/症状

业主实测（T-0406 病灶①）：自选报价编辑弹窗（WatchlistModal）尾部被「④ 待开仓」
瓦片盖住，保存按钮不可点。`.modal-overlay` 明明 `position:fixed; inset:0;
z-index:200`，却既没覆盖视口也没压过平铺卡。

## 根因或策略解释

CSS 规范：`backdrop-filter`（同 `filter`/`transform`/`perspective`/`contain:
layout`/`will-change` 族）使元素成为**后代的 fixed 定位包含块**，同时生成
**叠层上下文**——

- `.card`（Liquid Glass 件，`backdrop-filter: blur(20px) saturate(180%)`）
  内任何 `position:fixed` 后代的 `inset:0` 以**卡框**为参考系而非视口；
- 后代 `z-index:200` 只在该卡叠层上下文内竞争——兄弟卡（同为 .card=各自
  叠层上下文、DOM 序靠后）整卡盖在弹窗子树上。

即「fixed+高 z-index 依然被平铺件盖住」的反直觉病灶；弹窗越高越明显
（尾部越出卡界被遮）。

## 处置

浮层改为**卡外兄弟节点**渲染（QuoteBoardView 返回 `<>`：Card + 卡外
WatchEditHost），overlay 的 fixed 归视口、z-index:200 压过全部平铺卡
（侧 nav z≤30）。备选径 createPortal→body（本仓未引，domtest SSR 面更简
的先赢）。同票补 `.modal{max-height:90vh}+列内滚` 保任意列高下保存键可达。

排查同族阱先 grep 祖先链 `backdrop-filter|filter|transform|perspective|
contain|will-change|content-visibility`——本仓 `.page/.main/.duo/.cols-col`
均净，`.card` 是唯一阱位。

## 证据

- check: `cd web && node scripts/quoteboard-domtest.mjs`
- result: 「⑧浮层出卡渲（overlay 在 card 外）」断言=markup 中
  `.modal-overlay` 位于 `</section>` 之后（修前该断言红）
- ref: T-0406 收口 commit

## 反证条件

- check: 把浮层渲回卡内并移除 .card 的 backdrop-filter
- falsifies_when: 若移除 backdrop-filter 后卡内 fixed overlay 恢复视口行为，
  证因果成立（规范行为，非实现 bug）

## 适用边界

适用于一切「在带 filter/transform/backdrop-filter/contain 的祖先内渲
position:fixed 浮层」的组合；React/Vue/原生同。本仓当前唯一浮层=
WatchlistModal；后续新浮层若回卡内渲即复发，domtest 断言挡着。

## 伴生量表

- gauge_name: n/a（CSS 静态阱非运行时量表域——无适配 gauges.tsv 表；复发
  防=domtest 结构断言在案非巡检指标）
- class: n/a
- query: n/a
- threshold: n/a
