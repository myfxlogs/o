# T-0372 容错位静默组补报——usage_esc 复用推谱

- 状态：施工中（devin-t0372 @ 2026-11-05——业主手动外派）
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
