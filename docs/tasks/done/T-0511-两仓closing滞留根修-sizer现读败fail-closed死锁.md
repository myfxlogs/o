# T-0511 两仓 closing 滞留根修——sizer 现读败 fail-closed 死锁（外派急诊，业主令）

- 状态：完成（决策席裁决通过 2026-11-09——判词在案·取证=收口并master·机械轨票翻）
- 领取：外派 O-Dev（外派事件落账后由外席续跑）
- 优先级：P1（钱径滞留——两仓平仓卡死 16h+，同策略开仓权已自挂起）
- 收口移交归：外审席（业主令钦定外派审计；归宗裁=外审径）
- 立项：业主令急诊——升件面 仓#36/仓#38 closing 滞留 16h（ops_event #5398/#5399/#5568/#5569 族）

## 病灶（SG 生产取证三通道在案）

`fin_guard_close` 实拍（SG o-postgres）：

```
 position_id |  state  | trigger_kind | attempts | last_error                       | last_attempt_at
          36 | closing | delta_drift  |        2 | plan:sizer:平仓量现读败/对齐零   | 2026-11-08 09:35:54Z
          38 | closing | delta_drift  |        2 | plan:sizer:平仓量现读败/对齐零   | 2026-11-08 09:35:54Z
```

腿结构（fin_order_leg 实拍）：

```
仓#36 exec=1119 策略#287287 —— OKX 现货carry：
  okx MUU-USDT      buy  24.7986   filled
  okx MUU-USDT-SWAP sell  6.956    filled
仓#38 exec=1124 策略#287290 —— 跨所对冲：
  mt5 BTCUSD        sell  3.865    filled
  okx BTC-USDT-SWAP buy 24998.82   filled
```

机制定性（T-0046②/T-0428 判词回查）：`sizedCloseLegs` 按腿 venue 现读
定量 min(腿记录,venue 实读)⌊lotSz⌋——任一腿读败或对齐零=fail-closed
不构 intent，state 滞留 closing 按节拍重试。

**已否嫌疑**：Earn 收走余额（T-0428 判词类）——SG 日志实证
`[earn] boot 停泊快照缺席：earn venue 未挂接`，本部署 Earn 未挂接。

**排名嫌疑（施工须逐腿实证定位后根修）**：
1. **腿粒度死锁**——任一腿对齐零冻结*所有*腿：venue 侧实际已平
   （外部平/爆仓/下市）的腿读零=正确，但其余腿仍待平——「一腿零
   全腿冻」语义缺陷候选；
2. **MUU-USDT-SWAP 下市/过期**——funding -0.001695 翻面
   +lifecycle 连环开闸拒在案；下市乐器 ContractPos 读败；
3. **mt5 腿 sizer 读败**——AttachVenueSizer("mt5") 已挂（main.go:267），
   OpenedOrders 手数现读对 BTCUSD 3.865 手返回异常或乐器方言不匹配。

## 范围

1. `internal/guard/sizing.go`（sizedCloseLegs/alignCloseSz 腿粒度语义）；
2. `internal/guard/guard.go`（noteAttemptFail/closing 重试面）；
3. `cmd/core/main.go` sizer 装配面（Mt5Sizer/okxSizer——仅若根因落装配）；
4. venue 实拍脚本/桩测按病所需。

## 验收锚

① SG 生产实拍定到腿级根因（哪条腿、哪种败——读败 or 对齐零 or 下市）；
② 根修后两仓 close 推进（closed 或腿级独立推进——venue 已平腿记平，
   其余腿正常构 intent）；
③ 桩测钉死：腿级独立语义「一腿 venue 已平=该腿记平放行，不冻结余腿」
   红/绿双点；
④ `check_all` 绿戳对口；⑤ 不碰开闸/风控判词语义。

## 红线

- 只修 closing 卡死本体——不碰开闸判据/资金扩投/凭据域/真单语义；
- fail-closed 底线不破——读败仍不许构量（修的是「零量腿不再连坐」
  非放宽读败面）；
- venue 实拍禁落明文凭据入仓/票/日志；
- 已挂起的两策略开仓权不在本票解冻域（解冻结另走裁定席判词/业主令）。

## 结果回填

**腿级根因实证（三通道——SG 生产库+venue 实况 RPC+日志）**：
两仓**全为 `fin_exec_drill` 演练件**（verdict=pass，非真仓）——
drill ⑦步跑产线 `Warden.ScanOnce`：delta_drift 触发→`markClosing`
落 'closing' 行→sizer 对真 venue 读合成腿→MUU 系乐器已改名 XMUU
（ccy 缺席=读败）/mt5 OpenedOrders=0/两 SWAP ContractPos=0→
`plan:sizer:` 败；drill 清扫只写 `position.closed_at` **不关
guard_close 行**→scanOnce 按 `closed_at IS NULL` 滤仓永不再试，
而 loopProbe 无此滤→对死行每 2h 升假 P1+熔断挂起演练策略。

**实现摘要**：
- `internal/guard/loop.go` loopProbe 首行加 reconcile 先扫——
  position 已 closed 的 'closing' 行落 'closed'（last_error 记
  reconcile 源），阈键缺件也照扫；告警查询补 `p.closed_at IS NULL`
  纵深；
- `internal/trading/simdrill/{drill,matrix}.go` 清扫径同关
  guard_close（源头防生——夭折防护原注释本义补齐）；
- 生产两行已手动 reconcile（业主令，ops_event 处置行在案）。

**测试证据**：`TestLoopProbeReconcileClosedPos` 新桩（僵尸行→closed+
零升件零挂起零推送）红→绿；guard+simdrill 两包 52 tests 全绿。

**偏离说明**：
- 票面「腿级独立推进」方向**证伪后改道**——venue 实拍四腿全平，
  无需推进任何腿，病灶=演练件账面僵尸而非真平仓卡死；
- suspend 行（两演练策略）留置不解——演练策略本无开仓权语义，
  挂起与 degraded 同义无害，解冻结走常径。

**坑**：`reconcile:drill-cleanup-misses-guard-close`——
带外关账径（drill/人工）漏关 guard_close=僵尸假升件；
探针查询缺 `closed_at` 滤=死行升件放大器。条目同 commit 入库。
