---
schema_version: 2
id: env-multi-dsn-different-instances
kind: pitfall
risk: high
status: verified
keywords: [env, DSN, 多实例, META_PG_DSN, FIN_PG_DSN, 角色授权, 表名同形, 写错库, fin_condition_gap]
scope: [/opt/o-research/env 消费面, scripts/meta/*.sh, 跨库写径]
---

# 坑条目：env 多 DSN 指不同实例——表名同形写错库零报错

## 实证（2026-11-07 T-0438）

`/opt/o-research/env` 五个 DSN 非同库：`FIN_PG_DSN`=roster_rw@127.0.0.1:5544
（隧道→SG 正本 172.22.0.3:5432 db=o）；`META_PG_DSN`=o@127.0.0.1:5433
（**本地另一 PG 实例** 172.23.0.3）；`MD_PG_DSN`=:5435（第三实例，db 还不存在）。
两实例都有同名表 `ops_event`/`fin_condition_gap`（副本/测试残留）——T-0438
送达闸初版抓 `META_PG_DSN` 当「role o 箱全权径」，实写的是 :5433 副本库：
`INSERT…SELECT FROM ops_event WHERE id=3268` 在那边零行匹配，**INSERT 静默
零行 RETURNING 空、rc=0**——写错库不炸不报，差点整段回填落空。

## 病根

① 多实例共存+表名同形：SELECT 查得到表结构就以为在对的库；
② `INSERT…SELECT` 源侧零行=零行插入=成功语义（rc0），无「你本来想插一行」断言；
③ 角色名直觉误导——`META_PG_DSN` 用户是 `o`（superuser）看着像「正本全权」，
   实际只全权于副本。

## 律

① 跨库写径先对身份三件套：`current_user`+`inet_server_addr()||':'||inet_server_port()`+
   库内锚行存在性（如正本独有的判词行 id）——三件齐才动笔；
② `INSERT…SELECT` 回填类必须 `RETURNING` 且断言返回行数=预期行数
   （零行=源侧失配，fail 而非成功）；
③ DSN 取名≠库域：`META_`/`MD_` 前缀是消费域不是库域——SG 正本 fin 库的
   业主域写径唯一容器内 role o（o-core `DB_DSN`），主机面 meta 壳件写正本
   业务表走 `FIN_PG_DSN`+迁移柱级授权（fin_0113/fin_0139 形），禁抓
   `META_PG_DSN`/`MD_PG_DSN` 跨写。

## 教训

「连上了+有这张表」≠「在对的库」。多实例环境下每一个跨库写径都要一句
身份断言开局；回填类 SQL 的成功判据是 RETURNING 行数不是 exit code。
