# T-0170 dcd/gapscan 认证读面恢复——venue 私有 GET 经 SG 中继或只读钥（T-0165 投产回归件）

## 态：✅ 验收毕投产（merge ac592ed——readcred端到端通、328对实证、零401；dcd产品面feed在飞观察项见部署实证补记）
## 领取：zcode-glm-t0170 @ 2026-10-27
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

- **径 A（钦定——最优）**：只读 OKX key→dev 端 `NewClientReadOnly`。
  **边际暴露≈0**：research_rw=SELECT 全表角色，dev 已持「读穿基金全账」
  凭据——只读钥面是其严格子集。下载出口 dev、存储落 dev（PG-md 后）。
  **存放面终裁（业主「还是落 env？」问 2026-10-26）**：不落 env——
  key 写 `app_secrets.okx_read.*`（密文唯一真源，与交易钥同店不同架）；
  o-core `GET /v1/readcred/okx`（127.0.0.1:8095 解密出）+隧道
  `-L 5545:127.0.0.1:8095`；o-research boot 拉取**仅进内存**
  （dev 磁盘零副本）；**自重取**：feed 401→重取一次→再败 fail-closed——
  轮换自愈零值守。**业主授权在案，等钥物料（到手直写 app_secrets）。**
- ~~径 C 表作中继~~（撤）：SG 调 API→SG 落表→dev 隧道读——fetch 带宽+
  表盘双压 SG，与分离目的直接反向（优化错轴：凭据纯度压过了真目标）。
- ~~径 B relay~~（降兜底）：同上 SG 双跳，仅未来「同步即时读」场景可用。

## 桩锚

- dcd feed 经新凭据/中继→产品面 200 行；两臂 fail-closed 日志消失、产出恢复。
- env 凭据缺→现 fail-closed 行不变（不回归）。

## 验收锚

- `md_dcd_product` ts 恢复刷新；fin_surface_gap 新行；研究席六臂全活。

## 结果回填

### 施工面（径 A 全链——SG 解密专口+隧道+dev 内存取+401 自重取）

- SG 端 `cmd/core/readcred.go` 新件：`GET /v1/readcred/okx` @ :8095 loopback
  专口（独立 server 不挂 50051 总线，startReadcred 随 ctx 收停）——
  app_secrets `okx_read.api_key/api_secret/passphrase` 三件解密直出 JSON
  （no-store）；主钥缺/物料缺/解密败=503 fail-closed 不吐部分物料。
  main.go 挂线一行调用（297 行，行数棘轮内）。
- compose：o-core ports 增 `127.0.0.1:8095:8095`（host loopback pin 同
  50051 形；sg overlay 无 ports 覆写=合并即生效）。
- dev 端 `cmd/research/readcred.go` 新件：authClient 持守件——boot 拉
  READCRED_URL（仅内存零磁盘）→`NewClientReadOnly`（D-0013 只读镜像
  结构闸）；feed 认证拒（HTTP 401/信封 50103 双判——401 形状实拍
  journalctl 实证）→重取一次→换件重试；再败=原错上抛 fail-closed
  （与无凭据现态同形让位）；并发拍单飞双检；boot 败=WARN 降级不炸
  boot（后续 401 自重取自愈——轮换零值守）。
- 装配面：researchArms/dcdFeedFor/gapProbeFor/get429/okxCoinsFor 参数
  `*okxexec.Client`→`authGetter` 窄面（域内零改——闭包对凭据件透明）；
  main.go READCRED_URL 缺省=无凭据件原行为（桩锚②不回归）。
- deploy：`o-readcred-tunnel.service` 新件（5545→SG 8095，
  o-collecto-tunnel 同形同钥）+`o-research.service` After/Wants 随挂+
  env 模板 READCRED_URL 行；env-allowlist `READCRED_URL` 入册（引导件
  类——只载地址非凭据本，SG_PG_DSN 族同型先例）。
- 数据流.md 同 commit 更新（反向流 readcred 行+凭据面 SG/dev 两行）。

### 测试证据

