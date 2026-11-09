# T-0507 CEO 自运感知·票A——层扫窗（mech_tick 新窗+五片不变量集）

- 状态：已批（CEO 管理令放行 2026-11-09——提案-CEO自运感知三缺
  分票 ABC 序之首；业主令「做上去」锚在案）
- 级别：P2 机械轨感知面——提案档：`docs/infra/提案-CEO自运感知三缺-层扫窗诊断径动作包线.md`

## 范围

1. `scripts/meta/mech_tick.sh`——新「层扫」窗（节拍同现窗，
   零新守护件零新 unit）；
2. `scripts/meta/layer_scan.sh`（或 mech_tick 内函数族——施工
   裁形）——五片不变量机械扫：

   a. **队列一致**：dispatched 龄>30min 无拉起∧无在飞 unit；
      pulled 无存活 o-worker unit；review/approval 老化
      （龄>2h 初值）；draft 滞留>1h；
   b. **相位↔实体**：相位非终而票件无（幽灵行）；票件
      done/ 而相位非 done；相位=done 而无收口事件链；
   c. **容量水位**：连续 3 拍 spawn rc≠0；候选集非空而
      拉起=0 超 2 拍；worker 退出后无收口事件链>30min；
   d. **账本真值**（抽查非全量）：最近完成票的收口 commit
      在 master 可解、票面回填存在；
   e. **外事聚合**：origin/master 落后龄>2h；deploy 批龄
      >12h——已有探针者不重复，本窗只做聚合判断。
   f. **用户端点健康**（修订补面——2026-11-09 驾驶舱 502 实证
      业主令加）：按用户同路径探端点——dashboard SPA 根+
      `/o.dashboard.v1.CoreService/<轻量RPC>` 期望码对拍；
      **部署窗豁免语义**：ops_event 近 N 分有部署行且端点
      已复绿=不报（image 部署重启窗内 502 属预期瞬断），
      重启后仍红>5min 或无部署行而红=升件类目「端点失健」。
      端点清单随票列明（初期=dashboard 一处，扩面另立）。
   g. **无籍相位行**（修订补面——2026-11-09 T-9001 幻影实证
      业主令加）：ops_ticket_phase 存在行而 docs/tasks(+done/)
      无票件=升件类目「无籍行」——成因族=session id 撞票号
      域（T-9xxx 测试夹具段同病）/键闸过形不过实；判据按
      dedup 24h 一行，真票缺失不自动补造。
   h. **升件失领**（修订补面——2026-11-09 管道面板
      escalation_unclaimed 38h19m 实证业主令加）：
      ops_event 升件行超龄（阈≥4h）无处置/认领消费行
      =升件类目「升件失领」——面板挂行≠闭环，升件本身
      需二次升件进裁队；判据认「同 ticket 后续处置词行」
      为认领，dedup 同对象 24h。

   每破例→`ops_event` 升件行（seat=MECH，note=「层扫
   定性<类目>:<对象>」——走 T-0506 语义闸钦定 seat；
   dedup=同对象同类目 24h 一行不刷屏）。
3. **digest 行**：每窗落 `event='层扫'` 行（seat=MECH，
   note=各面计数+异常计数）→进 ceo_queue 收件面；
   `scripts/meta/ceo_queue.sh` seat 白名单补 `'MECH'` 或
   照现有 seat 口径归一（施工按现行词表钦定）。
4. `scripts/meta/tests/`——层扫桩测新件。

## 判据

1. 回放三病灶各报出：人造 dispatched-无拉老化/pulled-无unit/
   票件幽灵行→对应类目升件行落库且 24h dedup 生效；
2. digest 行落库且 `ceo_queue.sh` 出列含层扫行；终态行
   （处置/裁决）后正确出队；
3. 全绿态层扫=零升件+digest 照落（静默≠缺席，锚在案）；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 层扫**只报不判**——机械枚举不变量，定性词限类目级
  （「dispatched-无拉」非「为什么无拉」），诊断归票 B；
- 枚举阈值写常量带出处注，禁魔术数；
- 不碰钱径/下单径；扫面只读 ops_event/票件/systemd 状态，
  零写任何生产面。

## 结果回填

