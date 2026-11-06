# T-0414 DM机械层出会话——票翻/台账/派单降 shell 轨

- 状态：完成（决策席裁决通过 2026-11-07——机械轨首拍实证另面：ledger A面饥饿病灶已立 T-0417 P1 修）
- 领取：t/0414-dm-shell-lane @ /opt/o-wt/t0414
- 优先级：P1——持续成本件（每班都烧，省的是token+上下文窗）
- 收口移交归：决策席
- 关联：T-0413（派单依赖闸=机械出会话第一步，本票拆剩余面）

## 背景（实锤病灶）

DM 班次=devin 会话，但实证它把会话时花在零判断活上：
11-06 班内实录——`git add docs/tasks/T-0413`（票翻）、
T-0084/0089/0103/0189/0190 五连「台账早已闭」补登、派单行
拉起——全机械动作全在会话里跑。**烧的是双重成本：token
费+上下文窗（杂务挤占裁词容量，加速compact）**。

而 dm_tick.sh 壳层已有全套机械机制（单例锁/心跳/队列推导/
心跳行写）——shell 轨本来就在，只是没把活接过去。

## 钦定语义

```
留会话(判断轨):  审计裁词/敌意回放/merge裁决文/驳回意见
               ——要脑的才烧token
降shell(机械轨): ①票翻git add+commit+push(格式钦定
               [角色:决策][T-NNNN]类,机械串)
              ②台账对账补登(ticket在done/无完成行→
               机械补行,零裁量)
              ③派单轨与0413合流(本来就dispatch_worker.sh)
触发:        kick直驱不走会话——ops_event新行→shell件
           即办;会话开班只为判断件
```

## 范围

1. 票翻降 shell：票面 commit/票状态翻 机械函数（dm_tick.sh
   同式锁+账行同拍；冲突/败→升件不静默）；
2. 台账补登降 shell：对账补登件（done/有票无完成行→补
   ops_event+票翻）成周期/触发的 shell 函数；
3. 会话面收窄：SKILL 班次节改判——会话只接「待裁集非空」
   （判断件），机械集空时零烧空拍现状保持；
4. 竞态闸：shell git 写与会话 merge/push 的序——同台写径
   互斥（复用部署锁型或 PG advisory 二锁），败即升件；
5. 文档：SKILL/流程节改口径——「班=判断才烧，机械走壳」。

## 验收

- 实证形①：落一张 done 票无完成行→shell 补登+票翻完成，
  全程会话零启动（ops_headless 无新 devin session 行佐证）；
- 实证形②：待裁集非空→会话正常开班裁词（判断轨没退化）；
- 账形：机械行 actor=shell 轨可辨（ops_event actor 或 note
  前缀——查账能分谁写的）；
- 竞态桩：shell 写与会话写并发→互斥生效无双写；
- 绿戳对口+全链。

## 结果回填

实现摘要：

- **机械轨件 `scripts/meta/dm_ledger.sh`**（新）——两互补收敛规则：
  A 台账→文件（票翻）：latest=完成 或 裁决/通过且收口已证并 master
  （收口类 note SHA∪票支 tip∪wt HEAD∪merge commit 四层取证）→mv 入
  done/+状态行机械模板翻完成+`[角色:决策]` 机械串 commit+check_all
  绿戳+git_push；裁决未证并=压下拍，龄>2h→升件。B 文件→台账（补登）：
  done/ 票+收口类事件史+无完成行+latest 非负词→补 META-DM 完成行
  （机械轨 actor 可辨，advisory xact+行内重查防重）。状态行识别三形
  （`- 状态：`/`> 状态：`/`## 状态`+**词**）；作废/废弃/取消态不翻字。
- **写径锁 `scripts/meta/dm_repolock.sh`**（新）——部署锁型 mkdir 原子
  +holder+陈锁抢（rc0/3/6），锁目=`git-common-dir`/dm-repo.lock——
  壳轨票翻与会话 merge/push 同台互斥；忙=让位，抢陈锁/脏树/冲突/
  推败→升件不静默。
- **dm_tick.sh 收窄**：会话开班闸由 `dm_queue.sh`（all 面）改 `review`
  面（判断件）；班后收班判同改 review；尾窗挂 dm_ledger 每拍自跑
  （kick 直驱不走会话——ops_event 新行→shell 件即办）。
- **文档**：决策席 SKILL 班次节=判断轨口径+机械轨禁做清单+写径锁规；
  collab.md 席面分家=判断轨/机械轨分轨+dm_repolock 互斥口径。
- **桩测**：`dm_ledger_test.sh` 五景（B补登同拍翻/A-完成归档/A-裁决
  证并归档/未证升件/锁忙零写）+`dm_repolock_test.sh` 五态；dm_tick
  三测试接 O_DM_LEDGER_SH 桩全绿。

**调试事故全披露（诚实记账）**：施工中 dm_ledger 未合并码两次裸跑
（env 覆写漏带）——done_file 空判恒真病灶（`cmd|head` 管道 rc 恒 0）
致 B 面越 done/ 铁证误落完成行 35：无档键/活票误登 10 行已按 id
精删（BASESWP/DBP3a/DEPLOYNF/DOOR4/GATEFIX/PIPEFIX/T-0249/T-0264/
T-0266/T-0409），余 25 行属票面钦定补登语义保留；作废态 6 件翻字
已回滚。补偿行：T-0409 落「移交」复队回待裁，9 无档票落「异常」
作废声明，T-0414 落「异常」事故自报行——台账在案。
根修三钉：①done_file 判存在验串非空不凭 rc；②B 面加 T-NNNN 格式
闸；③状态翻加 terminal-alt（作废/废弃/取消）守。

测试证据：meta 桩测 16/16 全绿（dm_ledger 五景+repolock 五态+
dm_tick 三谱+depgate/suspend/queue 等回归）。
