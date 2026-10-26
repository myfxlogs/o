// stream.ts —— O Connect Watch 主刷新面（T-0156：SSE shim+保底轮询撤——
// 数据更新唯一驱动=CoreService.Watch 流帧（ev_* 脊，事件驱动律）；
// 断流 3s 重连=传输维护，非数据轮询）。
import { client } from "./api";

type Fn = (v?: unknown) => void;
const subs: Fn[] = [];
let booted = false;

// channel→topic：TRX 现金行（租金/领奖/隔夜息）落 fin_cashflow=ev_cashflow
// ——TRX 页组件的 dataRefresh 主题；其余通道原名透传。
const topic = (kind: string): string => (kind === "ev_cashflow" ? "trx" : kind);

async function pump(): Promise<void> {
  for await (const ev of client.watch({})) {
    subs.forEach((f) => f({ topic: topic(ev.kind) }));
  }
}

function run(): void {
  void pump()
    .catch(() => {}) // 断流吞错（读面自愈——下一帧自然回填）
    .finally(() => setTimeout(run, 3000));
}

function boot(): void {
  if (booted) return;
  booted = true;
  run();
}

export function onCase(name: string, fn: Fn): () => void {
  boot();
  if (name === "dataRefresh") subs.push(fn);
  return () => {
    const i = subs.indexOf(fn);
    if (i >= 0) subs.splice(i, 1);
  };
}
