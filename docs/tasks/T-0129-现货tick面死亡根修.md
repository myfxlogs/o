# T-0129 现货 tick 面死亡根修——md_ticks 现货行齐冻 15:58

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 优先 | **高**——行情面断流 1h+ 未被活性探针捕获；直接造成 T-0128 误杀 |
| 出处 | T-0126 验收期取证（仓 23 误杀溯源上游病灶） |

## 病灶

SG 实拍 2026-10-25：md_ticks 全部现货行（BTC-USDT/ETH-USDT 等 4 行）
最后更新冻结于 **15:58:02**，SWAP 行正常 30s 级刷新。okxws 订阅日志
`subscribed syms=507` 每重连照发——订阅在发、现货 tick 不进。

疑向：T-0122 渠道分工改造后现货品种只挂 `index-tickers`（推送低频/载荷
落 md_mark_index 非 md_ticks）或批帧中现货 args 丢失。施工须实核
`internal/data/okxws/` 订阅构建+消息路由双面。

## 修法方向（施工裁）

- 实核现货 inst 当前订阅渠道集合；现货腿定价需要 tickers 级实时价
  （index-tickers 语义是指数价非成交价，不可代用）。
- 现货 tick 恢复后，活性探针应能照见此类断流（md_ticks 全现货行超龄
  = 面级告警，非单键静默）。

## 验收锚

- md_ticks 现货行恢复秒级刷新
- 面级断流有告警通路（不靠人盯发现）
- 与 T-0128 联合验证：断流窗口内 guard 不误杀

## 结果回填

- 根因定性：`okx.symbols="*"` 扩张只返 SWAP+季度期货——现货腿标的全掉
  出订阅集（15:58 翻键激活时点即现货 tick 冻结时点，订阅日志照发=
  订阅在发标的已换）。修法=ExpandSymbols 同返 SWAP 基币现货对（去重
  dedup 面）——现货腿定价回 tickers 面（index-tickers 指数价不可代）。
- 季度期货 args 不挂 index-tickers（-USDT-YYMMDD 尾=tickers 唯订）。
- 面级断流告警：collector staleWatch 60s 拍——订阅标的 md_ticks 全行
  超龄 3min=ALERT [okxws] 一次抑刷屏恢复清标（面死显形非单键静默）。

## 审计判决

审计席：headless @ 2026-10-30

- 锚①现货秒级刷新——通过。修法链两段落码：a6b15019 宇宙扩件增 SWAP
  基币现货对 dedup 入订阅集；83c6f3ad 活跃集 tickers 动态增删订。
  master 现态 activeSet 含 dcd/basis 策略现货腿归一入集，tickers 帧→
  INSERT md_ticks 写径实存。生产实拍（SG 只读 2026-10-30 15:48Z）：
  现货行 BTC-USDT ts 龄 3s 秒级鲜；ETH-USDT 陈行=非活跃集无 tick 订
  （T-0132 设计态非病灶）。
- 锚②面级断流告警通路——通过。stalewatch.go 实存（60s 拍、宇宙键域
  md_ticks 全行超龄 3min→面死告一次、恢复清标），collector.go 挂拍、
  al.Alert→notify.Alerter→飞书推送接线（AttachAlerter）+日志兜底，
  不靠人盯。注记边缘态：活跃集全空时 fresh=0 会发面死告警（理论误报
  缘，建议另票裁，非本锚否项）。
- 锚③断流窗口内 guard 不误杀——通过。新鲜度闸 diff 实证（tick/funding
  龄超阈=缺 fail-closed facts_missing 不杀），TestStaleTickNoKill/
  TestFreshTickRealDriftKills 桩测在库；生产 fin_guard_close 实拍：
  delta_drift 触发全止 2026-10-25 17:00:02（原案时点），部署后零新
  触发——含 ETH-USDT 陈 17h 窗口。
