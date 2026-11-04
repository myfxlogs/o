# T-0348 venue 新品哨兵——在册 venue 产品目录+公告面 diff 探测

- 状态：施工中（业主令外派直派——O-Dev 施工 run，锁凭据=分支 t/0348-venue-watch）
- 领取：o-dev-t0348 @ 2026-11-04（worktree `/opt/o-wt/t0348`，branch `t/0348-venue-watch`）
- 级别：P2 进水口面——机会探测断链=新品窗口静默错过（研究员第二触发器无上游事件源）

## 病灶（实查在案，勿再诊）

研究员职卡第二触发「venue 新产品入站→新族提案」**无上游事件发生器**：
全仓 grep「新品/new_product/new_surface」零命中——无 instruments 目录 diff、
无公告面、无 RSS。`internal/research/scout` 只采 `fin_venue_candidate`
measuring/recommended 态 venue 的 perp funding 行（9 适配器），**已注册 venue
（OKX）不在其视野**——scout 是「要不要注册」度量臂，非「已注册 venue 上新
了什么」探测臂。实证：OKX 上线简易期权（2026-11-04 业主发现），系统无感。
职卡允许「决策席/CEO 手动代发同类事件」——即今日唯一触发径=业主本人。

## 任务（修法供裁）

1. 新哨兵拍 `scripts/meta/venue_watch.sh`（照 `gauges_tick.sh` 零 LLM 纯判据
   对形；env 径读 FIN_DSN；`META-VENUEWATCH` 心跳落 ops_headless）：
   - **数据源①（主）= OKX 公开公告端点** `/api/v5/support/announcements`
     （零凭据；annType 参数族实探测全）。关键词集滤新品类——词集钉脚本头
     常量（option/期权/Earn 新品/new product 族——精确词面勿正则地狱），
     检出项带公告标题+URL 出处。
   - **数据源②（备）= instruments 目录 diff**：`/api/v5/public/instruments`
     instType 全集快照 vs 上次水位（seen-set 存 $ST 文件或 PG——施工裁，
     幂等键=venue+instId/surface）；新 instType/新 surface 类=新品候选。
     注：简易期权若为 app-only 面 API 不可见，公告面兜——**协议面取证纪律
     适用：下「API 无此面」结论前消息体全文核证**，不许猜。
   - 检出 → `ops_event` 行 `seat='venue哨兵' event='新品' note=venue:产品:出处`。
     **同品幂等零重发**（seen-set 判重）；事件即研究员触发信号，哨兵职责
     止于事件+心跳（不碰 fin_opp_hypothesis——那是研究员窄门）。
   - 结构留扩展位：venue→探测函数 map（OKX 首发；Binance/MT5 同类端点
     后续可挂，不必本票实装）。
2. 失败显形：每拍 hb alive；取数败=hb note fail 表名（不静默绿）——照
   gauges_tick 取数败同构。
3. **timer 与脚本同 commit 落地**（T-0331/0332 孤儿脚本教训——脚本不带电
   =部署锚再漏）：`deploy/o-venue-watch.{service,timer}` 正本 +
   `~/.config/systemd/user/` 装机（1h 拍——公告非秒级面；
   OnBootSec=15min Persistent=true 对形）。
4. 首天然锚=OKX 简易期权：落地首拍应检出（公告面窗口期内）或如实记
   「API 不可见面」取证结论。

## 验收锚

- [ ] 合成注入干跑：伪公告含期权词→`ops_event` 新品行落账+幂等重放零重复
- [ ] 真实探测：哨兵对 OKX 简易期权检出（公告或目录面），或协议面取证
      全核证后如实记「不可见面」结论
- [ ] 断网干跑 hb fail 显形非绿
- [ ] timer 装机实证（`list-timers` NEXT/LAST 活）+ deploy/ 正本在仓
- [ ] `check_all` 绿

## 红线

- 零凭据公开端点 only——不碰私有 API/任何凭据面
- 只产 `ops_event`+心跳——`fin_opp_hypothesis`/钱径/模板面零触
- 禁 LLM——纯判据拍
- 新品事件幂等——同品零重复触发（事件刷屏=升件面污染）
