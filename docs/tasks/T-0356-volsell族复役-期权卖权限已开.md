# T-0356 volsell 族复役——期权卖权限已开

- 状态：已批（COS 例行代理批——业主令「期权的权限已打开」+「出维修
  提示词我外派」定案）
- 级别：P2 产线残缺——评估空转族：vol_sell_single_call/strangle
  模板在册 grammar 臂枚举不停，而 fin_opp_eval 行止于 10-30，
  8.6k 模板零评估产出=枚举烧算力零收益面

## 病灶（实查在案，勿再诊）

- vol_sell 评估归 **grammar 枚举臂实评**（cmd/research/dcd.go:188
  「T-0215 vol-sell 实评 coins 缝」）——无 enabled 闸臂在跑但
  eval 行 10-30 起断流，与 dcd 退役同日=**连坐嫌疑或凭据拒面**，
  根因待工人亲证（禁止先假写）；
- 执行面 `cmd/core/volsell_open.go` T-0215 在案；
- **业主已开 OKX 期权卖权权限**——「结构性不可执行」前提消失，
  复役条件成立；
- band.go 白名单已正确排除 vol_sell（非近确定族——卖权有尾部
  风险，保 0.06 纯阈档；`research.volsell.min_apr=0.06` 在册）。

## 任务（修法供裁）

1. **根因先证**：vol_sell eval 断流真因——连坐 dcd 闸/凭据 401/
   期权 instruments 目录面断/other，证据锚落回填；
2. 复役照 dcd 模板：如需 enabled 闸补 `research.volsell.enabled`
   键钉 SG 正本+fail_streak 复役条件书落票面（T-0353 同款），
   若根因是凭据/目录面则按实修；
3. **担保腿域分流**：期权卖=保证金占用（裸卖 USDT 担保/备兑
   BTC 腿）——业主资产域腿走条件请求闸（T-0353 件④同款径），
   禁自动开仓直通；
4. vol_sell 阈档零动：保 `research.volsell.min_apr=0.06`，
   禁入 band.NearCertain 白名单。

## 验收锚

1. eval 断流根因证据行落票面回填（查询/日志/代码锚）；
2. vol_sell 族新 eval 行实出（修复后 runner 拍出）；
3. 担保腿条件请求分流实证（合成提案→盒内非 armed）；
4. band.Audit 对 vol_sell 仍按 0.06 档判（测试断言）；
5. check_all 绿戳对口提交 SHA。

## 红线

- band.NearCertain 白名单零改——vol_sell 永禁进中间档
  （尾部风险族，业主裁 B 钉死「只放近确定族」）；
- 不动 executor 参数/下单径/阈数值；
- venue 全零 mutating——复役只到评估/提案面，开仓仍走
  armed→闸→业主资产域条件链；
- fin_opp_template 既有行零回填改写。
