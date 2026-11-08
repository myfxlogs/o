# T-0480 simdrill -matrix 产跑——T-0461 部署前置格凭据产批（部署令序执行件）

- 状态：待验（原批：CTO 立票直批 2026-11-08——T-0461 部署令序裁定执行件）
- 领取：无头施工席常径（swe-2-flash-high）
- 优先级：P1——T-0145 部署令硬前置（矩阵凭据未绿前部署令不发，
  业主令 2026-11-08 钱径批催裁）
- 收口移交归：决策席
- 立项：T-0461 验收锚「六格全谱跑绿凭据在案（fin_exec_drill cell_key
  各行 verdict=pass）」——真库跑批=部署前置运维动作（票面钦定），
  CTO 禁 fin_ 写域故派施工

## 背景

- T-0461 矩阵实现已 merge 2f302586 入 master 并部署（o-core 镜像含）；
- fin_exec_drill 现拍=0 行——格凭据全缺 → armed→open 格凭据闸
  fail-closed 全 hold（13 张 armed 提案滞留=「盈利等下米下锅」首堵点）；
- 跑批域=纸面 simvenue（venue 三面全仿真——零真 venue 写零凭据读，
  票面红线）；写面=fin_exec_drill.cell_key 行+ephemeral 策略行
  （template_ref='simdrill:cell:<cell_key>'，24h TTL 自洁同现径）。

## 钦定语义

```
执行：cd /opt/o && CGO_ENABLED=0 go build -o /tmp/simdrill-0480 ./cmd/simdrill
     DB_DSN=<生产写户> /tmp/simdrill-0480 -matrix
凭据：生产写户 DSN 取 SG 侧源（o-core DB_DSN 同值，host 对 SG 本机
     =localhost:15433）——禁抄明文入票/日志/会话
闸效：任一格 fail=非零退出+ops_event 升件——收口行须携
     fin_exec_drill cell_key×verdict 直查表
```

## 范围

1. build+scp 至 SG 跑批（源码树 /opt/o @ e51b973b3 起任一 ≥2f302586 祖先含）；
2. 跑后直查 fin_exec_drill：cell_key 非空行全 verdict=pass 落票面回填
   （fail 格名+detail 携行=排雷实据非罪状——armed→open 闸本就该拦，
   排雷成立也算交件由 DM 读面裁续）；
3. 零代码改动零 merge——纯生产凭据产批，不收 check_all 戳。

## 验收

- simdrill -matrix 执行输出+rc 落收口行；
- fin_exec_drill cell_key 行直查表（SG docker exec o-postgres psql）落收口行；
- ephemeral 策略行自洁确认（24h TTL 现径不手清）。

## 红线

- 纸面域：零真 venue 写/零凭据读域——simvenue 三面仿真外零接触；
- 禁动 fin_strategy/fin_proposal 非 ephemeral 行；
- 凭据不落明文。

## 结果回填（施工 2026-11-08）

### 执行实录

- 源=/opt/o @ 5ab4ef6a5（2f302586 祖先含——合规窗内）；
  `CGO_ENABLED=0 go build -o /tmp/simdrill-0480 ./cmd/simdrill`
  （x86_64 静态件）→ scp SG:/tmp/simdrill-0480。
- 跑批：SG host nohup `/tmp/simdrill-0480 -matrix`；DB_DSN 取
  `docker inspect o-core` env 同值、host 段 o-postgres:5432→
  127.0.0.1:15433 重写后注入——凭据零明文落票/日志/会话。
- 批跑毕：`矩阵批跑毕：格档=6 fail=1` → rc=1（main.go runMatrix
  Failed>0 os.Exit(1) fail-closed 语义——fail 格在案故非零为预期）。
- 免演登记跳过×3：dcd_call / vol_sell_single_call /
  vol_sell_strangle（词表零占——不产凭据不静默漏，excell 钦定径）。

### 凭据直查（SG `docker exec o-postgres psql -U o -d o`）

| cell_key | verdict | detail |
|---|---|---|
| dcd_put@okx/dcd_put#usd=50 | pass | shape=dcd inst=BTC-USDT-261127-67000-P 全径绿(open=#1118 settle=#1118) |
| funding-hedge@okx/short_perp_long_spot#usd=50 | pass | shape=okx_two_leg inst=MUU-USDT-SWAP 全径绿(open=#1119 close=#1120) |
| perp_quarter_spread@okx/perp_quarter_spread#usd=50 | pass | shape=okx_two_leg inst=BTC-USD-270326 全径绿(open=#1121 close=#1122) |
| quarter_basis_long@okx/basis_quarter#usd=50 | **fail** | shape=okx_two_leg inst=BTC-USD-270326 开仓执行败: leg2 败/歧义→LEGGED 回滚 leg1: state=absent |
| xvenue-carry@mt5xokx/long_perp_short_cfd#usd=50 | pass | shape=xvenue inst=BTC-USDT-SWAP 全径绿(open=#1124 close=#1125) |
| xvenue-carry@mt5xokx/short_perp_long_cfd#usd=50 | pass | shape=xvenue inst=BTC-USDT-SWAP 全径绿(open=#1126 close=#1127) |

- fail 格升件落账：ops_event id=4228（seat=演习 ticket=T-0426
  esc_class=执行径演习 note=矩阵演习 quarter_basis_long@okx/
  basis_quarter#usd=50：…开仓执行败…）——非零退出+升件双闸齐。
- 排雷实据：quarter_basis_long@okx/basis_quarter#usd=50 fail——
  leg2（BTC-USD-270326 期货腿）开仓败/歧义→LEGGED 回滚 leg1。
  armed→open 格闸本就该拦——凭据=fail 即闸判据，排雷成立与否
  归 DM 读面裁续（票面钦定语义，非罪状）。

### 写面与自洁核验

- 写面实测：fin_exec_drill cell_key 行×6（上表）+fin_strategy
  ephemeral 行×6（template_ref='simdrill:cell:<cell_key>'，
  state='degraded'）+演习子位（fin_execution/fin_position/
  fin_ledger 随 ephemeral strategy_ref）；fin_strategy/
  fin_proposal 非 ephemeral 行零触；venue 三面全仿真零真写
  零凭据读——红线三条全守。
- ephemeral 自洁确认：演习行 state='degraded' 由现径
  lifecycle.retireStaleDegraded（strategy.degraded_ttl_hours
  键缺省 24h）超龄自动 Transition→retired——24h TTL 自洁，
  未手清；fin_position 挂起行（closed_at IS NULL）实测=0
  （runCell 夭折防护 defer 生效）。
- 零代码改动零 merge——本票 diff 仅票面回填一件。
