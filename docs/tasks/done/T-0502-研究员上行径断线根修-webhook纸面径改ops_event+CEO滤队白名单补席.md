# T-0502 研究员上行径断线根修——webhook 纸面径改 ops_event+CEO 滤队白名单补席

- 状态：完成（决策席裁决通过 2026-11-08——判词在案·取证=收口并master·机械轨票翻）
- 领取：devin-o-worker-T-0502 @ 2026-11-09
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

## 结果回填

- 实现摘要：
  1. `scripts/meta/ceo_queue.sh`——seat 白名单 IN 集补 `'研究员'`
     （头注判据行同步）；终态消队语义零改动；
  2. `skills/CEO席/SKILL.md`——无头形态「收件箱改投」seat 记述补齐
     至码面全量集（原枚举缺 COS/O-Dev交互席/CEO/CEO上行/管理席五席，
     本次一并对齐=文档与码面一致判据）；
  3. `skills/研究员/SKILL.md`——「每件即送」送径 webhook 纸面径退役，
     改 `scripts/meta/escalate.sh 研究员 <ticket> '' '' '<产出摘要>'`
     直写 ops_event（seat=研究员,event=升件,ticket=提案/venue 标识；
     escalate.sh 自带 T-0499 键卫生闸+无类升件=永不入判例面）；
     「待送清单」回执降级为写径失败兜底注记；禁线「写径收窄」白名单
     同步补 escalate.sh 径（零手写 SQL 语义不破）；
  4. `scripts/meta/tests/ceo_queue_test.sh`（新件）——真 PG scratch
     库（test-pg 池，case_writeback 同构）全真 SQL 语义桩测。
- 测试证据：
  - **先红**（T-0388 律——修复类票）：`git stash` 退 ceo_queue.sh
    白名单改动后同桩测真跑——`FAIL seat=研究员 升件入 CEO 队`
    （出队输出空=白名单缺席病灶实证），同断言修后绿；
  - **修后绿**：`ceo_queue_test.sh` 3/3——①seat=研究员 升件入队+
    非白名单席（探针自检）升件仍拒；②裁决行落库（ts 显式钉序）
    终态消队语义不破；
  - 回归：`ceo_northstar_test.sh`（同消费 ceo_queue SQL 面）PASS；
    `scripts/meta/tests/` 全谱 42 件跑毕——零新增 FAIL/SKIP
    （5 件红+1 SKIP=check_all_scope/dm_shiftend/dm_tick_worker_resume/
    t0403_gate_converge/testdb_lease/heartbeat_dead_caliber，master
    同跑全数同红同跳=环境既有非本票引入；本票新件 3/3 绿）。
- 偏离说明/范围外发现（不动手呈裁）：
  - 送径选 escalate.sh 而非裸 psql/report.sh：票面「直写 ops_event」
    经既有带类升件写径落地——T-0499 键卫生闸随径在案、INSERT-only
    权限面同守、与研究员「写径收窄」禁线兼容（脚本调用非手写 SQL）；
    report.sh 虽亦可写但带 relay/飞书双写副作用面，escalate 为升件
    本义正径；
  - 触发侧观察（禁线①不动触发定义，仅记录）：venue_watch.sh 落
    ops_event '新品' 行 seat=venue哨兵系研究员触发信号源——哨兵→
    研究员拉起消费链是否在位本票未核，若断则为触发侧同型断线，
    留决策席/业主裁是否另立票。
