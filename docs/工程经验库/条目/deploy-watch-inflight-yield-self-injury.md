---
schema_version: 2
id: deploy-watch-inflight-yield-self-injury
kind: pitfall
risk: medium
status: verified
keywords: [deploy_watch, 部署守望, 在途让位, 竞锁, deploy_blocked, deploy_push_blocked, 首试rc1, 自伤升件, mirror_push, 绿戳, 单写者]
scope: [deploy_watch.sh 自醒守望, deploy_image.sh 锁闸, mirror_push 绿戳推, 部署类升件分诊]
summary: 部署守望自醒拍与人工/镜面径竞态=确定性自伤升件：会话部署持 .deploy-lock 时 watch 同发 deploy_image 撞锁 rc=1（DEPLOY 账实证每次部署首试 rc=1→重试 rc=0 三序列同型）；merge 未盖戳窗盲推撞 pre-push 闸=deploy_push_blocked；本地 merge 已合而 origin 未推窗抢跑=「HEAD 不在 origin/master」rc=1。修法=在途让位三态：锁持跳部署/尖未盖戳跳推送/推窗让位镜面——真实滞差照升不误发。
first_seen: 2026-11-10
last_verified: 2026-11-11
skills: []
evidence_refs:
  - type: doc
    ref: scripts/deploy_watch.sh（让位三段:锁持/尖未盖戳/推窗）
  - type: doc
    ref: scripts/mirror_push.sh（绿戳 sha 三断言:龄/祖先/HEAD包含）
  - type: server_fact
    source: postgres
    query: SELECT id,note FROM ops_event WHERE esc_scope IN ('deploy_blocked','deploy_push_blocked') ORDER BY id DESC LIMIT 10
    observed_at: 2026-11-11T00:00:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

deploy_blocked/deploy_push_blocked 升件销后三轮复燃（#9600→#9689、
#9505→#9687）。DEPLOY 账取证：sha 45c2f5524507/bacbb5865904/
e2997d3f8e42 三序列全为「首试 rc=1→重试 rc=0」同型——非随机
瞬态是结构性首试败。

根因三态皆竞态自伤：
1. **锁竞**——会话 deploy_image 持 `/opt/o/.deploy-lock` 时 watch
   自醒同发 deploy，`mkdir` 锁撞 rc=1→升件。
2. **戳竞**——merge 已合而 check-green 戳未落窗内，watch 推
   master 撞 pre-push 闸（未绿禁推）rc=1→push_blocked 升件。
3. **推窗竞**——本地 merge HEAD 已合而 mirror_push 未推窗内，
   watch 抢跑 deploy 撞「HEAD 不在 origin/master」rc=1→升件
   （升件#9811 实证）。

## 修法（在途让位——让非压）

- 部署锁持=另径部署在途→本拍 exit 0 让位（不部署不升件）；
- LOCAL 无 check-green 戳=合流在途→让位盖戳流（不推不升）；
- LOCAL 领先 origin/master 无戳=推窗→让位镜面推；
- 真实滞差/真推败（上述皆不命中）→照升不误发。

## 判例边界

- 竞态类升件看 DEPLOY 账 rc 序列同型性——「首试 rc=1 重试
  rc=0」重复出现=结构性病灶非瞬态，禁按瞬态销。
- 让位语义是「本拍不动」非「永不报」——条件消即正常径恢复，
  与「静默抑报」分界在 escEdgeU 沿：真病灶滞留超窗照升。
- 同类竞态普适：任何「守望拍×人工径×镜面径」三写者面，
  守望必须让位在途——竞锁不是错，撞锁报警才是错。
