# T-0354 DM 班次制——事件 kick + drain 自退

- 状态：待验
- 领取：claude-glm/O-Dev @ 2026-11-05（业主令外派，T-0353 同型——票面随支自
  t/cos-duty@5cc592c6 cherry-pick 带入）
- 级别：P2 工程效率——事件成簇时重复付全量冷启税（注入+上下文/次），
  且 30min 兜底拍是唯一唤醒径（移交后最长呆 30min）

## 病灶（实查在案，勿再诊）

- `dm_tick.sh` 每有活拍拉起独立会话：注入角色卡+读 collab+读票+扫收件箱
  ≈全量上下文税/次；事件成簇日（2026-11-05 实测 T-0350/351/352/353
  移交零散到达）同上午重复付 N 次；
- 空拍已零 LLM（T-0343 判据闸）——常驻救不了空拍，要省的是
  **在途批次的重复冷启**；
- 队列=`ops_event` 派生账（SG fin PG，行只增不删）——票丢不了，
  kick 语义=门铃非事实源，kick 哑兜底拍照收（队列推导逻辑
  `scripts/meta/dm_tick.sh:28-60` 在案）。

## 任务（钦定修法——班次制语义）

**事件落行 → kick dm 会话 → drain 到队列空 → 自退。**
忙时收敛成常驻（队列不空会话不退），闲时收敛成零成本（无会话可养）。

### 件①kick 径（权威源=ops_event 表，非 report.sh 旁报）

- ops_event insert 白名单事件（`移交`/`收口`/`收口待审`/`升件`）
  → 拉起 dm 会话；
- 实现面施工可裁最简可靠件（建议：PG trigger→NOTIFY→轻 listener→
  systemd-run；listener 应 bash 级小件或复用既有 daemon——禁新造重栈）；
- kick 失败不丢票：行在账在，兜底拍重推——kick 是纯加速件。

### 件②drain 循环（班次核心）

DM 会话 runbook 改：scan 待裁/待派集→逐件裁→rescan→**队列空才退**；
会话期内新到移交被自然吸收=批次冷启一次。

### 件③单例闸（防双开，T-0343 双工事故同款防）

会话起先抢锁（PG advisory lock 或 ops_headless 席态行二选一，施工按
现构贴）——kick 连发/与兜底拍撞车=后到者即退。

### 件④重裁幂等

会话内每票裁前先查「已合/已裁」态——崩后重进不重复 merge/flip/push。

### 件⑤兜底不退化

30min 定时拍保留原样——kick 哑=退化成现状非更差。

## 验收锚

1. 合成 `移交` 行→dm 会话秒级拉起→裁决→队列空自退（state=exit）；
2. kick 连发二次实证不双开（锁拒实证）；
3. listener 杀死后兜底拍仍收单（kick 非唯一径反证）；
4. ≥2 件移交同会话内连裁=冷启一次（ops_ticket_usage 或会话日志实证）；
5. 空队列 kick 零触发（白名单外事件不起会话）；
6. check_all 绿戳对口提交 SHA。

## 红线

- 不改裁决语义/票态机/ops_event 写入面（kick 只读行不发行）；
- 不新造常驻重栈——listener 件量级钉死；
- 兜底拍语义不动；report.sh 里程碑词面不动。

## 结果回填（O-Dev-t0354 @ 2026-11-05）

### 实现（前 run 半成品续作完成——接续不推倒）

- **件①kick**：复用既有总线链零新栈——`cmd/eventbus/main.go`「d」源
  水印 SQL 白名单补 `'升件'`（移交/收口/收口待审/升件四件全）；
  eventbus 3s 轮→metabus kind=dm→`systemctl --user start --no-block
  o-decision-seat`→dm_tick.sh（T-0312 既有径，本票只扩白名单）。
- **件②drain**：`skills/决策席/SKILL.md` 新增「班次节」——班次=本会话
  一次存续，scan→裁→rescan→队列空才退，会话期新到移交自然吸收；
  dm_tick.sh 拉起 prompt 同步指班次节+重裁幂等；队列推导单源
  `scripts/meta/dm_queue.sh`（新件，拍壳闸与班内 rescan 同式防漂）；
  班后队列空→ops_headless `state=exit`（fin_0093 六态正常终态）。
