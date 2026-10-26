# T-0170 dcd/gapscan 认证读面恢复——venue 私有 GET 经 SG 中继或只读钥（T-0165 投产回归件）

## 态：已批
## 立项：决策席 @ 2026-10-26（T-0165 投产实证回归）
## 决策档：D-0029/T-0165/T-0102

## 病灶（投产实证）

o-research 起后两臂 fail-closed：dcd feed 调 `sfp/dcd/currency-pair`+`products`
=OKX **私有读面**（unauth→50103），票面「公开读面零凭据」对此两臂不成立。
现 dcd/gapscan 每拍 401 让位（不崩）——DCD 发现链+gap 扫描暗、
`md_dcd_product` 冻结、244 paper 策略 fill 随报价陈化自然停。

## 施工（二径——先等业主只读钥回执；超时走 relay）

- **径 A（钦定首选）**：业主 OKX console 开**只读权限 API 钥**→
  `~/o-research.env` 加 `OKX_API_KEY/SECRET/PASSPHRASE` 三件→
  `cmd/research` okcli 构造改读 env 凭据（`NewClientReadOnly`——读钥天然配
  ReadOnly client，写径 client 侧亦不可达）。研究域持**只读**钥不违 T-0102
  （红线本意=不碰钱径；只读钥 venue 侧无写权）。
- **径 B（工程兜底）**：o-core 加 `GET /v1/venue-get?path=` 中继——SG 本地
  凭据签名+GET-only+path allowlist（`sfp/dcd/*`+gap 探针面）+bind 127.0.0.1；
  隧道加 `-L 5545:127.0.0.1:PORT`；research feed 加 relay 模式
  （okcli 无签名直指 relay）。

## 桩锚

- dcd feed 经新凭据/中继→产品面 200 行；两臂 fail-closed 日志消失、产出恢复。
- env 凭据缺→现 fail-closed 行不变（不回归）。

## 验收锚

- `md_dcd_product` ts 恢复刷新；fin_surface_gap 新行；研究席六臂全活。
