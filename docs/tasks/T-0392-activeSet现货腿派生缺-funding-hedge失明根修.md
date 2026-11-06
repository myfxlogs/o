# T-0392 activeSet 现货腿派生缺——funding-hedge 腿失明根修

- 状态：施工中（2026-11-06 DM 常径机派出工）
- 领取：devin-t0392 @ 2026-11-06
- 优先级：P1——钱径旁感知件（现无爆因 BTC/ETH 现货被
  dcd/basis 派生臂顺带喂着——巧合非设计）
- 收口移交归：决策席

## 病灶（代码实锤）

`internal/data/okxws/subscribe.go:75-83` activeSet UNION：
①fin_proposal/fin_strategy inst_id（=SWAP 键本身）；
②dcd uly→-USDT 现货对；③basis 期货键→-USDT 现货对。
**缺第四臂**：funding-hedge（short_perp_long_spot）的
`X-USDT-SWAP`→`X-USDT` 现货腿无派生——新开在 BTC/ETH 以外
的币→现货腿零 tickers→腿估值/预警/basis 供价失明
（carry_diff 同族=「有物无人喂」静默盲）。

## 范围

1. activeSet UNION 补臂：模板属 fundingHedgeFamily
   （short_perp_long_spot/long_perp_short_spot 向）的在仓/
   在途/live 策略 inst_id 派生 `regexp_replace(inst_id,'-SWAP$','')`
   ——归一链与 dcd/basis 两臂同语义；
2. 派生合法性：派生现货键 ∩订阅宇宙(isUSDTInst+valid 闸既有)
   ——虚构键（无该现货对上市的币）照 fail-closed 不订不炸；
3. 桩测钉死：funding-hedge 策略 inst=X-USDT-SWAP→activeSet
   含 X-USDT 断言；非 funding 模板不派生断言；虚构键滤除断言；
4. **腿-行情覆盖探针（防再发——业主令「不可能次次我来提」）**：
   挂 hungerTick 同槽（T-0384 族沉默探针同位），每拍对拍
   「live/paper 策略各腿所需行情键×md_ticks 有无新鲜行」——
   腿缺行情=ops_event 覆盖失明行+升件，只报不修（探针律）。
   腿键集=fundingHedgeFamily 派生现货∪inst 本键∪dcd uly∪
   basis 键——归一链复用 activeSet 同源表达式（禁自造第二套）。

## 验收锚

① 桩测三断言（派生/不派生/虚构滤）全绿；
② 生产谱面零误报径审计（派生面=dry 对拍可核）；
③ 覆盖探针桩测：腿缺行情→ops_event 行断言+有行情零误报断言；
④ check_all 绿戳对口；采集件随发窗批。

## 红线

- 只补派生臂——订阅节流/幽灵滤/退订语义零动；
- 不扩大基础宇宙（SWAP 全谱+FUTURES 现状不动）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
