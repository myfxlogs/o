# T-0151 黑名单回扫覆面补 probe/degraded 态（T-0147 验收衍生）

## 态：完成（判过并码在案，11-01 清账翻面·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**完成**——审计通过落票在案，漏网翻面
审计席：headless @ 2026-10-30
## 领取：zcode-o-7d3f @ 2026-10-26
## 域：执行径 / venue 终局分类
## 源出：T-0147 验收生产窗实案——0061 trigger 谓词只覆
  live/paper/proposed 三态，策略 286181（SHEIN-USDT-SWAP）probe 态
  漏扫滞留（决策席手动退役清尾，transition 行已落）。

## 病灶
`fn_untradeable_retro()` + `hedgeOpener.untradeable()` 的 live 守卫
均不覆 probe/degraded 两态——黑名单 inst 上的 probe 策略滞留：
probe→live 晋升后首开仓才被 opener 黑名单闸退役，窗口期策略
在册活着=状态机谎言（venue 已证开不了）。degraded 同形。

## 施工
- 0062 迁移：trigger 谓词集 IN ('live','paper','proposed','probe','degraded')
  ——retired 外全覆（闭集拒自挡已退役行）。
- `hedgeOpener.untradeable()` live 守卫同扩（显式迁移先于入册序位
  不动——仅谓词集对齐）。
- probe→live 晋升径顺带核：promotion 闸若读黑名单更优（未读=靠
  opener 兜底，可接受但记档）。
- 桩锚：probe/degraded 态策略+黑名单入册→trigger 回扫退役+
  transition 行落。

## 验收锚
- 五态全覆盖桩绿；存量黑名单 inst 上任意非 retired 策略=零。
- 生产实证：probe 态幽灵（如再出现）入册即退。

## 结果回填

- ①0062 迁移（migrations/0062_untradeable_retro_probe_degraded.sql，登记
  migrate_table.go {62,…}）：fn_untradeable_retro 谓词集扩五态
  IN ('live','paper','proposed','probe','degraded')——六态闭集减 retired
  （0007 CHECK 域亲证），幂等二重承 0061（PK 冲突走 UPDATE 不触发+状态
  谓词拒自挡已退役行）。
- ②存量清尾随迁（验收锚「存量=零」的达成面——trigger 只打新入册行，
  0061 生效期漏网 probe/degraded 由迁移一次性回扫）：transition 行
  reason=venue_untradeable_retroactive:0062_backfill 溯源+armed 提案
  fin_retired；谓词自挡=重跑幂等。
- ③opener 容退守卫对齐（cmd/core/open.go untradeable()）：守卫谓词
  state='live'→IN 五态，序位不动（显式迁移先于入册）。probe/degraded
  →retired 均合法迁移（machine.go legal 亲证）。
- ④晋升闸核查记档（票面「顺带核」项）：paper.go promote() 的 probe→live
  闸不读 fin_venue_untradeable——0062 后黑名单入册时点即五态全退，
  probe 态幽灵无处滞留到晋升；纵深仍三层（universe 滤/入册回扫/opener
  双兜底：黑名单开仓败退役+untradeable 显式迁移），票面预设「未读=靠
  opener 兜底可接受」成立，不改晋升径。
- 桩锚三件全绿（go test ./cmd/core/ -count=1，O_TEST_PG_DSN=5434 实例）：
  TestRetroTwoStates（五态逐腿入册即退+retroactive 溯源行+已 retired
  闭集自挡零 transition）、TestUntradeableGuardTwoStates（probe/degraded
  显式迁移 venue_untradeable:<code> 非 retroactive——守卫放行直证，回退
  live 单态守卫必红）、存量 TestRetroactiveScanTrigger 等四件回归绿。
- 偏离说明：存量清尾为验收锚②推定的必做面（施工清单未单列）——非越单，
  语义与 trigger 体同构。
- 生产实证锚（待验席合并部署后可亲证）：黑名单新入册任一非 retired 态
  策略=入册即退（桩测同形直证）；存量侧合并时 0062 清尾自动落。

## 判决注记（审计席逐锚对账，收口 86982d6d diff+生产只读取证）

- 锚①五态全覆盖桩绿：0062 trigger 谓词 IN 五态 diff 亲证；桩码实存
  untradeable_test.go——TestRetroTwoStates 五态逐腿入册即退+retroactive
  溯源行+retired 闭集自挡零 transition、TestUntradeableGuardTwoStates
  probe/degraded 显式迁移非 retroactive 行恰一（回退单态守卫必红设计）。
  状态闭集=0007 CHECK 六态亲证、谓词集=六态减 retired 恰合；五态→retired
  全合法（lifecycle/machine.go legal map 亲证）；helper（setupDB/t51001/
  hedgeOpener 字段）签名吻合。存量=零：0062 迁移内一次性回扫（transition
  溯源行 reason=…:0062_backfill+retired+fin_armed→fin_retired 谓词自挡
  幂等），prod 实测 nonretired_on_blacklist=0、schema_migrations v62 已应用、
  pg_proc 函数体含五态谓词——机制在产非仅码面。
- 锚②生产实证（条件锚「如再出现」）：prod trigger 函数五态谓词已投产，
  fin_venue_untradeable 现空=幽灵当前不存在；桩测同形直证入册即退语义，
  条件满足。
- 顺带核项：paper.go promote() 不读 fin_venue_untradeable 亲证（函数
  全文无该表引用）；universe.go:70 滤面读黑名单实存——纵深三层在，票面
  「未读=靠 opener 兜底可接受」判语成立，记档一致。
- 工程面：迁移登记 migrate_table.go {62}（经 T-0167 归一化名 fin_0062
  与文件一致）；open.go 297 行未超闸、*_test.go 豁免；无凭据入仓、无
  快照/定时器/轮询违例、无 env 直读新增。
- 判：审计通过。
