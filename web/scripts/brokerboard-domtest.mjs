// brokerboard-domtest.mjs —— T-0407 broker 接入卡 DOM 断言（范式同
// quoteboard-domtest.mjs：renderToStaticMarkup+esbuild 捆扎，零新依赖）。
// 断言锚：①在册列表渲（venue/协议/login/接入态人话/在跑策略/末拍注记）；
// ②「添加 broker」表单字段齐（venue/proto 下拉 MT5+MT4/login/password/
// server/note——票面字段钉）；③凭据字段不回显——password input type=
// password 且零 value 渲染（值断言=凭据面钉）；④提交径=onSubmit 回调闸
// 门面（有回调表单在/无回调表单隐——同 QuoteBoard 回调闸面范式）；
// ⑤MT4 族=「MT4 链未接」如实显形+败径=接入败 chip+无态=凭据在册候发现；
// ⑥空态=「在册 broker 空」不假行。
// 运行：node scripts/brokerboard-domtest.mjs（web/ 下）。
import { build } from "esbuild";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { pathToFileURL, fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const comp = join(webRoot, "src", "react", "components", "brokerboard.tsx").replaceAll("\\", "/");

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
	beatNote: opts.beatNote ?? "",
	lastBeatAt: opts.beatAgeS !== undefined ? { seconds: BigInt(nowS - opts.beatAgeS) } : undefined,
	beatFailed: opts.beatFailed ?? false,
	strategies: BigInt(opts.strategies ?? 0),
});
const render = (rows, extra = {}) =>
	renderToStaticMarkup(h(BrokerOnboardView, { rows, onSubmit: () => {}, ...extra }));

const rows = [
	row("exness", "mt5", { login: "3001157", state: "probed", beatNote: "beat=探测 Connect ok", beatAgeS: 60, strategies: 2 }),
	row("fxmt4", "mt4", { login: "777" }),
	row("badbro", "mt5", { login: "9", state: "registered", beatNote: "败径 fail-closed：网关不可达", beatFailed: true, beatAgeS: 30 }),
];
const html = render(rows);

// ①在册列表渲
check("①venue/login 渲出", html.includes("exness") && html.includes("3001157") && html.includes("fxmt4"));
check("①接入态人话（probed→已探测）", html.includes("已探测"));
check("①在跑策略数", html.includes(">2</td>"));
check("①末拍注记渲出", html.includes("Connect ok"));
// ②表单字段齐
check("②venue 字段", html.includes('name="venue"'));
check("②proto 下拉 MT5+MT4", html.includes('name="proto"') && html.includes("MT5") && html.includes("MT4"));
check("②login 字段", html.includes('name="login"'));
check("②password 字段", html.includes('name="password"'));
check("②server 字段", html.includes('name="server"'));
check("②note 字段", html.includes('name="note"'));
check("②提交钮", html.includes("提交凭据") && html.includes("添加 broker"));
// ③凭据字段不回显
check("③password input type=password", html.includes('type="password"'));
check("③password input 零 value 渲", !/name="password"[^>]*value=/.test(html));
check("③凭据字段不落可见值", !html.includes("pw-桩"));
// ④提交径=onSubmit 回调闸门面
const noCb = renderToStaticMarkup(h(BrokerOnboardView, { rows }));
check("④无 onSubmit=表单隐", !noCb.includes('name="password"'));
// ⑤如实显形
check("⑤MT4 族=链未接显形", html.includes("MT4 链未接"));
check("⑤败径=接入败 chip", html.includes("接入败"));
// ⑥空态
const empty = render([]);
check("⑥空态=在册 broker 空", empty.includes("在册 broker 空"));
check("⑥空态表单仍在", empty.includes('name="password"'));

console.log(`—— ${n - fail}/${n} PASS`);
process.exit(fail ? 1 : 0);
