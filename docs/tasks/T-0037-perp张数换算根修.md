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
