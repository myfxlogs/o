// stream.ts —— O Connect Watch 单泵（T-0170 单泵归一；T-0220b v2：带载荷帧
// （data）按 kind 类型化解析（fromJson 零手拼）直写 store 分片——一跳渲染
// 零逐卡重拉；无载荷帧保持 topic 信号（未载荷化面 refetch 径）。断流重连=
// 指数退避+抖动（T-0269——原固定 3s 在死路径上成重连风暴轰炸同一新 h2
// 会话，事故放大器病灶）=传输维护，非数据轮询——事件驱动律不变）。
import { fromJson } from "@bufbuild/protobuf";
import {
  GetHeadlessResponseSchema,
  GetHostMetricsResponseSchema,
  GetPipelineResponseSchema,
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
    case "ev_headless":
      store.applyHeadless(fromJson(GetHeadlessResponseSchema, ev.data));
      break;
    case "ev_pipeline":
      store.applyPipeline(fromJson(GetPipelineResponseSchema, ev.data));
      break;
  }
}

function setLive(v: boolean): void {
  if (live === v) return;
  live = v;
  statusSubs.forEach((f) => f(live));
}

// 重连退避梯（T-0269）：1s→2→4→…封顶 30s，±25% 抖动防雷群；帧到=流已证活
// →梯子复位（连接即死的路径无帧→梯子持续升——风暴自动降频）。
let backoffMs = 1_000;
const BACKOFF_MAX_MS = 30_000;

async function pump(): Promise<void> {
  for await (const ev of client.watch({})) {
    backoffMs = 1_000;
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
      const delay = backoffMs * (0.75 + Math.random() * 0.5);
      backoffMs = Math.min(backoffMs * 2, BACKOFF_MAX_MS);
      setTimeout(run, delay);
    });
}

function boot(): void {
  if (booted) return;
  booted = true;
  // T-0268：流建立让出首绘帧——握手 fetch 异帧发起，首绘不等 server-stream
  // 握手（订阅 effect 同步调 boot 也不与首绘同框）；断流/挂起期 UI 退化为
  // unary+store 存量快照态（LiveDot 黄点显形），零阻塞零轮询。
  setTimeout(run, 0);
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
