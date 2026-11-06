# T-0386 ops_event 防篡改账——hash 链化与禁 UD

- 状态：待验（2026-11-06 业主令——grounder 防篡改账移植：
  裁决账不可追改=自治可信根）
- 领取：devin-o-worker-T-0386 @ 2026-11-06（DM 常径机派）
- 优先级：P1——账完整性硬固（自我迭代加固族①）
- 收口移交归：决策席

## 病灶

ops_event 是全公司裁决/派单/参数调整唯一账——写进程同
角色可 UPDATE/DELETE，账可被追改。自治面越大，账不可篡改
越是可信根（grounder 判例：append-only 脊+SHA 链）。

## 范围

1. 迁移（fin 链水位+1）：ops_event 增 `prev_hash`/`hash`
   两列（text）——从现 max(id)+1 起链（老行不回填，
   创世行 prev_hash=NULL 锚注明）；
2. 链写：INSERT 侧 app 层算 hash=sha256(id|ts|seat|event|
   ticket|note|prev_hash)——写面集中在 report/直插两径
   全改走同一链函数（或 DB trigger 兜,择一注释引证）；
3. 禁 UD：迁移内 REVOKE UPDATE/DELETE ON ops_event FROM
   应用角色（写角色册查——grant 面同迁移修,注释明）；
4. 验链拍：慢拍（挂 hunger/矿工同槽正径）全链重算→
   断链/伪戳=ops_event `账链断`行+升件（自报可自证）；
5. 验链脚本 `scripts/check_ops_chain.sh` 入 check_all
   （或入 bridge-audit 段,择一）。

## 验收锚

① 桩测：插3行链连续+篡改中间行→验链检出+自报行；
② REVOKE 断言：应用角色 UPDATE 被拒；
③ 创世锚+链起点断言；④ check_all 绿戳对口；
⑤ 镜像面件并入发窗。

## 红线

- 不改既有行内容（链从新行起——历史不可变已是事实）；
- 不破坏写面（INSERT 照旧所有席可写）；
- 验链拍纯读零写；
- 不动真单径。

## 结果回填

### 实现摘要

- **迁移 fin_0120**（fin 链水位+1）：ops_event 增 `prev_hash`/`hash` 两
  text 列——从迁移后首个新行起链（老行不回填，创世行 prev_hash=NULL）。
- **链写=DB trigger 兜**（票面择一引证）：`ops_event_chain_link` BEFORE
  INSERT——链头=最新已链行 hash，`pg_advisory_xact_lock` 串行链写；
  `hash=sha256(id|ts|seat|event|ticket|note|prev_hash)`（ts 定式
  `YYYY-MM-DD"T"HH24:MI:SS.US"Z"` UTC 微秒）。取 trigger 非 app 层
  统一链函数：写面异构多径（report.sh psql/Go 服务族/meta 壳脚本/
  部署尾拍）无单一 app 收口点，库内闸才全径必经；且写者显式供
  hash/prev_hash 被 trigger 覆写——伪戳入库即不可能。
- **禁 UD**：`ops_event_lockdown()` 幂等锁权函数——应用角色册
  （roster_rw/research_rw/review_rw/o_fin_ro/analyst_rw/collector_rw/
  projector_rw/research_ro）+PUBLIC 收编 UPDATE/DELETE/TRUNCATE
  （prod 实测现役写角色=roster_rw 持 INSERT+SELECT——写面照旧；
  TRUNCATE 同族收=毁行同义改账）。链写触发器 INVOKER 须 SELECT
  链头——写册角色 SELECT 补授同迁移守「不破坏写面」。
- **验链**：`ops_chain_verify()` 库内单源全链重算——创世锚/伪戳/断链/
  裸行四类断（前链行=最近非空 hash 行，跨裸行不冤枉）。
  `internal/platform/opschain`：Verify 纯读零写 + ReportBreak 落
  `账链断`+`升件`（账链断类/ops_event 域）自报双行，同窗 24h 去重；
  挂 lifecycle 饥饿拍同槽（chainV 缝，置零即跳）+tools/hungertick
  同径——败情让位不阻主拍，验链假败走 hungerEscalate 自报径。
- **机检**：`scripts/check_ops_chain.sh` 入 check_all ops-chain 段——
  迁移/登记/接线三面静态断言+`OPS_CHAIN_DSN|FIN_PG_DSN` 条件 LIVE
  段（函数未落地=跳过不炸）。判类册补「账链断」类；数据流.md
  防篡改链段同 commit；import 白名单补 lifecycle→opschain 边。

### 测试证据

- `internal/platform/opschain` 三锚桩测全绿（O_TEST_PG_DSN 实库）：
  ① `TestTamperDetectedAndReported`——3 行链连续验链零断→篡改中间行
  note→`伪戳` 检出；篡改链针→`断链` 检出；ReportBreak 落 账链断+升件
  双行且自报行自身入链、同窗再报去重；
  ② `TestLockdownRevokesUD`——授权漂移景重锁→roster_rw UPDATE/DELETE/
  TRUNCATE 权无、INSERT/SELECT 照旧；SET ROLE 功能面 UPDATE/DELETE
  实拒、INSERT 实通（含 AFTER INSERT touch 触发器 INVOKER 面）；
  ③ `TestGenesisAnchorAndLinkage`——首链行 prev_hash=NULL+逐行
  prev_hash=前行 hash+显式伪戳供值被覆写。
- 接线桩 `TestHungerTickChainVerifySameSlot`（lifecycle）：验链同槽
  每拍先调、断链→ReportBreak+Detect 照常、假败→升件自报行落。
- check_all 全链绿戳对口（含 ops-chain 段+go test 全谱）。

### 偏离说明

- hash 域=票面钦定七字段——`esc_class`/`esc_scope`（判例键列）不在链面，
  该两列追改不在验链检出域；扩式另票，本票不擅动钦定公式。
- REVOKE 面加 TRUNCATE 同族收——票面「禁 UD」同义面（毁行=改账）。
- 残余限度在案：持权者全链前向重算型追改（改行+全链重算重写）不在
  检出域——grounder 同类限度；本票缓面=应用角色 UD 已收编+写者伪戳
  不可入库+周期验链。
- 镜像面件（Go 码+迁移+脚本）并入发窗——不合并不部署按票面。
