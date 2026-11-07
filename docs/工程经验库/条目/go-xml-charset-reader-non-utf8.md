---
schema_version: 2
id: go-xml-charset-reader-non-utf8
kind: pitfall
risk: high
status: verified
keywords: [encoding/xml, CharsetReader, windows-1252, cp1252, ffcal, ForexFactory, 静默 parse_err, 非UTF8声明, 同坑双面]
scope: [research, ctx, xml 解析, 公开面采集]
summary: 上游 XML 头声明 windows-1252（ForexFactory 周历实测）时 Go encoding/xml 无 CharsetReader 直接报错——采集面 Unmarshal 败走 parse_err 静默分叉=源零行史死无人知；修法=Decoder.CharsetReader 显式挂（x/net/html/charset.NewReaderLabel）；同坑双面复发案——修坑必须全仓扫同形面
first_seen: 2026-10-27
last_seen: 2026-11-07
evidence_refs:
  - type: ticket
    ref: T-0183
  - type: ticket
    ref: T-0429
  - type: commit
    ref: t/0429-ctx-inlet
---

## 定案

`nfs.faireconomy.media/ff_calendar_thisweek.xml` 响应头 `<?xml version="1.0"
encoding="windows-1252"?>`（curl 实测 200/40KB）。Go `xml.Unmarshal` 遇非
UTF-8 声明且 `Decoder.CharsetReader=nil` 即返
`xml: encoding "windows-1252" declared but Decoder.CharsetReader is nil`。

## 同坑双面复发案（本条核心教训）

- **首见 2026-10-27（T-0183）**：o-core `notify/calendar.go`（财经日历
  飞书推送面）24h 恒败高影响事件预推全灭——CharsetReader 接入修绿。
- **复发 2026-11-07（T-0429）**：同源同坑在研究侧 `research/ctx/collect.go`
  （md_ctx_fact 采集臂）**漏扫未修**——parse_err 静默分叉（零日志零行零
  升件），ffcal 自上线双表零行史死，靠票面勘查才浮出。
- **律：修坑必须全仓扫同形面**——同 URL 同解析形的多消费面，一处修了
  其余照病。T-0183 收口只修了撞见的面，未 grep 同端点消费位（`ff_calendar`
  全仓两处）。

## 修法（两面包围现形）

`xml.NewDecoder`+`CharsetReader=charset.NewReaderLabel`（x/net/html/charset
——T-0183 同款，全 label 覆盖，不另造轮）；伴随拉取败/解析败 log 灯行
（T-0311 返修⑤同形）。

## 教训


- 上游换头（编码声明）对采集臂是**隐形破坏**——XML/JSON 解析败必须带
  灯行+行龄探针双保险，纯 rej 计数（内存门）对断供不可见。
- 「解析败静默 continue」只该用于**行级**容错（坏行跳过不拖拍），**源级**
  解析败（整文件拒读）必须显声——源级静默=断供同形。

## 证据

- check: /tmp 探针复刻（xml.Unmarshal 对 windows-1252 声明 payload）；
  OCI curl 实拉 40386B 真件头；测试 TestFFCharset1252 修前红（no rows）
- result: 修前 parse_err 零行；挂 CharsetReader 后 High/Medium 双落行
- ref: T-0183（notify 面）；T-0429 收口 commit 链（ctx 面）

## 反证条件

- check: ffcal 连续两周 ctx_fact 周级新行
- falsifies_when: FF 换 UTF-8 头且解析恒通=本条退化为历史案（CharsetReader
  位留置无害——未知 label 路径仍在）
