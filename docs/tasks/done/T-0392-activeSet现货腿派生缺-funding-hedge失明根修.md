# T-0392 activeSet 现货腿派生缺——funding-hedge 腿失明根修

- 状态：完成（决策席裁决通过 merge 340dedd7——全链绿戳对口）
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

实现摘要：

- **派生臂唯一真源归并**：新包 `internal/platform/legset`——三族现货腿
  派生臂（basis/dcd/funding-hedge）+ `ActiveSetUnion` 全谱 UNION +
  `IsUSDTInst`/`DatedFutRe` 归一；族册归并 `execfam.FundingHedge`
  （lifecycle opener.fundingHedgeFamily 本地册删除，opener/retire 同册
  读）——「归一链复用/禁自造清单」结构面落实。
- **订阅面第四臂**：`okxws/subscribe.go` activeSet 改读
  `legset.ActiveSetUnion`（fundingHedge 族非 retired 策略 inst_id 剥
  `-SWAP` 尾→现货对——在仓/在途/live/paper 全覆盖退役即出）；加订
  闸 `tickGate` 提取可测面（isUSDTInst∩valid∩refused∩subTick 语义原样，
  虚构键照闸滤除不订不炸——节流/幽灵滤/退订语义零动）。
- **同病灶复制面同臂补齐**：`projector.refreshSets` 镜像谓词同补
  funding 臂（原归一双臂改为 legset 三臂同源——只补订阅面=桥端仍滤
  在桥外，经验库 bridge-predicate-misses-normalized-legs 在案同病灶）。
- **腿-行情覆盖探针**：`hunger.ProbeLegCoverage` 挂 hungerTick 同槽
  （ProbeFamilies 同位）——腿键集=ActiveSetUnion 同串 ∩ okx.symbols
  宇宙 ∩ IsUSDTInst，md_ticks.ts>10min 新鲜行对拍；缺行情腿逐键
  `覆盖失明` 行+观测盲升件（24h 按键去重），只报不修；宇宙=* 形
  fail-closed 让位报错不虚构。
- `docs/数据流.md` 谓词段同 commit 更新；经验库条
  bridge-predicate-misses-normalized-legs 升格（结构防漂移）。
- import 白名单 +4 边（okxws/hunger/projector→legset、legset→execfam，
  全平台层单边）。

测试证据：

- 先红桩：两断言修前 FAIL（AAA-USDT/FAKE9-USDT 未派生——病灶钉死）；
- 锚①三断言全绿（派生/非族零派生/退役零派生/虚构键闸滤）；
- 锚③探针五测全绿（盲腿行+升件/双供零误报/宇宙外零误报/陈行盲/
  24h 去重）；接线测每拍调+败情隔离升件绿；
- projector TestActiveSetNormalizedLegs 扩钉 funding 臂（DOGE-USDT 入集/
  ADA-USDT 退役离集）；execfam 域判据测改读同册绿；
- check_all 全链绿戳对口（机检记录见收口段）。

偏离说明：

- 补臂状态谓词取 `s.state<>'retired'`（票面「在仓/在途/live」——
  paper/probe/proposed 同含，与 dcd 臂/镜像谓词同律；退役即出零供）；
- projector/LocalWrite 侧：refreshSets 同臂补齐（同归一链复制面——
  不补=SG md_ticks 仍零行，探针即自报）；cmd LocalWritePred 谓词
  （position∪策略裸 inst）**未动**——SG 本地写窄域系既有钦定形
  （basis/dcd 现货腿同未含），现货腿 SG 落行正径=桥投影面，记此申报
  交决策席裁（若钦本地写亦含派生腿，另票对齐）。
