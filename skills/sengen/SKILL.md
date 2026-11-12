---
name: sengen
description: sensenova 文生图/图生图 CLI——UI 资产生成件(floor/其他视觉件)。
  gen=文生图,edit=参考图编辑;key 在 ~/.config/sensenova/key(不入仓)。
  需要生成/改图片资产时调用。
---

# sengen —— sensenova 生图工具(devin 专用,不入 O 业务码)

## 用法

```bash
# 文生图
/home/mluser/tools/sengen gen "提示词" [尺寸 auto|WxH] [输出文件]
# 图生图(参考图编辑)
/home/mluser/tools/sengen edit "编辑指令" <参考图路径> [尺寸 auto|WxH] [输出文件]
```

仓内副本 `.devin/skills/sengen/sengen` 同源——`/home/mluser/tools/sengen` 是首选执行径。

## 参数(实测在案 2026-11-12)

- model=`sensenova-u1.5-fast`,response_format=`b64_json`(免 24h 临时链二跳)
- size: `auto` 或 `WxH`(32 倍数,512-4096;常用 1024x1024 / 2720x1536)
- 成本:~5700 tokens/张;`watermark=false` 公测期免费无水印
- 超时:生成 ~30-60s,脚本内 120s
- edit:参考图本地文件→dataURL 直传,至多 5 张(脚本现用 1 张)

## key

`/home/mluser/.config/sensenova/key`(0600,入 env `SENSENOVA_API_KEY` 亦可)。
**禁入仓/禁入 commit**——工具属 devin 私有,与 O 业务无关(业主令 11-12)。

## 抠底管线(rembg——本地件)

- venv `/home/mluser/.openharness-venv`
- **`U2NET_HOME` 必设 `/home/mluser/.rembg`**——名字虽叫 u2net 实为全模型根
  (历史遗留变量名);不设则找 `~/.u2net` 空目→联网重下卡死(实证 10min 零产出)
- 模型固化位 `~/.rembg/models/<名>/`:
  - `u2net`(170MB)——泛用粗件
  - `birefnet-general`(~973MB,aria2 -x16 拉 ~12min)——边缘锐一档,
    但**全显著物打包**:裁片混多物时连椅带台一起抠(实证);
    对法=裁框先精裁到单物再喂它;多物粗框反用 u2net(偏主件)
  - `u2net_human_seg`——人像件(椅子会被吃,勿用于家具)
- 用法:`new_session('birefnet-general'|'u2net')` 一次建 session 复用;
  `remove(im, session=s)`
- rembg CLI 有 watchdog 依赖缺——**走 python API 别走 CLI**
- 矩形件(墙屏板/窗)不用 rembg——直接裁+羽化即 sprite
- 发丝级软 alpha 顶配:U²-Net 粗 mask→造 trimap→ViTMatte(两段式,未装)

## 坑

- API key 只放本机 config——工具脚本本身零凭据,可进仓
- 透明底 PNG:生成白底居多→rembg(u2net)抠底管线在 venv `/home/mluser/.openharness-venv`
- 出件统一先落 /tmp 验视再入 `web/src/react/assets/`——部署树 `/opt/o` 禁手放(打脏树闸)
