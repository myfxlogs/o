---
id: mt5-bridge-deadwindow-forensics-path
title: MT5桥死窗取证径——OCI user-unit journalctl+SG跳板+票面时点≠journal事件时点（#20105 断因=venue侧断）
kind: pit
date: 2026-11-18
source: 工单#20105 mt5-bridge-ev1 纯取证（父票#16226 挂起条件=子票落据）
---

# MT5 桥死窗取证三件套：径、时点口径、日志方言

**坑形**：票面「三死窗 06:49/10:14」时点在 journal 里**不存在**——
那是探针侧 md_ticks `recv_ts` 陈旧口径的检出时点。journal 事件时点
= 真帧静默满 `tickSilentAfter=150s` 才打 `WARN 推流静默死`，往前推
~150s 再叠重连/风暴段（最劣 16min）。按票面时点直查 journal 会扑空
误判「无此事」——前会话 #20974 挂起「SSH 凭据缺」被外审 #21488
勘正后，本会话实证第二坑=时点口径差。

**取证径**（dev→OCI 全通，本机无凭据也能走）：

```bash
source /opt/o/scripts/oci_ssh_route.sh   # 探直连→死转 SG 跳板，自愈
ssh $O_OCI_JARG ubuntu@149.118.58.98 \
  "journalctl --user -u o-collecto --since '<UTC±30min>' --no-pager -o short-iso" \
  | grep -E 'mt5bridge|Started o-collecto|Stopped o-collecto'
```

- 桥腿**不是独立 unit**——`o-collecto.service` 是 **user** systemd
  unit（`systemctl --user`，ubuntu uid=1001，`~/.config/systemd/user/`），
  系统级 `systemctl list-units` 看不到；进程本体 `/home/ubuntu/o/collector`。
- 前会话误判「OCI SSH 凭据缺」（#20974）被外审勘正（#21488）——
  正解=`oci_ssh_route.sh` 自带路由（直连死→SG `root@192.229.85.35` 跳板），
  id_ed25519 直登/SG 跳板均通。
- `md_ticks` 是**快照表**（PK venue+inst_id upsert），无历史——
  死窗史只能走 journal，PG 侧只查现状（`recv_ts` 新鲜度=流在产判据）。
- journal 噪声大（sigdist/okxscan 分钟级），mt5bridge 稀疏（~26 次/日）——
  **必须 grep `mt5bridge|Started|Stopped`**，否则 head 截断关键行。

**日志方言→断因映射**（11-14/15 实证）：

| journal 行 | 含义 | 归类 |
|---|---|---|
| `WARN 推流静默死（~150s 零真帧）` | venue 订阅侧哑——流不返 err、Recv 永阻塞；gRPC keepalive 救不了应用层哑 | venue 侧断 |
| `会话终 ... ConnectEx err: ... refusing logins / SERVICE_NOT_AVAILABLE / Server closed the socket ... send login` | broker（Exness-MT5Real）login 拒绝风暴，cluster IP 轮换（:3001157=EXR 户） | venue 侧断 |
| `会话终 ... OnQuote 流断: ... EOF ... goaway NO_ERROR` | venue 主动关流 | venue 侧断 |
| `订阅败 ... SERVICE_NOT_AVAILABLE (:0)` | venue 订阅面暂不可用（辅助签名） | venue 侧断 |
| 会话终→Ns→`桥腿起`→目录拍 `mid 快照 367 符` | 重连径工作（自愈实证） | 桥径在案 |

**架构断代锚**：`ConnectEx err` 行=**11-15 前旧架构**（桥腿自 login）
证据；11-15 共享会话接管后（`internal/mt5/bridge.go:1-11`，sharedAttach
读 `mt5.session_token`、永不 ConnectEx，owner=SG 汇市巡独占 login）
journal 再无 ConnectEx 错——读旧日志先断代再归因。

**参数读面**（internal/mt5/，全码内 const 非热配）：

