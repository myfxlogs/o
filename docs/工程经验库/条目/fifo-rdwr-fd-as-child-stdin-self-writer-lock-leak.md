---
schema_version: 2
id: fifo-rdwr-fd-as-child-stdin-self-writer-lock-leak
kind: pitfall
risk: high
status: verified
keywords: [fifo, O_RDWR, advisory lock, psql, stdin, 孤儿进程, 锁泄漏, dm_tick, bash fd]
scope: [scripts/meta/dm_tick.sh, bash fifo 模式, PG advisory lock 持锁件]
summary: bash `exec 8<>fifo`（O_RDWR）再以 `<&8` 作子进程 stdin——子进程自身即写端持有者，父脚本被杀后孤儿子进程永不见 EOF（自写自读），PG advisory lock 永泄=功能死锁；修法=另开 `exec 9<fifo`（O_RDONLY）作子 stdin，写端只归父 fd
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: t/0354-dm-shift-kick-drain
  - type: doc
    ref: scripts/meta/dm_tick.sh:29-50（T-0354 单例闸 holder 段）
---

## 事实/症状

T-0354 dm_tick.sh 单例闸原稿：`exec 8<>fifo`（O_RDWR）→ `psql ... <&8 &`——
psql stdin=dup(fd8) 共享 O_RDWR 描述符，**psql 自己就是 fifo 写端持有者**。
正常退出 trap kill psql 无碍；但脚本被 SIGKILL/崩溃（EXIT trap 不及）时：
fd8 关闭≠写端清零（psql fd0 仍 O_RDWR）→ psql 永不见 EOF → 孤儿 psql 永生
→ PG advisory lock 永持 → 后续所有 dm_tick 闸拒即退=决策席静默全灭。

## 根因或策略解释

FIFO EOF 语义=「所有写端关闭才 EOF」。`exec N<>file` 开 O_RDWR 单描述符
兼读写——dup 给子进程后，子进程 fd 继承同一 O_RDWR 描述，计入写端集合。
「父死子活」场景下子进程自持写端=EOF 永不至。教训一般化：**凡以
「父亡→子 EOF→子退→资源放」为释放链的 fifo/pipe 持锁件，子进程 stdin
必须是只读面（独立 O_RDONLY fd），写端只归父进程独占**。

## 处置

```bash
exec 8<>"$DM_GATE_DIR/pipe"   # 父独占写端（O_RDWR——自身也防读阻塞）
exec 9<"$DM_GATE_DIR/pipe"    # 子 stdin 只读面（fd8 已在=writer 故不阻塞）
psql "$FIN_DSN" -qAt ... <&9 & # 子不再是写端持有者
```

残余窗口=继承 fd8 的长命子进程（devin 会话本体=锁应有的存续期，语义正确；
stray 孤儿=其寿命为界，systemd cgroup kill 全灭即清）。SIGKILL 实证：
父杀后锁仍持（孤儿 sleep 持 fd8）→孤儿退出→psql 见 EOF 退→锁放（t）。

## 证据

- check: `kill -9 <dm_tick bash>` 后 `pg_try_advisory_lock(42540354)`
  t+1s=f（孤儿 q-sleep.sh 持 fd8）→孤儿退后=t（psql EOF 退锁放）。
- result: 旧形态（stdin=O_RDWR fd）同测=锁永 f；新形态有界释放实证。
- ref: T-0354 收口支；scripts/meta/dm_tick.sh:29-62。

## 反证条件

- check: 复现「父 SIGKILL+无任何 fd8 继承者」→新形态应即时放锁；
  若仍永锁则本条目结论有误。
- falsifies_when: 实测孤儿 psql 在全部写端关闭后仍不退（如 psql 行为变更）。

## 适用边界

bash fifo+后台持锁会话模式；systemd 服务 cgroup kill（全员同灭）场景下
新旧形态等价——坑只在「单点被杀、子进程成孤儿」径显形。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a（豁免由：shell 模式坑非产线数据面；监测面=advisory lock
  持有者存活巡检人工径）
