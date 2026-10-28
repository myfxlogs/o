// stream.ts —— O Connect Watch 单泵（T-0170 单泵归一；T-0220b v2：带载荷帧
// （data）按 kind 类型化解析（fromJson 零手拼）直写 store 分片——一跳渲染
// 零逐卡重拉；无载荷帧保持 topic 信号（未载荷化面 refetch 径）。断流 3s
// 重连=传输维护，非数据轮询——事件驱动律不变）。
import { fromJson } from "@bufbuild/protobuf";
import {
  GetHostMetricsResponseSchema,
  ListClosedPositionsResponseSchema,
  ListHedgeFlowResponseSchema,
  type WatchEvent,
} from "../gen/dashboardv1/core_pb";
import { client } from "./api";
import * as store from "./store";

type Fn = (v?: unknown) => void;
const subs: Fn[] = [];
const statusSubs: Fn[] = [];
let booted = false;
let live = false;

// channel→topic：卡片粒度信号——ev_cashflow→trx（TRX 面）、watch_open=就绪
// 信号全刷；载荷面更新走 store 分片通知（不经 topic）。
const topic = (kind: string): string =>
  kind === "ev_cashflow" ? "trx" : kind;

// T-0220a 三载荷面：kind→分片类型化写入（响应消息 protojson↔fromJson 直转）。
function ingest(ev: WatchEvent): void {
  if (!ev.data) return;
  switch (ev.kind) {
    case "ev_cashflow":
      store.setHedgeFlow(fromJson(ListHedgeFlowResponseSchema, ev.data));
      break;
    case "ev_position":
      store.setClosedPositions(fromJson(ListClosedPositionsResponseSchema, ev.data));
      break;
    case "ev_hostmetrics":
      store.setHostMetrics(fromJson(GetHostMetricsResponseSchema, ev.data));
      break;
  }
}

function setLive(v: boolean): void {
  if (live === v) return;
  live = v;
  statusSubs.forEach((f) => f(live));
}

async function pump(): Promise<void> {
  for await (const ev of client.watch({})) {
    setLive(true);
    ingest(ev);
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

// onData 订阅事件帧信号（topic 过滤在订阅侧 done；载荷更新另走 store 分片）。
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
