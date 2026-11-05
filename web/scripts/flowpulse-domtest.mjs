// flowpulse-domtest.mjs —— T-0362 活水段流动骨架 DOM 断言（范式同
// metaseats-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①管道形（pipe-link 连接线+▸ 箭头数=段数−1，态点四色 s0~s3 齐档）；
// ②滞后全人话档（无裸 m/h 小数）；③判词行实出且断点定位正确（灰中性段跳过）；
// ④饿殍期分支（eval 态标派生）；⑤数据零新增（只渲 FlowStage 现字段）。
// 运行：node scripts/flowpulse-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync } from "node:fs";
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
const stage = (key, label, state, lagS, count24h, { ok = true, status = "", expectS = 60 } = {}) => ({
	key, label, state, ok, status,
	lagS: BigInt(lagS), count24h: BigInt(count24h), expectS: BigInt(expectS),
	lastTs: lagS >= 0 ? { seconds: BigInt(nowS - lagS) } : undefined,
});
const render = (stages) =>
	renderToStaticMarkup(h(FlowPulseView, { stages, rosterCount: 3, rosterWorstLagS: 90n }));

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

// —— 全活场景：7 段全绿 ——
const labels = ["采集", "桥接", "评估", "提案", "执行", "报表", "侦察"];
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
	stage("bridge", "桥接", 3, 30, 0),          // 灰中性——跳过不算断点本身
	stage("eval", "评估", 2, 7200, 0),
	stage("proposal", "提案", 3, 30, 0),
	stage("exec", "执行", 3, 30, 0),
	stage("report", "报表", 1, 500, 4),
	stage("scout", "侦察", 3, -1, 0, { status: "扫描中" }),
];
const htmlBad = render(broken);
check("③断点定位=首个红段(桥接→评估)", htmlBad.includes('class="pipe-verdict s2"') && htmlBad.includes("断在 桥接→评估"));
check("③红段卡态点 s2 在案", htmlBad.includes("dot s2"));
check("③黄段卡态点 s1 在案", htmlBad.includes("dot s1"));
check("③灰中性卡态点 s3 在案", htmlBad.includes("dot s3"));
check("③7200s→2小时前(无裸露小数)", htmlBad.includes("2小时前") && !htmlBad.includes("2.0h"));
check("③无行段→status 摘要显形", htmlBad.includes("扫描中") && htmlBad.includes("无行"));

// —— 饿殍期：全绿/灰但 eval 带态标 ——
const famine = keys.map((k, i) =>
	stage(k, labels[i], i === 6 ? 3 : 0, 60, 0, i === 2 ? { status: "评估连续全拒（阈未达）" } : {}));
const htmlFam = render(famine);
check("④判词行=饿殍期·eval态标", htmlFam.includes("饿殍期·评估连续全拒（阈未达）"));
check("④判词 s0 中性色", htmlFam.includes('class="pipe-verdict s0"'));

// —— 读败段：ok=false → 缺/查询败 ——
const dead = [stage("collect", "采集", 0, 30, 5, { ok: false })];
const htmlDead = render(dead);
check("⑤查询败段→缺+查询败", htmlDead.includes('pipe-card-num">缺') && htmlDead.includes("查询败"));

console.log(`\n${n - fail}/${n} PASS${fail ? ` —— ${fail} FAIL` : ""}`);
process.exit(fail ? 1 : 0);
