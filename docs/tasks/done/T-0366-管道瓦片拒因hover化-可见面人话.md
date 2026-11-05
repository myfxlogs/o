# T-0366 管道瓦片拒因 hover 化——可见面人话

- 状态：完成（2026-11-05 决策席裁决通过——合 master @4f8aa239）
- 领取：o-worker-T-0366（2026-11-05 领单，worktree /opt/o-wt/t0366）
- 级别：P3 显示面修缮——T-0362 骨架反馈件

## 事实

- 提案段现显：`提案·闲·评估全拒（末因 net_apr<阈）`——
  括号内为评估臂内部词（net_apr=净年化/阈=族阈值），业主
  面读不懂；
- 长括号串把瓦片撑错位（布局病同诊）；
- T-0362 骨架已含状态点/三段式，此件=拒因呈现层精修。

## 任务

1. **可见面人话短化**：拒因括号改人话短句——如
   `评估全拒（饿殍期·无达标机会）`/`无达标机会`——
   业主扫一眼得结论不发术语（短句映射表静态钉死：
   net_apr<阈→无达标机会 等常见拒因各配人话）；
2. **悬停也是人话（业主令二件 2026-11-05——「前端是人
   在看不是机器在看」）**：机器原文**禁入前端任何层**——
   hover 显示=人话详述（可读句子+数值，如「评估 47 个
   机会，最高净年化 3.1% 未达阈 6%」——映射器把机器串
   翻成人话，数值参数带进详述）；`net_apr`/阈 等原文只
   留 DB/日志面；
3. **错位修**：短化+hover 化后瓦片不得被拒因串撑宽/
   换行错位（七段等宽布局不破）。

## 验收锚

1. 可见面无 `net_apr`/`阈` 等技术原文，人话短句在；
2. 悬停显**人话详述**（可读句+数值）——DOM 断言机器
   原文 `net_apr` 等不出现在渲染树任何层；
3. 瓦片宽度/对齐不随拒因长度错位（不同拒因长度下渲染锚）；
4. check_all 绿戳对口提交 SHA。

## 红线

- 纯渲染层——proto/RPC/查询/映射数据源零动；
- 人话映射表静态钉死随代码走，不接 LLM；
- 七段结构/状态点/判词行零动——只修拒因呈现。

## 顺手件（worktree 域）

- 施工同支落 KB 条目 `cockpit-owner-face-plain-language`
  （业主令「cockpit 可见面=人话面，机器原文不入前端任何
  层」原则入册——部署树 D-651 禁 skills/索引写故归本票
  worktree 径，kbgen gen+skills 双索引重建同收口）。

## 施工回填（2026-11-05 收口）

- **面**：web/src/react/components/pipeStatus.ts（新——humanStatus
  单一翻译闸：态标串→{可见短句,悬停详述}；rejectPlain 静态映射
  net_apr→无达标机会/below_earn→不及保底收益/basis_below_min→
  基差未过线/apr_est_missing|missing→缺数据/stale→数据陈旧/
  liquidity→深度不足/unprofitable→算不过账；armed→在飞提案、
  （阈未达）→（未过放行线）；machineLeft 哨兵=snake_case+armed+阈
  残留即整句兜底「评估未放行/评估判据未放行」——未识词不露原文）；
  flowpulse.tsx（mid/verdict/title 三面过闸，evalCount=评估段
  count24h 同源进详述「评估 N 个机会全部未过放行线」——数据源
  现字段零新增）；20-cards.css（pipe-card flex:1 1 0+min-width:0
  七段等分，num/lag ellipsis+nowrap 长串截断全文走悬停）；
  flowpulse-domtest.mjs（⑥人话面 7 锚——可见短句/机器原文禁入
  渲染树任何层/悬停详述带数值/执行段态词/活态/未识兜底/等宽截断，
  ④饿殍期判词预期随人话化改「未过放行线」）。
- **顺手件**：docs/工程经验库/条目/cockpit-owner-face-plain-
  language.md（invariant——哨兵兜底原则+边界注 DB/日志面不受束）；
  kbgen gen 重建索引.md 落行（skills 索引已当前零漂移）。
- **自测**：web npm test 23/23 PASS（flowpulse-domtest 全锚+metaseats
  12/12）；tsc --noEmit 绿；vite build 绿。
- **红线**：纯渲染层——proto/RPC/查询/映射数据源/七段结构/状态点/
  判词行派生语义零动；映射表静态钉死不接 LLM；机器原文只留 DB/日志。
- **观察项**：服务端态标词集扩展时 rejectPlain/machineLeft 词表须
  同票同步扩（KB 条目反证条件注记）。
