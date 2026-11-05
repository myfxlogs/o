# T-0368 token 台账写径根修——psql -c 不替换变量

- 状态：完成（2026-11-05 决策席裁决通过——合 master @dfc06ef3）
- 领取：o-worker-T-0368（2026-11-05 领单，worktree /opt/o-wt/t0368，支 t/0368-psql-stdin）
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

## 结果回填（施工席 o-worker-T-0368 @ 2026-11-05）

- 实现：`scripts/meta/ticket_usage.sh` 喂径 `-c "$SQL"` →
  `-f - <<<"$SQL"`（stdin 径走 :var 词法层）；SQL 本体一字未动
  ——delta/去重/advisory lock 语义零动。两测试 psql mock 同步
  支持 stdin 读径（ticket_usage_test sql 空则 cat；dm_tick_usage
  query==` - `则 cat 且 USAGE_INSERTS 补记 SQL 本体）。
- 锚①实证：psql 16.15 本机同 DSN 复跑——
  `-c "SELECT :'ticket'"` → `ERROR: syntax error at or near ":"`；
  `printf | psql -v ticket=T-0368 -f -` → `T-0368`（值非原文）。
- 锚②实证：最小桩 transcript（session=t0368-verifystub）真库喂径
  → ops_ticket_usage id=94 落账 1000/50/400（note 携 watermark）；
  同快照复跑 → `no new tokens` 零行（去重语义在）；累计改
  1400/65/520 再跑 → id=95 落差分 400/15/120（delta 语义零动）。
- 锚③回归：meta 干跑全绿——ticket_usage/dm_tick_usage/
  dm_tick_worker_resume/dm_suspend/worker_runtime 五件 PASS，
  dm_tick usage 行桩侧 `inserted` 不再 write failed。
- 锚④：check_all 全链 PASS——实现件对口戳
  `check-green-2cbc6a2f9cf52f8dc0eafb152c9327ba3dc371cc`；
  票面回填后对收口 tip 重跑取戳在案（.git/worktrees/t0368/
  check-green-<tip>）。
- 历史缺口如实报：ops_ticket_usage 末行 2026-11-01 08:00Z
  （旧径断供 T-0346 已记 11-02 起）；新径 ticket_usage.sh 生于
  T-0346 cde4a944@2026-11-04 23:36+08——出生即全败。实证留存
  transcript：worker 侧 8 件（T-0353/T-0361/T-0362/T-0363×2/
  T-0364/T-0366/T-0367，均在 11-05）+ dm_tick 侧 15 件真
  transcript（16 个 usage 目录其一 transcript 缺）≈23 场采集
  写败≈23 条账行缺口；另 11-02→11-04 旧径断供窗缺口未单独
  清点（T-0346 票面已记十余场）。回填与否由业主另裁，本票零动。
- 坑处置：已入库 psql-c-variable-interpolation-absent（T-0348
  条目适用边界钉「-f 径不受此坑影响」），条目照用零改。
- 涉及面：ticket_usage.sh、ticket_usage_test.sh、
  dm_tick_usage_test.sh、本任务单——红线三件全零动。
