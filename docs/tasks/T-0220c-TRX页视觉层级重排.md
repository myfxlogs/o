# T-0220c TRX 页视觉层级重排——核心指标条+次级瓦片+明细层三档

| 字段 | 值 |
|---|---|
| 状态 | 完成（判过并码在案，11-01清账翻面） |
| 立项 | `[角色:决策]` @ 2026-10-28（设计档 `docs/handoff/T-0173-最优解设计修订.md` §三/§五；票面文件由施工席依决策席 dispatch+设计档原样物化——b46be9d 只落设计档未落票文件，来源留痕交验收核） |
| 领取 | glm-t0220 @ 2026-10-28 13:01Z |
| 优先级 | 高 |
| 涉及面 | `web/src/react/pages/Trx.tsx`、`web/src/react/components/TrxKpiStrip.tsx`（新件）、`TrxYieldRadar.tsx`（组合两格上移后瘦身）、`styles/20-cards.css`（KPI 条/明细层段） |

## 背景

现状四瓦片等宽等高：核心数字（年化/待领取/强平距离）与次要标签视觉权重相同。设计令三档层级：核心指标条（大数字优先）→次级瓦片→明细层。

## 目标

```
核心指标条：组合 7d 年化 | 组合 30d 年化 | 待领取 | 强平距离（大数字 2em+呼吸灯+独占首行）
次级瓦片：收益率雷达（能量 7d/30d+投票当前/30d 四格）‖ 对冲结构（双列并排）
明细层：统一流水（全宽 13px）
```

## 验收标准

- [ ] 核心指标条 4 KPI 卡独占首行：大数字 2em（tabular-nums）+呼吸灯伴随+卡背景微调（较常规卡加重一档）；窄屏回落 2×2
- [ ] KPI 取值：组合 7d/30d=GetTrxYieldCurve annualized、待领取=GetTrxVoteStatus pendingTrx、强平距离=HedgeStructure legShort.distPct；挂载取数+trx 信号帧重取（径同现役卡）
- [ ] 呼吸灯状态色零新前端判据：强平距离灯随服务端 platformAlertState（normal=绿呼吸/warn=黄/breach=红静止）；年化/待领取=正负号显色（现有 u-good 先例）
- [ ] 收益率雷达瘦身为四格（组合两格上移 KPI 条，不双显）；对冲结构+雷达双列并排（.duo 先例）
- [ ] 明细层统一流水全宽、表格字号 13px
- [ ] `tsc --noEmit`+`vite build` 绿；四态目检（就绪/加载/空/错）过

## 范围外

数据面（T-0220a/b）；玻璃材质全局件（T-0220d——KPI 卡复用 .card 材质自动继承）；Overview/System/Review 页布局。

## 参考

- 设计档 §三（版式 ASCII+三档规则）；现状 `Trx.tsx`/`TrxYieldRadar.tsx`（6 格 grid+carry 行）/`TrxHedgeCard.tsx`/`HedgeFlow.tsx`
- 呼吸灯语：`10-shell.css` breatheCalm（绿缓息/黄急促/红静止全站同款）

## 结果回填

- 实现摘要：`TrxKpiStrip.tsx`（新件）——4 KPI 卡（组合 7d/30d=curve.annualized×2、待领取=voteStatus.pendingTrx、强平距离=structure.legShort.distPct）；挂载取数+trx 信号帧重取。`Trx.tsx` 三档布局：KPI 条独占首行→`.duo` 雷达‖对冲双列→统一流水全宽。`TrxYieldRadar` 6 格→4 格（组合两格上移不双显）。CSS `.trx-kpi-strip`（4 列→窄屏 2×2）+`.kpi-v` 2em tabular-nums+`.hedge-flow-table` 13px。
- 灯语判据：强平距离灯=服务端 platformAlertState（normal 绿缓息/warn 黄/breach 红静止）零新前端阈值；年化/待领取=正负号显色（现役先例）。
- 测试证据：`tsc --noEmit`+`vite build` 绿；TRX 页真数据目检截图——KPI 条（7.64%/5.49% 绿大数字+呼吸灯+样本不足注+待领取 0.00 琥珀静止灯+强平距离 127.2%）/雷达 4 格/流水 13px 全过。
- 偏离说明：待领取取 GetTrxVoteStatus.pendingTrx（proto 注「同 trx_allowance/rewardNum 源」口径一致）；KPI 条错误态=单卡读败文案（下一流帧自动回填），不假零。
- 坑处置：无坑。

---

## 验收注（`[角色:决策]` @ 2026-10-28，接管验收：原验收席 claude 卡住，业主令转本席；收口合并 `050b180`，实现件随 `26bf968` 在 master）

**裁：验收通过，票翻完成。** 票面物化来源追认（b46be9d 只落设计档未落票文件，施工依设计档 §一~§五 原样物化=合规）；回填四件（实现摘要/测试证据/偏离/坑处置）详实；偏离在案均认可；测试证据与生产目检（业主实见新前端）互证。

---

## 审计判决（无头单飞）

审计席：headless @ 2026-10-31

**判：审计通过。** 逐锚对账（基 origin/master，实现件随 `26bf968` 系 diff 实核）：

1. **首行四卡+大数字+窄屏回落**：`TrxKpiStrip.tsx`（新件 126 行 ≤300）渲 4 张 `.card.kpi`；`20-cards.css` `.trx-kpi-strip` grid repeat(4,1fr)、@media ≤900px 落 repeat(2,1fr)=2×2 ✓；`.kpi-v` font-size 2em+font-variant-numeric:tabular-nums ✓；每卡 kpi-head 挂 StateDot 呼吸灯伴随 ✓；`.kpi` 背景 surface 58% vs 常规 `.card` 46%——加重一档方向实证 ✓。
2. **取值径**：`fetchAll` 并发取 getTrxYieldCurve→annualized7d/30d、getTrxVoteStatus→pendingTrx、getHedgeStructure→structure.legShort.distPct（缺省 null→显 "—" 不假零）；useEffect 挂载取数 + onData topic==="trx" 帧重取——径同现役卡、事件驱动无轮询/定时器 ✓。
3. **灯语零新前端判据**：alert 映 platformAlertState（normal→s0+breathe=breatheCalm 绿缓息 / warn→s1+breathe=breatheQuick 黄急促 / breach→s2 且 breathe=false=红静止——与 10-shell.css「红死无呼吸」业主裁一致）；年化/待领取 value 正负号挂 u-good、dot 态随号——唯符号判据无新阈值 ✓。
4. **雷达瘦身+双列**：`TrxYieldRadar` 6 格→4 格（grid 3 列→2 列，组合 7d/30d 两格删除留注「上移不双显」）；`Trx.tsx` 雷达‖对冲包 `.duo`（既有先例，≥1000px 双列窄屏单列）✓。
5. **明细层**：HedgeFlow 出 `.duo` 独占全宽行；`.hedge-flow-table` font-size 13px，HedgeFlow.tsx 表实挂该类 ✓。
6. **机检+四态**：`tsc/vite build` 绿——CI 链 npm run build 先于 go build（embed 依赖），实现件在 master 全谱过编译（HEAD 78be20c6 ci success 兜证）；四态覆盖在码（err→kpi-err 卡 / 首拍→null / 数据→渲染 / ErrorBoundary 隔卡），目检目见以生产实拍互证在案，无头态不复跑如实标。
