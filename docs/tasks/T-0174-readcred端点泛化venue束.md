# T-0174 readcred 端点泛化——venue 束回包（第二家交易所出现前最便宜的改法）

## 态：✅ 完成（票内决策席验收注+b690fc5 投产销号在案——支 c5e682b 已合 master；票面欠翻补正 @10-28）
## 领取：zcode-glm @ 2026-10-27
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

## 结果回填（zcode-glm @ 2026-10-27）

**实现摘要**
- SG `cmd/core/readcred.go`：`GET /v1/readcred`（无 venue 段）venue-map 回包——
  `strpos(key,'_read.')>0` 枚举 `<venue>_read.*` 行族，`strings.Cut` 切
  venue/field，master key 解密直出 `{<venue>:{<field>:<明文>}}`。白名单=
  app_config 键 `readcred.entitlement`（venue 字符串数组，fin_0075 种子
  `["okx"]`；键缺/形坏=503，空数组=显式空面）。fail-closed 分级：主钥缺/
  白名单缺/查库败/解密败=503（系统性，禁 200 空 map 冒充）；行族残缺如实
  透传（SG 无 venue 凭据形面知识——「加行即面扩零代码」语义不可破），形面
  校验归消费臂。
- dev `cmd/research/readcred.go`：`fetchReadcred` 改收 map；`authClient` 加
  venue 维度（`newAuthClient(url, venue)`），refetch=取 map→`venueCreds`
  按臂提件换件——401 自重取同形，map 级粒度（okx 臂 401 只换 okx 件）。
  `venueCreds` okx 臂三件齐校验，缺所/缺件/未支持 venue=错 fail-closed。
- env：`deploy/o-research.env.example` READCRED_URL→`.../v1/readcred`。
- 旧 `/v1/readcred/okx` 直接切（dev 唯一消费方同票改，无兼容壳）；路由 404
  有测试锚。
- `docs/数据流.md` readcred 三处同 commit 更新。
- **rebase 对账**：施工期 master 前移 60+ 提交（T-0173~T-0183 并入），
  `fin_0071` 号被 T-0173 host_metrics 占用——本票迁移让位重编
  `fin_0075_readcred_entitlement.sql`（撞区解沿 T-0177「master 跳 71 让位」
  先例），种子行带 updated_by='fin_0075'（T-0178 写者身份惯例）；rebase 后
  gen 面重生成，build/gofmt/vet/staticcheck/check_migrations 七断言全过。
  **【决策席验收注】再撞：fin_0075 被 T-0185 cashflow_kind_class 先落库——
  本票二次让位终编 `fin_0076`（迁移名/种子 updated_by/码注全改）；
  若不重号：SG 已应用 75→migrate 探 version 即跳→白名单行永不插入=静默
  fail-open。**

**偏离说明（交决策裁）**
1. 票面「entitlement 白名单表」落为 app_config 键 `readcred.entitlement`
  （数组行），非新表——「SG 配置面」读作 app_config 本面（D-0005 配置唯一
  入口），另立表=第二配置面重复；「加行」=数组加元素（仍一行操作）。
2. 残缺行族透传 vs 503：SG 泛化后无法零代码判 venue 形面完整性，选透传+
  臂侧 fail-closed（research 端有缺件锚对拍）。
3. 部署序（条件请求）：SG 先部署（boot 迁移自动落 fin_0075 白名单行）→
  dev 机 env READCRED_URL 路径改一次（`.../5545/v1/readcred`，去 `/okx`）
  +research 重启。顺序反了不炸（readcred 404→WARN 无凭据降级→401 自重取
  自愈），但正序零 WARN 窗。
4. **条件请求·立单**：测试实例 schema GC 缺位（testutil.Pool 每测试建
  `t_*` schema 用后不清；前身 test_db_gc.sh 按 arb_* 库+arb 角色，与 O
  拓扑脱钩且 cron 日跑日 auth failed——实例积 18365 schema/32GB 实证）。
  本次应急逐条 DROP 清至 ~7千；根治须立单（testutil 自清或 GC 脚本换面）。
  同坑条目已入工程经验库（test-schema-per-test-no-gc-instance-suffocation）。

**测试证据**
- `go test ./cmd/core/ ./cmd/research/`（O_TEST_PG_DSN 独立实例）：readcred
  锚 8+6 全 PASS——①三件齐 map 直出+noise 行不入+旧路径 404；②桩锚/验收
  锚：伪所 bnb_read.* 三行+白名单加元素→200 含 bnb 件零代码；③entitlement
  控面（未授权缺席/空数组显式空面）；④残缺透传两件；⑤坏密文 503；⑥白名
  单键缺 503；⑦主钥缺 503；⑧非 GET 405。dev 端：map 解形+多所对号提件
  （伪所不串件）/缺所·缺件·未支持 fail-closed/401 自重取换件（map 多所只
  取本臂件）/重取败原 401 上抛/非认证错零重取。rebase 后 core 91s/research/
  pgstore 三包复绿。
- 全仓 `go test ./... -p 2`：39 包绿；**存量红两包（非本单引入）**——
  `internal/portfolio/lifecycle`（TestOpenLifetimeRejectsFlip/TestOpenBreakevenRejectsThin
  ——rejected 评估卡未落）/`internal/research/sigdist`（TestBasisAnn）在
  master 本尊（1166dd7）同挂实证，涉及面外，另案交决策。
- 偏离测试坑一枚（自坑自纠）：`json.Unmarshal` 复用非空 map 不清底——
  二次解形 `{}` 残留首轮键，测试需新 map 承接（条目已入库）。
