# 待业主裁决打包件 2026-11-04 o_fin_ro-condition-gap-grant

| 项 | 值 |
|---|---|
| 条目 | 源③/施工新证——T-0322 施工中发现：COS 收件箱源②「条件请求件」= `fin_condition_gap` 请求盒（D-0014/T-0177），但钦定 o_fin_ro 白名单不含该表 |
| 背景一句 | 白名单缺此行则 COS 源② SQL 上线即 permission denied——收件箱三源缺一角；施工提示词停线钉「勿自扩名单」，故打包上呈不擅补 |
| 选项 | A. 白名单补 `fin_condition_gap`（一行 GRANT 随续单迁移落）——收件箱三源闭环；B. 源②改读面（如 ops_event 镜像行——须各臂增写径，重）；C. 不动——COS 源②长期缺席，条件请求件回人肉转发（与本包减负目标相背） |
| 建议 | A——`fin_condition_gap` 是业主接口机制承载表，正是秘书席最该读的面；SELECT-only 白名单补一行零写权零凭据暴露，与 o_fin_ro 设计同源 |
| 证据锚 | `migrations/fin_0072_condition_gap.sql`（请求盒建表）；`docs/prompts/T-0322-COS秘书席施工-*.md` 停线条 + 收件箱三源口径；`internal/platform/condreq/condreq.go`（写径唯一）；本票回填段偏离说明 |
| 业主裁 | （空位——业主填） |

> 路由注：本件技术域归决策席（白名单补授非红线三件）——打包件「选项+建议+证据+裁空位」格式实证用真实条目；正案路由=升件行回决策席补裁（已落 ops_event `升件` 行，见 T-0322 回填段）。
