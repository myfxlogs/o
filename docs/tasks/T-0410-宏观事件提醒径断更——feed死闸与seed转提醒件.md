# T-0410 宏观事件提醒径断更——feed 死闸与 seed 转提醒件

- 状态：已批（2026-11-06 业主令——「当然要立，这都已经瞎了」）
- 领取：待施工
- 优先级：P1——感知面断更两天零告警，非农夜裸奔实证
- 收口移交归：决策席

## 病灶（业主对拍实证 2026-11-06 非农夜）

- **直播 feed 断更 ~2 天零告警**：`md_calendar_push` 零行自
  2026-11-05（末行=11-04 推送覆盖 11-03~11-05 事件）——
  今晚 NFP（11-06 13:30 UTC）整个日历面无一行推送；
  断更无任何死闸升件，靠业主人工对拍发现（与
  dispatch-spawn-silent-failure 同族「缺席无感」）。
- **seed 行有但不转提醒**：`md_macro_event` 确有
  `2026-11-06|nfp|NFP 非农就业月度` 行——但它是例行占位
  （无时刻/无预期值/无提醒动作），无任何件把它转成
  「今晚有事件」的推送。知识在库里，提醒径断了。
- **策略层失感知**：macro/regime 面本喂策略生成（regime
  分类/risk-on-off/事件窗避险）——feed 断=策略层对事件
  窗裸奔（非农前开仓=事件风险敞口零标记）。

## 范围

- **死闸**：`md_calendar_push` 行龄巡检——最新行
  `sent_at` 超阈（建议 24h，feed 本应日更）零新行
  →ops_event 升件「日历 feed 断更」呈批（缺席发现器
  同形，与 T-0403 族调度缝互补）。
- **seed→提醒件**：日拍扫 `md_macro_event` 今/明日
  event_date 行→推提醒（ops_event/飞书径同现有告警
  扇出，禁旁路裸写）——内容=事件 kind+title+date，
  时刻缺则如实标「日级精度」。
- **可选延拓（票面注明不阻塞）**：提醒行附 regime 标签
  （risk-event 窗标记）供策略层消费——策略侧消费改造
  不在本票。

## 验收锚

- 先红桩：桩断言「feed 陈>阈+零升件行」修前红——
  模拟断更场景升件行必落。
- 提醒实证：造 `event_date=明日` 行→下一拍提醒行落
  （渠道=现有告警扇出径）。
- 零静默：feed 拉取失败/空返=升件不静默（事实律）。
- 全链绿戳对口。

## 结果回填

- 状态：待验
- 实现摘要：
  1. **死闸** `internal/portfolio/hunger/feedstale.go`——`ProbeFeedStale`
     挂 hungerTick 同槽（ProbeFamilies/ProbeDeps 同款败情隔离位，接口
     `hungerSensor`+stubSensor 同扩；拍壳 tools/hungertick 同挂）。
     `notify.feishu.calendar.enabled` 开态下 `md_calendar_push` 末行
     `sent_at` 龄>24h（票面建议阈）∨零推送行史 →`feed断更` 行+升件
     （esc_class='观测盲' esc_scope='CALENDAR-FEED'，24h 同窗去重）。
     关态=feed 应停非病灶不探。
  2. **seed→提醒件** `internal/platform/notify/macro_remind.go`——
     `MacroRemind` 日拍扫 `md_macro_event event_date IN(今日,明日)`→
     飞书 SendText（price_alert/venue_req 同款 Sender+chatID+enabled
     装配面，禁旁路裸写）；文案=今日/明日+date+kind+title+「日级精度」
     如实标。已提醒锚=ops_event `宏观提醒` 行（键=event_date——送达才
     记账 T-0093 序，推败/凭据缺不落锚下拍重试）。cmd/core/event_push.go
     挂 24h ticker+起拍首扫（econ_digest 先例）；开关键
     `notify.feishu.macro_remind.enabled` 种 fin_0127 默认开（照
     fin_0101/0119 先例业主钦定要收）。
  3. **零静默** `internal/platform/notify/calendar.go`——拉取败/非 200/
     解包败/200 空返四相各挂 `escalate` 升件（esc_class='读败'
     esc_scope='CALENDAR-FEED'，note 前缀 24h 同窗去重）；pool 缺席=
     静默退不炸（桩测面 NewCalendar(nil,...) 存量形零回归）。
- 测试证据：
  - 红→绿实证：feedstale_test 五形（陈行病灶/鲜行零报/关态不探/零行史
    同报/24h 去重）先红（cfg 快照 Load 序坑修正后）全绿；calendar_escalate
    五形（传输错/非200/空返/畸形解包/nil 池不炸）全绿；macro_remind 五形
    （明日推+幂等/今日推/窗外静/关态不占位复归补推/推败不落锚败愈补推+
    升件自报）全绿。
  - 接线桩测 TestHungerTickFeedStaleEveryBeat：同槽每拍先探+探针假败
    →升件自报行落+Detect 照常（败情隔离位实证）。
  - 开关键种值断言（price_alert 同钉防 fixture 与生产分歧案）：
    `notify.feishu.macro_remind.enabled` 种='true'。
  - check_all 全链绿戳对口（绿戳 check-green-89cdd10d——提交面基线戳，
    收口 commit 后重跑盖终戳）。
- 偏离说明：
  ① 行龄断≠fetch 死——「无高冲击事件窗」同形陈（feed 是事件驱动推送面
  非日更流水面）：升件 note 如实标两可+「对拍归人肉/上游活性面」，探针
  只报不修同族铁律；
  ② 死闸挂 hungerTick 同槽（票面钦「缺席发现器同形」=探针族挂点位），
  6h 饥饿拍距内 24h 阈下检出一拍内——非日拍专槽；
  ③ 提醒锚取 ops_event 行键去重非新表（drafter/onboard 同窗去重先例；
  不复用 md_calendar_push——提醒行落该表会把死闸 max(sent_at) 刷鲜
  自遮病灶，刻意避开）；
  ④ 良构但非日历语义的 XML（如 <html>）落「空返」支非「解包」支
  （Go xml 宽容解成空集）——两相皆升件语义等价；
  ⑤ 可选延拓已带：`标签=risk-event` 挂 ops_event 提醒 note（零本附带；
  策略侧消费改造不在本票）。
