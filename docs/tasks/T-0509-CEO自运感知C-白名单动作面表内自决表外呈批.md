# T-0509 CEO 自运感知·票C——白名单动作面（表内自决/表外呈批）

- 状态：已批（CEO 管理令放行 2026-11-09——ABC 序之三，
  **附条件三钦定在案须随票呈现逐项呈裁**；依赖 T-0507/T-0508）
- 级别：P2 动作面——本批最敏感件，表初版内容=本票交付物一部分

## CEO 附条件三（钦定，判据内嵌）

1. **白名单条目随本票逐项呈裁后才生效**——票面列初版表全文，
   决策席逐项过目批注，未批项不入表；
2. **条目变更须再呈裁**——表内容变更走升件呈批径，禁表内
   自扩；
3. **败径升件不自愈**——白名单动作执行失败落升件行等裁，
   禁自动重试/降级路径。

## 范围（待逐项呈裁的表初版——施工照此列明）

| # | 动作 | 触发形 | 界 |
|---|---|---|---|
| 1 | 相位归位 | pulled 尸（无存活 unit）→重派（决策席 seat 转正） | 同票 24h≤1 次防循环 |
| 2 | 相位归位 | error 伪键/无票件→作废归档（键形/票件双查可判时） | 不可判者升件不猜 |
| 3 | 醒票 | suspended 且 await 锚已至→重派 | 锚判定须机械可证 |
| 4 | 席位 kick | oneshot 席 stall 而队列非空→start（同事件 kick 语义） | 单例闸现成 |
| 5 | 界内微调 | 白名单键值域（如 O_WORKER_PARALLEL_LIMIT∈{1,2}） | 键集随表列明，超界呈批 |

执行器形：mech_tick 窗或独立件（施工裁形）；每动作行落
ops_event（seat=MECH 或钦定 seat，note=「规则#N 命中」引证——
自决≠无账）。**红线三件（资金扩投/新平台/席位立换）永不入表**
——票面须明文重申。

## 判据

1. 表初版全文随票呈裁——决策批注落票后实施项才生效；
2. 人造尸 pulled→规则#1 命中重派+事件行引证；同票 24h 二次
   不重复动作（防循环闸绿）；
3. 表外动作触发→呈批行非自决；执行失败→升件行且零重试；
4. 全谱零新增 FAIL/SKIP。

## 禁线

- 表外一切=呈批不猜；动作面零碰钱径/venue/资本面；
- 不发明新 seat——动作行 seat 归现行钦定集；
- 规则命中判定须纯机械（查询结果非 LLM 读面）。

## 结果回填

