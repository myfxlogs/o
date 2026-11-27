---
id: dialect-canonical-namespace-identity
title: dialect.canonical 命名空间——借净名落行=策略 inst_id 永不可达
kind: pit
date: 2026-11-27
source: O-41379（O-41378 mint 臂 canonical=reg.Canonical 净名落行实证断层）
---

# canonical 借名落行断层（O-41379 根因层）

**坑形**：O-41378 mint 臂把 `fin_symbol_dialect.canonical` 落成
`reg.Canonical`（registry 净名 GOLD/BTCUSD）——但本系统唯一活命名空间
是**策略 `fin_strategy.inst_id`=venue 原符**（'USDSEK#'/'GOLD#'/
'XAUUSDm' 全谱在案）。执行径 `mtexec.Place→translate(leg.InstID)→
DialectStore.Lookup(canonical=inst_id)`——净名行对 inst_id='GOLD#'
**结构性永不可达**：mint 落得越勤，执行面越盲。

**次生互斥**：`fin_symbol_dialect` 双唯一键——PK(venue,account,
canonical)∧UNIQUE(venue,account,venue_symbol)。净名行与旧普查恒等行
（canonical=venue_symbol）同 venue_symbol 互斥，upsert 撞
UNIQUE(venue_symbol) 23505 实证炸 mint 拍（ON CONFLICT 只兜 PK 面）。

**修法（业主令 2026-11-27 钦定 a 径）**：
- `dialect.canonical = venue_symbol` **恒等式**——mintOne 落行键
  `Canonical: m.VenueSymbol`；`reg.Canonical` 降为验证归属族证物位
  （审计 note/evidence.registry/census.canonical_ref），不落键。
- registry 语义降级=「验证规范」（aliases 配对键+ctVal/family 对拍锚），
  「跨 venue 官名」是未实现的上游概念，当前无消费者——净名保留在
  registry.canonical 备将来 eval 出官名 inst_id 时复用。
- 手工面 verifySymbolMapping 不动（席位挑 canonical=inst 语义本就同源）。

**判别/推广**：
- 落行键先问「谁来查」——Lookup 的参键=inst_id=venue 原符，落行键
  必须与查键同命名空间；「更干净的名字」不是落键理由。
- 双唯一键表上 ON CONFLICT 只兜声明目标——另一唯一键撞=23505
  不兜，写前先画两键的互斥图。
- 恒等式验收=全谱 `canonical=venue_symbol` 一条 SELECT 可判；
  证物位（命中族/验证规范）走旁路列不占键。
