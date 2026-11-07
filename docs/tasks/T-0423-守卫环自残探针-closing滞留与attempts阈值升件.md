# T-0423 守卫环自残探针——closing 滞留/attempts 阈值升件（T-0419 范围 2b 遗留）

- 状态：已批（决策席立票直批 2026-11-07——T-0419 裁决同拍遗留件补立）
- 领取：待派
- 优先级：P2——钱径监护面缺口（非新发 bug，循环已靠 gave_up 兜底但中途零告警）
- 收口移交归：决策席

## 背景（生产实证 2026-11-07）

fin_position #34 CORE-USDT-SWAP：delta_drift 幻觉触发平仓循环，
`fin_guard_close state='closing' attempts=5` 连绞烧手续费，
**中途零告警**——既有 `close_gave_up` 仅在 attempts+1≥max（当前
`guard.retry_max_attempts=5`）才火，循环前 4 轮纯静默。
T-0419 票面范围 2b 钦定此探针，收口件未实现且偏离说明未声明——
本票补立，原票根修件不受影响已并。

## 钦定语义

```
守卫自残枚举：state='closing' 且 (attempts≥N 或 now-last_try_at>Xh)
             →探针行+Alert 升件——平仓循环本身是事故，不等耗尽才说
取值钦定：   N、X 走 app_config dm 类阈键（guard.loop_alert_attempts /
             guard.loop_alert_hours 族，种子 N=3、X=2h——可自主调域）
```

## 范围

1. closing 滞留/超次探针：fin_guard_close `state='closing'` 且
   `attempts≥guard.loop_alert_attempts` 或
   `now()-COALESCE(last_try_at,建仓刻)>make_interval(hours=>guard.loop_alert_hours)`
   → Alert（kind=guard_close_loop 族）+ ops_event 行；同仓告警去重
   （已报仓不连拍重复刷——节流或幂等键自定）；
2. 阈键种子迁移：app_config 两键默认值写入（迁文件序号接续）；
3. 红桩：fixture closing attempts=4 或滞留>阈→升件行/Alert 断言
   （修前零行红、修后绿）；反向锚：attempts<阈且未滞留=零告警。

## 验收

- 红桩：超次/滞留两形各断言升件迹，未达阈零告警；
- 阈键经 app_config 可调（直写 SQL 禁——生产调参走 tools/dmconfig）；
- 绿戳对口+全链。

## 结果回填

- 状态：待派