- 状态：待验
- 实现摘要：
  1. **白名单表** `scripts/meta/whitelist_actions.tsv`（新件）——
     六列 TSV（rule_id/action/param/bound/status/note），初版五条=
     票面表初版全文全「待裁」——判据①逐项呈裁后才翻「生效」；
     status 列为唯一机读判据（note/bound=呈裁留档面判定不读）；
     空字段 '-' 占位（bash read 折叠连续 tab 丢位错列——桩测实证
     根修）；param=;分隔 k=v 承载 seat_kick units=/cooldown_s= 与
     config_set key=/domain= 机读界。条目变更=本文件 commit 即呈裁
     面，表内自扩结构性不存在（判定层只读 status）。
  2. **执行器** `scripts/meta/whitelist_actions.sh`（新件，355 行）——
     三层全机械：检测层五规则只读 ops_ticket_phase/ops_event/
     systemd is-active+InactiveEnterTimestampUSec/票件存在性零写；
     判定层 TSV status——「生效」→执行、「待裁/停用/缺席」→表外
     呈批（event='呈批' seat='MECH' ticket=''——票外件不入相位投影，
     对象身份落 note 前缀「表外动作呈批：规则#N:<对象>」24h 同键
     去重；ceo_queue 以「呈批#<ops_event.id>」合成行入 CEO 队——
     议题#id 同构，终态事件 note 引证「呈批#id」出队，席域钉 MECH，
     票外呈批对裁决席结构性可见）；执行层动作本体+审计三件。败径：ops_esc 升件前缀
     「白名单动作败：规则#N:<对象>」seat=CTO（mech_esc/dm_suspend
     机械升件钦定口径），同（规则,对象）升件在案且其后无 lift 事件
     （生命周期+裁决注记族 60 词）=等裁期零重试——lift 事件集含
     自决/参数调整，自限每裁决周期一次；不可判分支（规则#2 真票
     有件/DIAG 键）「白名单不可判」升件不猜不闸 status。查询败
     全 fail-closed（宁可本拍不动作不冒重复动作/重试）。
  3. **五规则判据**（全纯机械）：①pulled 尸=phase='pulled'∧龄>
     600s∧^T-NNNN∧票件在∧manual_lane rc1∧o-worker-<tid> 非活
     （active/activating/deactivating 除外）→dispatch_worker+
     拉起行（seat=CTO 同补拉窗钦定口径）note「规则#1命中」，
     acted_recent 24h 防循环（钦定界）；②error=伪键/无票件→作废
     行 seat=MECH，真票有件/DIAG-*→不可判升件；③suspended 非
     rate-limit:auto（seat=施工席∧note 前缀钉同 dm_suspend）∧
     await=绿窗档复跑绿→续跑（判据逐字同 dm_suspend，本窗在其后
     序跑结构性不重；「拉起」晚于「挂起」=已消解跳过；acted_recent
     3 防自环）；④oneshot 席=param units= 表内席 stall（非活∧
     InactiveEnterTimestampUSec 龄>stall_s，空/0=未跑过=满龄）∧
     队列探针非空（cto-seat→dm_queue review/ceo-seat→ceo_queue/
     verdict→fin_*_lesson pending——无探针=不可判跳过不猜）→
     systemctl start --no-block+自决行 seat=MECH ticket=<unit>，
     cooldown_s 内已有关#4 行=防抖；⑤界内微调=key=/domain=——
     候水 24h≥3∧现值>域底→步降一档，候水=0∧现值<域顶→步升一档
     （域序相邻步进非跳极），目标∉domain=呈批不猜，覆写档
     $(o_worker_runtime_dir)/parallel-limit-override 原子写+
     参数调整行（seat=MECH ticket=''——镜像 ops_decision 同径
     在案），24h dedup+suppressed 双防抖。
  4. **接线面**：mech_tick.sh verdict_deliver 窗后新增白名单窗
     （同 FIN_DSN 门控+fail-soft mech_esc 径）；worker_runtime.sh
     o_worker_parallel_limit 前置覆写档采信（有效值 1|2 才认——
     坏档/缺档回落 env 缺省律，env>档语义反转为 档>env=运行时旋钮
     本位）；ceo_queue.sh 席位集补 'MECH'+票外呈批「呈批#id」
     合成行分支（ticket='' 票外件不入相位投影故主支不可见——
     议题#id 同构出列，note 引证「呈批#id」消队；不补则呈批对
     裁决席结构性不可见）；ceo_tick.sh 开班 prompt 补呈批#N 处置
     口径；docs/数据流.md 机械轨段补 T-0509 窗正本段。
  5. **桩测** `scripts/meta/tests/whitelist_actions_test.sh`（新件，
     40 断言）+ceo_queue_test.sh 场景③④⑤（MECH 票外呈批呈批#id
     入队/引证消队/非 MECH 席域钉死——真 PG 径）。
- 自测：bash -n 全改动件绿；whitelist_actions_test 40/40 绿
  （尸重派+引证行+24h 防循环/表外呈批零自决+去重/执行败升件+
  抑制期零重试/error 作废与不可判双支/suspended 绿续跑+红零动作
  +rate-limit 排除+已消解跳过/kick 唤起+自决行+队列空活跃冷却
  表外败径全谱/微调步降步升+域底顶零动作+表外呈批+写败升件）；
  邻面回归 mech_tick/dm_suspend/ops_esc/ticket_key_gate(23)/
  worker_runtime/ceo_queue(6)/tolerate_silent_batch 全 PASS；
  check_tolerate_report 绿。
- 域界呈裁（决策席裁时注意）：①T-0507 层扫窗未在本树——本件检测
  层直读 ops_ticket_phase/ops_event 既有投影不依赖层扫产出，
  层扫上线后异常→诊断草案链（T-0508）与本件表内自决面正交；
  ②规则#1「无存活 unit」判据=无 o-worker-<tid> 活 unit——交互
  式 Devin session 拉起的票无 unit 亦呈尸相，dev_run 会话恢复
  径（o_worker_session_state resume）兜底不双开冷启，但交互会话
  在飞票仍可能被唤——此边界建议决策席批注时一并裁；③T-0506
  (seat,event) 合法性矩阵未实施——本件新增在产组合
  （MECH,呈批/自决/参数调整/作废）（CTO,拉起引证）供其枚举对拍；
  ④规则#4 units=/stall_s/cooldown_s 与规则#5 key/domain 值均为
  呈裁初版建议值，决策席逐项批注可改——status 未批前全表不生效
  故无行为风险；⑤**check_all 已知红（与本件无关，呈裁席注意）**：
  `internal/ledger/report TestReportMacroEventsCount` 在设
  O_TEST_PG_DSN 的真测试库下确定性红——根因已实证：report.go
  `event_date>=$1 AND event_date<$2` 的 time.Time 参被 PG 按
  **date** 推断比较（显式 ::timestamptz 对照 count=1 vs 无 cast
  count=0），窗口塌缩为 [date(Start),date(End))——周一/周二
  End 与 Start 同日或仅隔一日→期内事件数≠2（周一实测=0）。
  即每周一二定期假红、本件 diff 对 internal//migrations/ 零改动
  （git diff 实证），系 T-0029 期遗留周界雷非本件引入——建议
  另立小单修（report.go 参钉 ::timestamptz 或夹具改用中周锚）。
