---
schema_version: 2
id: checkall-stamp-cfiles-race-midrun-tree-mutation
kind: pitfall
risk: medium
status: observed
keywords: [check_all, 绿戳, HEAD 竞态, 合并潮, 跑步机死局, MODE误判, 幻影红归因, repolock, 戳绑SHA]
scope: [scripts/check_all.sh, 交付流, pre-push 闸, 部署树写径]
summary: check_all 的 STAMP 名绑「起跑时 HEAD」而 CFILES/测面采样在后——合并潮中另一席中途合 commit 改树，脚本继续跑却测「非戳 sha 之树」：docs-only 票翻件被误判 full 跑全谱+go test 红在他人新码（幻影红归因错位）；check-all-stamp-binds-head-sha 的对偶面，T-0467 跑步机死局机制层实证
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [外审席]
evidence_refs:
  - type: commit
    ref: 外审 T-0460 裁——票翻件 4e8162224(docs-only)check_all 起跑后 CTO 持锁合 t/0441+t/0463 三 commit 改树，CFILES 采样含其码面→MODE=full；go test 红 o/internal/platform/notify(TestRegimePush/TestVenueReqPush 文案断言)=T-0441 合并树红非本件红
  - type: doc
    ref: scripts/check_all.sh——STAMP=$(git rev-parse HEAD) 在脚本前段、CFILES=git diff merge-base...HEAD 与 go test 在后段，两处 HEAD 采样非原子
  - type: doc
    ref: docs/tasks/T-0467-部署收敛面根修票立.md——「跑步机死局/推已绿sha非HEAD」钦定修票（本条目=其机制层旁证）
---

## 事实/症状

多席并发期 /opt/o 部署树在 check_all 跑中（30-40min 全谱窗）被另一席
合法合 commit（repolock 陈旧接管径），HEAD 前移 →

1. **戳名锚错位**：`.git/check-green-<STAMP时HEAD>` 落名是起跑 sha，
   但脚本后段 `git diff merge-base...HEAD`/`git status --porcelain`
   采的已是新 HEAD——本实证 docs-only 票翻件（1 个 .md）因 CFILES
   含他人合并码面 → MODE=full 白跑全谱；
2. **测面漂移**：go test 测的是**文件系统当前树**（合并后新树），
   非戳 sha 之树——他人新码的红被记到本件跑上（幻影红归因错位）；
3. 反向面同理可幻影绿（更险——他人坏码盖本件名戳）。

条目 check-all-stamp-binds-head-sha 治「先跑后 commit」同席时序；
本条治「跑中他席改树」跨席竞态——跑步机死局（T-0467）机制层。

## 根因或策略解释

check_all 非事务脚本：HEAD 采样三处（STAMP 名/MODE-CFILES/go test
文件面）各自其时——git 非快照隔离，长跑窗内写径合法前移=读面漂移。
repolock 护的是写序互斥不护读窗冻结；T-0467 钦定根修（推已绿sha+
commit时闸）正对此型。

## 处置

- 判红先问「跑中 HEAD 动否」：`git log --oneline <戳sha>..HEAD` 非空+
  红在他人范围文件=漂移幻影红候选，勿径归因戳 sha 内容；
- 复审/审裁读戳面：戳 sha 只证「该跑起跑时 HEAD」非「实测树」——
  合并潮期戳词须配 HEAD 漂移对拍；
- 根治归 T-0467（commit 时闸/推已绿sha径）——本条目只立判词防误裁。

## 证据

- check: `git log --first-parent <stamp-sha>..HEAD` 跑后对拍
- result: 2026-11-08 实证——4e8162224 戳跑中 +3 commit(422c09f5f/
  811d0efeb/60a3dbbad)，notify 两测红=T-0441 合并树红
- ref: relay 14:47 CTO 持锁合并行（repolock holder=CTO-T0441-T0463）

## 反证条件

- check: T-0467 落地后同场景复跑
- falsifies_when: commit 时闸生效后跑中 HEAD 不再漂移仍出同型红

## 适用边界

仅 /opt/o 部署树共享写径多席并发窗；单席/无并发窗不适用。
「幻影绿」反向面未被实证——发生时同判词处理。
