# T-0233 瓦片让位（FLIP）前端引入

| 字段 | 值 |
|---|---|
| 状态 | 已批 |
| 立项 | `[角色:决策]` @ 2026-10-29 |
| 领取 | （session 标识 @ 时间；空=未领） |
| 优先级 | 中 |
| 涉及面 | `web/src/react/motion.ts`（新增，FLIP hook + 时序 token；或并入 `hooks.ts` 施工定一处）；`web/src/react/pages/Overview.tsx`（MoneyTree 接线）；`web/src/react/styles/20-cards.css` 或 `22-effects.css`（token 变量与 reduced-motion 降级）；`web/package.json`（**只读锚：不得新增依赖**） |

## 背景

业主指名 scaffold-master 的瓦片让位交互（打开一个瓦片，其余瓦片平滑让位）。机制选型已决 **D-0037**：无依赖 React FLIP hook + CSS 让位式布局，**不引 GSAP / framer-motion**。

## 目标

Overview 页 MoneyTree 树头展开/收起时，同级树头与下方内容平滑让位（无瞬跳）；FLIP 机制沉淀为可复用 hook，后续瓦片面（占行展开等）直接取用。

## 验收标准

- [ ] `web/package.json` 无新增依赖（hook 零第三方动画库）
- [ ] MoneyTree 树头展开/收起：同级节点以 transform 过渡滑到新位（约 200–350ms），无瞬跳
- [ ] `prefers-reduced-motion: reduce` 下展开直落终态，无过渡
- [ ] 连续快速开/合（打断）不错位、结束后元素 computed transform 为 none（不残留）
- [ ] 键盘触发（Enter/空格开合）与指针行为一致
- [ ] `npm run build` 与 `npm run typecheck` 全过（exit code 贴进结果段）
- [ ] hook 注释写明使用约束：只动 transform/opacity、勿与同属性 CSS transition 混挂、测量须在 useLayoutEffect 内

## 范围外

- 不改 MoneyTree 信息架构与数据面（proto/Connect 不动）
- 不做跨页共享元素过渡、拖拽重排
- 不新增瓦片展开位（`grid-column: 1/-1` 占行让位待有真实内容面另立单）
- 不动既有呼吸灯（22-effects / 10-shell）

## 参考

- 决策档：`docs/decisions/D-0037-瓦片让位交互引入.md`（含源项目机制拆解与码坐标）
- FLIP 技法：First-Last-Invert-Play——量旧位 → 更新 → 量新位 → 反向 transform → 过渡清零
- 源项目（只读勘案，Vue 件不可照抄）：github.com/wbr664393425-lgtm/scaffold-master `scaffold-vue/src/composables/motion.ts:488`

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
