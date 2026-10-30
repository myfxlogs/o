---
schema_version: 2
id: cutover-orphan-service-checklist-gap
kind: pitfall
risk: high
status: verified
keywords: [cutover, 迁移断档, 孤儿服务, 清道夫-md, cmd/sweeper, 心跳判活, last_beat, 服务清单, OCI盒档案, systemd unit]
scope: [采集层迁移 (10-28 OCI cutover), cmd/sweeper, 部署清单纪律, roster beat 判活面]
summary: 服务跨机迁移按「本机在跑进程」清单搬易漏无 unit 孤儿——清道夫-md（cmd/sweeper，独立件非 systemd unit）10-28 cutover 未随迁，心跳 10-28 21:14 断、驾驶舱红灯 2 天无人修因「红灯被当作已知显示问题挂账」；迁移清单须以注册表/册心跳面反向核对（谁在 beat=谁在跑），迁移后逐席对拍
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: [生产取证]
evidence_refs:
  - type: server_fact
    source: postgres
    ref: SG fin_employee_registry 清道夫-md last_beat=2026-10-28 21:14:29Z（30s 拍断 2d+）；PG-md fin_data_retention audit_keep 行 last_swept_at=NULL（清道从未收尾）；OCI/dev 两机 ps+systemctl 零 sweeper 面
  - type: doc
    ref: docs/tasks/T-0241-驾驶舱钱面D0038标尺.md v4③（红灯误挂账为「取证源错位」显示件——真因=进程缺席）
  - type: doc
    ref: docs/运维/OCI盒档案.md 服务清单（无 sweeper unit——迁移后清单面缺此件）
---

# 迁移断档：无 unit 孤儿服务不进迁移清单——红灯被误读为显示问题

## 症状

驾驶舱/System 席位面「清道夫-md」持续红 2 天+。红灯被两层误读：T-0241 定性为「o-core 取证源错位」（显示件挂账），任务书猜「与信号臂盲眼同根因（蒸馏层死亡）」——都假。真相=**进程根本不存在**。

## 根因

1. **迁移清单按「本机看得见的 unit/容器」盘点**——cmd/sweeper 这类裸进程件（无 unit 无容器）从旧机消失时无人在册感知。
2. **红灯钝化**：持续红 2 天被归因为显示层问题挂账（T-0241 v4③），无人对 beat 断点时刻（10-28 21:14=cutover 当日）做关联。

## 判据（复用）

- **谁在 beat=谁在跑**：迁移前后各拉一次 `SELECT name,last_beat FROM fin_employee_registry WHERE status='active'`，逐席对拍——beat 断=孤儿候选。
- 红灯诊断序：先查「进程在不在」（ps/systemctl/容器三面），再查「判据对不对」——先入显示层=方向性误诊。
- 断点时刻关联法：last_beat 断点时刻 ↔ 迁移/重启事件时刻表对拍（10-28 21:14 恰=cutover 窗）。

## 关联

- 驾驶舱v3 任务书 A 件（定性+如实红+说明）；修复径=OCI 用户级 o-sweeper.service（MD_PG_DSN=本机 5435+SG_PG_DSN=隧道 15433）——急诊线候管理席。
- FIX-02（同日）同教训变体：面冻结≠采集死（镜像层）；本条=红灯≠显示病（进程层）。两坑合读：**生产面异常先做「真源-进程-显示」三层定位，勿先入最顺手的一层**。
