// hooks.ts —— 读面钩子（T-0170：快照抓取+Watch 主题局部失效+加载/错误态
// 归一——原每组件手写 useEffect 样板收拢；诚实空态律=错误显形不假零）。
import { useCallback, useEffect, useRef, useState } from "react";
import { onData } from "./stream";

export type Snap<T> = {
  data: T | null;
  err: string | null;
  loading: boolean;
  reload: () => void;
};

// useSnap：抓一次 + 每逢 topics 命中的 Watch 帧重抓（topics 空=不订阅事件）。
export function useSnap<T>(
  topics: string[],
  fetcher: () => Promise<T>
): Snap<T> {
  const [data, setData] = useState<T | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    fetcherRef
      .current()
      .then((d) => {
        if (!alive) return;
        setData(d);
        setErr(null);
      })
      .catch((e: unknown) => {
        if (!alive) return;
        setErr(e instanceof Error ? e.message : String(e));
      })
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [tick]);

  useEffect(() => {
    if (topics.length === 0) return;
    return onData((v) => {
      const topic = (v as { topic?: string } | undefined)?.topic;
      if (topic && topics.includes(topic)) setTick((t) => t + 1);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, topics);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, err, loading, reload };
}
