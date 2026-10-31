# T-0161 eval↔proposal 联动清扫+陈旧提案回收（滞留类病灶）

## 态：审计通过
审计席：headless @ 2026-10-30
## 领取：zc-glm-t0161 @ 2026-10-26
## 域：组合生命周期 / 提案队列治理
## 源出：T-0150 验收观察（SHEIN 评估早已 rejected 而 armed 提案滞留靠手清）
+ T-0160 盘点（244 张 dcd_call fin_proposed 陈旧滞留——product 到期即僵尸）

## 病灶
评估与提案两张表各管各：eval 翻 rejected/退役后，其 armed/proposed 提案
无联动退位机制；提案本身无时效概念——引用的 product/行情过期后提案成
僵尸仍占 fin_proposed/armed 位（黑名单 trigger 只覆盖 venue_untradeable
一类的联动面）。

## 施工
1. **eval 翻面联动**：eval→rejected 时同 eval_id 的 fin_armed/fin_proposed
   提案→fin_retired（reason=eval_rejected 溯源行；幂等）。
2. **陈旧回收**：fin_proposed 超阈（如 24h 或 proposal 关联对象过期）→
   fin_retired（reason=stale_expired）——日拍或事件径自择（清道夫拍可借
   T-0153 骨架）。
3. armed 提案陈旧同样裁（armed 超窗未执=提案失效，executor 侧重评估）。
4. 桩锚：eval 翻拒→armed 提案退位+溯源行；陈旧提案过窗→退位幂等。

## 验收锚
- 生产实态：eval rejected→其提案当日退位零滞留。
- fin_proposed 不再无限堆积（陈旧回收日志在案）。
- fin_proposal 计数分布回正（非靠手清）。

## 结果回填

- 0065 迁移（migrations/0065_proposal_retire_trace.sql，已登记）：fin_proposal
  补退位溯源列 retire_reason（写方语义闭集：eval_rejected/stale_expired/
  armed_ttl_expired；存量行 NULL=机制前溯不虚构）+retired_at 时点列；两个
  部分索引只盖活行（live_eval=eval_id WHERE 活态、proposed_age=created_at
  WHERE fin_proposed）——fin_proposal 全表 63.5 万 vs 活行 439（生产亲证），
  卫生闸逐拍跑，不建=每拍 seq scan 洪泛。
- 施工项①eval 翻拒联动（lifecycle/retire.go retireEvalRejected）：eval
  verdict='rejected' → 同 eval_id 的 fin_proposed/fin_armed 提案即拍退役
  +eval_rejected 溯源行。触发径=事件（T-0162 0064 实质变触发器——verdict
  翻面即醒 lifecycle 本拍即扫，当日退位零滞留；未走日拍）。幂等=state
  谓词只命中未退行。live/probe 策略与持仓零触（提案只是派生凭据）。
- 施工项②陈旧回收（retireStaleProposed）：fin_proposed 超 TTL 未吸收→
  retired+stale_expired。TTL=config lifecycle.proposal_stale_ttl_h 键缺/
  非正=缺省 24h（armed_ttl_h 同例不配键不瘫）。径自择=事件径（同上），
  未借 T-0153 日拍骨架——单 Owner 归 lifecycle（提案状态机既有家），
  ev_eval 唤醒频率（T-0162 实测值变 ~73 行/4min）足保当日性。退后
  verdict 仍 proposed 的合法机会由评估写方下拍重新起草（open 闸只数
  未退行）——dcd 僵尸类 244 张（T-0160 盘点）首批即清。
- 施工项③armed 超窗同裁：T-0136 armed_ttl_h 机制原已在岗（本票勘查
  确认），补 armed_ttl_expired 溯源列。T-0069 同键只留最新退役径未动
  （范围外——票面未列，该径存量行为零溯源保持）。
- 桩锚四绿（lifecycle/retire_test.go）：锚①eval 翻拒→proposed+armed
  联动退位+溯源行+幂等二扫+对照零误伤；锚②陈旧过窗回收 stale_expired
  +新鲜行不动+幂等；锚③armed 超窗落因+新鲜 armed 不动；锚④scan 序
  免误开户（eval 已翻拒提案先退位不建策略行）。
- 机检：build✓ vet✓ gofmt✓ migration contract✓ migrations 四断言✓
  env✓ staticcheck 新增零告警（pre-commit 实跑）；lifecycle 全包 46s 绿
  +pgstore 4.6s 绿（迁移登记面）——既有夹具（全 proposed 形）零干扰亲测。
- 偏离说明：施工项②票面「日拍或事件径自择」择事件径（理由如上）；
  T-0069 退役径未补因（票面未列，留存量行为）。

## 判决注记（审计席 headless @ 2026-10-30，收口=72610be，origin/master 在案）

- 锚①eval rejected→提案当日退位零滞留【过】：链路亲证——fin_opp_eval
  挂 ev_eval_trg（0064），lifecycle Runner 订 ev_eval 醒 scan()，
  scan 首步即 retireEvalRejected（state 谓词幂等）+retireStaleProposed。
  生产（o-postgres:5433 只读亲查）：138 行退位印 retire_reason=
  eval_rejected+retired_at=2026-10-27 10:58:46Z（部署后首拍补清算式集中
  退役）；当前 JOIN fin_opp_eval verdict='rejected' 而提案仍
  fin_proposed/fin_armed 者=0——零滞留在实。
- 锚②fin_proposed 不再无限堆积【过】：当前 fin_proposed=0 行、
  created_at>24h 陈旧行=0；retireStaleProposed 已挂 scan 第二拍（TTL
  =lifecycle.proposal_stale_ttl_h 缺省 24h）且桩锚过窗/幂等绿。
  诚实注记：stale_expired 行生产实测=0——该径至今无独占目标（zombie
  主流形态=eval 已拒，被 scan 序首拍 eval_rejected 先捕；余者 24h 内
  armed/翻面），非机制缺位；退役行本身即落痕在案（reason+时点列）。
- 锚③计数分布回正非手清【过】：fin_proposal 分布=823 fin_retired/
  75 fin_armed/0 fin_proposed；机制后退役 399 行全部带机写
  reason+retired_at（eval_rejected 138+armed_ttl_expired 261，
  10-27→10-28 跨拍散布=持续在岗非一次性手清）；424 行 NULL=机制前
  存量（回填自明不虚构）。
- 码面核对：retire.go/run.go/retire_test.go/迁移内容在 origin/master
  全在（迁移后随 T-0167 分链改名 fin_0065/md_1065 加 IF NOT EXISTS，
  T-0258 增补策略面退役——后续演进非本票缺失）；两部份索引
  live_eval/proposed_age 生产实存；迁移 contract 闸对 retire_reason
  无强制（键域白名单不含此类列，闭集由写方字面量保证）——非违例。
- 注记两则（不报缺）：①138 行中 45 张其 eval 今已翻回 proposed——
  票面自明设计（评估写方下拍重新起草新行），非误伤；②o-core 容器
  现处 Exited（~21h 前净退出）——运维面观察，锚外事实，报管理席知。
