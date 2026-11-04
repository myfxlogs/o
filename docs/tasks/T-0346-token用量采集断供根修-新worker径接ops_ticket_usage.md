# T-0346 token 用量采集断供根修——新 worker 径接回 ops_ticket_usage

- 状态：施工中（业主令直派，代 DM）
- 级别：P2 观测面——额度账全黑=配额决策无据（owner_interruption/预算域食粮）
- 领取：codex-o-dev-20261104 @ 2026-11-04（worktree `/opt/o-wt/t0346`，branch `t/0346-token-usage`）

## 病灶（实查在案，勿再诊）

`ops_ticket_usage` 表在（T-0267 件 A 钦定面：票→token 映射落 ops 表），
**写手死在旧径**——用量上报曾挂在 `devin-headless-run.sh`/`audit.sh`
wrapper 收口段（source='atif'，末行 2026-11-01 08:00Z）。T-0324 换
`dev_run.sh`+systemd-run、T-0343 换 per-票 session 续跑径，两次重构
均未带采集钩——11-02 起全部 run 账上零行（今日 T-0334/0339/0340/
0342/0343 十余场 SWE-2 无一行）。

数据在：`~/.local/share/devin/cli/transcripts/*.json` 逐 session 记
token 总数（local-watercress 实证 total_prompt/completion/cached 齐）；
T-0343 又给每票独立 session 目录+`devin list --format json` 发现径——
票→session→token 映射天然干净。

## 任务（修法供裁，施工据实选源并记取舍）

1. **采集钩接回新径**：`dev_run.sh`（或 worker_runtime）收口后步——
   在票 session 目录取 session 标识 → 解析对应 Devin transcript 取
   `total_prompt_tokens`/`total_completion_tokens`/`total_cached_tokens`/
   `model` → 与票内上次记录做 **watermark 差分**（同 session 续跑=累计
   值，防 resume 重复计）→ `INSERT ops_ticket_usage`（ticket/kind=run/
   model/三 token 列/source='devin-cli'）。决策席 verdict run 同径接
   （dm_tick 的 devin 调用面——施工裁挂点）。
2. **每席可读**：席列（seat）落行或给出「票→席」聚合读面——业主问
   「每席用多少」须一条 SQL 可答（施工裁列式或视图式）。
3. **静默纪律**：采集失败不阻断施工/裁决（旁路记 relay+ops_event 异常
   行可选），采集本身零 LLM。
4. 弃用面注记：OTEL `api_request` 事件径（config 未配）记为备选不接。

## 验收锚

- [ ] 下一真实 worker run（T-0344/T-0345 即候选）落 `ops_ticket_usage`
      行——真值非桩，三 token 列+model 齐
- [ ] 同票 resume 续跑差分记账——累计值转增量零重复（watermark 实证）
- [ ] 「每席用量」一条查询得：施工席/决策席分列汇总实录
- [ ] 采集败=施工不挂（干跑断 transcript 路径实证仍正常收口）
- [ ] `check_all` 绿

## 红线

- 不动 `ops_ticket_usage` 历史行（append-only）
- 采集不引入重依赖（jq/psql 级即可；OTEL 备选非必选）
- 不回改旧 wrapper 径——采集挂点是**现行** launcher 面
