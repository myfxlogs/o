# T-0356 vol-sell 复役证据与启用规程

施工留档；生产操作只在本文件所列安全顺序后由运维执行。任务票面保持不变。

## 根因证据（锚①）

- 只读命令：`psql "$FIN_PG_RO_DSN" -X -c "BEGIN READ ONLY; SELECT template,max(ts),count(*) FILTER (WHERE ts >= date '2026-10-30') FROM fin_opp_eval WHERE template IN ('vol_sell_single_call','vol_sell_strangle','dcd_call','dcd_put') GROUP BY template; SELECT venue,surface,enabled,endpoint_family FROM fin_capability_surface WHERE venue='okx' AND surface='dcd'; ROLLBACK;"`
- SG fin 只读结果：dcd/volsell 四族最后 eval 均在 2026-10-30；单 call 8573 行、strangle 1782 行、dcd_call 8677 行、dcd_put 7973 行。其他族仍有 2026-11-05 新 eval（funding-hedge 630 行、两 basis 族各 3 行）。
- 同次 SG `o_fin_ro` 目录面投影读取 `okx/dcd.enabled=false`，endpoint family 为 `/api/v5/finance/sfp/dcd`。`internal/research/grammar/grammar.go` 中 vol_sell 两族均 `NeedFace: okx/dcd`；`Enumerate` 要求所需面全部 enabled 才返回族；`grammar.scan` 只评估返回族。`cmd/research/dcd.go` 显示 grammar runner 不受 `research.dcd.enabled` boot 分支控制。因此直接断流机制是共享目录面关闭造成 vol_sell 家族不枚举；不是 dcd enabled 布尔键直接连坐。`fin_capability_surface` 的正本在 PG-md；本地 `MD_PG_DSN` 未连到生产 PG-md，生产正本值仍须在部署后用 MD 正本核验。
- 凭据错误分支不会造成“整族零 eval”：`okxCoinsFor` 读失败会返回错误；`evalVolSell` 将其转为 nil 余量，`dcd.Evaluate` 产 `unevaluable/coins_absent`，随后 `persistLegEval` 仍落 eval 行。历史表现是没有新 eval 行，故不符合单纯 401 的落行形态。未发现或声称有凭据 401 日志。
- `fin_capability_surface` 没有更新时间/变更审计列，能证明 SG 投影当前目录面关闭及其足以造成断流，但不能证明正本行何时被翻为 false。此次 md migration 1089 只在新 research binary 启动迁移时恢复正本读面。

## 复役条件书

1. `research.volsell.fail_streak_n=N`（fin_0114 种子 3），vol_sell 扫描连续读败/错件严格 `>N`，触发 `fin_condition_gap(kind='permission',venue='okx',surface='volsell-fail-streak')` 与同事务 `ops_event` 升件，停止 vol_sell 扫描。请求盒跨进程持久；只有业主明确处理为 `fulfilled` 才复扫。research_rw 无 `app_config` 写权，自动退役不翻 SG 键，持久停扫为 fail-closed 等价。
2. DCD products 目录端点结构性失效：同源 DCD 枚举臂按其连败条件停扫并升件；是否为官方下架/鉴权或结构改版由人核证，退役由升件处置，不将瞬态错误误判为结构性故障。
3. 业主令可随时退役；未触发上述条件时复役为常态，不另启试役/复议。

## SG 正本安全启用步序（本次未执行）

`research.volsell.enabled` 是 SG fin 正本键；research 进程在 boot `config.Load` 时读 PG-md 第九桥副本快照。旧 binary 若在担保腿条件闸部署前看到 true 并重启，会裸奔，所以严格按序：

1. 应用 fin migration 114，核 SG 正本 `research.volsell.enabled=false` 与 `research.volsell.fail_streak_n=3`；此时不开 vol_sell。
2. 部署含 T-0356 担保域条件分流及连败闸的新 research binary；其启动时执行 md migration 1089，将 `okx/dcd` 目录读面恢复。核对运行 SHA 为新 binary、面为 enabled、SG enabled 仍 false。
3. 在 **SG fin 正本**执行 `UPDATE app_config SET value='true' WHERE key='research.volsell.enabled'`；等待 projector 将键复制至 PG-md `app_config`。
4. 重启新 research binary，让 boot 快照读到 enabled=true；核 SHA、SG 键与 MD 副本 generation/value，再看 vol_sell 新 eval、担保域条件请求和零 armed。SHA/复制/检查任一失败，SG 键保持 false 并停止流程。

本施工只提交代码、迁移和本运行规程；不改生产配置、不应用迁移、不重启或部署。
