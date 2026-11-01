# T-0291 armed→strategy 晋升领取件停摆定位+复起（急诊 P1）

| 字段 | 值 |
|---|---|
| 状态 | 审计通过 |
| 立项 | `[角色:管理]` 急诊 P1 直派 @ 2026-11-01（票面由施工代录，派单原文见下；**CTO 事后会签待**） |
| 领取 | claude-kimi-t0286 @ 2026-11-01 |
| 优先级 | **急诊 P1**——armed 提案 4h TTL 空转 7d×347 件全过期=符合条件机会白白放过（T-0200 二期定性） |
| 涉及面 | promote/claim 组件码面（定位后回填）、复起部署径（走 `scripts/deploy_image.sh` 正规闸，施工零直部署） |
| 执行径 | 无头外派 |

## 背景（T-0200 二期补拍定性在案）

- `fin_proposal.retire_reason='armed_ttl_expired'` 7d×347 件——armed 后恰 4h 无人领，全 TTL 过期。
- T-0202 闸修毕已验（10-30 14:53Z 部署，dcd_call 拒因绝迹）但 10-28 后零执行尝试=闸开无人走。
- T-0200 续派指针：晋升领取件停摆定位+复起立票。

## 派单原文（管理席急诊 P1，2026-11-01）

> [角色:施工] 急诊P1——armed→strategy晋升领取件停摆定位+复起。取证锚已备：fin_proposal.retire_reason='armed_ttl_expired' 7d×347件（armed后恰4h无人领全过期）；T-0202闸修毕已验但10-28后零执行尝试=闸开无人走。定位promote/claim组件停摆因（进程死/谓词错/依赖断）+复起+一笔armed→live实走自证。先读/opt/o/AGENTS.md+docs/collab.md，worktree支t/02xx名自取，收口翻待审。边界：零部署零money路径直改，复起走正规部署闸。全程外派。

## 目标

1. 定位 promote/claim 组件停摆因：进程死 / 谓词错 / 依赖断，三选一（或并）定性+证据行。
2. 修复（码面/配置面，worktree 支内）。
3. 复起走正规部署闸——**施工零直部署、零 money 路径直改**。
4. 一笔 armed→live 实走自证（晋升链真通一件）。

## 验收标准

- [ ] 停摆因定位到具体组件+具体证据（日志行/进程相/谓词/依赖），三类定性有据
- [ ] 修复落地（若有码面改动），自测过
- [ ] 复起动线明确且走正规部署闸（不自起进程、不绕 `scripts/deploy_image.sh`）
- [ ] 一笔 armed→live 实走自证（提案→策略晋升成功一行实证）
- [ ] 票面结果回填+收口标记

## 范围外

- 零部署（生产复起由部署闸执行者执行）
- 零 money 路径直改
- T-0255 执行面 10-28 ambiguous 对账（他票在飞）
- entry→execute 断链（T-0200 已定性归 T-0255 系）

## 结果回填

施工 claude-kimi-t0286 @ 2026-11-01（全程外派勘查+亲验定性；取证=SG o-postgres/o-core 只读实拍，批 1+批 2）

### 定性裁决（进程死/谓词错/依赖断 三选一）

- **进程死：证伪**。o-core healthy（started 10-31 23:56Z restarts=0 image=b245dc24bacb）；命官 30s 心跳在拍（11-01 03:46）；scan/promote/retire 全在产痕（armed 退役 01:52、transition 尾 01:08）。
- **谓词错：领取/晋升谓词健康**。7d 领取时滞中位 **2.2min**（p90 9.9min；315/323 件 30min 内领讫）；7d transition 通量 proposed→paper **2062** / paper→probe **1191** / probe→live **1190**。openLive breakeven 闸逐拍拒 23 张 live（carry ~4.5bps<摩擦 32×1.5）=无利可图如实拒、非误拦——但 `research.funding.maker_fee_bps`/`guard.breakeven_safety` 键缺走 T-0247 双 taker 回退档（摩擦=4×(5+3)=32bps），参数校准归决策席（票外①）。
- **依赖断：成立但在件外**。**(a)** 上游产供断流：末提案 10-31 21:52，72h 到达面仅 BTC 季度对儿每 6h 两件、funding-hedge 起草=0（票外②转决策）。**(b)** 同键吸收环：paper 行原无退役径永占同键槽→上游重起草被吸收→4h TTL 退——BTC-USD 季度对儿 3 天 24 提案连环实证（24 孤兒全此因，same_key_active_now=1 逐行核）。

