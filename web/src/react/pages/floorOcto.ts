// floorOcto.ts —— 器官章鱼的实时信号面(Floor 行数棘轮拆件):
//   腕抽 = 器官域新行到达(id 水位比对,0.9s)  脑闪 = 任何非量表新行
//   在办 = 开检查点真源(cockpit.doing——unit粒度服务端闭账;fallback=认领行解析)
//   动向 = 最新非量表事实行(它刚干了什么)
import { useEffect, useMemo, useRef, useState } from "react";
import { plainText } from "../components/plain";
import type { Fact } from "./floorFocus";
import type { CogDoingRow } from "../../gen/dashboardv1/core_pb";

type OrganLite = { i: number; gh: { keys: string[] } };

export function useOctoFeed(facts: Fact[], organs: OrganLite[], doings: CogDoingRow[] = []) {
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
    // 真源=cockpit.doing 开检查点(服务端闭账):在干行优先,纯悬账=猝死残态显形;
    // 多场并行→最新场文本 +N场 计数。无开检查点→退认领旧径→「收棒中」。
    const open = doings.filter((d) => d.state === "在干");
    const pool = open.length ? open : doings;
    if (pool.length) {
      const d = pool[0];
      const tag = d.state === "在干" ? "在办" : "悬账";
      const more = pool.length > 1 ? ` +${pool.length - 1}场` : "";
      return `${tag} ${plainText(d.doing).replace(/\s+/g, " ").slice(0, 16)}${more}`;
    }
    const c = facts.find((f) => f.event === "认领");
    const m = c?.note.match(/工单#(\d+)\s*认领[：:]?(.*)/);
    return m ? `在办 #${m[1]} ${plainText(m[2]).replace(/^开修——/, "").slice(0, 18)}` : undefined;
  }, [facts, doings]);
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
