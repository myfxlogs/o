---
schema_version: 2
id: simdrill-shape-default-misroute
kind: pitfall
risk: high
status: verified
keywords: [ShapeOKXSingleLeg, runLegged, default分派, fundingLegs双腿构, perp-only乐器, simdrill]
scope: [internal/trading/simdrill, 工单#37649, 升件#37635]
summary: 矩阵 runLegged shape switch 缺 ShapeOKXSingleLeg 臂落 default=fundingLegs 双腿构——perp-only 乐器恒 fail 于现货腿缺、带现货乐器产出 two_leg 凭据与格标 single_leg 不符，全程零报错；修法=显式 case 走产线同码单腿径+default 应 fail-closed
first_seen: 2026-11-24
last_verified: 2026-11-24
evidence_refs:
  - type: ticket
    ref: 工单#37649（票面取 a 补单腿径，被否 b 格级 skip 豁免不取）
  - type: ticket
    ref: 升件#37635（simdrill 毕业径凭据空窗每扫复火——86/246 纸面 dir 乐器 perp-only）
---

# shape 枚举与执行分派双面漂移（matrix_exec 实证）

**坑形**：`runLegged` 的 `switch cell.Shape` 缺 `ShapeOKXSingleLeg`
分支，落 `default:`=fundingLegs 双腿构——该径硬性要求现货腿在册。
双向皆错且全程零报错：①perp-only 乐器（如 O-USDT-SWAP——venue live
在册而对应 spot O-USDT 实缺）恒 fail 于「现货腿缺」；②带现货的乐器
则产出 `two_leg` 凭据，与格标 `single_leg` shape 不符——毕业径拿错形
凭据照跑，判据面与执行面各说各话。

**判别**：①升件按扫面节奏复火、销案词指向「同格同因」；②矩阵格 detail
载双腿径字样但 shape 列标 single_leg——格标与凭据形不符即分派漂移；
③枚举面（`drillshape.go` Shape* 全集）与分派面（`runLegged` case
全集）逐枚对拍，枚举有而分派无臂=default 在吞新形。

**修法**：补显式 case 走产线同码单腿径（DirSide→side、全额名义合成
合约张数、ExecuteDir 开仓、CloseExec 单腿平、两锚对账）；被否的
「格级 skip 豁免」径不取——豁免修不了凭据真形。行数棘轮拆件：
新增执行函数独立成 `matrix_dirleg.go`，宿主文件回棘轮线内。

**推广**：凡「枚举→分派 switch 且 default=某条固定执行径」的结构，
default 即隐形兼容债——每加一个枚举成员，未补分派臂者**静默继承错
径**而非报错。纪律：default 应 fail-closed 显式拒（未知 shape 直接
verdict=fail+原因），宁响不哑；新 Shape 枚举落地须双面对拍（枚举面
全集 vs 分派面 case 全集），对拍闸可机检。
