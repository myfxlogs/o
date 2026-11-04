# T-0322 施工提示词——COS 秘书席：升件队列压缩层 + o_fin_ro 窄读

> 依据链：票面 `/opt/o/docs/tasks/T-0322-业主面减负替代包-o_fin_ro窄读与COS秘书席与草案开放立票.md`（已批）> 本提示词 > collab/工程纪律。
> 决策钦定落形（一席两形态案同语义）：COS=**人边界压缩层**，非机器环
> 齿轮——盯「待业主裁决」集→一页打包上行→批回转席执行，**准备决策
> 不做决策**。勿自扩权。

## 工作树/分支

`/opt/o-wt/t0322` 分支 `t/0322-cos-pack`（领单四步序：report.sh 领单
→git pull→worktree add→provision_worktree.sh）。

## 三件落法（钦定形，勿改计）

### A. `o_fin_ro` 只读角色——`migrations/` 新迁移件

- `CREATE ROLE o_fin_ro` + `GRANT SELECT` 白名单：
  `fin_ledger_snapshot`/`fin_position`/`fin_strategy`/`fin_opp_eval`/
  `fin_opp_hypothesis`/`fin_opp_template`/`fin_eval_reject`/
  `fin_proposal`/`fin_capability_surface`/`ops_*` 全族；
- **结构性黑名单**：`app_secrets` 及凭据物料列**不授**（REVOKE/不
  GRANT——写进迁移注释：凭据域结构性隔离，非配置项）；
- NO 写权一切（INSERT/UPDATE/DELETE/TRUNCATE 一律不授）；
- 实测贴票：白名单 SELECT 通、黑名单 SELECT 拒、写操作拒——三行
  psql 实录进回填段；
- `docs/数据流.md` 凭据面同 commit 加 `o_fin_ro` 行（**数据流正本
  同 commit 更新是硬律**——漏了等于没做）。

### B. COS 职卡 + 拍壳——**beat 席非常驻**

> 定位钉死（业主令 2026-11-04）：**COS 非新席——O-DM 席即秘书席**，
> 本件造的是它的**无头形态**（同一席两形态模型：交互=O-DM pane
> 能讨论能派单，无头=cos_tick beat）。职卡写法照此：一席两形态
> 并书，勿写成独立新编制。

- `docs/员工名册.md` 职卡四件齐：
  - **职责**：盯三类收件箱——①ops_event `升件`行 ②条件请求件
    ③草案滞留超 24h（与新章「升件排队 COS 兜底」同构）；每件产
    「选项+建议+证据」一页打包件上行业主；
  - **产出**：打包件（ops_event `COS上行` 行+飞书通知径）；业主
    批复（O-DM/飞书回）→ COS 转对应席执行（写 `派单`/落票）；
  - **合格判据**：打包件含「业主裁」空位+证据锚全引实件非口报；
  - **越权线**：写面唯一=待业主裁决队列/通知径——动钱/审批/部署/
    代拍板一律越权（票面禁线原文照录）。
- `fin_employee_registry` 在册行（runtime 花名册——照既有席行对形）。
- **拍壳**（照 dm_tick.sh 对形）：`scripts/meta/cos_tick.sh`+
  `deploy/o-cos-seat.service`（oneshot）——闸=收件箱三类非空才烧，
  空拍 `hb alive`（ops_headless ticket=`META-COS`）+exit；拍=
  `devin -p --model <swe-2-high|swe-2-medium> --respect-workspace-trust
  false --permission-mode dangerous -- "读 /opt/o/skills/COS/
  SKILL.md 打包协议 拍待业主裁决件"`。
- `skills/COS/SKILL.md` 新职卡（跑在 COS 席上的协议）：收件箱
  三源SQL钦定口径（最新刻集式——照 T-0315② 判法）+打包件模板
  （选项|建议|证据锚|业主裁空位）+写面窄白名单（`COS上行`/`派单`
  ops_event行+通知径；禁钱径/审批/章面）+上行=ops_event行+飞书
  （report.sh 同款三通径可复用组件）。

### C. collab.md §2 草案开放立票（文档面）

- §2 任务单章加一段：「**草案(draft)任角色可立**——提案入口全开放；
  `草案→已批` 批准闸仍归决策席交互形态独占。票面『立项』字段记实
  际立票席+业主令出处」——与 §1 两形态写权矩阵衔句（禁冲突）。

## 验收锚（票面外追加钦定锚）

- [ ] A 三行 psql 实录（白名单通/黑名单拒/写拒）——非口报。
- [ ] B 职卡四件齐+registry 行+cos_tick 空拍绿+**一次真打包包**
  （现成真实条目即可——决策席升件行/滞留草案任选喂它）。
- [ ] C collab §2 段落落字与 §1 无冲突（自查衔句）。
- [ ] 数据流.md 凭据面+席面同 commit。

## 停线

- o_fin_ro 要授权的表不在白名单→**停**——回票挂起勿自扩名单；
- cos_tick 拍壳非 oneshot/缺心跳/收件箱口径自造→停——照 dm_tick
  逐字对形是唯一合法形；
- 碰见要「代业主拍板」的写法→停——那是禁区不是功能。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | B 件拍壳+cos_tick 属新席实体化——票面只写了职卡 | 范围扩张风险 | 钦定落形写明（beat席非常驻——决策拍板含在提示词里，施工勿再裁形态） |
| 2 | 业主批复→转执行席的回收径未定义 | 缺口 | 钦定：O-DM pane/飞书回→COS 拍读 ops_event+收件箱，批复落 `业主批` 行即触发下一拍 |
| 3 | `派单` 事件语义两席都可能写 | 写面撞车 | COS 写面白名单含 `派单` 行仅限「业主批回转执行」语境——与决策席常径派单语源不同，行内注明 `cos-relay` 标 |
