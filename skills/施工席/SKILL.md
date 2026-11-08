---
name: 施工席
description: 无头施工席拉起式 run 职卡（领单四步/施工三禁/收口三步同拍/停挂自报）；瞬态 systemd-run 载 o-worker-<tid>（T-0324）
domain: 工程
---

# 施工席 SKILL——无头施工协议（拉起式瞬态 run，o-worker-<tid> 载）

## 领单四步序
1. `report.sh 领单 T-NNNN`（自报落账=领据）；
2. `cd /opt/o && git pull`；
3. `git worktree add /opt/o-wt/tNNNN -b t/NNNN-<slug>`；
4. `cd /opt/o-wt/tNNNN && bash /opt/o/scripts/bootstrap_worktree.sh`——
   **新 wt 先 bootstrap**（T-0375 件②）：buf 正径再生 gen（mtapi 模板
   裸 wt 即产；全谱 buf 须先 `npm ci`），buf 缺席/面缺降级 rsync 部署
   树快照（明示非静默）；dist=快照仅供闸/dev。

## 领单即召回（提案-KB向量检索-经验库召回面 首阶）

领单后在本树跑 `bash scripts/meta/kb_recall.sh docs/tasks/T-NNNN-*.md`
（worktree 优先——票面以本枝版为准）——输出 top-K 经验库条目摘要
（eng+econ 双库机械匹配），对号读 `docs/*/经验库/条目/` 正文；
召回件/物料缺席=零输出照跑（fail-open 增强非闸件）。无头派单
提示词已由 dispatch_worker.sh 注入同段——本步=人工位/会话席
同径补齐，别跳过。

## 施工
- 票面自含读单勿再诊；diff ⊆ 票面涉及面；
- **三禁**：红线三件（资金扩投/新平台/席位立换）、凭据域（app_secret/
  env 件写径）、改票本体；
- 自测 `go build+vet+test+check_migrations` 全绿才收口。
- **迭代戳跑径（OCI 档A，提案-check_all甩OCI 2026-11-09 活验在案）**：
  返修迭代期 `scripts/oci_ci_push.sh <分支>`——推 OCI scratch→ARM runner
  scoped 跑（活测 5m30s/30包 vs 本盒全谱 35min 量级），回报落
  `.git/oci-report-<sha>`，`scripts/oci_ci_accept.sh <sha>` 核签+
  bridge-audit 延后闸本盒补跑绿→check-green 戳；迭代红在 ARM 烧不烧
  本盒墙钟。fail-open：OCI 缺席/超时/畸形=本盒径不变不炸。
  **收口终盖恒本盒**——rebase 后 `scripts/check_all.sh` 盖戳语义不变，
  master/merge sha 不走 OCI。

## 收口三步同拍
0. **收口前置（GW-001 E2 戳复用轨）**：`git fetch origin && git rebase
   master` 先归并到**集成顶**（本地 master 常领先 origin/master——
   ff 判定对的是将合目标非远端名；枝落后则 rebase 变 SHA），再跑
   `scripts/check_all.sh` 盖戳——戳绑 rebase 后 HEAD，外审席 ff-only
   合入时戳复用零复跑；rebase/补丁在戳后再动=戳作废须重盖；
1. 票面回填翻待验；
2. `git commit -o -- <路径>`（共享 index 坑在案——禁裸 -A 跨 worktree）；
3. `report.sh 收口`+`report.sh 移交` **必调**——漏调断链案 T-0317 在案
   （「收口」字样 commit 另有 post-commit 钩兜底）。

## 停与挂
- 三轮自试不过/外因阻塞→`report.sh 挂起 <备注一句话定性>`；
- 熔断级→`report.sh 异常`；
- 挂起备注鼓励结构化 `await=<档>:<参>`（绿窗/凭据/外因/裁量四档——
  绿窗档=`await=绿窗:<[pkg:]>TestName`，DM 看门狗机械复跑该测试已绿
  即续跑派工，T-0365；无 await/非绿窗档超 4h 未消自动升件呈业主）。

## 域界
- 瞬态 run 只做一票（票域独占天然隔离）；O-Dev pane 人工位不属本卡。

## 退休纪律（T-0331 钦定——坟场防复发）

- 票翻完成→顺手 `git worktree remove <树>` + `git branch -d <枝>`
  （合入即删——未合枝保留记档，-d 天然拒删未合）；
- GC 兜底：`scripts/meta/wt_gc.sh [--dry-run]`——unit 活+树龄双闸
  禁碰活 run 工树；清场数 ops_event `GC` 行录。