- 状态：待验（施工收口——回决策席裁决径；施工席不自验）
- 实现摘要（票面六面逐件对位）：
  1. **层扫窗**（`scripts/meta/layer_scan.sh` 新件 400 行+
     `mech_tick.sh` 收敛窗前插唤段）：节拍复用 mech_tick 既有
     advisory lock/FIN_DSN 闸，零新守护件零新 unit；本体非零退
     →mech_esc「层扫本体败」一行+不挡后续窗（drift 先例同式）。
  2. **六片不变量**（类目级定性词只报不判）：
     a. 队列一致——dispatched 龄>30min 无拉起∧无在飞
        o-worker unit=「dispatched-无拉」；pulled 有机器拉起证
        无存活 unit=「pulled-无unit」；review/approval 龄>2h=
        「review-老化/approval-老化」；draft 滞留>1h=「draft-滞留」；
     b. 相位↔实体——相位非终而票件无=「幽灵票件」；票件 done/
        而相位非 done=「done-翻漏」（7d 扫窗 dm_handoff 同式首跑
        不洪史）；相位 done 而近 7d 无收口事件链=「done-无收口链」；
     c. 容量水位——dm_queue dispatched 候选集剔「候水」标记票；
        拍间拉起/领单=0 streak≥2=「拉起-空转」、≥3=「spawn-连败」
        阶梯；worker 退出无收口链>30min=「worker-无收口」；streak
        态存 layer_scan.state 跨拍，查询败不推 last_scan=未知≠空集；
     d. 账本真值——最近 5 张 done 相位票抽查：收口 commit
        master 可解（git log --grep）+票面状态行含完成|待验；
     e. 外事聚合——origin/master 落后龄>2h=「origin-落后」；
        deploy 批龄>12h=「deploy-批龄」（deploy_window_due.sh
        同源只读复用，OPS_ESC_SH=/bin/true 压其升件——已有探针
        不重复，本窗只做聚合判断）；
     f. 用户端点健康（票面修订补面——业主令驾驶舱 502 实证）——
        用户同路径探 `https://o.myfxlogs.org/` SPA 根+
        `/o.dashboard.v1.CoreService/Ping` 期望码 200 对拍
        （O_LAYER_SCAN_ENDPOINT 可覆写）；部署窗豁免——最新
        ops_event event='部署' 行龄≤5min 内端点红=image 重启
        瞬断不报，仍红>5min 或无部署行而红=「端点失健:dashboard」
        （初期清单 dashboard 一处钦定，扩面另立）；部署行查询败
        按无部署行判=fail-closed 向报不向隐。
     g. 无籍相位行（票面修订补面——T-9001 幻影实证业主令，返修1
        落）——ops_ticket_phase 行在而 docs/tasks(+done/) 两处无
        票件=「无籍行」，与 b① 幽灵票件对象域分判钉死：b①=合法
        号域 T-[0-8]xxx 非终相位无件（既有实现 T-[0-9] 收窄至
        T-[0-8]）；g=b① 补域全键——T-9xxx 夹具段（session id
        撞票号域成因族）/非 T-票形伪键（闸过形不过实）/合法域
        终相位无件；真票缺失不补造。归口 entity 面（相位↔实体
        不一致同族）。
     h. 升件失领（票面修订补面——escalation_unclaimed 38h19m
        实证业主令，返修1 落）——ops_event 升件行超 4h（常量
        ESC_UNCLAIMED_S=14400 钦定阈带出处注）无同 ticket 后续
        认领词行（处置/裁决/完成/驳回/消/挂起——ceo_queue 终态
        词表同源）=「升件失领」同 ticket 分组计件+最老龄二次
        升件进裁队；空 ticket 系统面升件对象键=「无票键」稳字
        面（票面「同 ticket」含空集同值径）；查询败=空集跳过
        本面 fail-closed 向静默。归口 queue 面（裁队队列项积压
        同族）。
  3. **升件径**：每破例→ops_esc.sh（seat=MECH，
     note=「层扫定性<类目>:<对象>——<详情>」）24h 前缀去重=同
     对象同类目 24h 一行不刷屏；类目/对象稳定字面供 T-0508
     signature=「层扫:<类目>:<对象>」对接（诊断归票 B 本票不做）。
  4. **digest 行**：每成功扫窗落 `event='层扫' seat='MECH'
     ticket='层扫'` 行（各面计数+相位直方图+unit/cap 降级标），
     全绿态照落=静默≠缺席锚在案。
  5. **ceo_queue.sh 事件级白名单**：`(event='层扫' AND
     seat='MECH')` 单列入队——MECH+升件仍不入=机械升件留 CTO
     壳轨不串味（CEO席职卡钦定不破）；digest ticket='层扫' 伪键
     单槽最新体检行，处置/裁决等终态行出队后新 digest 再入队。
  6. **降级面**（败≠空集）：FIN_DSN 缺=静默退零写；相位投影
     查询败=exit1（mech_esc 本体败行接住）；systemctl 败=unit
     面不可判 digest 载 unit=unk 不误报无 unit；dm_queue 败=
     cap=skip streak 保留不清；时间回拨=保 streak 不评窗。
