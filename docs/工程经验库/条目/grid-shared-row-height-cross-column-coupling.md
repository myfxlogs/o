---
schema_version: 2
id: grid-shared-row-height-cross-column-coupling
kind: pitfall
risk: low
status: verified
keywords: [CSS grid, 行高耦合, 列联动, 瓦片间距, align-items start, .quad, 驾驶舱版式, exch-head 展开]
scope: [web, CSS 版式容器, System 页, Cols]
summary: 多列 grid 同行行高=行内最高卡——一卡展开撑高整行，他列瓦片被下推+短列卡间撑出大空白；align-items:start 只防卡身拉伸不防行轨耦合。修法=独立列容器（每列独立 flex 上下文，列内 gap 恒定）
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: [施工席]
evidence_refs:
  - type: doc
    ref: web/src/react/styles/20-cards.css（.cols/.cols-col/.quad 面）
  - type: doc
    ref: web/scripts/quadcols-domtest.mjs
  - type: doc
    ref: docs/tasks/T-0389-驾驶舱版式-瓦片间距与列联动解耦.md
repro_or_symptom: 业主实屏两症——①瓦片间空白过大（短列卡底到下一卡顶被行轨撑开）；②采集卡 exch-head 展开时左列整列被向下拉、整页抻高
root_cause: .quad 旧构=grid 两列多行——grid 行轨高恒取同行最高卡，行内两列物理共享行高；短卡列的卡间距离=行高-卡高+gap 被动放大；一卡展开→整行增高→下一行两列同沉。align-items:start 只令卡身不拉伸（白留在卡外行轨里），行轨耦合本身不除
fix_and_prevention: 独立列容器（Cols 件：每列一 .cols-col flex 上下文，列内 gap 恒 24px）——他列几何物理不可达；grid 只留列宽分配（≥1100px 1fr 1fr+align-items:start），行轨语义退位。凡多列瓦片面要「各列独立排布」时禁多行 grid，列归组用 Cols
notes: floats 不等价——CSS2.1 §9.5.1 规则5：float 顶高受一切先前行内 float 顶约束（非仅同侧），非对称高度下仍有阶梯耦合；CSS columns（.masonry）列归组由 balance 算法动态分配，固定列语义（生产|采集分列）不可保
---

## 现象

System 页 `.quad`（2 行 2 列 grid）业主实屏两症：①「2个瓦片中间的空白太多」——短列卡底到下一卡顶被行轨撑出大空白；②采集卡内 exch-head 展开（exch-rows 块插入）时左列主机资源/管道整列被向下拉、整页抻高。

## 根因或策略解释

grid 布局中**行轨（row track）高度=该行所有格内最高项**——行内两列瓦片物理共享行高，此为单根双症：

- 短卡列：行高被对列高卡顶起后，本列下一卡必须从下一行轨起排——卡底到下一卡顶的距离=行高−本卡高+gap，被动放大且随对列内容波动（症①）；
- 展开时：右列卡增高→行轨增高→下一行两列同沉（症② 左列被下拉）。

`align-items: start` 只令卡身保持固有高不拉伸，空白仍在共享行轨里——不解决行轨耦合本身。

## 处置

改 `Cols` 独立列容器（web/src/react/components/ui.tsx）：每列一 `.cols-col`（flex column、gap 恒 24px、min-width:0），各列成独立格式化上下文——他列几何物理不可达；grid 只留桌面列宽分配（`.quad` ≥1100px `1fr 1fr`+`align-items:start`）。凡多列瓦片面钦定「各列独立排布零联动」时，列归组用 Cols，禁多行 grid 平铺。

## 证据

- check: `node web/scripts/quadcols-domtest.mjs`
- result: 11/11 PASS——左列各卡 y 在右列首卡 80→400 展开下零变；本列后续卡 y 增量=展开量；列内间距恒 24px
- ref: T-0389 收口 commit（t/0389-cockpit-col-decouple）

## 反证条件

- check: 对 `.quad` 子级回退平铺 grid（去 cols-col 壳）再跑桩测
- falsifies_when: 若平铺 grid 下左列 y 仍不变，则行轨耦合断言传过头（几何桩分组断言先红——当前桩对结构红）

## 适用边界

- 仅适用「各列独立堆叠」语义面；真正需要同行对齐的格面（表格式对齐）仍用共享行轨 grid；
- 窄屏单列回落时 DOM 序=列主序（整列1先于整列2）——行序交错与列独立物理不可兼得，钦定语义取后者。
