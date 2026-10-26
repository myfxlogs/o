# T-0161 eval↔proposal 联动清扫+陈旧提案回收（滞留类病灶）

## 态：已批
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
