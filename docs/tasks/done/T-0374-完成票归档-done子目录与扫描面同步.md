# T-0374 完成票归档——done/ 子目录 + 取号/唯一性扫描面同步

- 状态：已批（2026-11-05 COS 转录业主令——「这正是代码，目录
  整洁的范围」——票号不重排只归档，扫描面必须同步防
  唯一性破口）
- 领取：业主手动外派 @ 2026-11-05（外部工人 run——提示词 docs/prompts/T-0374-O-Dev施工提示词.md，DM 勿自动领）
- 级别：P3 目录整洁——含功能耦合件（扫描面同步），非纯搬动

## 事实

- docs/tasks/ 平铺 360+ 票文件，活跃票（已批/施工中/待验）
  难捞——业主令整洁归档；
- **耦合坑（COS 勘查在案）**：三消费面以 `docs/tasks/T-*`
  glob 取号谱——
  - `scripts/next_ticket.sh`：远端 docs/tasks/ 最大号扫描；
  - `scripts/check_ticket_numbers.sh`：唯一性闸（存量查重
    /staged/remote 三模）；
  - `scripts/meta/dispatch_worker.sh`：`TASK_DIR=/opt/o/
    docs/tasks`（单票文件径——已派票在飞面，搬动须错峰
    或确认零在飞）；
  → 平移到 `docs/tasks/done/` 而不改消费面=两闸看不见
    归档号，**新票可复用旧号=唯一性破口**；
- collab.md 查队列行 `grep -l '状态.*已批' docs/tasks/T-*.md`
  不递归——done/ 天然不进队列扫，整洁目标天然达成。

## 任务

1. `docs/tasks/done/` 新建，`git mv` 全部 状态=完成/作废 票
   入册（活跃态原地不动——已批/施工中/待验/挂起全留）；
2. 扫描面同步：`next_ticket.sh` + `check_ticket_numbers.sh`
   取号谱 glob 扩含 `docs/tasks/done/`（或改递归扫——二
   择一票面定，以改动最小者）；`dispatch_worker.sh`
   TASK_DIR 单文件径确认在飞零搬动（施工前核）；
3. collab.md 同步：查队列行+取号仲裁条补「done/ 归档位」
   注记；
4. 归档后实证：check_ticket_numbers 无参存量扫覆盖数=
   全部票号（含 done/）；next_ticket 返回 max+1 与归档
   前一致；队列查只现活跃票。

## 验收标准

- [ ] 完成/作废票全部入 done/（git mv 保史——git log
      --follow 可追）；
- [ ] 两扫描件号谱覆盖含 done/（实证：归档后返回号不变
      +查重覆盖数=总票号数）；
- [ ] 队列查/docs/tasks 顶层只现活跃票；
- [ ] check_all 绿戳对口提交 SHA（含 tolerate-report lint）。

## 范围外

- 票号不重排不回收不改内容；
- 票文件正文零动（纯位移）；
- 不动活跃票/dispatch 语义/事件推导。

## 参考

- 扫描件：`scripts/next_ticket.sh`、`scripts/check_ticket_numbers.sh`、
  `scripts/meta/dispatch_worker.sh:31`；
- 仲裁条：`docs/collab.md` §2（取号仲裁+查队列行）；
- 唯一性破口防判：T-0373 档（号=身份非序号——归档≠复用）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