- `tickSilentAfter=150s`（bridge.go:225，对齐下游 xvenue 120s 帧龄闸+余量）；
  巡查 ticker 15s；真帧=bid>0∨ask>0 非空报价帧（心跳形空帧不喂狗）。
- 重连退避 `reconMin=2s→reconMax=60s` 指数（collector.go:34-35，
  bridge.go:60-75；实测序列 2s/4s/8s/16s/32s/60s 封顶在案）。
- `SessionBackoff` 会话冲突指纹→照 mtapi 冷却睡（bridge.go:65-67）。
- gRPC keepalive `Time=30s Timeout=20s PermitWithoutStream=true`（client.go:53-55）
  ——传输层保活在位仍被订阅哑击穿=应用层 venue 问题非传输层。

**实证结论**（#20105 验收件，断因归类=**venue 侧断**）：

- 票面「当日」=**11-14**（父票 #16226 filed 2026-11-14T10:57Z）。
  journal 全天 ~26 次静默死（密集段 01:00-03:40 每 20-40min 一发）。
- **06:49 窗（票面 12m 自愈）**：journal 06:52:14 静默死→订阅败
  SERVICE_NOT_AVAILABLE×3→06:55:50 再静默死→**ConnectEx 拒绝风暴
  06:56:54-07:07:31 十连败**（broker 明文：refusing logins paused /
  SERVICE_NOT_AVAILABLE / Server closed the socket send login /
  refused a login on every cluster member，cluster IP 轮换
  8.209.63.83/48.222.251.98/35.178.63.121/43.211.3.14）→07:08:33
  桥腿起自愈 ≈16min。=venue 侧 login 拒绝+订阅不可用（旧架构桥腿
  自 ConnectEx 暴露面）。
- **10:14 窗（票面 40min+ 未复 #16153）**：journal 10:14:34 OnQuote
  EOF/goaway→16s 重连→10:14:51 桥腿起→10:17:24 目录拍 mid 367 符
  =流已复。journal 侧重连径正常；票面「40min 未复」与 journal 复流
  证据口径矛盾（探针 recv_ts 陈旧窗长/其后 11:13 静默死复发段）——
  两口径并列如实报，裁归唤知。=venue EOF 主动关流，重连径已愈。
- **第三发**=NAV 残差 mt5 无 bill 同形态（票面自注，非 socket 窗——
  同根 venue 断流→采集缺帧→下游残差的「一因多果」表现，不属本票）。
- **频次/时长改善实证**：11-14 ~26 静默死/日、最劣风暴窗 16min
  →11-15T11:00Z 后 ~57h journal **零静默死零 ConnectEx 错**，
  11-17T17:42 唯一会话终=干净 EOF→5s 愈（健康对照组）。净窗
  07:23 起早于两次重启（09:31/11:00）——重启非归零直接因；
  归因双候选（共享会话接管消双 login 竞速 vs venue 波动窗口自然
  结束）施工不独断，时序证据已载，git log 考接管 commit 时点
  可补细分（`git log -p internal/mt5/bridge.go --since=2026-11-14`）。

**三类归类证伪对照**：桥重连径缺=否证（看门狗+退避+重订全链在案，
11-14 当日自愈十余次，常规窗 ~1-2min）；OCI 修复环未注册=否证
（o-collecto enabled+active+MainPID 在跑，进程内看门狗即修复环，
journal 无 panic/OOM/crash，三次重启均干净对）。他端面同期
okxws/okxscan/sigdist 全正常=本端网络/进程健康，断点在 venue。

**判别口诀**：venue 推流哑三签名=「真帧零至但流不返 err」+
「RPC 面同会话照常」（目录拍 mid 367 符仍落行=非连接死、订阅面
选择性哑）+「重订即愈」。见此三签先归 venue 侧，勿再查桥径/OCI 环。

**复跑**：`journalctl --user -u o-collecto | grep mt5bridge`；
对照 `internal/mt5/bridge.go`（会话终/看门狗日志族、断代注释头）
与 `internal/mt5/client.go`（keepalive 参数）。
