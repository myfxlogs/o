# T-0138 paper/probe 策略退出 tickers 活跃集（带宽卫生件）

## 实证

- 活跃集口径=全量非退役策略（state<>'retired'）——paper 114+probe 91 虚拟/试探策略同吃 tickers 成交级流 ~1.4Mbps
- paper 虚拟撮合读 `LatestPx` 联集——桥供给的 mark_price（秒级鲜）已足精度，tickers 对纸面判仓属过度实时

## 修法

`activeSet` 谓词收紧：fin_strategy 侧只收 `live`/`degraded` 态（钱径实体）；`paper`/`probe` 退出 tickers——其 px 需求由 mark_price 联集行供（判仓龄闸 120s 对 60s 轮询兼容）。

## 验收锚

- 活跃集 tickers 订阅数≈live+armed 在途（<30）
- `okxws/tickers` netmeter 字节速率降半以上
- paper 策略虚拟撮合不因缺 tickers 报错（mark_price 联集行供价实证）
- 棘轮/机检全绿

## 结果回填

- activeSet 谓词收紧：fin_strategy 侧只收 live/degraded 态（钱径实体）——
  paper/probe 退出 tickers（mark_price 联集行供价，精度对判仓龄闸兼容）；
  fin_position/fin_proposal armed 在途面零动。
