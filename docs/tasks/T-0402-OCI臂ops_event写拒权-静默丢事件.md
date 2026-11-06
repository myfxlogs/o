# T-0402 OCI 臂 ops_event 写拒权——静默丢事件

- 状态：已批（2026-11-06 业主令——部署窗实证日志：
  o-research `[onboard] mt5 推进败: ERROR: permission
  denied for table ops_event (SQLSTATE 42501)`——OCI 侧
  事件行静默丢，业主可见性面漏）
- 领取：待施工
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

（施工写：定性 / 修法 / 测试证据 / 偏离说明）
