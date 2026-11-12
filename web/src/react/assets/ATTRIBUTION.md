# floor-holo.jpg

- 业主供图（2026-11-12 floor v3 钦定底图），源链:
  https://cdn.publish0x.com/prod/fs/images/45377939e7b372aadd8c5e219f21a1820ca3eb6f599bcd34915c47295059c11b.jpg
- AI 渲染全息舱图——作 floor 舞台氛围底,权属未证,内部运营面板使用;
  若外发/公开部署须换自产底图(B 径:按挂载位表重生成)。

# bot-seat.png / bot-stand.png
- 业主生件(模型:sensenova 系,2026-11-12 floor 件化钦定),
  原稿 /tmp/robot-command-quarantine-11812.png、robot-terminal.png;
  bot-seat 经抠边处理(白底洪泛+轮廓带+封闭囊清,胸徽/按钮保)。

# holo-globe.png
- AI 生件(sensenova-u1.5-lite,watermark:false,2026-11-12 floor 件化),
  提示词=全息地球线框(经纬网+轨道环+粒子+下射蓝辉光)。
  模型出**黑底**而非白底 → 走黑底辉光支路:alpha 直取亮度(Lmax/255),
  纯黑底全透、亮线全实,叠舱底即加色辉光。权属未证,内部面板使用。

# robot-chair.png
- AI 生件(同上口径,watermark:false),提示词=空座指挥椅(深皮+拉丝金属座
  底+青边光条,前四分之三视,全身可见)。白底洪泛抠边(阈值 240+中性饱和度门,
  只切与外沿连通的白底)。权属未证,内部面板使用。

# holo-dashboard.png
- AI 生件(同上口径,watermark:false),1024x1024。提示词=墙面空屏全息看板:
  拉丝金属八边形框+细青边辉光,屏内纯空深藏青无字无图,轻微透视倾斜。
  白底洪泛抠边(阈值 212,连地面软投影一起吃)+`--kill-floor-shadow`。

# bridge-hall.jpg
- AI 生件(sensenova-u1.5-lite,watermark:false,2026-11-12 floor 件化),1536x864 16:9。
  提示词=空指挥舱内景:圆形抛光暗金属地+中心大青环,空墙+蓝色氛围灯条,无屏无人无家具。
  整幅环境底,无透明需求,故存 JPG(q90)。权属未证,内部面板使用。

# conf-table.png
- AI 生件(同上口径),1024x1024。提示词=圆形会议桌(略俯视,深色金属面+内沿青色环
  光槽,桌面全空,中心镂空)。白底洪泛抠边(阈值 240)+`--kill-pocket=400000` 清中心
  镂空白袋+`--kill-floor-shadow` 清底投影+`--rim-loose` 吃浅灰描边。
  注:青环上的白色星点是源图高光非抠图残,叠舱底可接受。

# bot-stand.png(v2 白机)
- stockcake.com AI 库件(futuristic-robot-heart-stockcake.jpg,业主钦定
  11-12),rembg/u2net 抠底;替换 v1 sensenova 黑机(业主评:风格偏离
  floor-holo 原图白瓷机群)。
# bot-seat.png(v2 白机带椅)
- 从 floor-holo.jpg 原图抠件(业主评钦定原图风:米色 Eames 椅+白瓷机,
  rembg/u2net;伸向桌面前臂融桌光被食,显示尺寸内读作手搭膝)。

# bridge-hall.jpg / holo-dashboard.png
- 业主生件(11-12):空舰桥底图(件#6)+屏框(件#3,rembg 抠白底)。
  至此七件全位:bot-seat/bot-stand/holo-globe/robot-chair/conf-table/
  bridge-hall/holo-dashboard。
