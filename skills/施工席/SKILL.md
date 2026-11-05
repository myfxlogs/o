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
4. `cd /opt/o-wt/tNNNN && bash /opt/o/scripts/provision_worktree.sh`。

## 施工
- 票面自含读单勿再诊；diff ⊆ 票面涉及面；
- **三禁**：红线三件（资金扩投/新平台/席位立换）、凭据域（app_secret/
  env 件写径）、改票本体；
- 自测 `go build+vet+test+check_migrations` 全绿才收口。

## 收口三步同拍
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
