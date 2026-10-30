# T-0259 dcdOpen 拒因持久化——venue 码落库，容器重建不毁证

> 状态：**待派** | 领取：虚位 | 优先级：**P1（MANAGE-151 钦定立案——取证面
> 病灶，10-28 故障潮定性曾因日志毁损绕行侧表）** | 执行径：**无头**
> 立项：决策席 @ 2026-10-30（管理令 20261030-105000-MANAGE-151）

## 面况

- o-core 容器重建毁容器日志（10-29 17:02Z 实证）——dcdOpen venue 拒因
  （产品级 `52927` no-quote 恒拒，T-0255 两窗实证 1049+7063 次成交率 0%
  的定性证据）只活在容器 stdout，重建即灭，取证被迫绕行侧表。
- `fin_order_leg.venue_err_code/venue_err_msg` 列已在（flow.go:231
  INSERT 带两列），但**腿未建即拦的上游拒**不落腿表——T-0255 实证
  「18/1069 执行有腿，绝大多数 failed/ambiguous 根本未下单、在上游被拦」。
- dcdOpen 径：`okxexec/dcd.go` DcdQuote（POST /finance/sfp/dcd/quote）
  → DcdTrade——quote 步 52927 拒时执行行如何落、拒因落哪，施工先勘查
  执行面写点（fin_execution 列面/腿建序），再定落点。

## 钦定界

- dcdOpen 执行径（quote 步与 trade 步）venue 拒码+拒文**持久化入 DB**——
  落点取执行面既有同构位优先（fin_execution 列面扩 或 fin_order_leg
  前移建腿 或 fin_eval_reject 同构拒因表，施工勘查定，结果段记取舍）；
- 持久化点须在「失败标定之前或同事务」——容器任意时点重建，已拒事实
  不丢证；
- venue 原始码（sCode/数字码）+原始 msg 均落，禁只落归一化枚举。

## 范围外

非 dcdOpen 执行径拒因面的普查改造（同型面若顺手且同构可顺带，结果段
声明交裁）；告警/展示面。

## 验收锚

1. 桩测：dcd quote 注入 52927 拒→DB 可查 venue 码+msg（非仅日志）；
2. 桩测：trade 步 venue 拒→同上持久；
3. 持久化与失败标定同事务或先行（乱序重建不丢证）；
4. check_all 绿戳；票面回填+产坑 KB（若有）。
