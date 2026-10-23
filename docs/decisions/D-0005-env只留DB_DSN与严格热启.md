# D-0005 env 只留 DB_DSN 与严格热启

| 字段 | 值 |
|---|---|
| 状态 | 生效 |
| 决策 | `[角色:决策]` @ 2026-10-23 |
| 批准 | 业主令 2026-10-23（原话：「只留 DB_DSN 就够了吧（仅解决鸡生蛋）？不要一堆东西都放到 env 里，我们需要严格的热启」） |

## 背景

nox D-515 收口后生产 env 剩 4+1 键（DB_DSN/MASTER_KEY/ENVIRONMENT/AUTH_TOKEN/ALERT_BRIDGE_TOKEN），~110 业务键已迁库。业主令再收紧到一键。

## 取舍

| 候选 | 取因 / 弃因 |
|---|---|
| env 只留 DB_DSN（选中） | 鸡生蛋唯一不可库化项；其余全 DB 化=改配置零重启 |
| MASTER_KEY 留 env | 弃——env 经 `docker inspect`/`/proc` 可读；主机密钥文件（600 权限挂载）更严 |
| MASTER_KEY 入库 | 弃——库解密钥入库=加密失效 |
| 测试 DSN/工具参入库 | 弃——被测库不能装自己的地址；ratchet 白名单保留此例外面 |

## 结论

- 生产 runtime env=`DB_DSN` 一键；`MASTER_KEY` 走主机密钥文件；`ENVIRONMENT`/auth/告警 token 入 `app_config`/`app_secrets`。
- **严格热启**：config client 唯一入口 + `LISTEN config_changed` + 代次列；禁 boot 一次性捕获；断线重连快照重载；读错 fail-closed。

## 连锁影响

- `工程纪律.md` env 面契约已按此重写；P1 schema 须含 `app_config`/`app_secrets` 代次列+notify 触发器设计
- `env-allowlist.txt`（T-0002 落位）白名单收窄至测试/工具入参
