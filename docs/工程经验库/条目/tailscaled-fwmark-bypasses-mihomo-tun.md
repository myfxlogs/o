---
schema_version: 2
id: tailscaled-fwmark-bypasses-mihomo-tun
kind: pitfall
risk: medium
status: mitigated
keywords: [tailscale, mihomo, fake-ip, TUN, fwmark, GFW, 注册超时, 代理]
scope: [ops, network, proxy, tailscale]
summary: tailscaled 自打 fwmark 走物理口裸直（防环设计）——GFW 掐 register POST
  致永登不上；netcheck/GET 假绿。修=drop-in 挂 HTTPS_PROXY 走本地代理口
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: journal
    ref: "tailscaled doLogin 每 ~40s 循环 register 超时（10-30 23:18-23:33 全灭）"
  - type: netcheck
    ref: tailscale netcheck UDP/DERP 全通（hkg 43ms）——假绿陷阱，与登录面无关
  - type: ip-rule
    ref: ip rule 5210「from all fwmark 0x80000 lookup main」——tailscaled 出站打标绕行 TUN
  - type: curl
    ref: 裸直 eno1→controlplane POST machine/register HTTP/2 stream 开即挂死 vs 经 7890 代理 400 应答 2s
  - type: fix
    ref: /etc/systemd/system/tailscaled.service.d/proxy.conf Environment=HTTPS_PROXY=http://127.0.0.1:7890
---

## tailscaled 控制面被 GFW 掐 POST——「每次 SSH 都要重登」根案

- **表象**：tailscaled 常驻且 enabled，但 `tailscale status` 恒 logged
  out/NoState；journal 里 doLogin 循环全超时。机主每次 SSH 上来都要重新
  登录 tailscale。
- **误诊径（三连假绿）**：①`netcheck` UDP/DERP 全通——DERP 探测是小请求
  过得去；②curl GET controlplane 302——GET 也过得去；③fake-ip 劫持
  以为是 mihomo 路由病，改 DIRECT 规则+hosts 钉真 IP 全无效。
- **真机理**：tailscaled 为防「流量进自己隧道成环」给**自己的 socket 打
  fwmark 0x80000**→`ip rule 5210` 送 main 表→物理口裸直，**全程绕开
  mihomo TUN**（/connections API 里永无其踪=实证）。裸直访问
  controlplane.tailscale.com：GET/UDP 过、**machine/register POST 被
  GFW 断流**（HTTP/2 stream 开了挂死）→注册永不完成=状态不落盘=永远
  NoState。
- **正解**：控制面强制走代理、数据面保持裸直——drop-in
  `Environment=HTTPS_PROXY=http://127.0.0.1:7890`（tailscaled HTTP
  client 认标准代理 env；DERP/P2P UDP 不受影响照走物理口）。一次注册
  后凭据持久 `/var/lib/tailscale`，重登永不需。
- **排障方法论**（可复用）：「在跑但登不上」类——先 `ip rule` 看 fwmark
  政策路由（应用自带绕行常藏这），再 ss/conntrack 抓其真实 dial 目标
  （/connections 里缺席=没走代理=实锤绕行），最后 curl 分面验证
  GET/POST/UDP 各自的通过性（勿拿单面绿推全链）。
- **关联坑**：mihomo fake-ip 下 DNS 全劫持（连指定上游 nslookup 也回
  fake-ip）——求真 IP 须 DoH/外部解（`dns.google/resolve` JSON 体是
  远端解析不被本地劫）。
