# O-41371 funding-hedge 全族禁喂裁定复审 — 审计报告

> 审计票非修码票：三问取证→判词三选一，禁直改规约行（裁定面归裁决席，本报告只出判词+证据链+规约处置建议）。
> 取证日：2026-11-26。证据面：fin_econ_rule/fin_strategy/fin_opp_eval/fin_proposal/fin_venue_untradeable（MD 正本+SG 镜像双面）+ops_event 裁定/判词/止血全链+修件 merge 实证。

## 案底

`fin_econ_rule` active 规约五行全拦 funding-hedge：

| rule# | match | 裁定 | 立案 |
|---|---|---|---|
| 1-4 | `{template_ref:funding-hedge, inst_id:四untradeable键}` | #16054-16057 | 2026-11-14 |
| 5 | `{template_ref:funding-hedge}` 全族 | #22452 | 2026-11-17 |

提案 622035（NES-USDT-SWAP short_perp_long_spot net_apr=+16%）实证 `fin_proposed` 滞留>econ_rule 拦下行在案——闸门机械执行无 bug，本案问「裁定本身还该在吗」。

## 取证一问：裁定当日的病灶是什么

裁定谱两层：inst 级四张（11-14，四 venue 死键 SHEIN/XPD/KSTR/CSOPSKHYNIX2L——`fin_venue_untradeable` code=51001「Instrument doesn't exist」实锤）先于族级一张（11-17，止血令 #22349→裁定 #22452→反射 #22453→rule#5）。族级病灶=**三重坐实**（自我判词 #21967/#21968 原文）：

1. **闸幻影（eval 成本低估 ~5 倍）**——#21129 外席实测：friction 计量只入 28 行 position_ref 非空费行（$0.3244÷$1400=2.32bps），同批真成交 95 次在 `fin_venue_bill` 合 $1.1232——约 $0.80 费行 position_ref 为空未归仓；`maker_fee_bps` 键缺 app_config→双 taker 兜底高估另一路。叠加 #19776 时代错配幻影：旧制 `fin_opp_eval` upsert 覆写，`pr.eval_id` 行被续拍刷新——「入场信念」实为入场后重估（新制 net≈-34% vs 旧读 +7.5% 全属旧制 cohort）。产物=幻影肥 proposed 卡（史峰值 ZEC +64.3%）喂出必亏策略。
2. **真亏 churn（armed 壳现货腿 ~12h 轮 churn）**——19 条 live 态 funding-hedge armed 壳零敞口设计态，每 ~12h 现货买卖秒平，费+滑点每轮实亏（仓88/89 -0.73/-0.30；30d `fin_cashflow` 现货往返 -4.74 即 stall 本体分解项）。病灶链=幻影 mint→薄真实 edge<churn 成本→慢性失血。
3. **尾险（churn 触残留清扫 TOCTOU→裸腿）**——仓#86：funding-hedge 对冲币被 `residual_clear` TOCTOU 当杂币扫→close sizer 读 spot≈0 fail-closed→SWAP 短腿裸持 venue。真尾险事故非理论。

inst 级四张病灶=venue 死键产幻影提案（perp-only 合成器无现货腿，肥数字全是空气——T-0150 实案）。

## 取证二问：病灶现在还在吗

**三成分全有部署实证修件，inst 级已被机械闸覆盖——病灶不在。**

| 成分 | 修件 | 部署实证 |
|---|---|---|
| 闸幻影·成本低估 | #21129 摩擦费测量根修（腿级 fee 测量列+按仓归集换源，merge `51072157d` 11-18）+#19797 eval 成本侧三分量实测合流（friction 实测替声明+MeasuredHoldDays 中位段+无段 fail-closed，merge `08a83dbef` 11-21，复审过 `280105d39`） | 修后 eval 证据面实测：cost_rt 13.2-13.6bps（修前 17.16 声明值）、hold_days=实测中位（NES=1.0007 vs 声明 3）、cost_apr 0.16-0.50 诚实口径 |
| 闸幻影·时代错配 | md_1097 eval insert-only 版本史（工单#19778——upsert 灭史病灶实案 -14.4% 预测实为入场后重估 -34%）+#19776 探针 era 守卫 | fin_opp_eval 自然键 UNIQUE 撤、每评估一版 id 递增=入场信念成史可查 |
| 尾险 TOCTOU | #21723 residual_clear TOCTOU 根修（部署 `f63e9be9b78c` 验销）+#21745 sizer 三态修+#22247 紧急出口 reduce-only 第二径（#22294 完工） | 仓#86 事故链全闭（#22124 判词：两修支并部署+closed_at 落账） |
| churn 机制 | 病灶链上游=幻影 mint——修件在 mint 闸：诚实 eval（上）+T-0521 毕业闸三件（friction 记账/confGate 族分流/entryCondition 进入条件判词）+spotLegOK 起草闸 | churn=经济症状非独立机械 bug——幻影源断则 churn 动机断；现存 armed 壳已随止血令全轮退（872 条 retire_postmortem 判词「设计态死亡非教训」在案） |
| inst 级四键 | `spotLegOK` 起草闸（persist.go:22-54——黑名单+现货腿目录实有双证，读败 fail-closed）+`fin_venue_untradeable` 在册 | 四键全在册 code=51001——proposed 卡在落卡前即被 `no_spot_leg`/`blacklisted` 机械拒，规约行=冗余防御纵深 |

