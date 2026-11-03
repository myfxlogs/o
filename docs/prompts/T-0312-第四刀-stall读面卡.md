# 施工提示词：T-0312 第四刀——①存活卡 META-* 无头席读面

**角色**：`[角色:施工]`　**票**：T-0312 ③件（验收锚：驾驶舱存活卡见
META-REGIME 行+staleness 标色）
**工作树**：`/opt/o-wt/t0312`　**分支**：`t/0312-regime-push` 续提
**单点**：`web/src/react/pages/Cockpit.tsx`「① 存活」卡——纯前端件，
数据面零改（`HeadlessRow{ticket,seat,state,last_activity}` 全备，
state 服务端五态已判）。

## 范围

①存活卡内加「无头席」小区块：hlRows 滤 `ticket` 以 `META-` 前缀行
（现两席：META-REGIME 分析席/META-VERDICT 裁定席——新席自动随行），
每行显：`seat` 名+state 标（HL_STATE 映射复用）+`last_activity` 龄
（`fmtLagMin(now-lastActivity)` 现成函数）+标色：

- state=在飞→绿；停滞/死亡/逾期→红；完工→dim（seat 长期态不报警）
- 龄>该席节拍（META-REGIME 30min/META-VERDICT 4h——拍表写死两行注释
  注「节拍照 tick.sh 源实」）→标色升红
- 现有 `dev 会话 {hlAlive} 在飞` 注行**退役**（被小区块取代——META-* 
  行才是真无头面；DM/Dev 票行不入此块）

## 验收锚

- 驾驶舱①存活卡见 META-REGIME+META-VERDICT 两行：席名/态/龄/色齐全；
- META-VERDICT 龄≈4h 拍频一致（昨首拍后行在）；
- `npm run build` 过+截图实迹回填票面。

## 停线

- 只动 Cockpit.tsx（+如需小 CSS 复用 kpi-note 类零新造）；
- state/龄判读零服务端改——前端纯映射。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 节拍常量写死前端会漂 | 口径险 | 两行注释钉「源=tick.sh」+龄红仅作显警不阻 |
| 2 | DM/Dev 票行 200+ 混入 | 显面噪 | META- 前缀滤——票行归管道瓦片不属存活卡 |
