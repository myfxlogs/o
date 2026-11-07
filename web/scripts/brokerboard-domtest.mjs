// brokerboard-domtest.mjs —— T-0430 broker 接入卡 DOM 断言（范式同
// quoteboard-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①三步向导步序（步1=类型+venue 步进件/步2=凭据字段/步3=预览+
// 提交——initialStep 注面驱动）；②返步（步2/3 上一步钮在）；③败因显示
// （initialErr 注面=操作未成行）；④MT4 选项不存（value="mt4"/<option>MT4
// 零渲染——存量遗留行显形除外）；⑤卡位（复盘页第三卡已撤/顶导航「接入」
// 路由注册/Onboard 页挂卡——源码 fs 断言）；⑥凭据不回显（password/
// api input type=password 或掩码——raw 值零渲染）；⑦列表 phase 态
// （verified/failed chip/CEX 行/MT4 遗留「链未接」如实显形）；⑧空态。
// 运行：node scripts/brokerboard-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const reactRoot = join(webRoot, "src", "react");
const comp = join(reactRoot, "components", "brokerboard.tsx").replaceAll("\\", "/");

const entry = `
import { renderToStaticMarkup } from "react-dom/server";
import { createElement as h } from "react";
import { BrokerOnboardView } from "${comp}";
export { renderToStaticMarkup, h, BrokerOnboardView };
`;

const res = await build({
	stdin: { contents: entry, resolveDir: webRoot, loader: "ts", sourcefile: "brokerboard-domtest-entry.ts" },
	bundle: true, write: false, format: "esm", platform: "node",
	jsx: "automatic", logLevel: "silent",
	banner: { js: `import { createRequire } from "node:module"; const require = createRequire(import.meta.url); globalThis.window ??= { location: { origin: "http://domtest" } };` },
});
const tmp = mkdtempSync(join(tmpdir(), "bbdt-"));
const outFile = join(tmp, "entry.mjs");
writeFileSync(outFile, res.outputFiles[0].text);
const { renderToStaticMarkup, h, BrokerOnboardView } = await import(pathToFileURL(outFile).href);

let fail = 0, n = 0;
const check = (name, cond) => { n++; console.log(`${cond ? "PASS" : "FAIL"} ${name}`); if (!cond) fail++; };

const nowS = Math.floor(Date.now() / 1000);
const row = (venue, proto, opts = {}) => ({
	venue, proto,
	login: opts.login ?? "",
	state: opts.state ?? "",
	phase: opts.phase ?? "",
	beatNote: opts.beatNote ?? "",
	lastBeatAt: opts.beatAgeS !== undefined ? { seconds: BigInt(nowS - opts.beatAgeS) } : undefined,
	beatFailed: opts.beatFailed ?? false,
	strategies: BigInt(opts.strategies ?? 0),
});
const render = (rows, extra = {}) =>
	renderToStaticMarkup(h(BrokerOnboardView, { rows, onSubmit: async () => {}, ...extra }));

const rows = [
	row("exness", "mt5", { login: "3001157", state: "probed", phase: "verified", beatNote: "beat=探测 Connect ok", beatAgeS: 60, strategies: 2 }),
	row("okx", "cex", { state: "", phase: "verified", beatNote: "venue=okx proto=cex balance ok totalEq=7.89" }),
	row("badbro", "mt5", { login: "9", state: "registered", phase: "failed", beatNote: "败径 fail-closed：凭据拒", beatFailed: true, beatAgeS: 30 }),
	row("fxmt4", "mt4", { login: "777" }),
];
const listHtml = render(rows);

// ⑦列表 phase 态
check("⑦verified 人话（已验证）", listHtml.includes("已验证"));
check("⑦CEX 行渲出（okx venue+CEX 协议列）", listHtml.includes(">okx<") && listHtml.includes(">CEX</td>"));
check("⑦failed chip（接入败）", listHtml.includes("接入败"));
check("⑦败因列渲出（凭据拒）", listHtml.includes("凭据拒"));
check("⑦MT4 遗留=链未接如实显形", listHtml.includes("MT4 链未接"));
check("⑦login 渲出", listHtml.includes("3001157"));

