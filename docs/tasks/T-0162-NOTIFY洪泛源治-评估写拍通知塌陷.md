# T-0162 ev_* 通知洪泛源治——评估写拍逐行 NOTIFY 塌陷（饿死钉死的供方根）

## 态：已批
## 域：事件脊 / PG notify 供方治理
## 源出：T-0154 返修实害根因链的供方半——消费侧读饥饿已修（专读排干），
  但 ev_eval/ev_strategy 通知率被评估 upsert churn 放大（T-0150 后合成器
  批量拒+枚举面 1014 标的每拍评估覆写→每行一 NOTIFY→log ~770 行/min 级）。

## 病灶
fin_opp_eval 每键每次评估写都发 NOTIFY（payload 含新 ts 使 PG 同事务去重
失效）——写拍频率远超事件消费者需要（lifecycle 一醒一拍即够）。后果：
notify 队列积压（PG 有上限，满则发送方 NOTIFY 报错）、扫描臂空转烧拍、
Watch 前端被无谓推帧。消费端解耦已止血，供方洪泛是同类病灶的另一半。

## 施工
1. **通知发射收敛**：ev_eval 改「实质变才发」——verdict/state/score 显著
   变化才 NOTIFY（覆写同值静默），或每事务/每扫描批一次合流 NOTIFY
   （payload 固定值即天然同事务去重）。评估写拍本身不动。
2. **ev_strategy 同查**：批量迁移/退役潮期间按批合流。
3. **notify 队列水位观测**：pg_notification_queue_usage() 入巡检/哨兵面
   （洪泛先于队列爆顶可见）。
4. 桩锚：同键重复覆写零 NOTIFY；批内多变一 NOTIFY；队列水位可观测。

## 验收锚
- 生产实测 ev_eval NOTIFY 率降一个数量级+（log/队列水位对照）。
- lifecycle 唤醒语义不变（实质变仍即时醒）。
- 供方侧零「notify queue full」类错误。
