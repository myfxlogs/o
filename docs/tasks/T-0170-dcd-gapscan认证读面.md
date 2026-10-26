# T-0170 dcd/gapscan 认证读面恢复——venue 私有 GET 经 SG 中继或只读钥（T-0165 投产回归件）

## 态：已批（**钦定径 C 表作中继**——业主授权 A 作过渡备选；B relay 留即时读场景）
## 立项：决策席 @ 2026-10-26（T-0165 投产实证回归）
## 决策档：D-0029/T-0165/T-0102

## 病灶（投产实证）

o-research 起后两臂 fail-closed：dcd feed 调 `sfp/dcd/currency-pair`+`products`
=OKX **私有读面**（unauth→50103），票面「公开读面零凭据」对此两臂不成立。
现 dcd/gapscan 每拍 401 让位（不崩）——DCD 发现链+gap 扫描暗、
`md_dcd_product` 冻结、244 paper 策略 fill 随报价陈化自然停。

## 施工（三径——最优解裁 2026-10-26）

**两臂真实需求=两份慢变目录**（非任意 venue 读）：dcd feed
`currency-pair`+`products?baseCcy=`；gapscan `OKXCatalog` 两面
（currency-pair+lending-rate-summary）。周期性同步即足。

- **径 C（钦定——表作中继）**：o-core 起「venue read sync」岗——按面清单
  （OKXCatalog+dcd products 展开式）周期调认证 GET→原样落 `md_venue_read`
  表（venue/face/raw_json/ts，迁移一链）；dev 侧 dcd feed+gap probe 改读表
  （ts 陈=面死=臂让位——血统语义复用）。**dev 零凭据物料+零新攻击面**；
  扩面=sync 岗加行；PG-md 落地后经 projector 桥表投影零返工。
- **径 A（过渡备选——业主授权在案）**：只读钥→env→NewClientReadOnly。
  面宽于需要（全账户读权换两目录）——C 落地前想先复明可用，C 落地撤钥。
- **径 B（留即时读场景）**：SG `/v1/venue-get` relay——仅当未来某面需
  同步即时读时启用；目录面不需要。

## 桩锚

- dcd feed 经新凭据/中继→产品面 200 行；两臂 fail-closed 日志消失、产出恢复。
- env 凭据缺→现 fail-closed 行不变（不回归）。

## 验收锚

- `md_dcd_product` ts 恢复刷新；fin_surface_gap 新行；研究席六臂全活。
