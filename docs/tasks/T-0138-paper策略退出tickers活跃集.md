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

## 验收（决策席 2026-10-26 @1125ff2 部署）

- 谓词落地：activeSet 仅 live/degraded 策略；md_ticks 511 行中仅 160 鲜（armed+live+现货对合法订阅者）
- `okxws/8443` 实测 1678 kbps（前 ~3.3Mbps——减半）；零错帧零重连
- 残余=armed 提案+live+现货对 tickers——开火判价合法需求，再降归排水账非订阅面
- 状态：✅ 验收合入部署

### 状态：审计通过
审计席：headless @ 2026-10-30

## 审计判决注记（逐锚对账——收口 diff=0c19a7d7 谓词一行+票面回填；合并 1125ff21；翻面 30a4dbcd）

- 锚①（订阅数≈live+armed<30）：谓词实跑——当前 activeSet SQL 出集=75=fin_armed 全集
  （live/degraded=0、fin_position open=0），集≈armed 在途零额外，结构成立；「<30」
  量级今日越界——75 armed 全 >4h 陈，属 T-0136 超龄退役面欠账致在途淤积，非本票
  谓词偏离。票面时代验收记录（md_ticks 鲜行收窄）与锚一致。
- 锚②（降半以上）：diff 令 ~205 paper/probe inst 退 tickers——机制自洽；验收实测
  1678kbps vs 前 ~3.3Mbps（-49%）在案；本机无采集容器，以票据实测+结构机制双证计。
- 锚③（paper 撮合不因缺 tickers 报错）：paperAccounting 读 md_funding（funding-rate
  宇宙订阅未动）与 tickers 无涉；价需走 LatestPx=md_ticks∪md_mark_index 60s 鲜闸
  （sizing.go:60-74），桥 FullSub 全 -SWAP 订 mark-price 续供；dcd/basis paper 现货
  腿由 T-0171/T-0216 activeSet 现货腿并集回补（subscribe.go:78-83 在案），T-0138
  时代 dcd 记账读产品目录 spot 字段非 ticks——无断供窗。
- 锚④（棘轮/机检全绿）：棘轮面过（pre-commit 十二道、文件 162<300 行）。机检注记：
  两推送 ci build-test 均 failure——败项 TestTrxRollupIdempotent（o/cmd/core，与本
  diff 零代码径）；红窗自 10-26T00:02Z 起先于本票首开，9ccb6b1f 票面已钉「T-0152
  TRX桩日界flake」、a8806ffe 根修——继承 flake 非本票致红。
- 注：projector refreshSets/localwrite pred/fin_active_set 触发器表仍含 non-retired
  策略项——投影/写滤/发布三层异语义（后续票分治），不改订阅面，不计本票违例。
