// flowpulse-domtest.mjs —— T-0362 活水段流动骨架 DOM 断言（范式同
// metaseats-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①管道形（pipe-link 连接线+▸ 箭头数=段数−1，态点四色 s0~s3 齐档）；
// ②滞后全人话档（无裸 m/h 小数）；③判词行实出且断点定位正确（灰中性段跳过）；
// ④饿殍期分支（eval 态标派生）；⑤数据零新增（只渲 FlowStage 现字段）；
// ⑥拒因人话面（T-0366——可见短句+悬停详述带数值，机器原文禁入渲染树任何层）。
// 运行：node scripts/flowpulse-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL } from "node:url";
import { fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "flowpulse.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { FlowPulseView } from "${comp}";
export { renderToStaticMarkup, h, FlowPulseView };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "flowpulse-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	// flowpulse.tsx 顶链引 api.ts（window.location.origin 建 transport）——垫伪窗
	// 只为捆扎求值过（断言走 FlowPulseView 纯件不触 client）。
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" } };` },
});
const tmp = mkdtempSync(join(tmpdir(), "fpdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, FlowPulseView } = await import(pathToFileURL(outFile).href);

const nowS = Math.floor(Date.now() / 1000);
// stub breathe 默认镜像服务端派生（T-0406）：ok 且（事件驱动中性段|绿灯）即呼。
const NEUTRAL_KEYS = new Set(["proposal", "strategy", "exec"]);
const stage = (key, label, state, lagS, count24h, { ok = true, status = "", expectS = 60, breathe } = {}) => ({
	key, label, state, ok, status,
	breathe: breathe ?? (ok && (NEUTRAL_KEYS.has(key) || state === 0)),
	lagS: BigInt(lagS), count24h: BigInt(count24h), expectS: BigInt(expectS),
	lastTs: lagS >= 0 ? { seconds: BigInt(nowS - lagS) } : undefined,
});
const render = (stages) =>
	renderToStaticMarkup(h(FlowPulseView, { stages, rosterCount: 3, rosterWorstLagS: 90n }));

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

// —— 全活场景：7 段全绿 ——
const labels = ["采集", "投影", "评估", "提案", "执行", "报表", "侦察"];
const keys = ["collect", "bridge", "eval", "proposal", "exec", "report", "scout"];
const allGreen = keys.map((k, i) => stage(k, labels[i], 0, 30, 12 + i));
const htmlOk = render(allGreen);
check("①管道形:pipe-card=7", (htmlOk.match(/pipe-card"/g) || []).length === 7);
check("①管道形:pipe-link 连接线+箭头=6", (htmlOk.match(/pipe-link/g) || []).length === 6
	&& (htmlOk.match(/pipe-line/g) || []).length === 6
	&& (htmlOk.match(/▸/g) || []).length >= 6);
check("②判词行=活水正常(s0)", htmlOk.includes('class="pipe-verdict s0"') && htmlOk.includes("活水正常"));
check("②滞后全人话:30s→刚刚", htmlOk.includes("刚刚") && !htmlOk.includes("0.5m") && !htmlOk.includes("0.0h"));
check("②中大数=count24h", htmlOk.includes('pipe-card-num">12') && htmlOk.includes('pipe-card-num">18'));
check("②态点全绿 breathe", (htmlOk.match(/dot s0 breathe/g) || []).length === 7);

// —— 断点场景：灰中性段跳过，首个红段定位，前段名带出 ——
const broken = [
	stage("collect", "采集", 0, 30, 100),
	stage("bridge", "投影", 3, 30, 0),          // 灰中性——跳过不算断点本身
	stage("eval", "评估", 2, 7200, 0),
	stage("proposal", "提案", 3, 30, 0),
	stage("exec", "执行", 3, 30, 0),
	stage("report", "报表", 1, 500, 4),
	stage("scout", "侦察", 3, -1, 0, { status: "扫描中" }),
];
const htmlBad = render(broken);
check("③断点定位=首个红段(投影→评估)", htmlBad.includes('class="pipe-verdict s2"') && htmlBad.includes("断在 投影→评估"));
check("③红段卡态点 s2 在案", htmlBad.includes("dot s2"));
check("③黄段卡态点 s1 在案", htmlBad.includes("dot s1"));
check("③灰中性卡态点 s3 在案", htmlBad.includes("dot s3"));
// T-0406 饿≠死：中性段零事件=呼吸待命（dot s3 breathe——活着的灰脉动），
// 与真死态（s2 静止/ok=false 缺）节奏可辨。
check("③中性待命=呼吸灰(dot s3 breathe)", (htmlBad.match(/dot s3 breathe/g) || []).length >= 2);
check("③7200s→2小时前(无裸露小数)", htmlBad.includes("2小时前") && !htmlBad.includes("2.0h"));
check("③无数据状态点人话", htmlBad.includes("扫描中") && htmlBad.includes("暂无数据"));

// —— 饿殍期：全绿/灰但 eval 带态标 ——
const famine = keys.map((k, i) =>
	stage(k, labels[i], i === 6 ? 3 : 0, 60, 0, i === 2 ? { status: "评估连续全拒（阈未达）" } : {}));
const htmlFam = render(famine);
check("④判词行=饿殍期·eval态标(人话化)", htmlFam.includes("饿殍期·评估连续全拒（未过放行线）") && !htmlFam.includes("阈"));
check("④判词 s0 中性色", htmlFam.includes('class="pipe-verdict s0"'));

// —— 读败段：ok=false → 缺/查询败 ——
const dead = [stage("collect", "采集", 0, 30, 5, { ok: false })];
const htmlDead = render(dead);
check("⑤查询败段→缺+读取失败", htmlDead.includes('pipe-card-num">缺') && htmlDead.includes("读取失败"));
// T-0406：死态不呼——查询败段态点静止（服务端 breathe=false）。
check("⑤死态不呼(无 breathe 类)", !htmlDead.includes("breathe"));

// —— 拒因人话面（T-0366：可见面短句人话/悬停人话详述带数值/机器原文
// 禁入渲染树任何层含 title/aria——net_apr|阈|armed|snake_case 断言）——
const rejected = [
	stage("collect", "采集", 0, 30, 100),
	stage("bridge", "投影", 0, 60, 60),
	stage("eval", "评估", 0, 120, 47),
	stage("proposal", "提案", 3, 30, 0, { status: "闲·评估全拒（末因 net_apr<阈）" }),
	stage("exec", "执行", 3, 30, 0, { status: "闲·无 armed" }),
	stage("report", "报表", 0, 300, 4),
	stage("scout", "侦察", 0, 400, 2),
];
const htmlRej = render(rejected);
check("⑥闲态大格单显闲且走饿色档", /pipe-card-num pipe-card-num-hungry">闲<\/div>/.test(htmlRej));
check("⑥机器原文禁入渲染树任何层(net_apr/阈/armed)",
	!/(net_apr|阈|armed|below_earn|basis_below|apr_est)/.test(htmlRej));
check("⑥悬停人话详述带数值(评估47个机会)",
	htmlRej.includes("评估 47 个机会全部未过放行线") && htmlRej.includes("测算净年化低于放行线"));
check("⑥闲态详述留在悬停", htmlRej.includes("测算净年化低于放行线") && htmlRej.includes("近 24 小时无成交/持仓类真金白银流水"));
const armed = keys.map((k, i) =>
	stage(k, labels[i], 0, 30, k === "proposal" ? 0 : 5, k === "proposal" ? { status: "活跃·3 armed" } : {}));
const htmlArm = render(armed);
check("⑥活态人话=3 个在飞提案+详述", htmlArm.includes("活跃·3 个在飞提案")
	&& htmlArm.includes("3 个提案已放行") && !/armed/.test(htmlArm));
const unknown = keys.map((k, i) =>
	stage(k, labels[i], 0, 30, 5, k === "proposal" ? { status: "闲·评估全拒（末因 exotic_new_reason_v2）" } : {}));
const htmlUnk = render(unknown);
check("⑥未识拒因→人话兜底不露原文", htmlUnk.includes("未过放行线") && !/exotic_new_reason_v2/.test(htmlUnk));
// 等宽锚：pipe-card flex 等分+态标截断规则在案（CSS 静态断言——布局契约随码走）。
const css = readFileSync(join(webRoot, "src/react/styles/20-cards.css"), "utf8");
const cardRule = css.match(/\.pipe-card\s*\{[^}]+\}/s)?.[0] ?? "";
const numRule = css.match(/\.pipe-card-num\s*\{[^}]+\}/s)?.[0] ?? "";
check("⑥七段等宽:pipe-card flex:1 1 0+min-width:0",
	/flex:\s*1\s+1\s+0/.test(cardRule) && /min-width:\s*0/.test(cardRule));
check("⑥态标截断:pipe-card-num ellipsis",
	/text-overflow:\s*ellipsis/.test(numRule) && /white-space:\s*nowrap/.test(numRule));

// T-0371：七段 tooltip 全谱采用语义表人话、事件驱动不贴固定节拍。
const expectedHover = [
	"OKX 资金费率行情采集臂；约每 5 分钟一次", "条行情数据",
	"行情投影入账桥（开发环境到生产镜像）；约每 10 分钟一次", "帧",
	"策略机会评估臂（净年化过放行线才出提案）；约每 30 分钟一次", "次评估",
	"已放行提案（待开仓或候选就绪待触发）；事件驱动，无固定节奏", "个提案",
	"真金白银流水（成交、利息、费用）；事件驱动，无固定节奏", "笔流水",
	"每日结算报表；约每天一次", "份报表",
	"全球交易所进水口侦察度量；约每天一次", "个交易所",
];
check("⑦七段悬停含段义/人话节奏/计数单位", expectedHover.every((s) => htmlOk.includes(s)));
check("⑦事件驱动段不显示周期或最近写入",
	!htmlOk.includes("预期节拍")
	&& !htmlOk.includes("已放行提案（待开仓或候选就绪待触发）；事件驱动，无固定节奏，最近一次写入")
	&& !htmlOk.includes("真金白银流水（成交、利息、费用）；事件驱动，无固定节奏，最近一次写入"));
check("⑦状态点悬停用最近一次写入/暂无数据/读取失败人话",
	htmlOk.includes("最近一次写入刚刚") && !/(末行|无行|查询败显缺)/.test(htmlOk));
check("⑦查询败悬停说明该段显示为缺", htmlDead.includes("读取失败——该段显示为缺"));
check("⑦闲态拒因详述仍可悬停查看", htmlRej.includes("测算净年化低于放行线"));

console.log(`\n${n - fail}/${n} PASS${fail ? ` —— ${fail} FAIL` : ""}`);
process.exit(fail ? 1 : 0);
