---
schema_version: 2
id: dm-ledger-nofile-terminal-ticket-batch
kind: pitfall
risk: medium
status: retired
keywords: [dm_ledger, LEDGER-NOFILE, 终态票无票件, tasks/done双缺, 升件噪声, 票翻不可为, 24h去重, 升件面污染]
scope: [scripts/meta/dm_ledger.sh A面扫描, docs/tasks, docs/tasks/done, ops_event 升件面]
summary: 23 件终态票 tasks/done 双缺——dm_ledger 每扫恒撞 LEDGER-NOFILE-<tid> 升件（prefix 携票号 24h 去重=日窗 ~23 行噪声）且票翻永不可为；无补档/豁免径，真升件被批量噪声淹
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [决策席, 裁定席]
evidence_refs:
  - type: doc
    ref: scripts/meta/dm_ledger.sh（A 面扫描 LEDGER-NOFILE 升件位）
  - type: server_fact
    source: postgres
    query: SELECT id,ts,ticket FROM ops_event WHERE note LIKE 'LEDGER-NOFILE%'——2026-11-06 18:15:54~18:16:15 单窗 23 行（T-0001/T-0118/T-0145/…/T-0295）逐秒批量升件
    observed_at: 2026-11-07T00:00:00Z
---

## 事实/症状

- ops_event 单窗 23 行 LEDGER-NOFILE-<tid> 逐秒升件
  （2026-11-06 18:15~18:16，T-0001…T-0295 共 23 票）——终态票
  tasks/done 双缺，票翻不可为；
- prefix 携票号=24h 去重键——每件双缺票日窗内只升一次，但 23 件全集
  =日噪声 ~23 行常驻升件面，真升件（同窗 REPO-DIRTY-INLOCK 等）被淹
  且矿工按签名聚类成案。

## 根因或策略解释

- dm_ledger A 面扫描对终态票 `task_file+done_file` 双缺即 esc+continue
  ——无补档径（票件史缺/migration 失落不补）亦无豁免名单，恒撞恒升；
- 双缺集是静态史账（T-0001 等票面生命周期早于文件台账制）——机械轨
  无「认不得的形不碰」同义豁免（rc2 判 A 只盖 status_done 不识形，
  未盖票件缺席形）；
- 噪声面结构性：升件逐票 prefix=去重不聚合——23 件同病灶在账面占 23
  行。

## 处置

- 双缺票补档或豁免除名二选一：补 stub 票件入 done/（台账全），或册面
  skip-list（LEDGER-NOFILE 已知集不升）——豁免件留「账不全」显位
  不静默；
- 升件聚合：同窗双缺集一行升件携票号全集（前缀 LEDGER-NOFILE-BATCH）
  ——去重键归一，日噪声 23→1；
- 判据层通则沿用 rc2 判 A 精神：「恒败不可为件」不属本轨翻面责任——
  不碰即不升，只留显位清单。

## 证据

- check: `SELECT id,ticket FROM ops_event WHERE note LIKE
  'LEDGER-NOFILE%'`（2026-11-06 拍窗）
- result: 23 行逐秒批量升件（#2885 T-0001 16:51 首发、#2977~#2998
  18:15~18:16 全集 23 件）——单窗实证批量噪声
- check: `ls docs/tasks/ docs/tasks/done/ | grep T-0001/T-0118`
- result: 双缺实证——票件两档皆无
- ref: fin_eng_lesson#9/#10（同簇两案并裁）

## 反证条件

- check: 双缺票补档/豁免落地后 LEDGER-NOFILE 升件归零
- falsifies_when: 若双缺集是活账缺口（新票仍缺件入终态）则病灶升级为
  票面写径缺陷——条目照改不退

## 适用边界

- 仅 dm_ledger A 面 LEDGER-NOFILE 位；LEDGER-DUP（票件双存）同位不同
  病灶另案；
- 豁免名单须显式册面——裸过滤=静默缺口违升件不静默律。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（候选=「LEDGER-NOFILE 日升件计数」：ops_event note
  LIKE 'LEDGER-NOFILE%' 24h 行数——meta 机械轨面无适配 gauges.tsv 表，
  待入表时随调度可靠性票补建）
- class: n/a（候选：淤积）
- query: n/a
- threshold: n/a
