# T-0170 dcd/gapscan 认证读面恢复——venue 私有 GET 经 SG 中继或只读钥（T-0165 投产回归件）

## 态：已批（径 A 业主授权 2026-10-26 升首选——等只读钥物料到位）
## 立项：决策席 @ 2026-10-26（T-0165 投产实证回归）
## 决策档：D-0029/T-0165/T-0102

## 病灶（投产实证）

o-research 起后两臂 fail-closed：dcd feed 调 `sfp/dcd/currency-pair`+`products`
=OKX **私有读面**（unauth→50103），票面「公开读面零凭据」对此两臂不成立。
现 dcd/gapscan 每拍 401 让位（不崩）——DCD 发现链+gap 扫描暗、
`md_dcd_product` 冻结、244 paper 策略 fill 随报价陈化自然停。

## 施工（两径——**径 A 业主授权升首选**：「可以给采集放只读api key，这个
是没有问题的，执行端使用交易权限就可以了」业主令 2026-10-26）

**凭据面裁**（业主问「必须 env？能放 PG？热启怎么办」）：

- app_secrets 密文需主权钥解——**主权钥永不出 SG**（出=全库密钥面失守，
  okx 交易钥/mt5/feishu 一泄俱泄）——故「key 放 PG 密文+dev 解」天然毙。
- env vs PG 可读表对 dev 节点**暴露面等价**（research_rw DSN 本就在 env
  里——持 DSN 即可读表；表仅省 /proc/environ 泄露一项）——争论存哪是次优解。
- **钦定 B=零物料**：dev 不存任何 venue 凭据，热启问题消失（无钥可热）。

- **径 B（降备选/工程兜底）**：o-core 加 `GET /v1/venue-get?path=` 中继——SG 本地
  凭据签名+GET-only+path allowlist（`sfp/dcd/*`+gap 探针面+日后研究面私有
  GET 白名单扩张位）+bind 127.0.0.1；隧道 unit 加 `-L 5545:127.0.0.1:PORT`；
  research feed 加 relay 模式（okcli 变体：GET 不签名直指 relay，响应原样透传——
  信封/限流语义不变；relay 侧限流沿用 get429 退避）。**防滥用**：path 白名单
  正则钉死+禁 body+禁 method≠GET；relay 不可作任意 URL 代理。
- **径 A（钦定首选——业主授权）**：业主 OKX console 开**只读权限 API 钥**
  （read permission，无 trade/withdraw 位）→物料到手后落 `~/o-research.env`
  三件（`OKX_API_KEY/SECRET/PASSPHRASE`，chmod 600）→`cmd/research` okcli
  构造改读 env 凭据+**`NewClientReadOnly`**（venue 侧无写权+client 侧写径
  硬闸双层）。凭据存放面：env 文件与 PG 可读表暴露面等价（research_rw DSN
  在 env），钦定 env（简单一致）；主权钥永不出 SG 不变。**等料中**——
  物料到位前 dcd/gapscan 维持 fail-closed。

## 桩锚

- dcd feed 经新凭据/中继→产品面 200 行；两臂 fail-closed 日志消失、产出恢复。
- env 凭据缺→现 fail-closed 行不变（不回归）。

## 验收锚

- `md_dcd_product` ts 恢复刷新；fin_surface_gap 新行；研究席六臂全活。
