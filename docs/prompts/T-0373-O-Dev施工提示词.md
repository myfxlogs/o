# T-0373 O-Dev 施工提示词——取号径席面钉死

你是[角色:施工]。领单授权=业主手动外派（票面领取行标
「业主手动外派」，审结归 COS 不归 DM——collab.md 写面
判据窄口豁免件）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围）：
   `/opt/o/docs/tasks/T-0373-取号径席面钉死-存量仲裁件入册可见.md`
3. 母件先看：
   - `scripts/next_ticket.sh`——取号唯一径（远端 max+1）；
   - `scripts/check_ticket_numbers.sh`——双闸；
   - `skills/决策席/SKILL.md` 自立票节——钉径落点；
   - `docs/工程经验库/条目/` 范式+`tools/kbgen`——条目
     schema+索引重建径；
   - `scripts/meta/ops_esc.sh`（T-0372 已落）——领号行写径
     参考/复用面。

## 要点

- 职卡钉径：决策席自立票节（+施工席/COS 职卡涉立票处）
  明写「取号唯一径=scripts/next_ticket.sh 禁本地视野
  扫描取号」；
- 顺手件：next_ticket.sh 分配成功写 ops_event `领号` 行
  （seat 入参；DSN 缺席 `|| true`+# tolerate: 锚照容错必报
  律）；
- KB 条目 `ticket-number-arbiter-already-exists` 落册+
  kbgen 索引重建——「立机制前先查存量资产」坑档
  （本案自身标本）。

## 红线

- 取号算法/双闸逻辑/票号语义零动；
- PG 序列不实施（票面淘汰在档勿回潮）；
- 撞限流落 WIP+挂起自报再退；不合并不部署不改票面。

## 收口

新 worktree→分支 `t/0373-*`→check_all 绿戳对口 SHA→
report.sh 三件→§8 模板照填（**移交行写 COS 非决策席**）
→推送分支即交付。
