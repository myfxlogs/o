# T-0037 perp 张数换算根修——ctVal 币本位×px（验审实勘缺陷）

[角色:决策] 立 2026-10-24 | 优先级：**阻断级（首单前必修）**

## 病灶（实勘）
- `BuildLegs`: `perpSz=floor(half/ctVal)`——ctVal=0.01 **BTC**（ctValCcy=BTC 币本位面值），被当 USD 面值算：$25 腿→2500 张≈$2M，正解≈0.03 张
- `BuildCfdLegs` 同款（perpCtVal 除法无 px）
- 现测试锚用了把 ctVal 当 USD 的 meta——断言没拦住（fixture 失真病灶同 T-0026 猜列名）
- OKX 实勘：BTC-USDT-SWAP ctVal=0.01 minSz=0.01 lotSz=0.01 ctValCcy=**BTC**；spot minSz=0.00001 BTC

## 修法
- 换算=contracts=floor(half/(ctVal×px_perp))，px_perp=md_ticks SWAP 腿最新价就近取（陈>60s 拒——同 LatestPx 径）
- ctValCcy≠USD/USDT 一律按币价换（凡 ctValCcy=base coin 皆此径；若遇 ctValCcy=USD 则 contracts=half/ctVal——实现按字段分流不假设）
- BuildCfdLegs 同修
- 测试锚改真实形 meta：ctVal=0.01 px=80950 → half=25 期望 sz≈0.03 非 2500；minSz 下拒锚保留

## 验收锚
- 真 meta 测试：BTC-USDT-SWAP $50 双腿→sz 落在 [0.02,0.04] 区间（非 2500）
- go test 全绿+既有锚不回退

## 追加（验审同轮实勘——同件同轮修）
openfirst 未挂 `.WithEarn(earnW)`——产线 Runner 有 earn 装配（cmd/core/main.go:152 WithEarn(earnW).WithAlerter），工具裸奔无赎回件。OKX 交易户实勘 USDT≈0（2129 全泊 Earn）——不挂=首单保证金必拒。修法=工具装配同产线：earn.NewWorker+AttachVenue(okxcli)+WithEarn 挂上（赎回额=notional 全闸径内）。
## 追加验收锚
- openfirst 装配含 WithEarn——dry 跑日志见赎回径装配（或真单前 RedeemToTrading 留痕行）

## 结果回填（[角色:施工] zcode-glm 2026-10-24）

### 实现

| 面 | 修 |
|---|---|
| BuildLegs | `perpContracts(usd,ctVal,ctValCcy,px)` 字段分流：ctValCcy=USD/USDT→usd/ctVal；币本位（BTC 等）→usd/(ctVal×px)；px_perp=SWAP 腿自身 md_ticks 最新价（LatestPx 60s 新鲜度门同径）。InstMeta 增 CtValCcy 字段（okxexec.InstrumentMeta 同步扩——字段实读不假设） |
| 次病灶（根修中实勘） | raw 张数普遍 <1（0.0309 张）——旧序 floor(raw)=0 再过 minSz 闸必拒：改 **floorLot 先落 lot 网格再过 minSz 闸**（网格取整=合法最小可下单量形） |
| BuildCfdLegs | perpSz 同 perpContracts 币本位径（×px——px 即 BTC 价双腿共用） |
| openfirst | earn.NewWorker+AttachVenue(cli)+WithEarn(earnW) 装配同产线（cmd/core:152 径）；运行时打印「earn 赎回径装配 OK」——dry 跑日志锚 |
| 测试锚 | 真 meta（ctVal=0.01/BTC/lotSz 0.01/minSz 0.01，SWAP tick 80950）：$50 双腿→sz=0.03∈[0.02,0.04]（2500=旧病灶反锚）；USD 面值分流锚（ctVal=10→2.5 张直除）；minSz 拒锚保留（0.5 notional）；CFD 锚（half=1000/(0.01×90000)=1 张，旧 10 万张反锚） |

### 机检

build✓ vet✓ gofmt✓ test✓（全量 19 包零 FAIL）race✓（trading 域）import✓（55/55）迁移 contract✓ lines✓ env✓

### 偏离说明

无——票面修法逐条落位；追加验收锚（openfirst WithEarn）以装配码+运行时日志行落位（真单前 RedeemToTrading 留痕行属首单执行时自然产出）。
