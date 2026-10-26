
## 验收（决策席 2026-10-26 @657fa16/merge d28fff5——投产实证）

- 0064 触发器：`ev_notify_eval` row-level 实质变闸（verdict/gross_apr/net_apr/
  legs/evidence IS DISTINCT 才发，ts 覆写静默；INSERT 恒发）——生产触发器形
  实拍在册；DO NOTHING 臂零行零 fire。
- funding 写拍批内合流：读面零写→pgx.BeginFunc 单事务落全批——同事务
  同值 NOTIFY 天然去重=一拍至多一醒（旧逐行 autocommit=洪泛根）。
- ev_strategy 查判：写方本为单 statement 批量+实质变（0028 statement 级
  触发器天然批合流）——不动证在票。
- 哨兵：CheckNotifyQueue——pg_notification_queue_usage()≥50% 告警臂
  （生产实测水位 0）；xmin 臂不受影响（14 LISTEN 会话全 ClientRead）。
- 桩：真 LISTEN 频道行为桩——同值覆写×3 静默/多变一事务恰一醒/DO NOTHING
  静默/实质变即时醒全谱；14 桩三包全绿。
- KB：「通知塌陷」+「PG18 upsert 触发器形状律」双条目入库。
