# T-0174 readcred 端点泛化——venue 束回包（第二家交易所出现前最便宜的改法）

## 态：可领
## 立项：决策席 @ 2026-10-27（业主问「新所又要写 env?」——答：不该写，泛化掉）
## 决策档：T-0170 存放面终裁延伸

## 病灶
`READCRED_URL=.../v1/readcred/okx` 把 venue 刻进路径——每新增交易所=
新端点路径+新 env 键+（若分口）新 permitopen，引导件随 venue 数线性膨胀。
venue 集合是**数据不是拓扑**，应走库面。

## 施工
1. SG `GET /v1/readcred`（无 venue 段）→回包 venue-map
   `{okx:{api_key,api_secret,passphrase}, <venue>:{…}}`——按
   `app_secrets` 里 `*_read.*` 键族前缀自动枚举（加行即面扩，零代码）；
   entitlement 白名单表控 venue（SG 配置面）。
2. dev `fetchReadcred` 改收 map——按各臂 venue 分发内存件；401 自重取
   同形（哪个 venue 401 重取哪件，map 级粒度）。
3. `READCRED_URL` 路径改 `/v1/readcred`（env 值更新一次，此后永不再动）。
4. 旧 `/v1/readcred/okx` 路径保留一版兼容或直接切（dev 唯一消费方）。

## 桩锚
- app_secrets 增伪 venue 行→端点回包含新键零代码；dev map 分发各臂对号。
## 验收锚
- 新 venue 上架径=「app_secrets 加行+白名单加行」两件——dev 零 env 变更实证。
