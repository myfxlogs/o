# T-0346 token 用量采集断供根修——新 worker 径接回 ops_ticket_usage

- 状态：完成（决策席裁决通过合 master@63bd06aa；锚①首条生产真 run 行留部署后验）
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

## 结果回填（施工席 codex-o-dev-20261104 @ 2026-11-04）

- 实现：新增 `scripts/meta/ticket_usage.sh`。解析 Devin `--export` transcript 的 `session_id`、`agent.model_name` 与三项 `final_metrics`；对同票+同 session 用 PostgreSQL advisory xact lock 读取上条账 note 中的累计 watermark，只追加新增 token 差分，重复采集不重复记账；计数回退时失败显形，不写负数。
- 施工席挂点：`dev_run.sh` 给每次 run 在该票私有 session 目录导出 transcript，模型返回后旁路调用 collector，记录 `kind=run`/`seat=施工席`/`source=devin-cli`。采集成功删 transcript；失败不挡 worker，保留受限 transcript 供恢复。
- 决策席挂点：`dm_tick.sh` 的 verdict Devin 批次同样 `--export` 并旁路采集，记录 `kind=audit`/`seat=决策席`。单票批次按票号入账；多票共用一次调用时只记一行 `META-DM` 并在 note 留票集，不把整批用量伪分摊到各票。失败不挡裁决，保留 transcript。
- 选源说明：未从全局 transcript 目录按文件时间猜票归属；实查 `devin list` 的 session id 与 transcript JSON 的 `session_id` 不同名（T-0345：`catkin-kicker` vs `local-watercress`），因此用每次调用显式 `--export` 到受限本地文件，读取文件自身 session_id，避免并行 worker 串票。
- 每席查询（一条 SQL）：

  ```sql
  SELECT split_part(split_part(note, ';seat=', 2), ';', 1) AS seat,
         sum(coalesce(prompt_tokens, 0)) AS prompt_tokens,
         sum(coalesce(completion_tokens, 0)) AS completion_tokens,
         sum(coalesce(cached_tokens, 0)) AS cached_tokens
  FROM ops_ticket_usage
  WHERE source='devin-cli' AND note LIKE '%;seat=%'
  GROUP BY 1 ORDER BY 1;
  ```

- 测试：`ticket_usage_test.sh` 首见 100/10/40、续跑累计到 150/17/55 只追加 50/7/15、重复快照零增量、回退 fail-closed、决策席 META-DM 归属 PASS；`dm_tick_usage_test.sh` 空/单票/多票/采集失败不阻断 PASS；`worker_runtime_test.sh` 限流续跑/采集失败仍收尾 PASS。各脚本 `bash -n` 与 `git diff --check` PASS。
- 文档：`docs/数据流.md` 已将预算账数据流从退役 wrapper 更新为 `dev_run.sh`/`dm_tick.sh`→`ticket_usage.sh`，同票/session 差分和多票决策聚合径对齐。
- 验收锚：② PASS（同 session watermark 差分干跑）；③ PASS（每席 SQL 如上）；④ PASS（两席 collector 失败均保留 transcript 且主流程继续）；⑤ `check_all` 全链 PASS：`bash scripts/check_all.sh`，结果戳 `check-green-cde4a9442a261dbcf7207712d4a44b142e367589`；任务单回填后将对最终 HEAD 重跑取戳。
- 验收锚① **未达**：需要合并/部署后的下一真实 worker run 向运行库写入三 token+model 实账。按业主指令本施工不合并、不部署、不写生产账；此项留决策席审计及部署后验收，不以 fixture 冒充生产行。
- 偏离：将 transcript 显式导出到权限受限的临时文件，而非从共享全局目录按 session 名猜文件；原因是实测 Devin 列表 id 与 transcript 内 id 不同名，且 worker 并行时共享最新文件存在错票风险。其余范围无偏离；未动历史账行/旧 wrapper/数据库 schema。
- 涉及面：`scripts/meta/ticket_usage.sh`、`scripts/meta/dev_run.sh`、`scripts/meta/dm_tick.sh`、三项 meta shell 干跑、`docs/数据流.md`、本任务单。
