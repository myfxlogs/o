# T-0502 研究员上行径断线根修——webhook 纸面径改 ops_event+CEO 滤队白名单补席

- 状态：已批（业主令「一切修」——D-0043 钦定设计「每件即送 CEO」的接线完工）
- 级别：P2 进水口接线面——断的不是优化是通路：研究员产出自出生起
  从未到过任何裁决席

## 病灶（2026-11-09 COS 三面实证）

`skills/研究员/SKILL.md` 钉死「产出当场送 CEO 裁决——送径＝meta 外部面
webhook；端点未就绪期＝会话尾回执记『待送清单』」。实测：

1. **webhook 端点不存在**——meta 盒全量检索无此端点；送径是纸面概念，
   落回执「待送清单」= 停在会话收据里无人扫；
2. **CEO 滤队 seat 白名单无研究员**——`scripts/meta/ceo_queue.sh` 升件/
   呈批 seat IN 集 = {COS上行,COS,O-Dev交互席,决策席,META-DM,CTO,CEO,
   裁定席,施工席,外审席,CEO上行,管理席}，研究员升件即使落库也进不了
   CEO 队列；
3. **ops_event 零条 seat=研究员 行**——两触发（T-0300 达阈/venue 新品）
   历史零触发，双断面从未爆雷=断线一直没被看见。

## 范围（CTO 钦定回填——depgate 缺清单补位）

1. `scripts/meta/ceo_queue.sh`——seat 白名单补 `'研究员'`；
2. `skills/CEO席/SKILL.md`——收件面 seat 记述同步更新；
3. `skills/研究员/SKILL.md`——送径 webhook 退役改 ops_event 直写；
4. `scripts/meta/tests/`——ceo_queue 出队桩测新件（升件入队/终态消队两断言）。

## 判据

1. `ceo_queue.sh` seat 白名单补 `'研究员'`（skills/CEO席/SKILL.md
   收件面 seat 记述同步更新，文档与码面一致）；
2. `skills/研究员/SKILL.md` 送径改写：webhook 径整体退役，改**直写
   ops_event**（seat=研究员,event=升件,ticket=提案/venue 标识）——
   「待送清单」回执降级为写径失败时的兜底注记，不再是主径；
3. 桩测：造一行 seat=研究员 升件→`ceo_queue.sh` 出队含该票；
   裁决行落库后出队不含（终态消队语义不破）；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 不动触发定义本身（两触发闭集是 D-0043 钦定，不开第三门）；
- 不碰钱径/venue 评估逻辑；不改 CEO 裁决权限边界；
- 写径遵守 T-0499 键卫生闸（若两票同拍落，本票写径走其闸件）。
