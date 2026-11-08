---
schema_version: 2
id: deploy-stamp-sha-not-head-treadmill
kind: pitfall
risk: high
status: verified
keywords: [绿戳, pre-push, stamp_ok, HEAD, 推已绿sha, 跑步机, mirror_push, deploy_at_sha, 部署收敛]
scope: [scripts/mirror_push.sh, scripts/deploy_at_sha.sh, scripts/git_push.sh, pre-push 闸, 部署窗]
summary: 绿戳验的是「被测 sha 的树」——推/发必须按戳 sha 不按 HEAD；HEAD 是分钟级移动靶（check_all 20-40min 期间 commit 照落），按 HEAD 取戳=永续跑步机（当日废戳 4 次实证）；正解=镜面发窗：取最新鲜可推绿戳 sha 推+按该 sha 部署，跑期新 commit 归下趟不冻不废
first_seen: 2026-11-08
last_verified: 2026-11-08
skills: [施工席]
evidence_refs:
  - type: doc
    ref: scripts/hooks/pre-push（stamp_ok "$local_sha"——戳验本就绑被推 sha 非 HEAD，机制原生支持推旧绿戳）
  - type: doc
    ref: scripts/mirror_push.sh（三断言选择口径：戳龄≤6h∧未合入∧origin/master 为祖先=ff可推；全灭=镜面空窗合法零动作）
  - type: doc
    ref: scripts/deploy_at_sha.sh（临时 worktree 检出目标 sha→bootstrap→该 sha 自带 deploy_image.sh——部署物=该 sha 树逐字节）
---

## 病灶

「随到随推」形制下：check_all 绿戳落在跑测时的 HEAD sha 上，
而 pre-push `git_push.sh` 缺省 `HEAD:master`——推送要求
**当前 HEAD** 持戳。gate 跑 20-40min 期间新 commit 分钟级落地
（docs 票面回填/他席 merge），HEAD 一动旧戳即对新 HEAD 无效
→ 废戳重跑 → 再跑期间又有新 commit → **数学死局：服务
（check_all 时长）< 到达（commit 频率）时队列永胀**。
2026-11-08 全天实证废戳 4 次；全无头形态下靠人肉等安静窗侥幸。

## 根因或策略解释

戳语义本是「该 sha 的树测过」——sha 是内容寻址的静态点，天然
不怕 HEAD 移动；病灶不在戳在**取件口径**：推送/部署面按
「当前 HEAD」这个移动靶取件，而非按「戳所指的那个 sha」取件。
pre-push `stamp_ok` 参数本就是被推 local_sha——机制从未要求
HEAD，只是调用侧 `HEAD:master` refspec 把靶钉死在了 HEAD 上。

## 处置（T-0467 钦定三面）

- **件①推已绿 sha**：`scripts/git_push.sh origin <stamped_sha>:master`
  ——戳验绑被推 sha，HEAD 移动不废戳；跑期新 commit 归下趟。
- **件②镜面发窗**：`scripts/mirror_push.sh` 周期拍（deploy/
  o-mirror-push.timer 15min）扫戳面 mtime 新→旧，首个
  「戳龄≤6h ∧ 未被 origin/master 含 ∧ origin/master 是其祖先
  （ff 可推）」者推之；全灭=空窗合法。**包非替**——复用
  git_push.sh 重试包装，master_push_converge.sh 合流闸原位。
- **件②部署半**：`scripts/deploy_at_sha.sh <sha>`——绿戳在案
  ∧已并 origin/master 双断言→临时 worktree 检出该 sha→
  bootstrap→跑该 sha 自带 deploy_image.sh（出处闸/部署锁/
  SHA tag/健康闸全继承；工具链亦取该 sha 版）。
- **件③部署树 commit 时闸**：`scripts/check_deploy_commit_seat.sh`
  由 commit-msg 调（subject 词形 pre-commit 物理不可见——
  git hook 序决定，commit-msg=commit 时链最后可见点，拒=
  中止=进不了历史语义等价）；判别域=[角色:施工] 前缀负向列
  （T-0370 语义一字不动；无标票立/他席标直落=既有合法径不收窄，
  origin/master first-parent 500 件实证）。

## 证据

- check: t0467_deploy_converge_test.sh 13 场景全绿——A 组镜面
  选择（最新戳中选/戳陈落次/全合入空窗/分叉跳过/支名无关）、
  B 组 pre-push 戳绑被推 sha（戳在 P1 HEAD 走至 P2 推 P1 过闸
  /无戳拒）、C 组部署树闸（施工拒/他标放/无标放/非 master 直通/
  非部署树直通/MERGE_HEAD 豁免）
- result: T-0467 收口 @分支 t/0467-deploy-converge——三面落地，
  全 check_all/既有三闸语义不破，新径弃置可回旧径（mirror_push
  零调用=旧径原样）

## 反证条件

- check: 若未来 check_all 时长 << commit 间隔（跑步机条件消失），
  镜面制仍成立但非必需——镜像窗语义不退化
- falsifies_when: pre-push 戳验改绑「工作树净+HEAD 戳鲜」联合
  判据后本条目需重审（戳绑 sha 语义是本修的地基）

## 适用边界

- 只规「推已绿 sha」与「部署树 commit 时闸」；merge 合流语义、
  check_all 档制、deploy_window_due.sh 两面判据均不动；
  deploy_at_sha 的 bootstrap 依赖 /opt/o 快照径（rsync gen/dist
  降级面与 bootstrap_worktree.sh 同）。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a（流程形制坑——「按 HEAD 取件」是口径错误非
  量化阈值病灶；跑步机数学=服务<到达排队论，非 gauge 域）
- class: n/a
- query: n/a
- threshold: n/a