- cmd/research 6 测绿：fetchReadcred 形面（200 映射/缺件/非200/坏形拒）+
  isAuthErr 判据（401 串/50103 真；他码他错 nil 假）+authClient 治愈/
  重取败保原错/非认证错零重取。
- cmd/core 4 测绿（testutil 独立 schema）：真钥三件 200 直出+删钥 503+
  主钥缺 503+非 GET 405。
- 机检：build✓（core+research+全树）vet✓（全树）gofmt✓ staticcheck
  新增零告警 check-env-reads✓（白名单 30 键）check_import_direction✓
  check-lines✓（main.go 297<300）。
- 测闸：涉面两包带 DSN 独跑绿（`go test ./cmd/core/ ./cmd/research/
  -count=1` exit 0——core 553.9s/research 0.004s，重构后终版）；earn/guard
  独跑绿。**存量红三件（非本票，净 master 树 4517cee 对拍同值红）**：
  sigdist TestBasisAnn（年化 bps 301.89 vs want 300.41——时敏断言漂）+
  lifecycle TestOpenLifetimeRejectsFlip/TestOpenBreakevenRejectsThin
  （rejected 评估卡未落——test-dsn 条目 T-0124 旧案同形夹具漂复发）；
  交决策席另案。全树 `./...` 印记归 CI（跨席并发同库全测争用坑与
  两段判红法→经验库 test-shared-db-cross-session-serialization 在案）。

### 部署序（决策席——live 验收锚前置）

1. SG tunnel@ authorized_keys permitopen 增 `127.0.0.1:8095`（现仅
   15433——实拍 2026-10-27；缺行则隧道单元 ExitOnForwardFailure 周期重启）。
2. SG 侧轮换（collab §11 四步；compose 增口随 --force-recreate 生效）+
   readcred 实拍：`curl -s 127.0.0.1:8095/v1/readcred/okx` →200 三件。
3. dev 侧：o-readcred-tunnel+o-research 单元更新（deploy/ 件）+
   `/home/mluser/o-research.env` 增 READCRED_URL 行；restart o-research。
4. 实拍验收锚：o-research 日志零 401/50103；`md_dcd_product` ts 刷新；
   `fin_surface_gap` 新行；六臂 beat 在册。
   ——注：钥物料已到位实拍（app_secrets okx_read 三件在库 2026-10-27），
   live 锚须决策席部署后实证，本席交付止于分支+部署序。

### 偏离说明

- 票面「o-core（127.0.0.1:8095 解密出）」：容器内 server 绑 :8095
  （容器内 loopback docker-proxy 不可达）——loopback pin 由 compose
  host 侧 `127.0.0.1:8095:8095` 承担（与 50051 同形）；SG host loopback
  独占语义不变。
- coins（account/balance）随 dcd 臂同持守件——非票面明列但同属私有
  读面（T-0095 归口判据源），不换则凭据到位仍 401 fail-closed；零新
  端点，纯面换 authGetter。

### 部署实证补记（决策席 2026-10-27 部署后验）

部署序 4 步全落：SG permitopen +8095/compose loopback pin/o-core 轮换 →
`o-readcred-tunnel` 起（dev:5545→SG:8095）→ `/opt/o-research/env` 增
READCRED_URL（单元实读件——~/o-research.env 双径坑再证实测，残影已
symlink 归一）→ 新二进制停换起（Text file busy 坑：禁 cp 覆写活件）。

live 锚实拍：o-research 起「读面=readcred 认证读」；重启后零 401/50103
（前置态=每拍恒败）；readcred 端到端实拍 dev:5545→200 三件；
`fin_surface_gap.last_seen` boot 拍刷新（gapscan 认证探针通）；独立
取证 OKX currency-pair 直返 328 对（只读钥有效）；六臂 beat 在册。
md_dcd_product 填充=feed 在飞中——328 对×300ms+429 退避（sfp 限频紧，
实测首对即 429），feed 是全集返回制，整批约 ~40min 窗口后落行；
任一单对退避尽=全面让位（T-0117 退避件既有语义，观察项）。