### T-0200 二期定性勘误（证据在案）

1. 「armed 能立≠有人领」不成立——码面 armed=已领/吸收终态（admitProposals 领取成功或同键吸收皆翻 fin_armed）；347 件中 **323 件已领**成 strategy 行。
2. 「无一被领成 strategy」实为：领取后 strategy 行被 DCD 污染清扫批量 retired（748+35 张，决策席令，transition 归因在案），非领取失败。
3. 「entered_at 被扫描器刷新≠真进场」误读——entered_at=迁移时刻（每次 Transition 刷新），11-01 01:08 恰是三张 probe→live 迁移时刻。
4. armed_ttl_expired×347 是 TTL 卫生闸对已领提案的例行回收——286894 全息样本：提案 armed 4h 退役时其 strategy 活着并在 3.5 天后毕业 live。

### 根修（票内，沙箱面零钱径）

`retireStalePaper`（retire.go）：paper 沙箱超 `lifecycle.paper_ttl_h`（**缺省 168h**）Transition 退役+落因 `paper_ttl_expired`；scan 序置于 admitProposals 前（run.go）——领取同拍先释放同键槽。缺省锚定实证：286157 paper 滞 **139h** 毕业、286894 滞 90.7h 毕业（11-01 实拍）——初拟 72h 会误杀慢毕业行，故 7d。无活动豁免（豁免=恒负 carry 每 8h 一 fill 行永生、锁键复现）。degraded 同病先例=T-0098。性质：吸收环解锁+漏斗防堵。

### armed→live 实走自证（生产链+测试回放双证）

- **生产**（transition 行号可查）：286894 NIGHT-USDT-SWAP=提案 403064（10-28 05:16:41 armed）→ strategy 286894（05:16:52，**领取时滞 11.4s**）→paper→probe（11-01 00:00:26）→**live（01:08:20，live_switch=true）**；同批 286216/286157 同径三跳（transition 5247-5252）。
- **测试**：TestPaperTtlRetire 三断言 PASS（超龄退役+transition 落因 paper_ttl_expired+同键新提案重开 paper）；全包 104s 绿；存量锚（TestDegradedTTLRetire/TestArmedTtl×3/TestLifecycleE2E）零回归。

### 机检

build✓ vet✓ gofmt✓ import✓ env✓ rejects✓ migrations✓ bridge✓（check_all 全链绿戳随推支）；lifecycle 包 go test 全绿（O_TEST_PG_DSN=o_test@5434）。

### 票外转决策（不越单）

1. **breakeven 摩擦参数校准**——maker_fee_bps 键缺→双 taker 32bps 档全拒 23 张 live。钱径参数面，决策席裁（补键 or 维持紧闸）。
2. **上游产供断流**——research 起草 72h 零 funding-hedge（另立票；o-projector 帧水位表 fin_projector_frame 不存在，水位取证径缺）。
3. **ledger 对账 mt5 读败**（「事件拍 fn 败: venue 现读: mt5 读败: account 空体」间歇）——对账面让位语义在守（fail-closed 不炸），独立小票候选。
4. armed_ttl 观测量勘误——停摆监控勿再用 armed_ttl_expired 计数（例行政回收），改用领取时滞分位+孤兒率（KB 条目 `armed-ttl-expired-misread-claim-stall` 在案）。

### 验收对账

- [x] 停摆因定位到组件+证据：三选一全裁决（上）
- [x] 修复落地+自测：retireStalePaper+TestPaperTtlRetire PASS+全包绿
- [x] 复起动线：码修随支→决策席审计→`scripts/deploy_image.sh` 正规闸生效；施工零直部署、零生产写、零 money 路径直改（本修纯沙箱面）
- [x] 一笔 armed→live 实走：生产三链（286894 全息）+测试回放双证
- [x] 票面回填+收口标记

### 收口标记

[施工收口:T-0291] @支头（推支部 t/0291-armed-promote-claim-revive）

**审计通过**——管理席 2026-11-01 复证：码面（接线序/Transition留痕/TTL168h锚于139h慢毕业实测/paper面零钱径）全过；生产实证（286894/286157毕业链+领取时滞2.2min+陈paper现量0）全对上；**审计席认勘误**：本席T-0200二期「晋升领取器停摆」三判被本票逐条证伪——armed=已领终态、entered_at=迁移时刻、armed_ttl=已领例行回收。真断点=上游产供断流+breakeven摩擦参数（票外转决策在案）。KB条目武装取读坑并入。
