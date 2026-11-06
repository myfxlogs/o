# T-0402 OCI 臂 ops_event 写拒权——静默丢事件

- 状态：已批（2026-11-06 业主令——部署窗实证日志：
  o-research `[onboard] mt5 推进败: ERROR: permission
  denied for table ops_event (SQLSTATE 42501)`——OCI 侧
  事件行静默丢，业主可见性面漏）
- 领取：待验（2026-11-06 施工收口——check_all 全链绿戳对口）
- 优先级：P1——事件账完整性（静默数据丢失类）
- 收口移交归：决策席

## 实证

OCI o-research 写 ops_event 被 pg 拒权（42501）——
onboard 推进类事件行丢。拒因=该进程连的库/角色无
ops_event INSERT 权（跨库写 SG 域表还是 md 域缺表待定性）。
同类面审计：collector/projector 同险（凡写 ops_event 的
OCI 臂都查）。

## 范围

1. 定性：拒权根因（DSN 指错库/角色缺授权/表不存在——
   三选一对账 DSN 与授权表）；
2. 修法按定性二选一：角色补授权（最小权 INSERT+SELECT
   id）或改写径（事件行经合法桥/API 不直库）；
3. OCI 三臂写 ops_event 面全谱审计（不只 research）；
4. 失败可观测化：写拒权类败→升件行或 WARN 级日志
   断言（静默丢=本次病灶，禁再静默）。

## 验收锚

① onboard 推进行成功落 ops_event（生产实证行）；
② 三臂同面审计结果落回填（各臂写径+权限态清单）；
③ 写败不再静默（桩测：撤权→升件/WARN 断言）；
④ check_all 绿戳对口。

## 红线

- 授权最小化——INSERT+必要 SELECT，禁宽授；
- 不改 ops_event 表构/链（hash 链件刚上）；
- 事件经桥径时禁绕锁另开写径。

## 结果回填

- **定性**：角色缺授权（三选一之②）。`o-research` `FIN_PG_DSN=review_rw`（隧道→SG fin 正本，库/表对——能到表才 42501）。`fin_0086` 只授 `research_rw` INSERT，`fin_0104` 补 `review_rw` 仅 SELECT（eventbus 水印读面），写面从未授——T-0379 onboard 上线时按「fin=ops_event 正本池」写径装配，授权漏配。
- **修法**：取「角色补授权」径——`fin_0124` 条件 GRANT `review_rw` INSERT+SEQUENCE USAGE+SELECT（最小权；SELECT=链写触发器 INVOKER 读链头所需，幂等）。不改表构/链（红线守）；不改写径（直库即合法径，票面无桥可绕）。
- **三臂写面审计**：
  - `o-research`：finPool=review_rw 写 ops_event 共三径——`onboard.trans`（venue接入行，onboard.go:238）、`dcd` 连败自停升件（gate.go:165）、`grammar` vol-sell 升件（volsell_gate.go:100）——三径同被拒；且 dcd/volsell 升件与 fin_condition_gap 同事务，42501 连请求盒行一并回滚=双重静默。ROSTER_PG_DSN=roster_rw 只写 fin_employee_registry 册不碰 ops_event；SG_PG_DSN=本机 md `o`（md 域无 ops_event 表，零写径）。
  - `o-collecto`：SG_PG_DSN=本机 md `o`、ROSTER_PG_DSN=roster_rw（可选）——**零 ops_event 写径**（码面无 INSERT，仅 roster beat/台账）。
  - `o-projector`：MD_PG_DSN md `o`、SG_PG_DSN=projector_rw——**零 ops_event 写径**（投影面不写事件簿）。
  - 结论：三臂唯一 ops_event 写方=o-research/review_rw，fin_0124 一迁全覆盖；collector/projector 无同险。
- **可观测化**：`onboard.trans` ops_event 写败 → WARN 级日志（venue/beat 锚）+`rej.Inc("write_err")`+升件行兜底（dedup 形；拒权时兜底同败、WARN 仍出不静默）。dcd/volsell 升件径授权复通后照常落行（tx 内条件请求+事件行同拍，拒权场景已由 WARN/ err 上抛覆盖——未加重复埋点，diff 守票面）。
- **测试证据**：桩测 `TestOpsEventWriteDeniedWarns`——撤权角色 `t0402_deny`（GRANT USAGE+SELECT 无 INSERT）写径 → trans 返 err + WARN 断言过；`internal/venue/onboard` 包全绿；`go build+vet+test ./...` 全绿；`check_migrations` 七断言过；`check_all` 全链绿戳落（4a07c5e0 基底）。
- **验收锚对位**：①生产实证行待部署后核验（迁移随 SG 部署落——本席无生产写径，留决策/部署窗实拍）；②审计清单上文；③桩测断言过；④绿戳对口。
- **偏离说明**：无。