- 测试证据：
  - `scripts/meta/tests/layer_scan_test.sh` 新件 12 场景全绿——
    ①回放三病灶（dispatched-无拉/pulled-无unit/幽灵票件）各报+
    digest 同窗落；②24h dedup 同对象同类目一行不刷屏；③零报面
    （unit 在飞/人工径/龄未超/无机器证）；④老化三类目正报+龄
    未超零报；⑤done-翻漏宽限+相位守卫+无收口链正报；⑥账本
    抽查三态（全绿/commit 不可解/回填缺席）；⑦外事聚合正报+
    收敛归零；⑧容量 streak 阶梯+候水剔出+归零；⑨端点健康四态
    （绿/无部署行红/Ping 独红/部署行≤5min 瞬断豁免/>5min 仍红
    报）；⑩全绿态零升件+digest 照落；⑪降级面四态（FIN_DSN 缺
    静默退/相位查询败 exit1/systemctl 败 unit=unk/dm_queue 败
    streak 保留 cap=skip）；⑫返修1补面——g无籍行四域分判（T-9xxx
    夹具段/非T伪键/合法域终相无件/T-9xxx有件不误报）+幽灵vs无籍
    对象域分判钉+h失领分组报出/认领零报/dedup。
  - `mech_tick_test.sh` 扩——层扫窗唤起断言+本体败→层扫本体败
    升件行不挡后续窗，全绿。
  - `ceo_queue_test.sh` 7/7——MECH+层扫 digest 入队、MECH+升件
    仍拒、处置终态消队、新 digest 再入队（真 PG scratch 库 SQL
    语义）。
  - 自测 `go build+vet+check_migrations` 绿；`go test ./...`
    全谱绿（check_all 内）。
  - check_all 门禁段全绿（build/vet/import/gofmt/env/rejects/
    config/migrations/bridge-audit/tolerate/ops-chain/seat-writes/
    ticket-seq 逐闸 OK）；**go test 全谱仅
    `o/internal/ledger/report TestReportMacroEventsCount` 红——
    预存量日期依赖潜在 bug，master 净树（/opt/o 无本 diff）同
    红复现在案，与本票 diff（纯 scripts/meta 壳件）零交集，
    判据「零新增 FAIL」满足、无绿戳如实记**：病灶=报表
    macro_events 计数 `event_date>=$1 AND <$2` 参数面 date
    语义截尾（date>=timestamptz 实测 '2026-11-09'::date vs
    '01:11+00'::tstz=false——当日事件被期界排他漏计）+夹具
    「+1d 事件」假定窗宽≥2d——周一/周二跑必红周三起自愈，
    缓存命中径掩盖（check_all 注释钦定陈判窗口）。修复归另
    立票（钱径外报表面小 bug 但每周一二冷缓存 wt 全撞——
    建议 P1）。返修1 复跑 check_all（HEAD 27d5495b5）同态——
    全谱唯一红仍该预存量件，g/h 面零新增 FAIL。
- 偏离说明：票面「五片不变量」+修订补面 f=六面（票面原文六项
  a–f，标题「五片」为修订前旧称）；阈值全常量带出处注钉死；
  ceo_queue 入队走事件级白名单非 seat 集扩列（票面「施工按现行
  词表钦定」授权径）。
