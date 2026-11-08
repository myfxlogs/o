# T-0457 量表 heartbeat_dead 口径修——死道剔除终态票道

- 状态：已批（CTO 班自立票——升件#3444 处置件）
- 归口：meta/量表面
- 领取：常径待派（DM 常径机派）
- 优先级：P3

## 病灶（升件#3444 在案）

`gauges.tsv heartbeat_dead`：`SELECT count(*) FROM (SELECT ticket,
max(last_activity) m FROM ops_headless GROUP BY 1) t WHERE
m<now()-interval '7 days'`——死道计数 242→247 单调积。病因：
ops_headless 道含**逐票心跳道**（META-Dev-T-NNNN 类——票终态后道
永缄属正常），量表把历史闭票道全计入「死道」=口径病灶非活性
病灶——连鸣踢席而无可处置对象。

## 范围

1. `scripts/meta/gauges.tsv` heartbeat_dead 行 SQL——死道集剔除
   终态票道（票道识别=票号存在终态行（裁决/完成/驳回）或 ops_event
   无该票近态；保活面=常驻服务道 META-*/o-*/席名 类纯守道）。
2. 同口径复查 gauges.tsv 全表有无同型「历史积道误报」行——一并修。
3. 桩/对拍：修后死道集只含应活未活道（可造具 ops_headless 行对拍）。

## 验收锚

- gauges_tick 拍后 heartbeat_dead 计数只计常驻道缄默（闭票道不入数）；
- escalation_unclaimed 连鸣面不再被本型噪声占额（阈=1 恢复有意义）。

## 备注

升件#3444 处置落行引证本票（口径病灶自立票——非活性病灶不挂起糊账）。