**复开径前置条件核对**（止血执行回执 #22455 原文：「复开径=解禁裁定行（修票#19797/#21129 落地后）」）：两修票均已 merge+部署在案——前置条件成立。

## 取证三问：机会成本账

**禁喂期（11-17→11-26）eval 臂产出谱**（MD 正本面）：

- 全史 proposed eval 仅 8 版：NES×4（11-18/11-19/11-25/11-26）、OFC×1（11-19）、BAND×3（11-23）；rejected 770,306 版——eval 臂全谱扫描 ~531 inst 诚实拒绝面。
- **修后（post-#19797 merge 11-21）proposed 5 版**：BAND×3（11-23，net_apr=+19.76%）、NES×2（11-25 +17.00%、11-26 +16.00%）→提案 479252/564802/622035 全部滞留 `fin_proposed`/`fin_retired` 被 rule#5 拦下。
- 卡片口径诚实声明：snapshot 评估（当前 funding_rate×settles_per_day×365+质押分量−实测成本摊销）——**regime 盲**（费率反转无建模，结构性非缺陷）；NES hold_days 实测=1.0 天中位=快速轮动假设下 cost_apr≈0.50 仍净正=费率真肥非口径幻影。

**历史 lethal 尸案净额对拍**：

- 1823 条 funding-hedge 策略全 retired——**全入旧制 cohort（10-24~11-13），新制零入场**（#19776 实证）；872 条 retire_postmortem 判词一致：「弃——设计态死亡非教训：positions=0 armed_ttl_expired 同窗代谢」——**尸案=TTL 设计态代谢非经济死刑**。
- 真经济亏=19 条 live armed 壳 churn（现货往返 -4.74/30d+费）+仓#86 裸腿事故——亏源=幻影 mint 的薄 edge 策略，非族机制本身。
- 机会成本：禁喂 9 天内 NES/BAND/OFC 复现肥卡被白拦——$50 名义 paper 级敞口预期（proposal→paper→probe→live 毕业链全闸不变，live_switch 委员会开关仍在）；对比史尸案净额：禁喂期零新增亏损（churn 源已断）vs 放行期预期=诚实 eval 下的 paper/probe 观察敞口。

## 判词：**(b) 病灶已修=可解禁**

三证径（#25562 外席预挂框架）逐项在案：

1. **新口径判词 ✓**——friction_at 实测刷新（#21129）+#19797 三分量合流后，funding-hedge proposed 行持续产出（BAND 11-23×3、NES 11-25/11-26，net_apr>min_net_apr=0.06 阈）——eval 臂在诚实成本下仍判肥=非口径冤杀残留。
2. **开闸放行/breakevenOK ✓**——#19797 钦定「T-0521 同源补臂」：eval cost_apr 与 opener breakevenOK 同一成本地基——post-calib proposed 卡按构造即过 breakeven（离线推演不必真撤：BAND cost_rt=13.22bps 实测+hold_days=3 中位→cost_apr=0.1608，gross_apr=net+cost=0.358→净 19.8% 过 Earn 动态底）。
3. **冤案证 ✓**——史败构成=设计态 TTL 代谢（872 判词一致）非经济死刑；真亏源=幻影 mint churn，幻影源（成本低估+时代错配）双修件部署在案。新制 eval 下 mint=诚实卡，冤案命题不成立。

**残余风险诚实声明**（不构成续禁理由，列档备裁决席参考）：

- snapshot eval regime 盲：费率反转风险结构性存在——由 paper→probe→live 毕业链（min_hours/net/maxDD+confGate+entryCondition+execDrill+shapeGate+live_switch）兜底，非 eval 单点责任。
- churn 机制无独立修件：经济症状随幻影源断而失动机；若解禁后诚实 eval 仍产 churn 型卡（edge<churn 成本），paper/probe 观察窗会如实记负——毕业链即第二道闸。
- inst 级四键=venue 死键（51001），规约行与 spotLegOK+untradeable 机械闸冗余——撤与不撤皆无功能差，建议同步撤（规约面归真=只留有独立语义的行）。

## 规约处置建议（裁定面归裁决席，本席只建议）

| 规约 | 建议 | 理由 |
|---|---|---|
| rule#5 全族 admit_block | **active=false 撤块** | 三成分病灶全有部署实证修件；复开径前置（#19797/#21129 落地）已满足；解禁=提案回 paper 开户径，live 毕业链不变=分阶段敞口非裸放 |
| rule#1-4 inst 级四块 | **同步撤（active=false）** | 病灶族内通用（幻影提案）已修；四键 venue 死键由 spotLegOK+untradeable 机械闸覆盖=规约冗余；逐张判词同（b）——独立判=同结论非连带 |

**附带产出核对**（spec 条款）：判词=(b) 解禁→eval 臂继续正常产卡且卡片恢复可行动性=**无需降频**（「不产不喂」建议仅在续禁情形下适用，本判词不触发）。若裁决席否决本判词维持禁喂，则建议 `research.funding.scan_interval_min` 降频（产卡白耗资源）——此为条件分支非本判词主张。

**禁线自查**：本报告零改 `fin_econ_rule` 行（裁定面归裁决席）；判词链到原病灶修件部署实证非「eval 卡看着肥」；inst 级四张逐张独立判（同结论各带独立证据）；无代码/规约/迁移改动——纯取证+判词文档件。
