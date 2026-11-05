# T-0368 token 台账写径根修——psql -c 不替换变量

- 状态：已批（业主令 2026-11-05——「肯定呀，不然如何继续」；
  代谢判词数据源修复，根因已实证定位）
- 领取：（待派）
- 级别：P1 数据源——ops_ticket_usage=代谢判词成本腿，
  写径全败=评审缺腿

## 事实（实查在案，勿再诊）

- 症状：dm_tick 每次收口报
  `[usage] ops_ticket_usage write failed`（今日 T-0366 实证）；
- **根因已实证**：`scripts/meta/ticket_usage.sh:106-109` 把
  含 `:'var'` 引用的 SQL 经 `psql -c "$SQL"` 喂——
  **psql -c 不做 psql 变量替换**（:var 词法层只在 stdin/
  交互输入走），全部 `:'ticket'` 等原文直达服务端→
  `syntax error at or near ":"`；
- 实锤：psql 16.15 本机复跑——`-c "SELECT :'ticket'"`/
  `:ticket`/:`"ticket"` 三形全 syntax err 原文回显；
  同 SQL 经 stdin 喂→`:'ticket'` 正常替换得值；
- 后果面：ops_ticket_usage 全量写败——token 成本台账
  一直空记/缺记，T-0360 代谢判词（月机器成本 vs 策略
  收入）成本腿无数据。

## 任务

1. `ticket_usage.sh` SQL 喂径 `-c` → stdin 径
   （`printf '%s' "$SQL" | psql -v ... -f -` 或 `<<<"$SQL"`
   ——实现任选，变量替换须实证生效）；
2. 修后写一行实测锚（真 transcript 或最小桩）；
3. 顺手核对历史缺口：ops_ticket_usage 近期待填行数
   如实报（是否全败史需补注——回填另裁）。

## 验收锚

1. stdin 径变量替换实证：`:'ticket'` 等到值非原文；
2. 真 transcript 喂→ops_ticket_usage 行落账（delta 语义
   不动）；
3. 回归：dm_tick 收口 usage 行不再报 write failed；
4. check_all 绿戳对口提交 SHA。

## 红线

- 只动喂径——delta/去重/advisory lock 语义零动；
- 历史缺账不自动回填（是否补由业主另裁）；
- ops_event/其他台账写径零动。
