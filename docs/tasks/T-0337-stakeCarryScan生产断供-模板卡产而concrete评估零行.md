# T-0337 stakeCarryScan 生产断供——模板卡产而 concrete 评估零行

- 状态：草案
- 立项：`[角色:决策]` devin 无头拍 @ 2026-11-04（T-0333 部署窗验项④红项根——断供类实案立票；同源先例 T-0326 升件行 id1593）
- 级别：P1 断供类——「第 1 分钱」通路生产端末节管道堵

## 病灶（生产亲证 2026-11-04）

`fin_opp_eval` `cmc:TRX` **0 行**（2h/全窗皆零——决策席 o_fin_ro
径亲证复算一致）；同库 `TRX-USDT-SWAP` 2 行 rejected 在案——
stakeCarryScan 臂活（升件行 1593：模板卡 `cmc/stake_carry_template`
@04:56 已产）但 **concrete 评估行零产**。

dev 链标定值 `0.12833773338730112`（T-0320 锚④桩测在案）——
模板卡到 concrete 的转化段在 dev 通、生产断，属产线面断供
（T-0332 谱系「断供」类：臂活产出=0 持续）。

## 任务

1. trace 模板卡→concrete 段在生产断点：stakeCarryScan 枚举源
   （staking 注册/cmc 实收/TRX 质押率三件套）在生产侧的输入面
   逐个对拍——哪一路缺席/静默
2. 根修至生产 `fin_opp_eval` 落 concrete `cmc:TRX` 行（右元对拍
   dev 标定值量级）；若根=生产输入缺席不可修（如 cmc 实收源无
   TRX staking 数据）→如实定性落票+拒因族码可溯
3. 断供类伴生量表随票（T-0332 模板新规：坑必修伴生表）——
   候选=「eval concrete 产出臂级计数」入 gauges.tsv

## 验收锚

- [ ] 生产 `fin_opp_eval` `cmc:TRX` concrete 行实录（或断点定性
  实录+不可修由）
- [ ] 断点归因落票（三件套逐项对拍值）
- [ ] 伴生量表入册（断供类——臂产出=0 持续即触）

## 范围外

- stake-carry 执行径建设（execfam 未在册族——开径归业主裁）
- dev 侧链（dev 桩测已绿——本票专治生产面）
