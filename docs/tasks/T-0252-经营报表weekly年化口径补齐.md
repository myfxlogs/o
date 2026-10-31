# T-0252 经营报表 weekly 年化折算口径补齐（T-0026 排查驳回案根修）

> 状态：**已立未派** | 领取：虚位 | 优先级：低（观测面非钱径——字段缺
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
