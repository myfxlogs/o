# T-0374 O-Dev 施工提示词——完成票归档 done/ + 扫描面同步

你是[角色:施工]。领单授权=业主手动外派（票面领取行标
「业主手动外派」，审结归 COS 不归 DM——collab.md 写面
判据窄口豁免件）。

## 开工前必读（按序）

1. `/opt/o/AGENTS.md` + `docs/collab.md` + `skills/施工席/SKILL.md`
2. 票面全文（唯一权威范围——耦合坑节必读：平移不改
   闸=归档号逃逸唯一性破口）：
   `/opt/o/docs/tasks/T-0374-完成票归档-done子目录与扫描面同步.md`
3. 消费面先看：
   - `scripts/next_ticket.sh`——远端 docs/tasks/ max 扫描；
   - `scripts/check_ticket_numbers.sh`——三模查重闸；
   - `scripts/meta/dispatch_worker.sh:31`——TASK_DIR 单
     票文件径（搬动前核在飞零撞）。

## 要点

- `docs/tasks/done/` 新建，git mv 全部 状态=完成/作废 票
  入册（活跃态原地不动）；git log --follow 保史；
- 两扫描件号谱 glob 扩含 done/（或递归扫——二择一
  改动最小者），归档后唯一性闸覆盖数=总票号数；
- collab.md 查队列行+取号仲裁条补 done/ 注记；
- 归档后实证：查重覆盖全谱+next_ticket 返回值不变+
  顶层只现活跃票。

## 红线

- 票文件正文零动（纯位移）；号不重排不回收；
- 不动活跃票/派工语义/事件推导；
- 撞限流落 WIP+挂起自报再退；不合并不部署不改票面。

## 收口

新 worktree→分支 `t/0374-*`→check_all 绿戳对口 SHA→
report.sh 三件→§8 模板照填（**移交行写 COS 非决策席**）
→推送分支即交付。
