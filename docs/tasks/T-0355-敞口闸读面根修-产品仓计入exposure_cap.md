# T-0355 敞口闸读面根修——产品仓计入 exposure_cap

- 状态：施工中
- 领取：devin-t0355 @ 2026-11-05（业主令外派复工 run）
- 级别：P2 风控语义——闸漏看真仓位：DCD 0.0207 BTC（~$2.2k）在
  exposure_cap 判据里零计，新单压线时闸会按欠账放行=超真敞口

## 病灶（实查在案，勿再诊）

T-0351 工人回填标的孪生洞（其披露③）：`internal/risk/check.go`
`exposure_cap` 闸仍读 `fin_position` 单账——产品仓（dcd_baseline
凭据族）不入判据。显示面已由 fin_position_view 统一（T-0351 合），
**闸面没跟**——同一仓位显示可见而风控不可见=口径分裂的最坏形态。

## 任务（修法供裁）

- `exposure_cap` 敞口计口径对齐统一持仓视图（fin_position_view 同径
  或 dcd_baseline 凭据 Σamt 加项——施工按现构裁最贴点位）；
- dcd 币量→USD 估值同 cockpit 径（最新 tick 折算，valCcy 同径）；
- **tick 缺价=fail-closed**：闸面估不出敞口=不许放（与显示面
  「无 tick=0 不虚计」相反——闸面保守取，读败/无价宁可拒放，
  语义钉票不许取显示面口径）；
- exec 仓原口径零动。

## 验收锚

1. 合成 dcd_baseline 凭据行→exposure_cap 敞口值含其 USD 估值
   （测试断言）；
2. 缺 tick 行情下闸面 fail-closed 实证（拒放非零计）；
3. exec 仓敞口计数零回归；
4. check_all 绿戳对口提交 SHA。

## 红线

- 不改闸参/阈数值/decision 语义——本票只改**敞口读径口径**；
- 不动持仓/凭据写径；venue 全零 mutating；
- fin_position_view 消费面已有先例（T-0351）——同源不新造。
