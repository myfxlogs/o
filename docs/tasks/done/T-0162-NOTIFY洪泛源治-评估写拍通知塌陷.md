# T-0162 ev_* 通知洪泛源治——评估写拍逐行 NOTIFY 塌陷（饿死钉死的供方根）

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面
审计席：headless @ 2026-10-30

## 审计判决注记（逐锚对账，收口 commit 657fa16 全量 diff 亲核+生产取证）

- 锚①（生产实测 NOTIFY 率降一数量级+）：生产 schema_migrations 64 行在册
  （2026-10-26 12:58Z 施）；ev_eval_trg 实拍=FOR EACH ROW +
  ev_notify_eval('ev_eval')，函数体与 0064 同文（verdict/gross_apr/
  net_apr/legs/evidence IS DISTINCT 才发）；本席实测径复验——生产 LISTEN
  ev_eval 90s 窗零通知（测法本机先证伪：stdin-管 psql 确能印异步行），同窗
  fin_opp_eval 写拍 ~13 行/min（300s 内 65 行）——旧逐行 autocommit 形应
  同率醒，现=0；pg_notification_queue_usage()=0。降幅越一个数量级。
- 锚②（lifecycle 唤醒语义不变）：lifecycle/run.go 仍 LISTEN ev_eval→
  r.scan；触发器保留 INSERT 恒发+实质变即发（verdict 翻面即醒径在）；
  notifyflood_test.go 桩断实质变即时醒——语义未改，只静默同值覆写。
- 锚③（供方零 queue full 类错误）：o-postgres 72h 日志 grep 零命中；
  o-core 72h 日志零命中；水位实测 0。
- 产物三证：0064 迁移件在（T-0167 分链改名 fin_0064_notify_flood_shrink.sql
  同文，migrate_table.go:73 在册）+md_1064 双链同文；persist.go 在，
  run.go 拍级 pgx.BeginFunc 单事务合流在；pgguard CheckNotifyQueue 在且
  装配于 cmd/core/main.go（≥50% 告警臂）；KB 条目+索引行在；
  ev_strategy_trg 生产实拍 FOR EACH STATEMENT——「不动」宣称成立。
- 偏离注：票面病灶描述「payload 含新 ts 致同事务去重失效」与码面不符
  （实际恒空 payload，真根=逐行 autocommit）——结果回填已如实自纠，
  治法对症，不影响锚面判决。
## 领取：zc-glm-t0162 @ 2026-10-26
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

## 结果回填

- 0064 迁移（migrations/0064_notify_flood_shrink.sql，已登记
  migrate_table.go）：ev_eval 触发器改 row-level「实质变才发」——
  INSERT（新键）恒醒；UPDATE（普通 UPDATE 与 upsert DO UPDATE 臂同径）
  verdict/gross_apr/net_apr/legs/evidence 任一 `IS DISTINCT FROM` 才
  pg_notify（ts=写拍元数据不计；DO NOTHING 臂零行零 fire）。空 payload
  +同事务同值去重律（0028）不动——批内合流天然在位。
- **形状律实证（PG 18.6 生产同版探针）**：statement 级 transition table
  触发器禁多事件（"transition tables cannot be specified for triggers
  with more than one event"）+INSERT 单事件触发器禁声明 OLD TABLE——
  upsert 写拍在 statement 级拿不到旧版；可行形=row-level 多事件触发器
  （探针实拍：DO UPDATE 臂 TG_OP='UPDATE' 且 OLD/NEW 完整可比）。已入
  工程经验库条目 pg18-upsert-trigger-material-change-shape。
- funding 扫描臂拍级合流（internal/research/funding/run.go）：读评分离
  （md_funding 排干在池连）+写拍整体单事务（pgx.BeginFunc）——逐行
  autocommit 旧形每行独立事务=同事务去重永不生效=洪泛根；改后全拍
  至多一醒+拍原子性（半拍态消除）。persist 改 tx 面（评估/起草闸/
  提案写读同事务一致）。dcd/xcarry/gapscan/grammar 写拍低频（生产
  实测 churn 100%=funding-hedge），留 0064 同值静默即治，不动。
- ev_strategy 同查结论（票面施工项②）：写方全是单 statement 批量
  UPDATE（statement 级触发器天然批合流）+状态迁移皆实质变——零改动。
- 水位观测面（internal/platform/pgguard）：Sentinel 10min 节拍加
  pg_notification_queue_usage() 探针（CheckNotifyQueue）——≥50% 告警
  一次（notify_queue_usage kind，走业主通道；恢复清标再报；线独立于
  xmin 臂）。爆顶=发送方 NOTIFY 报错，50% 线留处置余量。
- 生产亲证（验收锚对账基底）：
  - 病灶量：fin_opp_eval 6532 行，churn≈509 行/拍（100% funding-hedge），
    逐行 autocommit=同率 NOTIFY（票面 ~770 行/min 峰形同源）。
  - 值漂移率：4 分钟窗（恰一拍）509 覆写中 73 行实质变（verdict 变≈0
    ——600 行采样两拍间 verdict 变更=0）；700s 窗 600 行采样 3 行变。
  - 预测：0064 同值静默≈率降至实质变行（~14%/拍活跃窗~0.5%/拍静默窗）
    ；funding 拍级合流把上界钉在 ≤1 NOTIFY/拍——合计从 ~509/拍 → ≤1/拍
    （≥两个数量级，锚①达标线以部署后实测为准）。
- 桩锚（internal/platform/events/notifyflood_test.go 全绿）：
  - 结构锚：Migrate 后 ev_eval_trg=FOR EACH ROW+ev_notify_eval('ev_eval')。
  - 锚①同键同值覆写三连写零 NOTIFY（静默窗）；锚②批内多变一
    （一语句三行恰一醒+同事务三条逐行 upsert 恰一醒——拍级合流形）+
    重放全同值零醒；锚③INSERT 恒发；锚④实质变（verdict/评分/evidence
    三面各测）仍即时醒=lifecycle 唤醒语义不变线；边缘：DO NOTHING 零醒。
  - 哨兵桩锚（pgguard_test.go TestNotifyQueueSentinel）：水位可读可判、
    过线告警恰一次+arm 抑刷、清标再报、常态静默。
  - 行为测触发器重指每测独占频道（共享测试实例跨包噪声隔离），生产
    频道形由结构锚直验。
- 机检：build✓ vet✓ gofmt✓ migration contract✓ migrations 四断言✓
  kbgen gen/skills --check✓ env✓ staticcheck 新增零告警（pre-commit 闸
  实跑）；go test 全绿（本单触及包全绿+其余包隔离跑绿）。
  附注：全量并行跑下 guard/earn/cmd/core/trading-execution 在 600s
  超时崖随机翻车为既有环境病（改造前 run 同形翻车实测：cmd/core 545s/
  earn 556s 贴崖而过；本支隔离跑 guard 174s/earn 145s/cmd/core 147s
  全绿）；pgguard TestXminSentinelQuiet 并行跑一次红=共享实例跨包
  10min+ 残留事务被 pg_stat_activity（集群级可见）喂进哨兵判定的跨包
  噪声，隔离跑+复跑全绿——两处均非本单代码病，留观察。
- 偏离说明：票面「payload 含新 ts 使 PG 同事务去重失效」与码面不符——
  实际触发器自 0028 起恒空 payload（同事务去重有效），洪泛真根=逐行
  autocommit（每行独立事务）。按票面病灶方向（供方逐行 NOTIFY 塌陷）
  治，细节以本段生产亲证为准。

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
