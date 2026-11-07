---
schema_version: 2
id: dm-ledger-dirty-gate-untracked-ticket-stall
kind: pitfall
risk: high
status: verified
keywords: [dm_ledger, REPO-DIRTY-INLOCK, REPO-COMMIT-FAIL, REPO-PUSH-FAIL, 脏树闸, porcelain全量判脏, 钦定未跟踪票件, 票翻弃拍, 锁外写者, META-DM]
scope: [scripts/meta/dm_ledger.sh, META-DM 机械轨票翻/台账写径, /opt/o 部署树, docs/tasks]
summary: dm_ledger 锁内 `git status --porcelain` 全树判脏——钦定未跟踪票件+票面回填 WIP 属常态脏，闸不辨「钦定态」与「锁外写者」即升件 REPO-DIRTY-INLOCK 弃拍：票翻/对齐/翻面全跳；脏树面 push_needed 段照跑两试恒败——滞留 commit 断供自锁
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [决策席, 裁定席]
evidence_refs:
  - type: doc
    ref: scripts/meta/dm_ledger.sh（REPO-DIRTY-INLOCK/REPO-COMMIT-FAIL/REPO-PUSH-FAIL 三闸面）
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE id IN (2886,2999,3038,3041)——META-DM 12h 内 COMMIT-FAIL/PUSH-FAIL/物料化rc127/DIRTY-INLOCK 四连升件
    observed_at: 2026-11-07T00:00:00Z
  - type: server_fact
    source: postgres
    query: /opt/o git status 实证（2026-11-07）：HEAD 领先 origin/master 1 笔(cfeaef58 滞留)+T-0419/0420/0422/0425 票 ?? 未跟踪+票件回填 unstaged——钦定脏常态
    observed_at: 2026-11-07T00:00:00Z
  - type: doc
    ref: docs/工程经验库/条目/untracked-ticket-pre-push-number-gate.md（钦定未跟踪票态出处）
---

## 事实/症状

- 2026-11-06~07 META-DM 12h 内四连升件：
  REPO-COMMIT-FAIL（票翻 commit 败——件在工作树留待下拍，ops_event#2886）
  → REPO-PUSH-FAIL（推 origin/master 两试皆败——本地 commit 滞留，#2999）
  → 物料化径败 rc=127（ticketdraft 未跑净——pending 草案滞留，#3038）
  → REPO-DIRTY-INLOCK（持锁后部署树仍脏——锁外写者越闸疑案本拍不动收，#3041）；
- 裁定拍时实证：/opt/o HEAD 领先 origin/master 1 笔（cfeaef58 滞留未推）、
  T-0419/0420/0422/0425 票件 `??` 未跟踪、票件状态回填 unstaged——
  部署树正处「钦定脏」常态，锁内守卫所见即此面。

## 根因或策略解释

- 锁内守卫 `git status --porcelain`（全树无 pathspec）非空即判「锁外写者
  越闸」升件弃拍——但 untracked-ticket-pre-push-number-gate 钦定正形
  「票件留 /opt/o 未跟踪面、状态翻/结果回填直写票件本体」＝untracked+
  unstaged 脏是钦定常态，非越闸写者；
- 判脏粒度错配：判脏面（全树含 `??` 行与任意 modified）远大于翻面写面
  （docs/tasks 内 mv/翻行）——钦定脏恒命中守卫，票翻面在常态期结构性
  断供；
- 弃拍后果链：票翻/对齐/翻面全跳→完成票积压；锁段尾部 push_needed 照跑
  （在 else 外）但 check_all+git_push 在脏树/滞留面上两试恒败
  →REPO-PUSH-FAIL→本地 commit 滞留更久→树更脏——自锁环；
- 同窗 COMMIT-FAIL/物料化 rc=127 属同写径败面各点失守（闸门拦/工具链
  败），逐点根因未分案钉死——本条立的是「判脏粒度错配」结构性病灶。

## 处置

- 判脏面收窄：porcelain 限 docs/tasks 跟踪路径（`-- docs/tasks` 且滤 `??`
  行）或对 `?? docs/tasks/T-04xx-*.md`/票件 modified 白名单豁免——
  「钦定脏」与「锁外写」分类后才升件；
- 升件携 dirt 分类计数（tracked modified / untracked 票件分列）——
  定性权交席面，不再一行疑案；
- 滞留 commit 自愈径：push_needed 段两试败把 check_all rc 与 git_push rc
  分行落 detail——COMMIT-FAIL/PUSH-FAIL 各点败因可溯。

## 证据

- check: ops_event META-DM 升件面 12h 窗（fin_eng_lesson#6 evidence_ids）
- result: #2886/#2999/#3038/#3041 四连——COMMIT-FAIL/PUSH-FAIL/物料化
  rc=127/DIRTY-INLOCK
- check: `git -C /opt/o status --porcelain` + `git log origin/master..HEAD`
  （2026-11-07 裁定拍时）
- result: 票件 `??`×4+票件回填 modified×2+HEAD 滞留 cfeaef58×1——
  钦定脏全谱在场，与升件面互证
- ref: fin_eng_lesson#6、untracked-ticket-pre-push-number-gate（钦定脏出处）

## 反证条件

- check: 钦定未跟踪票件+回填 WIP 常态期 DIRTY-INLOCK 不再升件且票翻照跑
- falsifies_when: 判脏面收窄落地（pathspec/白名单）后守卫只在真锁外写时
  升件——彼时本坑失效，条目照退

## 适用边界

- 仅 META-DM 机械轨锁内判脏面（dm_ledger.sh TOKEN 段）；会话席人工写径
  不在此闸域；
- 同窗 COMMIT-FAIL/PUSH-FAIL/物料化 rc=127 逐点根因（pre-commit/pre-push
  闸拦、go run 工具链败）未分案钉死——复发时按上列自愈径取 detail 另案。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（候选=「滞留 commit 计数」：`git -C /opt/o log
  origin/master..HEAD --oneline | wc -l`>0 连拍——meta 机械轨面无适配
  gauges.tsv 表，待入表时随调度可靠性票补建）
- class: n/a（候选：淤积）
- query: n/a
- threshold: n/a
