# T-0170 dcd/gapscan 认证读面恢复——venue 私有 GET 经 SG 中继或只读钥（T-0165 投产回归件）

## 态：已批（**径 A 钦定**——最优解终裁 2026-10-26；C/B 对分离目的反向已撤）
## 立项：决策席 @ 2026-10-26（T-0165 投产实证回归）
## 决策档：D-0029/T-0165/T-0102

## 病灶（投产实证）

o-research 起后两臂 fail-closed：dcd feed 调 `sfp/dcd/currency-pair`+`products`
=OKX **私有读面**（unauth→50103），票面「公开读面零凭据」对此两臂不成立。
现 dcd/gapscan 每拍 401 让位（不崩）——DCD 发现链+gap 扫描暗、
`md_dcd_product` 冻结、244 paper 策略 fill 随报价陈化自然停。

## 施工（终裁——最优解按分离目的轴重判 2026-10-26）

**目的轴**：二级分离=SG 硬盘+带宽卸载——故 venue 下载须在 dev 出口、研究
存储落 dev（PG-md）。据此：

- **径 A（钦定——最优）**：只读 OKX key→`~/o-research.env` 三件→
  `cmd/research` okcli 读 env+`NewClientReadOnly`。**边际暴露≈0**：
  research_rw=SELECT 全表角色，dev 已持「读穿基金全账」凭据——只读钥
  面（账户读）是其严格子集。下载出口 dev、存储落 dev（PG-md 后）——
  唯一两面对齐分离目的的径。**业主授权在案，等钥物料。**
- ~~径 C 表作中继~~（撤）：SG 调 API→SG 落表→dev 隧道读——fetch 带宽+
  表盘双压 SG，与分离目的直接反向（优化错轴：凭据纯度压过了真目标）。
- ~~径 B relay~~（降兜底）：同上 SG 双跳，仅未来「同步即时读」场景可用。

## 桩锚

- dcd feed 经新凭据/中继→产品面 200 行；两臂 fail-closed 日志消失、产出恢复。
- env 凭据缺→现 fail-closed 行不变（不回归）。

## 验收锚

- `md_dcd_product` ts 恢复刷新；fin_surface_gap 新行；研究席六臂全活。
