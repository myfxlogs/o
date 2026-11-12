// floorOcto.ts —— 器官章鱼的实时信号面(Floor 行数棘轮拆件):
//   腕抽 = 器官域新行到达(id 水位比对,0.9s)  脑闪 = 任何非量表新行
//   在办 = 最新认领行解析  动向 = 最新非量表事实行(它刚干了什么)
import { useEffect, useMemo, useRef, useState } from "react";
import { plainText } from "../components/plain";
import type { Fact } from "./floorFocus";

type OrganLite = { i: number; gh: { keys: string[] } };

export function useOctoFeed(facts: Fact[], organs: OrganLite[]) {
  const [flash, setFlash] = useState<Set<number>>(new Set());
  const seenRef = useRef<Map<number, bigint>>(new Map());
  const initRef = useRef(false);
  useEffect(() => {
    if (!facts.length) return;
    const next = new Set<number>();
    for (const o of organs) {
      const newest = facts.find((f) => f.event !== "量表" && o.gh.keys.some((k) => f.note.includes(k)));
      if (!newest) continue;
      const prev = seenRef.current.get(o.i);
      if (initRef.current && (prev === undefined || newest.id > prev)) next.add(o.i);
      seenRef.current.set(o.i, prev !== undefined && prev > newest.id ? prev : newest.id);
    }
    initRef.current = true;
    if (next.size) {
      setFlash((s) => new Set([...s, ...next]));
      const t = window.setTimeout(() => setFlash((s) => { const m = new Set(s); next.forEach((i) => m.delete(i)); return m; }), 900);
      return () => clearTimeout(t);
    }
  }, [organs]);

  const claim = useMemo(() => {
    const c = facts.find((f) => f.event === "认领");
    const m = c?.note.match(/工单#(\d+)\s*认领[：:]?(.*)/);
    return m ? `在办 #${m[1]} ${plainText(m[2]).replace(/^开修——/, "").slice(0, 18)}` : undefined;
  }, [facts]);
  const lastAct = useMemo(() => {
    const f = facts.find((x) => x.event !== "量表");
    return f ? `${f.event} ${plainText(f.note).replace(/\s+/g, " ").slice(0, 22)}` : undefined;
  }, [facts]);

  const [brainFlash, setBrainFlash] = useState(false);
  const brainSeen = useRef<bigint | undefined>(undefined);
  const brainInit = useRef(false);
  useEffect(() => {
    const newest = facts.find((f) => f.event !== "量表");
    if (!newest) return;
    const prev = brainSeen.current;
    if (brainInit.current && (prev === undefined || newest.id > prev)) {
      setBrainFlash(true);
      const t = window.setTimeout(() => setBrainFlash(false), 900);
      brainSeen.current = newest.id;
      return () => clearTimeout(t);
    }
    brainSeen.current = newest.id;
    brainInit.current = true;
  }, [facts]);

  return { flash, claim, lastAct, brainFlash };
}
