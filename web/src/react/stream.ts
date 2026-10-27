// stream.ts —— O Connect Watch 单泵（T-0170：原双泵归一；事件→topic 局部
// 失效 + 连接状态显形（断流=顶栏黄点，不静默）。断流 3s 重连=传输维护，
// 非数据轮询——事件驱动律不变）。
import { client } from "./api";

type Fn = (v?: unknown) => void;
const subs: Fn[] = [];
const statusSubs: Fn[] = [];
let booted = false;
let live = false;

// channel→topic：卡片粒度失效——ev_position→持仓卡、ev_cashflow→流水/TRX、
// ev_account→健康/账户卡、ev_strategy→策略相关；watch_open=就绪信号全刷。
const topic = (kind: string): string =>
  kind === "ev_cashflow" ? "trx" : kind;

function setLive(v: boolean): void {
  if (live === v) return;
  live = v;
  statusSubs.forEach((f) => f(live));
}

async function pump(): Promise<void> {
  for await (const ev of client.watch({})) {
    setLive(true);
    subs.forEach((f) => f({ topic: topic(ev.kind) }));
  }
}

function run(): void {
  void pump()
    .catch(() => {})
    .finally(() => {
      setLive(false); // 断流显形——不静默（T-0170 痛点④：实时性可见）
      setTimeout(run, 3000);
    });
}

function boot(): void {
  if (booted) return;
  booted = true;
  run();
}

// onData 订阅数据事件帧（topic 过滤在订阅侧 done）。
export function onData(fn: Fn): () => void {
  boot();
  subs.push(fn);
  return () => {
    const i = subs.indexOf(fn);
    if (i >= 0) subs.splice(i, 1);
  };
}

// onLive 订阅连接状态（true=流活/false=断流重连中）。
export function onLive(fn: Fn): () => void {
  boot();
  statusSubs.push(fn);
  fn(live); // 立即回放现态
  return () => {
    const i = statusSubs.indexOf(fn);
    if (i >= 0) statusSubs.splice(i, 1);
  };
}
