# T-0372 容错位静默组补报——usage_esc 复用推谱

- 状态：待验（devin-t0372 @ 2026-11-05——业主手动外派）
- 立项：`[角色:决策]` 决策席班次 @ 2026-11-05（T-0369 裁决
  观察项落实——清单§2 七组静默位「败了谁知道=答不上」，
  同法理续票非判红项）
- 领取：devin-t0372 @ 2026-11-05（业主手动外派——外部工人 run）
- 级别：P2 机制补格——静默失败面清账（T-0369 容错必报律
  第一批次落地）

## 事实（T-0369 普查在案——docs/容错位清单.md §2）

- `dm_tick.sh:73` `PENDING_TICKETS=$(dm_queue.sh … || true)`
  ——队列推导败=空集与「真无待裁」同形，SKIP_DECISION
  直行+hb 照打 alive，**死人闸不可见**，全管道停滞无人知
  （清单头号静默位）；
- `dm_tick.sh:146` 派单查询 `2>/dev/null`——待派票滞留与
  「无票可派」同形（瞬时败下拍自癒可容，rc≠0 才报）；
- `dm_tick.sh:168` `dm_suspend.sh || true`——看门狗本体败，
  挂起票黑洞加深（本级=末级看门狗，结构性最后一环）；
- `dev_run.sh:93-100` usage 采集 `if/else` stderr——施工席
  token 台账写径败，与 T-0369 标本同病灶（当时票面首用
  只钉 dm_tick，施工席面留本票）；
- `arch_daily.sh:30` 保留期 DELETE 败、`deploy_sync.sh:18,20`
  ssh 取数败、`backup_push_sg.sh:21` sha256 scp 败——淤积/
  陈旧/缺料三面静默。

## 任务

1. 七组静默位逐位接自报径——复用 `usage_esc` 同式
   （ops_event 升件/写败行+note 前缀同窗去重）；
   dm_tick 三位可同文件同批落；
2. `usage_esc` 如需提为共享件（dm_tick/dev_run 两用）
   走既有 scripts/meta 面，禁新通知栈；
3. 每修一位同步更新 `docs/容错位清单.md` 定性档
   （❌静默→✅知情）+registry 机读块刷新
   （`check_tolerate_report.sh --emit-registry`）；
4. 自报率限延用 T-0369 形——同键同窗至多一行。

## 验收锚

1. 七组静默位全部接自报径（或票面逐位写明「留静默」
   裁定理由入册）；
2. dm_queue 败注入实证：队列查询败→升件行落账
   （非空集滑行）；
3. dev_run usage 写径败注入实证同 T-0369 锚③形；
4. 清单定性档同步+registry 重生成零 diff 漂移；
5. check_all 绿戳对口提交 SHA（含 tolerate-report lint）。

## 红线

- 容错语义零动——败了仍容错，只加自报；
- 自报走既有 ops_event/escalate.sh 白名单径；
- 半知情/末环位（escalate.sh 本体、report.sh 族）**不修**
  ——结构性限度 T-0369 已记录在案；
- 队列推导/相位语义/事件词表不动。

## 参考

- 机制母件：T-0369 `scripts/meta/dm_tick.sh` usage_esc
  节+`scripts/meta/escalate.sh` 带类升件径；
- 清单正本：`docs/容错位清单.md` §2 静默位表；
- 纪律条：`docs/工程纪律.md` 容错必报条。

## 结果（devin-t0372 @ 2026-11-05 施工回填）

七组静默位全接自报径——usage_esc 同式提共享件
`scripts/meta/ops_esc.sh`（ops_event 升件行+note 前缀 24h
同窗去重+escalate.sh 带类径在则走/不在直落同构 INSERT）；
容错语义零动——败了仍滑行只加报。清单§2 七行定性全翻
✅知情，registry 机生块重生成零漂移。

### 逐位落法

| 位 | 实落 |
|---|---|
| dm_tick 队列推导（闸+drain 复核两调用面） | dm_queue.sh 改透传 psql rc（原 `| awk` rc 恒 0=静默根）；消费面 rc≠0→无类升件「队列推导败」 |
| dm_tick 派单查询 | 查询先捕获 rc 再喂 while——rc≠0→无类升件「派单查询败」 |
| dm_tick 看门狗本体 | dm_suspend.sh 扫描败 exit 0→1（原与空集同形）；消费面 rc≠0→无类升件「看门狗本体败」 |
| dev_run usage 两位 | 复用 ops_esc——「usage 采集败」写败/META-DEV-USAGE（判类册追记），跨席同前缀 24h 去重 |
| arch_daily 保留期清 | 内联 esc() 同式（OCI 单件部署面无 scripts/meta 树）——「arch 保留期清扫败」OCI-CRON 无类 |
| deploy_sync ssh/comm | rc 捕获≠0→「deploy_sync 取数败/差集败」DEPLOY 无类 |
| backup_push_sg sha256 | 内联 esc() 同式——「sha256 校验件推败」OCI-CRON 无类 |

### 验收锚对账

- 锚① 七组全接自报径：PASS——七行清单定性档全 ✅知情
  （无「留静默」裁定项）。
- 锚② dm_queue 败注入：PASS——`tolerate_silent_batch_test.sh`
  场景 A：O_DM_QUEUE_SH exit1 桩→dm_tick 照跑（空集滑行
  容错不动）+「队列推导败」升件调用实收。
- 锚③ dev_run usage 写径败注入：PASS——场景 D/E：usage 桩
  exit1→「usage 采集败」写败/META-DEV-USAGE；transcript 缺
  →同前缀「transcript 缺/空」；场景 F 正常径零自报。
- 锚④ 清单同步+registry 零漂移：PASS——`--emit-registry`
  重生成后 diff 比对 IN-SYNC；lint `check_tolerate_report.sh`
  ✅全谱有锚/在册。
- 锚⑤ check_all 绿戳对口 SHA：见收口行。

### 机检与自检

- `ops_esc_test.sh` 5 场景 PASS：带类落账/dedup 同前缀抑制/
  异前缀异窗/无 escalate 直落径/dedup 败 fail-open 照升。
- `tolerate_silent_batch_test.sh` 10 场景 PASS（A–J 全列上）。
- 回归：`dm_tick_usage_test.sh`/`dm_tick_worker_resume_test.sh`/
  `dm_suspend_test.sh` 全 PASS——usage_test 桩补两臂
  （DISTINCT ON=看门狗扫描空集、`l.ticket ||`=派单查询空集；
  旧桩派单查询误落 `*)` exit2 被管道吞=本票病灶的活标本，
  修复后显式空集）。
- 自检 A–I：A 复用 ops_event/escalate.sh 既有径零新通知栈；
  B 共享件单源三消费面（dm_tick/dev_run/deploy_sync）同式；
  C 无死码（usage_esc/dm_esc 双包装单引擎）；D 对抗断言=
  15 场景注入（七组败径+dedup+fail-open+正常零扰）；
  E 红线三件零触/队列推导相位语义事件词表零动/末环位不修；
  F 票面回填+清单册+判类册+registry 同 commit；
  G 五锚全 PASS 无虚报；H 病灶径亲读七位原文+T-0369 册在案；
  I n/a（无数据语义面）。
- 偏离 2 项：① OCI 两脚本内联 esc() 非共享件调用——单件
  部署面无 scripts/meta 树，逐字同式（dedup+INSERT）自带，
  部署面零新增依赖；② dm_tick drain 复核调用面同批接报
  （同一 dm_queue 败源第二消费面，票面「同文件同批落」内）。
