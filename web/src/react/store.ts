// store.ts —— 全局分片 store（T-0220b：useSyncExternalStore 分片订阅——
// 推帧只通知命中分片，订阅卡只重渲自己；useSnap 逐卡 fetch+state 样板退役）。
// 快照身份纪律：set 才换对象（React useSyncExternalStore 同时渲染一致性）；
// hydrate 与推帧竞态裁决=推帧优先（迟到回包弃陈，推帧语义=最新事实）。
import { useSyncExternalStore } from "react";
import type {
  GetHeadlessResponse,
  GetHostMetricsResponse,
  ListClosedPositionsResponse,
  ListHedgeFlowResponse,
} from "../gen/dashboardv1/core_pb";

export type SliceName = "hedgeFlow" | "closedPositions" | "hostMetrics" | "headless";
export type SliceView<T> = { data: T | null; err: string | null; loading: boolean };

type AnyView = SliceView<unknown>;
type Slice = { snap: AnyView; subs: Set<() => void> };

const freshView = (): AnyView => ({ data: null, err: null, loading: false });
const slices: Record<SliceName, Slice> = {
  hedgeFlow: { snap: freshView(), subs: new Set() },
  closedPositions: { snap: freshView(), subs: new Set() },
  hostMetrics: { snap: freshView(), subs: new Set() },
  headless: { snap: freshView(), subs: new Set() },
};

// 推帧已写入的分片：hydrate 迟到回包不再覆盖（推帧后分片恒新）。
const pushed = new Set<SliceName>();
const inflight = new Set<SliceName>();

function setView(name: SliceName, view: AnyView): void {
  slices[name].snap = view;
  slices[name].subs.forEach((fn) => fn());
}

export function subscribe(name: SliceName, fn: () => void): () => void {
  slices[name].subs.add(fn);
  return () => {
    slices[name].subs.delete(fn);
  };
}

// useSlice 分片订阅钩（React 18 原生 useSyncExternalStore——零依赖零样板）。
export function useSlice(name: "hedgeFlow"): SliceView<ListHedgeFlowResponse>;
export function useSlice(name: "closedPositions"): SliceView<ListClosedPositionsResponse>;
export function useSlice(name: "hostMetrics"): SliceView<GetHostMetricsResponse>;
export function useSlice(name: "headless"): SliceView<GetHeadlessResponse>;
export function useSlice(name: SliceName): SliceView<unknown> {
  return useSyncExternalStore(
    (cb) => subscribe(name, cb),
    () => slices[name].snap
  );
}

// hydrate 挂载首取（分片空且闲时才发；推帧/重入不重复请求）。
export function hydrate(name: SliceName, fetcher: () => Promise<unknown>): void {
  const cur = slices[name].snap;
  if (cur.data || cur.loading || inflight.has(name)) return;
  inflight.add(name);
  setView(name, { data: null, err: null, loading: true });
  fetcher().then(
    (d) => {
      inflight.delete(name);
      if (pushed.has(name)) return; // 推帧已新——弃陈回包
      setView(name, { data: d, err: null, loading: false });
    },
    (e: unknown) => {
      inflight.delete(name);
      if (pushed.has(name)) return;
      setView(name, { data: null, err: e instanceof Error ? e.message : String(e), loading: false });
    }
  );
}

// set* 推帧写入（stream.ts v2 专用——最新事实落地，err/loading 清面）。
function setPushed(name: SliceName, d: unknown): void {
  pushed.add(name);
  setView(name, { data: d, err: null, loading: false });
}
export function setHedgeFlow(d: ListHedgeFlowResponse): void {
  setPushed("hedgeFlow", d);
}
export function setClosedPositions(d: ListClosedPositionsResponse): void {
  setPushed("closedPositions", d);
}
export function setHostMetrics(d: GetHostMetricsResponse): void {
  setPushed("hostMetrics", d);
}

// applyHeadless seq 水印去重（T-0263——unary 快照与 ev_headless 流帧同径入：
// 快照 seq=max(行版) 单调轴，陈帧/重帧 seq≤水位即弃；断流后 unary 回补+接流
// seq>水位帧补洞，无重态无回退）。
export function applyHeadless(d: GetHeadlessResponse): void {
  const cur = slices.headless.snap.data as GetHeadlessResponse | null;
  if (cur && d.lastSeq <= cur.lastSeq) return;
  setPushed("headless", d);
}
