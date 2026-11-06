# T-0392 activeSet 现货腿派生缺——funding-hedge 腿失明根修

- 状态：已批（2026-11-06 业主问实锤——「如果有策略出来了，
  会不会造成没有报价的缺陷？」COS 代码取证=真缺口）
- 领取：常径待派（DM 常径机派）
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
   含 X-USDT 断言；非 funding 模板不派生断言；虚构键滤除断言。

## 验收锚

① 桩测三断言（派生/不派生/虚构滤）全绿；
② 生产谱面零误报径审计（派生面=dry 对拍可核）；
③ check_all 绿戳对口；采集件随发窗批。

## 红线

- 只补派生臂——订阅节流/幽灵滤/退订语义零动；
- 不扩大基础宇宙（SWAP 全谱+FUTURES 现状不动）。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