- **件③单例闸**：PG advisory lock（int8 键 42540354）——holder=psql
  挂 FIFO：fd8=O_RDWR 自持写端，fd9=O_RDONLY 作 psql stdin（续作期
  自审根修：原稿 psql stdin 复用 O_RDWR fd=自持写端，脚本被杀后
  孤儿 psql 永不见 EOF=锁永泄全席死——改只读面后写端只归 fd8，
  全部持有者亡即 EOF→psql 退→锁自放；SIGKILL 实证见下）。
- **件④幂等**：SKILL 班次节「重裁幂等」条——每票动手前查最新
  ops_event 已含裁决/完成/驳回或收口支已合 master 即跳。
- **件⑤兜底**：`deploy/o-decision-seat.timer` 30min 拍原样未动。

### 验收锚对账

1. **部分 PASS+部署后验**——kick 链结构实证：eventbus「d」源 SQL 现含
   升件（白名单四件全）；dm_tick 全程干跑（真 PG+桩件）：锁空闲 0.575s
   跑完 exit0、空队列 SKIP_DECISION 零 devin 调用（devin.log 缺席实证）、
   drain 场景班后队列空→hb `state=exit` 行落账（stub 测试 PASS）。
   「合成移交行→秒级拉起」生产端到端=部署后验项（kick 链 T-0312 既有
   径，eventbus/metabus/systemd 三段均在产）。
2. **PASS**——锁拒实证（真 PG）：外部持 `pg_advisory_lock(42540354)`
   时 dm_tick 0.5s 内 `[dm] 班次闸未得` exit0；stub 场景 locked 同断言
   （零决策零派单零心跳）PASS。
3. **PASS（结构+反证）**——kick=纯加速件：listener/总线哑=行在账在
   兜底拍照收（timer 件未动=语义不退化）；队列推导真 PG 实证：
   T-9Q07 仅升件行票→队列外（升件唤醒但 ticket='' 空挂不入队列，
   拍壳闸自挡零 LLM）。
4. **PASS（机制）+部署后验**——drain 语义=会话内 rescan 同式
   （dm_queue.sh 单源，班内复查与拍壳闸同 SQL）；「≥2 件同会话连裁
   冷启一次」生产 token 实证=部署后验项（ops_ticket_usage META-DM
   batch=tickets=A,B 行）。
5. **PASS**——空队列 kick 零触发：真 PG 合成行集实证（T-9Q01 移交在队/
   T-9Q03 已批在队/T-9Q05 收口在队；T-9Q07 升件/T-9Q08 已裁/T-9Q04
   派单毕/T-9Q02 裁决孤行/T-9Q06 完成孤行全在队外）；非白名单事件
   不推「d」源水位（SQL 过滤 event IN 白名单）。
6. 见机检行（绿戳=收口 commit SHA）。

### 偏离说明（供裁，未扩单）

1. **件③选型**：票面给「PG advisory lock 或 ops_headless 席态行二选一」
   ——选 advisory lock（锁语义=进程级互斥比行态更贴「后到即退」；
   holder 生死=连接生死免租约心跳）。续作期自审根修 FIFO fd 模态
   （上详）——原稿缺陷会致锁永泄，已修并 SIGKILL 实证。
2. **件①零新件**：票面建议「PG trigger→NOTIFY→listener→systemd-run」
   为可选实现面；续作发现 T-0312 总线链已覆盖 kick 全径（eventbus
   水位轮+metabus 唤醒映射+systemd start），只需白名单补升件——
   零新 listener/daemon，最贴「禁新造重栈」红线。
3. **测试桩同步改造**：`dm_tick_usage_test.sh`/`dm_tick_worker_resume_
   test.sh` 按新架构更新（锁桩+dm_queue 单源 SQL+locked/drain 新场景
   ——锁拒/收班 exit 两断言为本票新增覆盖）；两测干跑全 PASS。
4. **数据流.md 同 commit 更新**（修订义务——kick 链+班次节图面补段）。
