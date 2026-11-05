---
schema_version: 2
id: cos-inventory-before-asserting
kind: pitfall
risk: high
status: verified
keywords: [COS, 盘家仓, 断言前先查, 存量资产, 内部机制, 外部参照]
scope: [COS 席发言纪律, internal/ 实码, fin_* 实账, 机制存在性判定]
summary: COS 两次同型病灶——对外下「缺/没有/需要引进」断言前不先盘自家仓：①不知 next_ticket.sh 在案拟立 PG 序列（重复机制险造）；②论 Vibe-Trading 影子账户时称 O 缺中间态，实 fin_strategy paper/probe/live 阶梯全史跑过 2068 提案。修法=断言机制存在性前必查 internal/ 实码+生产实账双源
first_seen: 2026-11-05
last_seen: 2026-11-05
evidence_refs:
  - type: doc
    ref: docs/工程经验库/条目/ticket-number-arbiter-already-exists.md
  - type: doc
    ref: internal/portfolio/lifecycle/paper.go
  - type: doc
    ref: internal/portfolio/lifecycle/machine.go
---

## 病灶

COS 席对「公司内有什么机制」的覆盖不全，连续两次同型发作：

**发作①（T-0373 前科）**：业主问票号防撞，COS 不查仓即答「立
ops_event 领号行/PG 序列仲裁」——实则 T-0163 双闸（pre-commit
`--staged`+pre-push `--remote`）与 T-0219 `scripts/next_ticket.sh`
（fetch 后扫 origin/master max+1）早已在案且实证根治三次撞号。
险造重复轮子，靠施工前复核才改判。

**发作②（Vibe-Trading 对照评估）**：业主问外部仓可引进否，COS 答
「影子账户值得抄——O 的策略评估缺中间态」。实查 internal/
`portfolio/lifecycle`：`fin_strategy` 迁移闭集
`proposed→paper→probe→live` 一直在跑——SG 实账全史 2068 提案经
paper（42% 洗出率）、1208 晋 live、paper→live 越级禁绝+委员会
live_switch fail-closed。O 的影子件不但在，且比外部件更硬。

## 共性根因

对外部参照物（GitHub 仓/新机制点子）兴趣先行，**断言「O 缺 X」
时凭记忆下结论，未先做最低成本的两源核查**：
- 实码源：`grep -rn <机制名/领域词> internal/ cmd/`（机制是否在案）
- 实账源：生产库实表（机制是否在跑、跑多深）——本地 DSN 权限
  不够时走 `ssh root@SG docker exec o-postgres psql -U o -d o`

## 处置

COS 席断言纪律（生效即日）：

| 断言类型 | 前置义务 |
|---|---|
| 「O 缺 X / O 没有 X」 | 必查 internal/ 实码+fin_*/md_* 实账双源；查不到才准说缺 |
| 「该引进外部 X」 | 先列 O 同域存量件对拍；增量为零明说零 |
| 「该立新机制 X」 | 先查 scripts/+docs/tasks/ 历史票同域件（T-0373 法） |

外部参照评估范式：先答「它是什么」→「与我们同域的存量件是
什么」→「增量真在哪」。缺第二格=结论不可信。

## 回改点

- 业主两次亲验（11-05 同日两发）——「COS 对公司里有哪些部门
  机制了解得不够全面」当面指出。
- 二次发作后业主令「先落条目再回来讨论」——本条即立案。
