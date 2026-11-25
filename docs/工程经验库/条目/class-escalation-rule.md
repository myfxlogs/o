---
schema_version: 2
id: class-escalation-rule
kind: ops
risk: high
status: verified
keywords: [举一反三, 同形升件, 族级票, 个案修, 完成态无主, 泛化闸, 处置纪律]
scope: [唤知会话, 处置席, 升件处置, 工单立案]
summary: 处置席纪律——同一形态升件≥2起时禁再修个案,必升档「族级票」(消整个病族);泛化前须过复审闸防泛错
first_seen: 2026-11-25
last_verified: 2026-11-25
skills: [工单规约, 探针]
evidence_refs:
  - type: server_fact
    source: postgres
    query: SELECT seat,count(*) FROM ops_event WHERE event='升件' AND ts>now()-interval '30 days' GROUP BY seat
    observed_at: 2026-11-25T02:00:00Z
  - type: doc
    ref: /opt/o-research/prompts/O-39400-desk-lifecycle.md
---

## 事实/症状

同形病在不同器官上逐个复发的实证（2026-11-25 诊断链）:
「完成态落册∧终末消费者未登记」一族四显形——
fin_econ_lesson verdicted 矿(38942修) / fin_opp_template_roster candidate
族卡(39330修) / ops_ticket_desk 未闭票(39400修) / fin_cashflow USDT
计价腿(39310修)。每处个案都修对了,但病族活到第四处才被归类——
纯个案修法=永远追自己尾巴。

## 判因

处置动作缺「类化」一步:会话问「这个洞怎么修」不问「这类洞长什么
样」。病族在个案修复中被隐性确认却无人登记,等于每处修复都在给
下一处复发作证。

## 处置

处置席/唤知会话升件处置纪律(举一反三件):

1. **同形计数**:处置升件时顺手判「近30d是否有同根因升件」——
   同根≥2起→禁修个案,转立「族级票」;
2. **族级票形**:spec 须含「类判据」(这族病的共同谓词)+
   「消族件」(出生闸/通用探针——拦于出生非捞于腐后);
3. **防泛错闸**:类票spec落盘前过复审——「这几起真同因吗」
   由复审签判,泛错泛化比不修更贵;
4. **个案照修**:族级票在途期间,当前升件的个案仍须处置
   (族票防复发,不代个案止血)。

## 反例/禁线

- 禁「形不似因似」强归——表象同形≠同根因,泛化闸存在即为此;
- 禁族票拖死个案——升件止血优先级不降(同禁线律:处置件先行);
- 消族件只拦「病族」不拦正常增长——判据须可区分病/健
  (如:正常积压=有主有SLA,病态积存=无主无销号径)。

## 关联

- O-39400 desk-lifecycle(消族件样板:出生终末消费者钦定闸);
- O-39410 desk-anchor(同族后续);
- 策略评估/教训矿 canon 同构先例。
