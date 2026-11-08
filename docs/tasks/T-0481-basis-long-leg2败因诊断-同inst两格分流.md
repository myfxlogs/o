# T-0481 quarter_basis_long leg2 开仓败因诊断——同 inst 两格一分一败（T-0480 排雷续件）

- 状态：已批（CTO 立票直批 2026-11-08——T-0480 排雷续件，票钦「排雷成立与否归 DM 读面裁续」）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P1——T-0145 部署令前置面缺角（quarter_basis_long 凭据=fail 则该格部署令不发）
- 收口移交归：决策席
- 立项：T-0480 矩阵产跑凭据格 quarter_basis_long@okx/basis_quarter#usd=50
  verdict=fail——armed→open 格闸 fail-closed 持拦在案，败因未定性前
  该格凭据不可翻绿（禁重跑冲绿掩盖病灶）

## 实据锚（生产取证面已核）

- fin_exec_drill fail 行：`开仓执行败: leg2 败/歧义→LEGGED 回滚 leg1: state=absent`
- 升件：ops_event#4228（seat=演习 esc_class=执行径演习）
- 执行日志：SG `/tmp/simdrill-0480.log`（在案未清）——exec#1123
  `LEGGED→leg1 回滚成`，与 exec#1121/1122（perp_quarter_spread
  同 inst BTC-USD-270326 双腿成交）同拍对拍
- **分流产异**：perp_quarter_spread@okx 同 venue 同 inst（BTC-USD-270326）
  双腿全绿；basis_long 同 inst leg2 独败——方向/shape 异致歧，
  非 venue 能力缺

## 诊断域

1. leg2 败/歧义判据落点：execution 双腿序（clOrdId 定态/歧态势/回滚径）
   在 basis_long shape 下的 leg2 下单参数面——与 perp_quarter_spread
   双腿参数逐字段对拍，定位败因字段（方向/ordType/instId 方言/面值）；
2. simvenue 侧 leg2 拒因：simdrill simvenue 对 BTC-USD-270326
   卖向/合约面值的处理面是否真拒——若 sim 缺陷归 simvenue 修面
   （纸面域），若真径缺陷归 execution 修面（钱径域）；
3. `state=absent` 语义核实：leg2 订单在 simvenue 侧是否真未落
   （LEGGED 回滚正确性——回滚后零残仓是硬断言）。

## 边界

- 诊断票——产病灶定性+修法提案，实现修法归 DM 裁后续派；
- 纸面域全仿真——禁真 venue 写/禁真凭据读；
- 生产只读取证径：SG docker exec o-postgres psql + /tmp/simdrill-0480.log。

## 验收

- 败因定性判词（sim 缺陷/真径缺陷/参数方言 三取一，附代码行锚）；
- 与 perp_quarter_spread 成功格参数面对拍表；
- 修法提案（票件化——若属真径缺陷须标钱径域让 DM 按钱径道裁）；
- 禁重跑矩阵冲绿——凭据翻绿须待修法落批后重产。
