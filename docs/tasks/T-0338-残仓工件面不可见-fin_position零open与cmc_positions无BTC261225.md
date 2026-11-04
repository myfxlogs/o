# T-0338 残仓工件面不可见——fin_position 零 open 与 cmc_positions 无 BTC-261225-95000-C

- 状态：**完成**（决策席无头拍裁决通过 @2026-11-04——合 master@c1cad19b；三锚本席生产亲证复核一致）
- 立项：`[角色:决策]` devin 无头拍 @ 2026-11-04（T-0333 部署窗验项⑤红项根——幽灵/缺席类实案立票）
- 级别：P2 定性件——先定性「残仓已灭」vs「工件面断供」再定修否

## 决策席验收注记（无头拍 @2026-11-04）

- **锚① venue 实拍**：本席经 readcred 隧道径取只读钥直签 OKX
  `/finance/sfp/dcd/order-status?ordId=11448262` → `state=LIVE`
  复核一致；`account/positions` OPTION/SWAP 皆 `data=[]` 一致
  （dev 出向 403——签名本机、执行 SG 径，凭据未出内存域）。
- **锚② 定性**：残仓实存+工件面正常——生产 GetFlowPulse
  workItems 首行 `position|DCD BTC-USDT-261225-95000-C|残仓|
  2026-10-21T12:21:46Z` 逐字亲证（buf curl 本地 schema 经 SSH
  隧道——SG proto 滞版弃新字段坑绕行）；`fin_position`
  `closed_at IS NULL`=0 本席 psql 亲证；T-0327 挂锚复证销。
- **锚③ 口径落字**：两源并集（fin_position open ∪ venueRemnants
  −归属集）对码 `cmd/core/remnant.go`/`flow_pulse.go` 一致——
  「DB 零行⇒工件面无行」推断对残仓面不成立在案。
- 范围：diff 仅票面自身；码面零改门禁不受触。

## 病灶（生产亲证 2026-11-04）

- `fin_position` `closed_at IS NULL` 行=**0**（22 行全 closed，
  o_fin_ro 径亲证）——生产无任何 open 持仓行
- `fin_cmc_positions` `BTC-USDT-261225-95000-C`/`%261225%`/`%-C`
  全 0 行（T-0333 实录；cmc_positions 表本席白名单外未亲证）
- T-0327 裁词挂「GetFlowPulse 工件面残仓行可见」锚——工件面
  上残仓不可见属实

## 定性二择（查后落票）

- **甲·残仓已灭**：2025-12-26 到期 BTC call 残仓在 T-0327→今
  时间差内被 sweeper/到期机制自然消费——工件面无物可显=正确，
  残仓类入集码面无需动，结案性归档
- **乙·工件面断供**：残仓实体仍在 OKX 侧（venue 实拍）但
  fin_position/cmc_positions/GetFlowPulse 链不显——数据面断供，
  trace 断点根修（T-0327 类级入集逻辑或采集臂）

## 任务

1. venue 实拍先行：OKX 侧 `BTC-USDT-261225-95000-C` 持仓今实存否
   （okx-account/positions 径——mt5/okx 能力账在 skills/）
2. 甲定性→票面如实落「残仓已灭」+销 T-0327 挂锚即结案；
   乙定性→断点 trace+根修落同票或升件裁修向
3. 工件面口径对拍：GetFlowPulse 残仓集定义 vs fin_position open
   口径——零 open 时工件面应显何物落字

## 定性实录（`[角色:施工]` @2026-11-04——两择皆不成立，第三种实情）

**实情=残仓实存 + 工件面正常显形——「不可见」系查错面的误判。**

### venue 实拍（OKX 真户，readcred 只读钥径 `dev:5545→SG:8095` 直签 API）

- `GET /api/v5/finance/sfp/dcd/order-history?state=live` →
  `ordId=11448262 productId=BTC-USDT-261225-95000-C state=LIVE
  notionalSz=0.0207 BTC strike=95000 cTime=2026-10-21T12:21:46Z
  settleTime=2026-12-25`——**残仓本体实存**（DCD 产品单非
  positions 仓位——`account/positions` OPTION/FUTURES/SWAP 三型
  实拍皆空集属正常，本就不走该面）
- `GET /api/v5/finance/sfp/dcd/order-status?ordId=11448262` →
  `state=LIVE` 复核一致
- `GET /api/v5/account/positions[?instType=OPTION|FUTURES|SWAP]` →
  `data=[]` 全空——SWAP 仓位零在持

### 工件面实拍（SG 生产 `o-core` 镜像 `73e2433c670a`——已含 T-0327 码 `5a63a6a0`）

- `POST /o.dashboard.v1.CoreService/GetFlowPulse` workItems 实含：
  `{kind:"position", ref:"DCD BTC-USDT-261225-95000-C", stage:"残仓",
  sinceTs:"2026-10-21T12:21:46Z"}`——**T-0327 挂锚复证满足，销锚**
- `POST /o.dashboard.v1.CoreService/Overview` doing 实含：
  `DCD BTC-USDT-261225-95000-C — 持仓中，估值 $1623.70`；
  money 区 `结构性持仓 BTC-USDT-261225-95000-C` 行在列
- 生产 `o-core` 日志 24h 无 `dcd 持仓活读败` WARN——dcd 臂读件健康

### 误判根（T-0333 验项⑤红项复评）

病灶两条（`fin_position` 零 open / `fin_cmc_positions` 无该品行）
皆真但**不构成断供证据**：残仓臂设计=venue 活读
（`cmd/core/remnant.go` dcd/okx/mt5 三源）非 DB 表消费——
`fin_position` 只供「持仓中」工件行与归属集，`fin_cmc_positions`
是 CMC venue 采集表与 OKX DCD 无涉。查 DB 表判工件面=拿错镜。
（`fin_cmc_positions` 全表仅 `X-AFQUL` 一品 8390 行、停采于
2026-10-31——cmc 采集面自滞另案，不属本票面。）

### 口径落字（验收锚③）

GetFlowPulse 持仓类工件行=两源并集：

1. `fin_position closed_at IS NULL` 簿记仓→`kind=position/stage=持仓中`；
2. `venueRemnants` venue 实持（dcd live ∪ okx positions ∪ mt5 orders）
   − 归属集（未平仓 inst ∪ 其腿 inst ∪ probe/live/degraded 策略 inst）
   →`kind=position/stage=残仓`；各源读件缺→`stage=读件缺` 显缺行不静默。

故 `fin_position` 零 open 时工件面持仓区**仍可非空**（残仓臂 venue
直读承载）；双源皆空且无读件缺=工件面持仓区为空属正确，
**「DB 表零行 ⇒ 工件面该无行」的推断对残仓面不成立**。

## 验收锚

- [x] venue 实拍实录（残仓**实存**——DCD live 单 ordId=11448262，
  OKX API 直签可复现，径见上）
- [x] 定性落票：甲乙两择皆不成立——残仓实存+工件面正常，
  码面零改动结案归档；T-0327 挂锚复证销
- [x] GetFlowPulse 残仓工件面口径落字（见「口径落字」节）

## 范围外

- 残仓本身的处置交易（清仓/展期=钱径面——归业主裁不动）
- fin_cmc_positions 表级修法——乙定性后另票
