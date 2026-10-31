# OCI 盒档案——采集层宿主（落地 2026-10-28）

> 机器全谱档：身份/进法/服务/端口/凭据存放点/重灌径。
> 密件本体不存本文件——只记「它住在哪+怎么重生」（repo 明文=泄密面）。

## 机器身份

| 项 | 值 |
|---|---|
| 提供方 | Oracle Cloud Always Free（ARM A1） |
| 实例名 | `instance-20261028-1303` |
| 区域 | **Singapore West**（ap-singapore-2 域） |
| 公网 IP | `149.118.58.98` |
| 内网 IP | `10.0.0.76` |
| 规格 | aarch64 · 4 vCPU · 24G 内存 · 45G 盘 |
| OS | Ubuntu 24.04.5 LTS |
| 登录名 | `ubuntu` |
| 角色 | 采集+蒸馏+PG-md+投影臂+Reality 出口（数据流.md 全景图采集层宿主） |

## 进法（dev→盒）

```bash
ssh -i ~/.ssh/id_ed25519 ubuntu@149.118.58.98
```

- dev 侧两把键皆已入盒 `authorized_keys`：`~/.ssh/id_ed25519`（主）、`~/.ssh/id_rsa`（备，OCI 栏位挑剔时用）
- 盒私钥永不外拷；丢失=重生钥对+控制台重写 authorized_keys
- OCI 控制台兜底径：实例页 → Console Connection（需串口钥对，救援通道）

## 服务清单

| 单元 | 级 | 职责 |
|---|---|---|
| `o-collecto.service` | **用户级**(ubuntu) | 采集+sigdist 蒸馏（OKX 全宇宙 WS+MT5 桥） |
| `o-projector.service` | **用户级**(ubuntu) | 投影臂 dualwrite——盒 PG-md →SG 八桥镜像 |
| `o-research.service` | **用户级**(ubuntu) | 研究臂全族（funding/dcd/gapscan/grammar/假想官/review)——2026-10-28 自 dev 迁入，`~/o-research/` |
| `o-sweeper.service` | **用户级**(ubuntu) | PG-md 清道夫——fin_data_retention 分库账日拍+首拍即时（T-0275 建制恢复；曾 10-28 cutover 漏册断供 2.7d——无册面孤儿教训在 kb） |
| `o-sg-tunnel.service` | **系统级** | SSH 隧道 盒→SG（15433=PG-fin，8095=readcred） |
| `o-backup-fin.timer` | **系统级** | PG-fin 经隧道备份落盒（03:20 UTC，T-0173 跨机跨域） |
| `o-backup-md.timer` | **系统级** | PG-md dump+推 SG oci-inbox+配置 escrow（02:40 UTC，T-0234——SG 统推 R2 `oci/` 前缀） |
| `o-sentinel.timer` | **系统级** | 拨测哨兵（*/5min，红灯落 /var/log/o-sentinel/alerts.log） |
| `sing-box.service` | **系统级** | vless+reality 出口 :443（dev mihomo 上游） |
| docker `o-postgres-md` | 容器 | PG-md(timescaledb:2.30.2-pg18 arm64，卷 `o-pgdata-md`，绑 127.0.0.1:5435) |

- **linger 已开**(`loginctl enable-linger ubuntu`)——用户级件无会话不死，勿关
- env 件：`~/o-collecto.env` `~/o-projector.env`（含 DSN 口令，0600；`o-sweeper` 复用 `o-projector.env`——projector_rw 已含 Register+Beat 授权面，零新密件 T-0275 亲证）

## 端口面

| 端口 | 向 | 用途 |
|---|---|---|
| 22 | 入（OCI NSG+盒 iptables 双开） | SSH 管理 |
| 443 | 入（同上双开） | Reality 出口（伪面=www.oracle.com 握手） |
| 5435 | 本机 loopback | PG-md |
| 15433/8095 | 本机 loopback | 隧道转发→SG |

盒 iptables 手工补过 443 NEW ACCEPT——已 `iptables-save` 持久（重灌须补）。

## 凭据存放点（本体在哪，不拷进 repo）

| 件 | 住处 | 失则 |
|---|---|---|
| 盒 SSH 钥对 | dev `~/.ssh/`(ed25519+rsa) | 重生+控制台重贴公钥 |
| SG 隧道钥 | 盒 `~/.ssh/`（restrict+permitopen=15433/8095 无壳钥） | 重生+SG authorized_keys 重挂 |
| 备份推送键 | 盒 `/root/.ssh/id_ed25519_backup_push`（SG 侧 restrict+internal-sftp 无壳收口——T-0234） | 重生+SG authorized_keys 重挂（bootstrap 打印收口行） |
| Reality 私钥/uuid/short_id | 盒 `/etc/sing-box/config.json`（root 0600) | escrow 原配置回灌则零换；缺则盒上 `sing-box generate reality-keypair/uuid` 重生，dev mihomo 配置同步换 public-key/uuid |
| 备份锚 | SG `root@192.229.85.35:/root/oci-secrets/` | 信托锚在 SG——凭据副本统一归此，永不落 dev/repo |
| 配置 escrow | SG `/root/oci-secrets/escrow/`（o-md-escrow-*.tar.age 日推滚动 14 份——T-0234；age 私钥业主离线） | 业主解包→bootstrap `/root/bootstrap-env/` 值面供给 |
| PG-md 口令 | 盒 `~/o-*.env` | 同 env 重建（migrate -domain=md 重灌） |
| OKX 只读钥 | **不在盒**——venue 物料不出 SG(readcred 经 8095 取） | — |

## 重灌径（机器被回收/烧毁时）

**一键径=T-0234**：`scripts/oci_bootstrap.sh`（新盒 root 跑，`DRY_RUN=1` 干跑可推演）——下述八步全部机内化，前置 scp 清单/escrow 值面供给见脚本文件头。手工径（理解用/兜底）：

1. OCI 重开 ARM A1(SG West,Ubuntu 24.04)→贴 dev 公钥
2. `apt install docker.io postgresql-client-16` + `usermod -aG docker ubuntu`
3. 盒钥对生→SG authorized_keys 挂 restrict 钥→`o-sg-tunnel` 立起
4. `o-postgres-md` 容器起→`migrate -domain=md`(PG_DSN=127.0.0.1:5435)；有 escrow/dump 则末份 o-md dump restore 优先
5. GOARCH=arm64 交叉编 collector/projector→部署+env+用户级 unit+linger
6. app_config 灌 `okx.symbols=["*"]` 等键（源=dev 或 SG app_config 表）
7. sing-box 重装+Reality 钥对重生+dev mihomo `oci-reality` 换参（escrow 配置在位则零换）
8. OCI NSG+盒 iptables 双开 22/443

## 判读规则

- **随时可弃设计**：此盒=发现层宿主，钱径零依赖；烧掉按上表重灌≈半天
- **Claude Code 防回收探针在跑**（Always Free 闲置回收对策，勿停）
- 数据流正本=`docs/数据流.md`；此档只钉机器面，链路语义以正本为准
