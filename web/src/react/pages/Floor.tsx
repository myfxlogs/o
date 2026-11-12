// Floor.tsx —— 它在干啥（业主令 11-12 瓦片版：iso 场景全撤,回归站内
// v2 面板语言——hero+异常条+栅格面板;每块瓦片=事实+态色+点击详情,
// 态推导/事实锚/FocusBody 弹窗全保留,零装饰像素)。
import { useEffect, useMemo, useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { onLive } from "../stream";
import { Skeleton, ErrorState } from "../components/ui";
import { plainText } from "../components/plain";
import { navigate } from "../router";
import { FocusBody, GHOSTS, ghostFact, ghostSt, type Tip } from "./floorFocus";

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const RACKS = ["o-core", "PG", "collector", "projector", "sweeper", "eventbus"];
const ST_TXT: Record<string, string> = { ghost: "毛坯", pin: "已立项", build: "在建", done: "已完工" };

export function FloorPage() {
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () => client.getFaceGrid({}));
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 12 }));
  const [live, setLive] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const [focus, setFocus] = useState<Tip | null>(null);
  useEffect(() => onLive((v) => setLive(!!v)), []);
  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    const on = (e: KeyboardEvent) => { if (e.key === "Escape") setFocus(null); };
    window.addEventListener("keydown", on);
    return () => { clearInterval(i); window.removeEventListener("keydown", on); };
  }, []);

  const d = g.data, o = ov.data, s = d?.session;
  const facts = useMemo(() => t.data?.facts ?? [], [t.data]);
  const openEsc = useMemo(() => (t.data?.trail ?? []).filter((r) => r.open), [t.data]);
  const openReq = useMemo(() => (t.data?.condreqs ?? []).filter((r) => r.open), [t.data]);
  const vens = useMemo(() => { // money 行按 venue 分组合计
    const m = new Map<string, number>();
    for (const r of o?.money ?? []) m.set(r.venue, (m.get(r.venue) ?? 0) + (r.usd || 0));
    return [...m.entries()]; }, [o]);
  const probes = d?.probes ?? [];
  const openProbes = probes.filter((p) => p.open).length;
  const engCanon = Number(t.data?.mine?.engCanon ?? 0n);
  const econCanon = Number(t.data?.mine?.econCanon ?? 0n);
  const utc = now.toLocaleTimeString("zh-CN", { timeZone: "UTC", hour12: false }),
    bj = now.toLocaleTimeString("zh-CN", { timeZone: "Asia/Shanghai", hour12: false });

  // 器官态推导——同 facts 窗驱动（GhostRow 同一套,点击开详情）
  const organs = GHOSTS.map((gh, i) => ({
    gh, i, st: ghostSt(gh, facts), fact: ghostFact(gh, facts),
  }));
  const doneN = organs.filter((x) => x.st === "done").length;
  const buildN = organs.filter((x) => x.st === "build").length;
  const pinN = organs.filter((x) => x.st === "pin").length;
  const queue = [...openEsc.map((r) => ({ id: String(r.escId), note: r.note, open: true })),
    ...openReq.map((r) => ({ id: String(r.id), note: "条件请求·待处理", open: true }))];

  // 事件类计数——底条机速同源
  const evTop = (() => {
    const m = new Map<string, number>();
    for (const f of facts) m.set(f.event ?? "?", (m.get(f.event ?? "?") ?? 0) + 1);
    return [...m.entries()].sort((a, b) => b[1] - a[1]).slice(0, 6);
  })();

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;

  const bad: string[] = [];
  for (const p of probes) if (p.open) bad.push(plainText(p.escClass + "/" + p.escScope));
  for (const r of openEsc) bad.push("待办 #" + String(r.escId));
  if (!live) bad.push("数据流断");

  return (
    <div className="page v2">
      <section className="v2-hero">
        <div className="v2-eyebrow">
          <span className={`v2-livedot${live ? "" : " off"}`} />
          它在干啥 · LIVE
        </div>
        <h1 className="v2-title">
          {doneN} 工位完工 · {buildN} 在建 · {pinN} 立项 · {queue.length} 件待办
        </h1>
        <p className="v2-lede">
          NAV {o ? fmtMoney(o.navTotal) : "—"} · 探针未决 {openProbes} · 会话
          {s?.running ? "在班" : "下班"}(24h唤{String(s?.spawns24h ?? "—")}败{String(s?.fails24h ?? "—")})
          · UTC {utc.slice(0, 5)} · 北京 {bj.slice(0, 5)}
        </p>
      </section>

      {bad.length > 0 && (
        <div className="v2-alert">⚠ {bad.slice(0, 6).join(" · ")}{bad.length > 6 ? ` · 另${bad.length - 6}件` : ""}</div>
      )}

      <div className="v2-cols">
        {/* 器官工位阵——9 瓦片,态色左条=事实推导,点击开详情 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">器官工位</h2>
            <span className="ink-3">{doneN}/{organs.length} 完工</span>
          </div>
          <div className="ft-tiles">
            {organs.map(({ gh, i, st, fact }) => (
              <button key={gh.name} className={`ft-tile st-${st}${st === "build" ? " breathe" : ""}`}
                onClick={() => setFocus({ k: "gh" + i, x: 0, y: 0, title: `工位 · ${gh.name}`, body: [gh.hint], to: "/decision" })}>
                <span className="ft-tile-name">{gh.name}</span>
                <span className="ft-tile-st">{ST_TXT[st]}{st === "done" ? " ✓" : st === "pin" ? " 📌" : ""}</span>
                <span className="ft-tile-sub">{fact ? `据 #${String(fact.id)}` : "无事实行"}</span>
              </button>
            ))}
          </div>
        </section>

        {/* 钱径——NAV + venue 分腿,点击开详情 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">钱径</h2>
            <span className="ink-3 num">{o ? fmtMoney(o.navTotal) : "—"}</span>
          </div>
          <div className="v2-rows">
            <button className="v2-row ft-row" onClick={() => setFocus({ k: "nav", x: 0, y: 0, title: "NAV · 收银", body: [], to: "/" })}>
              <span className="dot s0" />
              <div className="v2-row-main">
                <div className="v2-row-title">NAV 总净值</div>
                <div className="v2-row-note">今日 {o ? (o.pnlToday >= 0 ? "+" : "−") + fmtMoney(Math.abs(o.pnlToday)) : "—"} · 累计 {o ? (o.pnlCum >= 0 ? "+" : "−") + fmtMoney(Math.abs(o.pnlCum)) : "—"}</div>
              </div>
              <div className="v2-row-sub num">{o ? fmtMoney(o.navTotal) : "—"}</div>
            </button>
            {vens.map(([vn, usd]) => (
              <button key={vn} className="v2-row ft-row" onClick={() => setFocus({ k: "ven:" + vn, x: 0, y: 0, title: `${vn} 资金`, body: [`合计 ${fmtMoney(usd)}`], to: "/" })}>
                <span className="dot s0" />
                <div className="v2-row-main">
                  <div className="v2-row-title">{vn}</div>
                </div>
                <div className="v2-row-sub num">{fmtMoney(usd)}</div>
              </button>
            ))}
          </div>
        </section>

        {/* 待办队列——openEsc/openReq 直读 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">待办队列</h2>
            <span className="ink-3">{queue.length} 件</span>
          </div>
          <div className="v2-rows">
            {queue.length === 0 && <div className="v2-row"><div className="v2-row-main"><div className="v2-row-note">板上干净——零积压</div></div></div>}
            {queue.slice(0, 8).map((q) => (
              <button key={q.id} className="v2-row ft-row" onClick={() => navigate("/decision")}>
                <span className={`dot ${q.open ? "s2" : "s3"}`} />
                <div className="v2-row-main">
                  <div className="v2-row-title">#{q.id} <span className="v2-row-note" style={{ display: "inline" }}>{plainText(q.note).slice(0, 60)}</span></div>
                </div>
              </button>
            ))}
            {queue.length > 8 && <div className="v2-row"><div className="v2-row-main"><div className="v2-row-note">…另 {queue.length - 8} 件,进「何事要裁」看全谱</div></div></div>}
          </div>
        </section>

        {/* 感知哨兵——探针排 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">感知哨兵</h2>
            <span className="ink-3">{openProbes ? `${openProbes} 报警` : "全静默"}</span>
          </div>
          <div className="v2-rows">
            {probes.slice(0, 8).map((p, i) => (
              <button key={p.escClass + "/" + p.escScope} className="v2-row ft-row"
                onClick={() => setFocus({ k: "probe" + i, x: 0, y: 0, title: plainText(p.escClass + "/" + p.escScope), body: [p.open ? "正在报警" : "静默在岗"], to: "/device" })}>
                <span className={`dot ${p.open ? "s2 breathe" : p.lastEsc ? "s1" : "s0"}`} />
                <div className="v2-row-main">
                  <div className="v2-row-title">{plainText(p.escClass)}</div>
                  <div className="v2-row-note">{plainText(p.escScope)}</div>
                </div>
              </button>
            ))}
            {probes.length > 8 && <div className="v2-row"><div className="v2-row-main"><div className="v2-row-note">…另 {probes.length - 8} 探,进「哪病了」看全谱</div></div></div>}
          </div>
        </section>

        {/* 常驻机 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">常驻机</h2>
            <span className="ink-3">{live ? "流活" : "流断"}</span>
          </div>
          <div className="ft-tiles">
            {RACKS.map((nm) => (
              <button key={nm} className="ft-tile st-rack" onClick={() => setFocus({ k: "srv:" + nm, x: 0, y: 0, title: `机柜 · ${nm}`, body: [], to: "/device" })}>
                <span className="ft-tile-name">{nm}</span>
                <span className="ft-tile-st"><span className={`dot ${live ? "s0" : "s2"}`} />{live ? "运行" : "断流"}</span>
              </button>
            ))}
          </div>
        </section>

        {/* 共享大脑——canon + 会话席 + 成长 */}
        <section className="v2-panel">
          <div className="v2-panel-head">
            <h2 className="v2-panel-title">共享大脑</h2>
            <span className="ink-3">eng {engCanon} · econ {econCanon}</span>
          </div>
          <div className="v2-rows">
            <button className="v2-row ft-row" onClick={() => setFocus({ k: "desk", x: 0, y: 0, title: "会话席", body: [], to: "/device" })}>
              <span className={`dot ${s?.running ? "s0" : "s3"}`} />
              <div className="v2-row-main">
                <div className="v2-row-title">会话席 · {s?.running ? "在班" : "下班"}</div>
                <div className="v2-row-note">24h 唤{String(s?.spawns24h ?? "—")} 收{String(s?.closes24h ?? "—")} 败{String(s?.fails24h ?? "—")}</div>
              </div>
            </button>
            <button className="v2-row ft-row" onClick={() => setFocus({ k: "shelf", x: 0, y: 0, title: "经验书架", body: [], to: "/decision" })}>
              <span className="dot s0" />
              <div className="v2-row-main">
                <div className="v2-row-title">经验书架 · canon</div>
                <div className="v2-row-note">工程 {engCanon} 盒 · 经济 {econCanon} 盒</div>
              </div>
            </button>
            <button className="v2-row ft-row" onClick={() => setFocus({ k: "ruler", x: 0, y: 0, title: "身高尺", body: [], to: "/" })}>
              <span className="dot s3" />
              <div className="v2-row-main">
                <div className="v2-row-title">成长位 · 婴→少年</div>
                <div className="v2-row-note">判据:点破率30d=0 + 年化≥8%</div>
              </div>
            </button>
            <button className="v2-row ft-row" onClick={() => setFocus({ k: "wall", x: 0, y: 0, title: "日记钉墙", body: [], to: "/decision" })}>
              <span className="dot s1" />
              <div className="v2-row-main">
                <div className="v2-row-title">日记钉墙 · ops_event</div>
                <div className="v2-row-note">窗内 {facts.length} 行 · {evTop.slice(0, 3).map(([e, n]) => `${e}×${n}`).join(" ")}</div>
              </div>
            </button>
          </div>
        </section>
      </div>

      {/* ops_event 尾流——整宽面板,事件计数+最新行 */}
      <section className="v2-panel">
        <div className="v2-panel-head">
          <h2 className="v2-panel-title">事实尾流 · ops_event</h2>
          <span className="ink-3">{evTop.map(([e, n]) => `${e}×${n}`).join(" · ")}</span>
        </div>
        <div className="tile-scroll">
          <div className="v2-rows">
            {facts.filter((f) => f.event !== "量表").slice(0, 12).map((f) => (
              <div key={String(f.id)} className="v2-row">
                <span className="dot s3" />
                <div className="v2-row-main">
                  <div className="v2-row-title">{f.event ?? "?"} <span className="v2-row-id">#{String(f.id)}</span></div>
                  <div className="v2-row-note">{plainText(f.note)}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 聚焦弹窗——点瓦片开详情(原 FocusBody 全保留) */}
      {focus && (<>
        <div className="iso-focus-back" onClick={() => setFocus(null)} />
        <div className="iso-focus">
          <button className="iso-focus-x" onClick={() => setFocus(null)}>×</button>
          <FocusBody f={focus} openEsc={openEsc} openReq={openReq} trail={t.data?.trail ?? []} facts={facts}
            probes={probes} openProbes={openProbes} engCanon={engCanon} econCanon={econCanon}
            nav={o ? { total: o.navTotal, today: o.pnlToday, cum: o.pnlCum } : undefined}
            money={o?.money ?? []} live={live} session={s} utc={utc} bj={bj} />
        </div>
      </>)}
    </div>
  );
}
