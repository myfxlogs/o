# T-0480 simdrill -matrix 产跑——T-0461 部署前置格凭据产批（部署令序执行件）

- 状态：已批（CTO 立票直批 2026-11-08——T-0461 部署令序裁定执行件）
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
