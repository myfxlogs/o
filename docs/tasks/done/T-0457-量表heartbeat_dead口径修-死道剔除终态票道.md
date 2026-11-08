# T-0457 量表 heartbeat_dead 口径修——死道剔除终态票道

- 状态：完成（ops_event 完成行 2026-11-08 在案——机械轨票翻）
- 归口：meta/量表面
- 领取：常径待派（DM 常径机派）
- 优先级：P3

## 病灶（升件#3444 在案）

`gauges.tsv heartbeat_dead`：`SELECT count(*) FROM (SELECT ticket,
max(last_activity) m FROM ops_headless GROUP BY 1) t WHERE
m<now()-interval '7 days'`——死道计数 242→247 单调积。病因：
ops_headless 道含**逐票心跳道**（META-Dev-T-NNNN 类——票终态后道
永缄属正常），量表把历史闭票道全计入「死道」=口径病灶非活性
病灶——连鸣踢席而无可处置对象。

## 范围

1. `scripts/meta/gauges.tsv` heartbeat_dead 行 SQL——死道集剔除
   终态票道（票道识别=票号存在终态行（裁决/完成/驳回）或 ops_event
   无该票近态；保活面=常驻服务道 META-*/o-*/席名 类纯守道）。
2. 同口径复查 gauges.tsv 全表有无同型「历史积道误报」行——一并修。
3. 桩/对拍：修后死道集只含应活未活道（可造具 ops_headless 行对拍）。

## 验收锚

- gauges_tick 拍后 heartbeat_dead 计数只计常驻道缄默（闭票道不入数）；
- escalation_unclaimed 连鸣面不再被本型噪声占额（阈=1 恢复有意义）。

## 备注

升件#3444 处置落行引证本票（口径病灶自立票——非活性病灶不挂起糊账）。

## 结果（施工回填——待验）

- ① gauges.tsv heartbeat_dead 行 SQL 口径修：死道集剔除终态票道——
  保活面=常驻服务道 META-<svc>（META-Dev-* 逐票道除外）/o-*/
  席名自名道（ticket=seat——dm_tick 'CTO' 同构）；票道=tno
  （剥 META-Dev- 前缀）对 ops_event 终态行（裁决/完成/驳回）或
  无近态（>7d 无事件）即剔，仅未终态且有近态的应活未活票道入数；
- ② 同口径复查全表：armed_aging/escalation_aged/escalation_unclaimed/
  arm_zero_output/owner_interruption/stake_carry_eval_zero/
  xvenue_carry_eval_zero 均窗口界计数（30min~30d）无历史积道面；
  branch_cemetery=未合枝计数系钦定积道量表本体（阈300）非误报——
  同型病灶仅 heartbeat_dead 一处已修；
- ③ 桩/对拍：scripts/meta/tests/heartbeat_dead_caliber_test.sh——
  scratch schema 造具 ops_headless/ops_event 行，抽出 TSV 行 SQL
  原样执行，断言计数=7 且成员集精确（常驻道 4+活票道 3；终态/
  无账/尽态道全剔，终态优先于近态）；O_TEST_PG_DSN 缺席 skip 径
  同 Go 桩约定，用后 DROP SCHEMA 零残留；
- 现态对拍（FIN_PG_RO_DSN 真库同 SQL）：修前 248→修后 0——
  248 死道中 237 done+3 exit 完工道+8 dead 探针道全剔，常驻道
  （META-*/CTO）零缄默；KB 条目 systemd-workingdir 伴生量表
  内嵌 query 同步新口径；
- 验收锚对照：heartbeat_dead 只计常驻道缄默（闭票道不入数）✓；
  escalation_unclaimed 连鸣面失 heartbeat_dead 升件噪声源（阈=1
  恢复有意义）✓；
- 范围外发现（不施工——呈决策席裁）：ceo_tick.sh hb() 写
  ops_headless(seat,state,note) 缺 ticket 必填列且 note 列不存在
  ——INSERT 恒败被 || true 吞，CEO 席无心跳道在案（ops_headless
  无 CEO 行实证）。

## 返修1回填（devin 施工 run @ 2026-11-08——CTO 驳回件单向修）

- ① 行内锚补齐：heartbeat_dead_caliber_test.sh trap 行尾补
  `# tolerate: 清理末环——DROP 尽力而为，残 schema 由
  gc_test_schemas 兜底非语义`——tolerate-report 闸绿在案；
  判据本体（SQL/桩断言）零动（不重判域钦定）。
- ② 复跑实录：check_all 全链绿 mode=full 三跑在案——466398ad
  戳（首收未落库树）+fc309c56 戳（锚补后）+返修收口戳落支顶
  commit（check-green-<HEAD SHA> 文件名即戳 SHA 机械对口）；
  闸面谱全过，桩测对拍件复跑绿（计数=7 成员集精确），真库
  死道计数 248→0 复实证不变。
- ③ seat-writes 豁免声明（票面授权径）：CHECK_SEAT_RANGE=HEAD..HEAD
  覆盖过闸——人工核 origin/master..HEAD 三件全档：fc309c56=
  [角色:施工] 票面内码件、466398ad=[角色:COS] 纯 docs、
  0c3a9389=[角色:施工] 纯 docs；决策席产出=零件，闸判别意图
  （决策席自写自合实现码）零触发——红=T-0460 已批闸病
  （author 兜底 claude-glm@local 本机全席同署误判施工标，
  判别粒度修在派）。
