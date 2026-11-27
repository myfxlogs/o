---
id: dialect-same-name-different-contract
title: 同名不同券——canonical 锚定选型与「每族一行」确认粒度
kind: pit
date: 2026-11-27
source: O-41378（XAU-USDT-SWAP=0.001oz 微型张 vs XM GOLD#=100oz 标准手 实拉对拍）
---

# 同名不同券（O-41378 四件套根因层）

**坑形**：canonical 官名锚在 OKX instruments 唯一源时，CFD 原生品种
（GOLD/SILVER/EURUSD/指数）OKX 无品→`verifySymbolMapping` 硬验根上验不了；
反向更毒——OKX 有金系品 `XAU-USDT-SWAP` ctVal=**0.001 XAU** 微型永续，
XM `GOLD#` ctVal=**100 XAU** 标准手——同名 underlying 面值差 10⁵ 倍。
拿它当 GOLD 锚=单位错配病自动化（approxEq 判据正是为此设的防线）。

**修法（四件落地）**：
- `fin_canonical_registry`：确认粒度「每 venue-symbol 一击」→「每族一行」
  （席位登记 canonical+family+ct_val+quote_ccy+aliases；机器只读，
  席位维护=迁移种子+运维行）。别名=显式配对键字典，禁模糊匹配。
- `fin_venue_symbols` 普查表：各平台 SymbolList+SymbolParams 全量落库
  （业主令②——人工分不清合约只能错配，机器分拣）；family_guess 机分类
  只收窄别名匹配域，落闸判据=参数对拍一件。
- mint 拍=表上纯函数（census×registry）：别名命中→参数对拍（判据
  mt5.* 单源与手工 verify 同尺）→GetQuote 探活→
  UpsertVerifiedWithAudit(actor=auto-mint)；败=flagged+map_reason
  落库可查，registry 补登记下拍自愈。
- verifySymbolMapping 参源泛化 registry∪OKX 双源——手工/自动同尺。

**判别/推广**：
- 「无某能力」结论前先查参源是否单源——OKX instruments 锚 CFD 官名
  是结构性盲区非数据缺；登记表=语义字典层补位。
- ct_val 种子必须 venue 自报表实返（survey JSON contract_size），
  crypto 族与 OKX SWAP 微型形各是各的 canonical 不互译。
- 配对键三型：base 单名形（GOLD#→GOLD）∪ base+quote 对形
  （EURUSD#→EUR+USD→EURUSD——FX 必须带 quote 否则 EUR 跨对误配）
  ∪ suffix m尾剥（XAUUSDm→XAUUSD）。
- 「每族一行」复用面：registry.aliases 撞多 canonical=alias_ambiguous
  如实 flag——同基异券的语义分岔由字典行数解决，靠模糊匹配=把语义
  事故自动化。
