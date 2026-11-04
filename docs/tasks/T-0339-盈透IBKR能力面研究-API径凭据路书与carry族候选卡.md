# T-0339 盈透 IBKR 能力面研究——API 径/凭据路书/carry 族候选卡

- 状态：完成（决策席无头拍裁决通过 @2026-11-04——合 master；四锚亲证复核一致；md_1088 待随窗落 PG-md）｜原状态：已批（业主令 2026-11-04「立呗」——COS 例行代理批；业主明示未申 API key、不知申请径与门槛——凭据路书为本票第一产出）
- 级别：P3 研究件——排「第 1 分钱」链（T-0336/0337）之后自然拉起
- 族：venue-scout 发现面同族（T-0312 谱系）——机器画地图，业主只按图点

## 背景

业主持盈透账号（凭据就绪=候选池唯一跳过注册瓶颈的 venue）。
IBKR 上 CME 加密期货基差=机构级 carry 族；闲置现金付息=法币侧
Earn 对标基准。本票只做**能力面枚举**——执行闸不因此松一毫米。

## 任务

1. **API 面枚举**：TWS API（socket）/Client Portal Web API（REST+OAuth）/
   FIX 三径——鉴权模型、rate limit、paper vs live 差异、哪径合 O 读面优先
2. **凭据路书（第一产出）**：申请/开通 API 的逐步清单——
   账户管理何处开通、资金门槛有无、行情订阅费项、交易权限前置、
   paper 账户能否零费完成 API 开发验证——落成 `docs/venue/ibkr-凭据路书.md`
   业主照单可点，每步注明出处（公开文档/官方页）
3. **carry 族枚举**：CME BTC/ETH 期货基差、FX carry 对、box/融资、
   闲置现金利息分档、证券借出（可选）——逐族记保证金/借贷成本/门槛
4. **候选卡**：按 scout 册既有形落 venue 候选卡（数值阈照铁闸——
   与现有 venue 同一评价面，不搞特批）

## 验收锚

- [ ] 三 API 径对照表实存（鉴权/限制/paper-live 差）
- [ ] 凭据路书落档——业主待办清单逐步+门槛全列+出处可溯
- [ ] carry 族枚举表（≥3 族，保证金/成本项有数不空栏）
- [ ] venue 候选卡入册（数值阈面，非描述面）

## 红线

- **禁申任何凭据**——业主申请动作在路书产出后由业主亲手做
- 禁下单/动钱/注册任何面——纯公开面+文档研究
- 业主现有 IBKR 凭证未提供前禁以任何形式假设已持 API 权限

## 回填

票面+路书/候选卡路径+锚逐项实录。

## 施工回填（devin-t0339 续领收口 @ 2026-11-04，分支 t/0339-ibkr-scout）

- 状态：收口待验
- **锚①三 API 径对照表** → `docs/venue/ibkr-api-对照.md`（TWS socket / CP
  Gateway REST+WS / FIX CTCI：鉴权/费用门槛/paper 差/速率面逐项+出处；
  结论=CP Gateway 读面首选，TWS 补充，FIX 本阶段出局 $1,500/月）
- **锚②凭据路书** → `docs/venue/ibkr-凭据路书.md`（业主待办四步：paper
  凭据→Futures-US 权限→CME L1 $1.55/月→gateway 拉起+IB Key；关键事实=
  CP 径无审批流、要求 funded 户、每日重登、paper 零费验证；FIX/OAuth
  留档不走）
- **锚③carry 族枚举 ≥3 族** → `docs/venue/ibkr-carry-族枚举.md`（五族：
  CME 加密基差 MBT 保证金≈22.6%/全成本≈3.2bps·边；闲置现金息 USD
  3.38%@NAV≥100k 首 10k 无息=法币 Earn 对标锚；FX carry spot≈3.33% margin
  美国户 ECP 限；box 融资/SYEP 借券记存在性）
- **锚④候选卡入册** → `migrations/md_1088_venue_candidate_ibkr.sql`：
  fin_venue_candidate 插 'ibkr' 行（kind=cfd-broker 四值 CHECK 最近拟；
  taker/maker=3.16bps MBT 单边全成本折；min_notional=6334；endpoints={}
  ——无公开免鉴权端点 metrics 注定空，state 留 candidate；evidence 携
  三产出件路径+抓取日）
- **自测**：check_migrations 七断言绿 + go build ./... 绿 + go vet
  pgstore 绿（docs+迁移件，无逻辑码）
