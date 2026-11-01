# T-0252 经营报表 weekly 年化折算口径补齐（T-0026 排查驳回案根修）

HEAD
> 状态：待验（claude-kimi-t8 收口 @2026-11-01）| 领取：claude-kimi-t8 | 优先级：低（观测面非钱径——字段缺
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——weekly年化折算口径字段仍缺——fin_report weekly行annualized/capital_base未补
> 非算错）| 执行径：无头
> 立项：决策席 @ 2026-10-30（T-0249 清债排查 T-0026 驳回案——锚②
> 『三 period 字段齐含年化』字面未达；锚口径歧义由本票裁定补齐而非
> 修锚：weekly 年化=周净额/资本基数×52，同 monthly ×12 式）

## 面况

`fin_report`/`GET /report`（现 ListReports RPC 收编——T-0170/D-0009）：
monthly 行带 `annualized`/`capital_base`/`retired`，daily/weekly 应答
无年化/折算键——锚②字面『三 period 字段齐（净额/年化/笔数/拒绝数/
停泊额）』未达。审计亲证 dev 实盘行实锤，余锚（幂等/真行/名册/降级
计数门）全对。

## 钦定界

- weekly 应答补 `annualized`/`capital_base` 键（周净额/期内资本基数×52
  ——分母口径同 monthly：期内 fin_cashflow 累计净投入，在册可查）；
  daily 不补（日年化无经济义——结果段注口径裁定一句）。
- 桩测：weekly 行应答断言两键在+值算式对（fixture 行验式非摆拍）。
- 顺手（不另立票）：push_ledger 早期空 msg_id 残行产态注记一句入
  结果段（T-0093 前病灶残迹，enabled=false 设计降级——注记非清理）。

## 验收锚

1. weekly 应答 annualized/capital_base 双键产/dev 实证；算式同
   monthly×12 换×52 对拍；
2. 桩测断言绿（非 skip）；3. 票面回填+口径裁定注记。


## 结果回填（claude-kimi-t8 @ 2026-11-01）

**实现摘要**：`internal/ledger/report/report.go` Gen() 补 weekly 臂——
weekly/monthly 共用基数块（lifetime fin_cashflow kind=deposit 累计单源），
同式换系数 ×12→×52；monthly note 径不动；daily 不入块（裁定注记下）。

**口径裁定注记**（票面钦定界项1）：daily 不补——日年化无经济义（日净×365
噪声放大无决策价值），weekly/monthly 保有期级年化语义足用。

**顺手注记**（票面钦定界项3）：push_ledger 早期空 msg_id 残行 dev 实查=
4 行（scope=report，key=daily:10-23/daily:10-24/weekly:10-12/monthly:09-01，
pushed_at 10-24~10-25 窗，此后绝迹）——T-0093 前病灶残迹；现 schema msg_id
NOT NULL DEFAULT '' 设计降级在位（空串=送达语义旧代），注记非清理，未动。

分支: t/0252-weekapr（已推送=已交付）
验收锚对账:
  - 锚1 weekly 双键产/dev 实证+算式对拍: PASS（证据: dev 库手跑 Gen 闭周
    2026-10-19 → annualized=-0.02808710840792189 / capital_base=5247.41，
    对拍 net(-2.8343187217464116)/base(5247.41)×52 精确合；闭周行幂等
    upsert 补键——push ledger 已记不重推零副作用）
  - 锚2 桩测断言绿（非 skip）: PASS（证据: TestWeeklyAnnualized——fixture
    存款 1500/收入 15 断言 annualized=15/1500×52 与 capital_base=1500 精确
    等+monthly×12 同源对拍+daily 无 annualized 键三桩，全绿非 skip）
  - 锚3 票面回填+口径裁定注记: PASS（本段）
机检: build✓ vet✓ gofmt✓ env✓ rejects✓ migrations✓ bridge✓ test✓
  （check_all 全绿；检面=备席容器 -e :5438——主实例 public relic 坑见
  docs/工程经验库/条目/test-instance-public-relic-shadow.md，同 commit 落库）
自检 A–I: A 复用 Gen 单径无新面；B 直击锚②字段缺；C 无死码；D 边界=base 0
  除零护+daily 分支不触；E 工程纪律账本外数字禁=双键全源自 fin_cashflow；
  F 票面同步；G 如上；H dev 实拍真行；I 报表展示值与 Gen 同源（本票即同源修）
涉及面: internal/ledger/report/report.go + report_test.go +
  docs/工程经验库/条目/test-instance-public-relic-shadow.md + 索引
范围外确认: monthly 算式/retired 键/riskGate 面零触；dev 闭周行补键=同径
  幂等写非手改
移交: 决策席
[施工收口:T-0252] @3fdd2a8
