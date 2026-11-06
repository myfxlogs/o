# T-0406 驾驶舱 UI 五修——弹窗遮挡与 UTC 时标

- 状态：待验（2026-11-06 业主实测令——「自选报价弹窗尾部
  被待开仓挡住无法点保存」「待开仓瓦片右上角加 UTC 时间」）
- 领取：o-worker-t0406 @ 2026-11-06（worktree `/opt/o-wt/t0406`，branch `t/0406-cockpit-ui-fixes`）
- 优先级：P2——前端可用性（业主实测病灶）
- 收口移交归：决策席

## 病灶（业主实证）

① 自选报价编辑弹窗（T-0391 WatchlistModal 语义件）尾部
   被「待开仓」瓦片遮挡——保存按钮不可点（z 序/滚动
   容器问题：弹窗应全屏置顶或列内滚动，不得被平铺件
   覆盖）；
② 「待开仓」瓦片右上角加 UTC 时间显示（当前时刻 UTC，
   与全局时标惯例一致——查现有时标件复用样式）；
③ 活水段「提案/策略/执行」三事件驱动段呼吸灯语义修——
   现 `state=3 恒中性灰+不呼吸` 视觉=死态（业主实测
   「这3个只是饿，并不是死」）；**饿≠死**：事件驱动段
   零事件=活着等≠死掉——改呼吸中性态（呼吸的灰/青脉动
   =活着待命），与真死态（state=2 红/ok=false 缺）可分辨；
   判态仍在服务端派生（flow_pulse.go neutral 段 breathe
   派生语义同修），前端按语义渲染不新造判据；
④ 自选报价瓦片精简——业主令「只用显示买家和卖家，
   前端那个价格我不知道取自那里，不用显示」：瓦片
   只显 bid/ask（买价/卖价），`last` 成交价不渲染；
   鲜度行（272h 前类）保留——诚实标注不动。
⑤ 「行情·价格提醒」卡内 bid/ask 色彩区分——业主令
   「需要颜色区分，按 cex 的定调」：CEX 惯例买=bid
   绿、卖=ask 红——两处渲面同修（QuoteBoard.tsx
   自选瓦片 `bid/ask` 行 + 提醒表 买一/卖一 列），
   复用语义色板 `.u-good`/`.u-bad`（=var(--good)/
   var(--delta-down) 族——禁新造色值禁内联 hex）。

## 范围

1. `web/src/react/components/QuoteBoard.tsx`（或弹窗所在
   件）修遮挡——模态 z-index/定位或列容器滚动约束，
   保存键恒可达；
2. 待开仓瓦片件加 UTC 时标（右上角——复用既有
   timestamp 样式约定）；
3. 活水段语义修：`cmd/core/flow_pulse.go` neutral 段
   breathe 派生+`flowpulse.tsx`/`pipeStatus`/样式面
   中性呼吸渲染——饿=呼吸中性态，死=原死态；
4. 报价瓦片去 `last` 渲染——只显 bid/ask 两行（QuoteBoard
   自选条内瓦片件）；
5. bid/ask 双色——自选瓦片+提醒表两处，bid=`.u-good`
   ask=`.u-bad`（CEX 定调买绿卖红）；
6. domtest 桩断言：弹窗保存键可点/UTC 时标渲染/
   neutral 段呼吸非死态渲染/瓦片无 last 价元素/
   bid/ask 元素带双色类。

## 验收锚

① 弹窗任何列高下保存键可点（domtest/实测截图证）；
② 待开仓右上 UTC 时标渲染正确；
③ 提案/策略/执行段零事件时=呼吸中性可视（非死灰），
   真缺/败段仍死态可辨；
④ 报价瓦片无 last 价元素，仅 bid/ask 行；
⑤ bid/ask 双色成——bid 绿 ask 红（CEX 定调）两处
   一致；
⑥ 前端 typecheck+domtest+check_all 绿戳对口。

## 红线

- O 组件体系内修——禁引新 UI 库；
- 不动瓦片数据语义——纯呈现层。

## 结果回填

**实现摘要**（五件全落 + 先红桩实证）：

① 弹窗遮挡根修——`.card` 的 `backdrop-filter` 使卡成 `position:fixed`
   包含块+叠层上下文阱（`.modal-overlay` 在卡内渲则 inset:0 只覆卡界、
   z-index:200 被困卡叠层内，后续平铺卡盖尾=保存键不可点病灶）。修径=
   WatchEditHost 改作 Card **外兄弟节点**渲（QuoteBoardView 返回 `<>`），
   overlay fixed 归视口 z:200 置顶；`.modal` 补 `max-height:90vh`+列表区
   `min-height:0` 列内滚（任意列高下保存键恒可达）。新增坑条目
   `docs/工程经验库/条目/card-backdrop-filter-traps-fixed-modal.md`。
② 待开仓卡右上 UTC 时标——`UtcClock` 原语升 `components/ui.tsx` 共享件
   （原 Overview「最近动作」卡角样式：`ink-3 mono` HH:MM:SS UTC 1s 拍），
   Cockpit ④ 卡 extra 挂 UtcClock+StateDot 并置；Overview 本地件去重改共享。
③ 活水段饿≠死——`FlowStage` proto +`bool breathe=10`（同 pipeline 件
   `step.breathe` 先例）；`flow_pulse.go` 服务端派生
   `breathe=ok&&(neutral||state==0)`，查询败段补 `state=2` 死红（原默认
   0=绿呼=死段装活同根修）；前端 `breathe={st.breathe}` 零判据；
   `.dot.breathe.s3`+`breatheIdle` 灰珠缓息脉动（2.4s 同绿活包络异色），
   死态（s2 红/ok=false 缺）静止可辨。
④ 自选瓦片去 `last`——只渲 bid/ask 两价+鲜度行（诚实标注不动）。
⑤ bid/ask 双色——瓦片 `num u-good`/`num u-bad` 分跨；提醒表 买一/卖一
   td 加 `u-good`/`u-bad`（CEX 定调买绿卖红，两处一致零新色值）。

**测试证据**：

- 先红桩（病灶存在断言）：修复件 stash 回退跑新断言——
  flowpulse FAIL×2（中性段无呼吸=死态视觉、ok=false 段仍呼=死段装活）、
  quoteboard FAIL×5（瓦片含 last/表无双色/浮层困卡内/保存键缺席/modal 无封顶）。
- 修后绿：`web npm test` 85/85 PASS（flowpulse 30/30、quoteboard 31/31、
  metaseats/quadcols 全谱）；`tsc --noEmit` 净；
  `go test ./cmd/core -run TestFlowPulse|TestDemandStage|TestReasonShort` 4/4 PASS
  （TestFlowPulseStages 增呼吸派生锚：空库中性段 ok=state3+呼、周期段呼吸与绿灯同、
  缺段死红不呼）；`scripts/check_all.sh` 全链绿戳对口。

**偏离说明**：

- Overview.tsx 本地 UtcClock 去重改共享件（复用径的自然归位——票面
  「查现有时标件复用样式」的落法，零视觉差）。
- 服务端查询败段 `state` 由默认 0 改 2——票面「真死态 state=2 红可辨」
  的语义对齐（修前查询败段渲绿呼=死段装活），domtest stub 喂态不受影响。
- 未推 master——分支 `t/0406-cockpit-ui-fixes` 待决策席验收合流。