// ①步序——步1 缺省：类型+venue 件在，凭据件不在
const s1 = render(rows);
check("①步1 类型下拉（MT5+CEX）", s1.includes('value="mt5"') && s1.includes('value="cex"'));
check("①步1 venue 件在", s1.includes('placeholder="broker 名（小写数线横）"'));
check("①步1 凭据件不在", !s1.includes('name') || (!s1.includes('placeholder="登录号"') && !s1.includes('placeholder="API Key"')));
check("①步1 下一步钮在", s1.includes("下一步"));
check("①步1 无提交钮", !s1.includes("验证并接入"));
// ①步2（MT5 凭据——initialDraft 注面）
const s2mt5 = render(rows, { initialStep: 2, initialDraft: { venue: "exness", proto: "mt5" } });
check("①步2 MT5 凭据件齐（login/password/server/host）",
	s2mt5.includes('placeholder="登录号"') && s2mt5.includes('placeholder="密码"') &&
	s2mt5.includes('placeholder="服务器名"') && s2mt5.includes('placeholder="host:port（选填）"'));
check("①步2 上一步钮在（返步）", s2mt5.includes("上一步"));
// ①步2（CEX 凭据）
const s2cex = render(rows, { initialStep: 2, initialDraft: { venue: "okx", proto: "cex" } });
check("①步2 CEX 凭据件齐（api 三件）",
	s2cex.includes('placeholder="API Key"') && s2cex.includes('placeholder="API Secret"') &&
	s2cex.includes('placeholder="Passphrase"'));
check("①步2 CEX 无 MT5 login 件", !s2cex.includes('placeholder="登录号"'));
// ①步3 预览+提交
const s3 = render(rows, {
	initialStep: 3,
	initialDraft: { venue: "exness", proto: "mt5", login: "3001157", password: "pw-桩", server: "Exness-MT5Trial9", host: "h:443" },
});
check("①步3 预览 venue/login/server 渲出",
	s3.includes("Exness-MT5Trial9") && s3.includes("3001157"));
check("①步3 提交钮（验证并接入）", s3.includes("验证并接入"));
check("①步3 上一步钮在（返步）", s3.includes("上一步"));

// ③败因显示（initialErr 注面——步3 验败即回）
const errHtml = render(rows, { initialStep: 3, initialErr: "验连败——凭据未落库: MT5 invalid account" });
check("③败因即回显形", errHtml.includes("操作未成") && errHtml.includes("MT5 invalid account"));

// ④MT4 选项不存（列表显形例外——选项零渲染）
check("④MT4 选项不存（value=mt4 零渲染）", !listHtml.includes('value="mt4"'));
check("④MT4 option 文本零渲染", !listHtml.includes(">MT4</option>"));

// ⑥凭据不回显
check("⑥步3 密码掩码（raw 值零渲染）", s3.includes("••••••") && !s3.includes("pw-桩"));
const s3cex = render(rows, {
	initialStep: 3,
	initialDraft: { venue: "okx", proto: "cex", apiKey: "ak-桩", apiSecret: "as-桩", passphrase: "pp-桩" },
});
check("⑥步3 CEX api 三件掩码", s3cex.includes("••••••") &&
	!s3cex.includes("ak-桩") && !s3cex.includes("as-桩") && !s3cex.includes("pp-桩"));
check("⑥步2 password input type=password", /type="password"/.test(s2mt5));

// ⑧空态
const empty = render([]);
check("⑧空态=在册 broker 空", empty.includes("在册 broker 空"));
check("⑧空态向导仍在", empty.includes("①venue"));

// ⑤卡位断言（源码 fs——复盘页第三卡已撤/顶导航「接入」路由/Onboard 页挂卡）
const reviewSrc = readFileSync(join(reactRoot, "pages", "Review.tsx"), "utf8");
check("⑤复盘页不再挂 broker 接入卡", !reviewSrc.includes("BrokerOnboardCard"));
const mainSrc = readFileSync(join(reactRoot, "main.tsx"), "utf8");
check("⑤顶导航「接入」路由注册（/onboard）", mainSrc.includes('"/onboard"') && mainSrc.includes('"接入"'));
const onboardSrc = readFileSync(join(reactRoot, "pages", "Onboard.tsx"), "utf8");
check("⑤Onboard 页挂 broker 接入卡", onboardSrc.includes("BrokerOnboardCard"));

console.log(`—— ${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
