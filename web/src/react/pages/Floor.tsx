// Floor.tsx —— 交易大厅页·等距立体景（业主令 11-11 活物世界观第二版）。
// 大厅=O 身体常驻：左墙门/哨兵窗，右墙白板/钉墙/书架/NAV屏/身高尺，
// 值班台=唤知会话，吊灯=数据流活性（断流灯暗不假亮），双钟=UTC+北京，
// 毛坯工位=待长器官。悬停出详情、点击物件到中央（floorFocus 承接）。
import { useEffect, useMemo, useState } from "react";
import { client } from "../api";
import { useLive } from "../hooks";
import { onLive } from "../stream";
import { navigate } from "../router";
import { Skeleton, ErrorState } from "../components/ui";
import { pbDate } from "../components/fmt";
import { AGE } from "../components/panel";
import { plainText } from "../components/plain";
import { P, pts, IsoBox, Person } from "../components/iso";
import { GHOSTS, FocusBody, type Tip } from "./floorFocus";

const fmtMoney = (v: number) =>
  "$" + v.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function FloorPage() {
  const ov = useLive(["ev_position", "ev_account"], () => client.overview({}));
  const g = useLive(["ev_account", "ev_position", "ev_pipeline"], () => client.getFaceGrid({}));
  const t = useLive(["ev_pipeline"], () => client.getDecisionTrail({ limit: 12 }));
  const [live, setLive] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const [tip, setTip] = useState<Tip | null>(null);
  const [focus, setFocus] = useState<Tip | null>(null);
  useEffect(() => onLive((v) => setLive(!!v)), []);
  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    const on = (e: KeyboardEvent) => { if (e.key === "Escape") setFocus(null); };
    window.addEventListener("keydown", on);
    return () => { clearInterval(i); window.removeEventListener("keydown", on); };
  }, []);

  const d = g.data, o = ov.data, s = d?.session;
  const openEsc = useMemo(() => (t.data?.trail ?? []).filter((r) => r.open), [t.data]);
  const openReq = useMemo(() => (t.data?.condreqs ?? []).filter((r) => r.open), [t.data]);
  const probes = d?.probes ?? [];
  const openProbes = probes.filter((p) => p.open).length;
  const engCanon = Number(t.data?.mine?.engCanon ?? 0n);
  const econCanon = Number(t.data?.mine?.econCanon ?? 0n);
  const wallNotes = (t.data?.trail ?? []).slice(0, 3);
  const utc = now.toLocaleTimeString("zh-CN", { timeZone: "UTC", hour12: false }),
    bj = now.toLocaleTimeString("zh-CN", { timeZone: "Asia/Shanghai", hour12: false });

  const hot = (tt: Omit<Tip, "x" | "y">, x: number, y: number, z = 0, cls = "") => {
    const [px, py] = P(x, y, z);
    return {
      onMouseEnter: () => setTip({ ...tt, x: px, y: py }),
      onMouseLeave: () => setTip(null),
      onClick: () => { setTip(null); setFocus({ ...tt, x: px, y: py }); },
      className: "iso-hot" + (cls ? " " + cls : ""),
    };
  };

  if (g.loading && !g.data) return <Skeleton rows={6} />;
  if (g.err && !g.data) return <ErrorState msg={g.err} onRetry={g.reload} />;

  const ROOM_W = 14, ROOM_D = 10, WALL_Z = 3, lampC = P(7, 5, 0);

  return (
    <div className="page v2">
      <div className={`floor-scene${live ? " live" : " dark"}${focus ? " focused" : ""}`}>
        <svg viewBox="-470 -150 940 560" className="floor-svg">
          <defs>
            <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#16233f" />
              <stop offset="100%" stopColor="#0b1220" />
            </linearGradient>
            <radialGradient id="lampGlow">
              <stop offset="0%" stopColor="#ffd76a" stopOpacity="0.55" />
              <stop offset="100%" stopColor="#ffd76a" stopOpacity="0" />
            </radialGradient>
          </defs>

          {/* 地板 + 格线 */}
          <polygon
            points={pts([P(0, 0), P(ROOM_W, 0), P(ROOM_W, ROOM_D), P(0, ROOM_D)])}
            fill="#242a36" stroke="#171c26" strokeWidth="1.5"
          />
          {Array.from({ length: ROOM_W + 1 }, (_, i) => (
            <line key={"gx" + i} x1={P(i, 0)[0]} y1={P(i, 0)[1]} x2={P(i, ROOM_D)[0]} y2={P(i, ROOM_D)[1]} stroke="#2e3543" strokeWidth="0.5" />
          ))}
          {Array.from({ length: ROOM_D + 1 }, (_, i) => (
            <line key={"gy" + i} x1={P(0, i)[0]} y1={P(0, i)[1]} x2={P(ROOM_W, i)[0]} y2={P(ROOM_W, i)[1]} stroke="#2e3543" strokeWidth="0.5" />
          ))}

          {/* 左墙（gx=0 面，+x 朝屋）——门/哨兵窗/UTC 钟 */}
          <polygon
            points={pts([P(0, 0), P(0, ROOM_D), P(0, ROOM_D, WALL_Z), P(0, 0, WALL_Z)])}
            fill="#313949" stroke="#171c26" strokeWidth="1.2"
          />
          {/* 右墙（gy=0 面，+y 朝屋）——白板/钉墙/书架/北京钟/身高尺 */}
          <polygon
            points={pts([P(0, 0), P(ROOM_W, 0), P(ROOM_W, 0, WALL_Z), P(0, 0, WALL_Z)])}
            fill="#3a4256" stroke="#171c26" strokeWidth="1.2"
          />

          {/* 门——左墙前段（gx=0 面上画平行四边形） */}
          <g {...hot({ k: "door", title: "门", body: ["它对外只有一扇门——飞书日行从这里递给你"], to: "" }, 0, 8.9, 1.1)}>
            <polygon points={pts([P(0, 7.8, 0), P(0, 10, 0), P(0, 10, 2.15), P(0, 7.8, 2.15)])} fill="#1d2432" stroke="#11161f" strokeWidth="0.8" />
            <polygon points={pts([P(0, 8.05, 0.05), P(0, 9.75, 0.05), P(0, 9.75, 2.0), P(0, 8.05, 2.0)])} fill="#55617a" stroke="#11161f" strokeWidth="0.6" />
            {(() => { const [kx, ky] = P(0, 9.55, 1.0); return <circle cx={kx} cy={ky} r="2" fill="#ffd76a" />; })()}
          </g>

          {/* 哨兵窗——左墙中段夜空窗，窗台哨兵排 */}
          <g {...hot({
            k: "sentry", title: `哨兵窗 · ${probes.length} 探在册`,
            body: [openProbes ? `${openProbes} 探在报警——点开看哪病了` : "全谱静默——窗瞪着，无事上报"], to: "/device",
          }, 0, 2.7, 1.9)}>
            <polygon points={pts([P(0, 1.3, 1.3), P(0, 4.3, 1.3), P(0, 4.3, 2.75), P(0, 1.3, 2.75)])} fill="url(#sky)" stroke="#4d5a75" strokeWidth="1.4" />
            <line x1={P(0, 2.8, 1.3)[0]} y1={P(0, 2.8, 1.3)[1]} x2={P(0, 2.8, 2.75)[0]} y2={P(0, 2.8, 2.75)[1]} stroke="#4d5a75" strokeWidth="0.8" />
            <line x1={P(0, 1.3, 2.02)[0]} y1={P(0, 1.3, 2.02)[1]} x2={P(0, 4.3, 2.02)[0]} y2={P(0, 4.3, 2.02)[1]} stroke="#4d5a75" strokeWidth="0.8" />
          </g>
          {/* 哨兵——窗下小像，探针态=头部灯色（报警红呼吸/报过黄/静默绿） */}
          {probes.slice(0, 6).map((p, i) => {
            const gy = 1.7 + i * 0.55;
            const col = p.open ? "#ff6b6b" : p.lastEsc ? "#ffd76a" : "#6fd08c";
            return (
              <g key={p.escClass + "/" + p.escScope}
                {...hot({ k: "probe" + i, title: plainText(p.escClass + "/" + p.escScope), body: [p.open ? "正在报警" : "静默在岗"], to: "/device" }, 0.9, gy, 0, p.open ? "fx-breathe" : "")}>
                <Person x={0.9} y={gy} tone={col} />
              </g>
            );
          })}
          {probes.length > 6 && (
            <text x={P(0.9, 5.2)[0]} y={P(0.9, 5.2)[1] - 8} className="iso-cap">+{probes.length - 6}</text>
          )}

          {/* UTC 钟——左墙 */}
          <g {...hot({ k: "utc", title: "UTC 钟", body: [utc + " UTC——机器时间"], to: "" }, 0, 5.8, 2.3)}>
            <ellipse cx={P(0, 5.8, 2.3)[0]} cy={P(0, 5.8, 2.3)[1]} rx="11" ry="6" fill="#0e141f" stroke="#8fa3c8" strokeWidth="1" />
          </g>
          {/* 北京钟——右墙 */}
          <g {...hot({ k: "bj", title: "北京钟", body: [bj + " 北京——你的时间"], to: "" }, 8.5, 0, 2.3)}>
            <ellipse cx={P(8.5, 0, 2.3)[0]} cy={P(8.5, 0, 2.3)[1]} rx="11" ry="6" fill="#0e141f" stroke="#8fa3c8" strokeWidth="1" />
          </g>

          {/* 收银屏——右墙 NAV 大屏（钉死钱径读面） */}
          <g {...hot({
            k: "nav", title: "收银屏 · NAV",
            body: [
              o ? `净值 ${fmtMoney(o.navTotal)}` : "—",
              o ? `今日 ${o.pnlToday >= 0 ? "+" : "−"}${fmtMoney(Math.abs(o.pnlToday))} · 累计 ${o.pnlCum >= 0 ? "+" : "−"}${fmtMoney(Math.abs(o.pnlCum))}` : "",
            ], to: "/",
          }, 5.8, 0, 1.9)}>
            <polygon points={pts([P(4.9, 0, 1.4), P(6.7, 0, 1.4), P(6.7, 0, 2.5), P(4.9, 0, 2.5)])} fill="#0a1020" stroke="#3ddc84" strokeWidth="1.2" />
          </g>

          {/* 日记钉墙——右墙纸条（ops_event 最新钉，绿=已办红=未决） */}
          <g {...hot({
            k: "wall", title: "日记钉墙 · 共享大脑",
            body: wallNotes.length ? wallNotes.map((r) => (r.open ? "🔴 " : "") + plainText(r.note).slice(0, 40)) : ["墙上干净——无新钉"], to: "/decision",
          }, 8.6, 0, 1.6)}>
            {wallNotes.map((r, i) => {
              const x = 7.6 + i * 1.0;
              const face = pts([P(x, 0, 1.15), P(x + 0.72, 0, 1.15), P(x + 0.72, 0, 1.85), P(x, 0, 1.85)]);
              const [px, py] = P(x + 0.36, 0, 1.78);
              return (
                <g key={String(r.escId)}>
                  <polygon points={face} fill="#e8e2d0" stroke="#8a8268" strokeWidth="0.5" transform={`rotate(${i % 2 ? 3 : -2} ${px} ${py})`} />
                  <circle cx={px} cy={py} r="2.6" fill={r.open ? "#ff6b6b" : "#3ddc84"} stroke="#0b0e14" strokeWidth="0.6" />
                </g>
              );
            })}
          </g>

          {/* 经验书架——右墙深段（canon 实体书脊，架空如实空） */}
          <g {...hot({
            k: "shelf", title: "经验书架 · canon",
            body: [`工程 ${engCanon} 盒`, econCanon ? `经济 ${econCanon} 盒` : "经济 空架——裁定 26 件无一入库"], to: "/decision",
          }, 12, 0.5, 1.2)}>
            <IsoBox x={10.4} y={0.12} w={3.3} d={0.7} h={2.4} top="#4d5a75" front="#3c4763" right="#2e3850" />
            {[0.75, 1.5].map((z, zi) => (
              <line key={zi} x1={P(10.45, 0.82, z)[0]} y1={P(10.45, 0.82, z)[1]} x2={P(13.65, 0.82, z)[0]} y2={P(13.65, 0.82, z)[1]} stroke="#1c2333" strokeWidth="1.4" />
            ))}
            {Array.from({ length: Math.min(engCanon, 10) }).map((_, i) => {
              const x = 10.6 + i * 0.3;
              const poly = pts([P(x, 0.82, 1.5), P(x + 0.2, 0.82, 1.5), P(x + 0.2, 0.82, 1.95), P(x, 0.82, 1.95)]);
              return <polygon key={"e" + i} points={poly} fill={i % 3 === 0 ? "#7aa2f7" : i % 3 === 1 ? "#3ddc84" : "#bb9af7"} stroke="#0b0e14" strokeWidth="0.4" />;
            })}
            {econCanon === 0 && (
              <text x={P(11.9, 0.82, 1.1)[0]} y={P(11.9, 0.82, 1.1)[1]} className="iso-cap dim">经济架·空</text>
            )}
          </g>

          {/* 身高尺——右墙角刻度（婴→少年→成年） */}
          <g {...hot({
            k: "ruler", title: "身高尺",
            body: ["现位：婴→少年（A-D 在建）", "判据：点破率归零 + 年化≥8%"], to: "/",
          }, 0.35, 0, 1.5)}>
            <line x1={P(0.35, 0, 0.4)[0]} y1={P(0.35, 0, 0.4)[1]} x2={P(0.35, 0, 2.5)[0]} y2={P(0.35, 0, 2.5)[1]} stroke="#8fa3c8" strokeWidth="1.4" />
            {[{ z: 0.75, lb: "婴" }, { z: 1.5, lb: "少年" }, { z: 2.3, lb: "成年" }].map((m) => (
              <g key={m.lb}>
                <line x1={P(0.35, 0, m.z)[0] - 5} y1={P(0.35, 0, m.z)[1]} x2={P(0.35, 0, m.z)[0] + 5} y2={P(0.35, 0, m.z)[1]} stroke="#8fa3c8" strokeWidth="1.2" />
                <text x={P(0.35, 0, m.z)[0] - 9} y={P(0.35, 0, m.z)[1] + 3} className="iso-cap" textAnchor="end">{m.lb}</text>
              </g>
            ))}
            <path d={`M ${P(0.35, 0, 0.95)[0] + 8} ${P(0.35, 0, 0.95)[1]} l 8 -5 l 0 10 Z`} fill="#ffd76a" />
          </g>

          {/* 白板——立式写字板（工单面钉上） */}
          <g {...hot({
            k: "board", title: "白板 · 待办",
            body: openEsc.length + openReq.length
              ? [...openEsc.slice(0, 3).map((r) => plainText(r.note).slice(0, 36)), ...openReq.slice(0, 2).map((r) => `条件请求 #${String(r.id)}`)]
              : ["板上干净——急件落账后钉在这里"], to: "/decision",
          }, 2.6, 1.6, 1.2)}>
            <IsoBox x={1.6} y={1.5} w={0.16} d={0.16} h={0.7} top="#5a4632" front="#4a3a29" right="#3d2f21" />
            <IsoBox x={3.4} y={1.5} w={0.16} d={0.16} h={0.7} top="#5a4632" front="#4a3a29" right="#3d2f21" />
            <IsoBox x={1.4} y={1.45} w={2.4} d={0.14} h={1.35} z0={0.62} top="#e8e2d0" front="#d9d2bc" right="#c9c1a8" />
            {Array.from({ length: Math.min(openEsc.length + openReq.length, 4) }).map((_, i) => (
              <line key={i}
                x1={P(1.65, 1.59, 1.72 - i * 0.24)[0]} y1={P(1.65, 1.59, 1.72 - i * 0.24)[1]}
                x2={P(3.55, 1.59, 1.72 - i * 0.24)[0]} y2={P(3.55, 1.59, 1.72 - i * 0.24)[1]}
                stroke="#7a6a4d" strokeWidth="1.6" />
            ))}
            {openEsc.length + openReq.length === 0 && (
              <text x={P(2.6, 1.59, 1.35)[0]} y={P(2.6, 1.59, 1.35)[1]} className="iso-cap dim">板上干净</text>
            )}
          </g>

          {/* 值班台——中央桌+椅+经理（在班出人形/下班空椅） */}
          <g {...hot({
            k: "desk", title: "值班台 · 唤知会话",
            body: [
              s?.running ? "在班——正在干活" : s?.lastSpawn ? `下班了（末唤起 ${AGE(pbDate(s.lastSpawn)!)}）` : "唤起机未写行",
              `24h 唤${s?.spawns24h ?? 0} 收${s?.closes24h ?? 0}${(s?.fails24h ?? 0n) > 0n ? ` 败${s?.fails24h}` : ""}`,
            ], to: "/device",
          }, 7, 4.8, 0.8)}>
            <ellipse cx={P(7, 5.6)[0]} cy={P(7, 5.6)[1]} rx="86" ry="26" fill="#1c2230" opacity="0.55" />
            <IsoBox x={5.4} y={4.6} w={3.2} d={1.5} h={0.9} top="#6b543c" front="#5a4632" right="#4a3a29" />
            <IsoBox x={6.6} y={3.6} w={0.8} d={0.8} h={0.55} top="#3a4256" front="#2e3648" right="#252c3c" />
            {/* 桌屏 */}
            <IsoBox x={7.6} y={4.7} w={0.85} d={0.12} h={0.55} z0={0.9} top="#0a1020" front="#0d1526" right="#0a1020" />
            {s?.running && <Person x={7} y={4.0} tone="#7aa2f7" breathe />}
          </g>

          {/* 毛坯工位——前排 8 位（虚线盒=未长，金=在建） */}
          {GHOSTS.map((gh, i) => {
            const x = 0.8 + i * 1.65, y = 8.3;
            const gold = gh.st === "build";
            const c = gold ? { top: "#5c5132", front: "#4d4329", right: "#403723" } : { top: "#2c3444", front: "#252c3a", right: "#1f2531" };
            const [nx, ny] = P(x + 0.6, y + 1.35);
            return (
              <g key={gh.name}
                {...hot({ k: "gh" + i, title: `毛坯位 · ${gh.name}`, body: [gh.hint, gh.st === "build" ? "在建" : "毛坯"], to: "/decision" }, x + 0.6, y + 0.4, 0.8, gold ? "fx-breathe" : "")}>
                <IsoBox x={x} y={y} w={1.2} d={0.8} h={0.7} top={c.top} front={c.front} right={c.right} dashed op={0.85} />
                <text x={nx} y={ny + 8} className={`iso-cap${gold ? " gold" : " dim"}`} textAnchor="middle">{gh.name}</text>
              </g>
            );
          })}

          {/* 吊灯——活性灯（断流=灯暗，不假亮） */}
          <line x1={lampC[0]} y1={-140} x2={lampC[0]} y2={lampC[1] - 66} stroke="#4d5a75" strokeWidth="1.2" />
          {live && <ellipse cx={lampC[0]} cy={lampC[1] - 46} rx="46" ry="20" fill="url(#lampGlow)" />}
          <path d={`M ${lampC[0] - 13} ${lampC[1] - 66} L ${lampC[0] + 13} ${lampC[1] - 66} L ${lampC[0] + 6} ${lampC[1] - 50} L ${lampC[0] - 6} ${lampC[1] - 50} Z`} fill={live ? "#ffd76a" : "#4d5a75"} stroke="#0b0e14" strokeWidth="0.8" />
        </svg>

        {/* 悬停详情卡 */}
        {tip && !focus && (
          <div className="iso-tip" style={{ left: `${((tip.x + 470) / 940 * 100).toFixed(2)}%`, top: `${((tip.y + 150) / 560 * 100).toFixed(2)}%` }}>
            <div className="iso-tip-t">{tip.title}</div>
            {tip.body.map((b, i) => <div key={i} className="iso-tip-b">{b}</div>)}
            <div className="iso-tip-to">点击→到中央</div>
          </div>
        )}

        {/* 聚焦——物件到中央，景压暗让位 */}
        {focus && (
          <>
            <div className="iso-focus-back" onClick={() => setFocus(null)} />
            <div className="iso-focus">
              <button className="iso-focus-x" onClick={() => setFocus(null)}>×</button>
              <FocusBody f={focus} openEsc={openEsc} openReq={openReq} trail={t.data?.trail ?? []}
                probes={probes} openProbes={openProbes} engCanon={engCanon} econCanon={econCanon}
                nav={o ? { total: o.navTotal, today: o.pnlToday, cum: o.pnlCum } : undefined}
                session={s} utc={utc} bj={bj} />
            </div>
          </>
        )}

        {/* 双钟读数（屏幕空间钉在钟面位——viewBox -470/-150 940×560 归一） */}
        {([["utc", P(0, 5.8, 2.3), utc], ["bj", P(8.5, 0, 2.3), bj]] as const).map(([k, [px, py], v]) => (
          <div key={k} className="iso-clock" style={{ left: `${((px + 470) / 940 * 100).toFixed(2)}%`, top: `${((py + 150) / 560 * 100).toFixed(2)}%` }}>{v.slice(0, 5)}</div>))}
        {/* NAV 屏读数 */}
        <div className="iso-nav num" style={{ left: `${((P(5.8, 0, 1.95)[0] + 470) / 940 * 100).toFixed(2)}%`, top: `${((P(5.8, 0, 1.95)[1] + 150) / 560 * 100).toFixed(2)}%` }}>
          {o ? fmtMoney(o.navTotal) : "—"}
        </div>
        <div className="iso-state">{live ? "厅灯亮——数据流活着" : "厅灯暗——数据流断了"}</div>
      </div>
    </div>
  );
}
