# T-0338 残仓工件面不可见——fin_position 零 open 与 cmc_positions 无 BTC-261225-95000-C

- 状态：草案
- 立项：`[角色:决策]` devin 无头拍 @ 2026-11-04（T-0333 部署窗验项⑤红项根——幽灵/缺席类实案立票）
- 级别：P2 定性件——先定性「残仓已灭」vs「工件面断供」再定修否

## 病灶（生产亲证 2026-11-04）

- `fin_position` `closed_at IS NULL` 行=**0**（22 行全 closed，
  o_fin_ro 径亲证）——生产无任何 open 持仓行
- `fin_cmc_positions` `BTC-USDT-261225-95000-C`/`%261225%`/`%-C`
  全 0 行（T-0333 实录；cmc_positions 表本席白名单外未亲证）
- T-0327 裁词挂「GetFlowPulse 工件面残仓行可见」锚——工件面
  上残仓不可见属实

## 定性二择（查后落票）

- **甲·残仓已灭**：2025-12-26 到期 BTC call 残仓在 T-0327→今
  时间差内被 sweeper/到期机制自然消费——工件面无物可显=正确，
  残仓类入集码面无需动，结案性归档
- **乙·工件面断供**：残仓实体仍在 OKX 侧（venue 实拍）但
  fin_position/cmc_positions/GetFlowPulse 链不显——数据面断供，
  trace 断点根修（T-0327 类级入集逻辑或采集臂）

## 任务

1. venue 实拍先行：OKX 侧 `BTC-USDT-261225-95000-C` 持仓今实存否
   （okx-account/positions 径——mt5/okx 能力账在 skills/）
2. 甲定性→票面如实落「残仓已灭」+销 T-0327 挂锚即结案；
   乙定性→断点 trace+根修落同票或升件裁修向
3. 工件面口径对拍：GetFlowPulse 残仓集定义 vs fin_position open
   口径——零 open 时工件面应显何物落字

## 验收锚

- [ ] venue 实拍实录（残仓实存/已灭——可复现出处）
- [ ] 甲/乙定性落票+（乙则）根修或升件在案
- [ ] GetFlowPulse 残仓工件面口径落字（防「该显无显」再误判）

## 范围外

- 残仓本身的处置交易（清仓/展期=钱径面——归业主裁不动）
- fin_cmc_positions 表级修法——乙定性后另票
