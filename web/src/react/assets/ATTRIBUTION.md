# floor v4 拆件包(2026-11-12 业主令:完全拆解原图,件件独立可调)

## 现役件

### room-shell.jpg —— 空厅底壳(整幅氛围底)
- 源链(原图): https://cdn.publish0x.com/prod/fs/images/45377939e7b372aadd8c5e219f21a1820ca3eb6f599bcd34915c47295059c11b.jpg
- 制程(v3·原图直抹): 原图→sengen edit 一遍抹「全部机器人+椅凳+线框球」,台/光斗/墙屏/工位/地原像素保留→1792×1006 q87 真 JPEG。权属未证,内部运营面板使用。

### rim-front.png —— 台缘前唇遮罩(纵深件)
- 从 room-shell v3 裁台缘环带(椭圆 rx36.5/ry16 心(50,59) 环带 0.72-1.05,前唇带)→alpha。
- 盖在座机下半身造「围台坐」纵深——机器埋台缘凹槽非浮台面。

### console-ring.png —— 环台带屏(独立件)
- 从 plate-v1 抠(rembg/u2net,1480×660 alpha 45%)——椭圆全息控制台+嵌屏+台缘清碎块。
- 独立于底壳,可挪/换/删。

### eames-chair.png —— 空椅(立项/毛坯座)
- sengen 生件(1024×1024 白底,watermark:false)→rembg/u2net 抠(591×877)。
  米色圆背转椅——与 bot-seat 的椅同款,状态语义:座摆人未坐。

### bot-seat.png —— 坐机低斗凳(完工/在建座)
- v2 AI 生件(watermark:false)——白瓷机坐奶油斗壳低凳(原图凳同款:
  斗壳座+短铬杆+平圆盘),侧视朝右,birefnet 抠底(599×867)。

### bot-stand.png —— 立机(常驻机×6)
- stockcake.com AI 库件(futuristic-robot-heart-stockcake.jpg,业主钦定 11-12),
  rembg/u2net 抠底。

### bot-seat-back.png —— 坐机低斗凳背影(前座朝台件)
- v2 AI 图生图件(watermark:false)——以 bot-seat v2 为参考出背后视角,
  同斗壳凳+铬杆圆盘,birefnet 抠底(1010×1783)。前 3 座专用(face-b)。

### holo-globe.png —— 全息地球(脑)
- AI 生件(watermark:false)——线框球+同心环核+浮粒子(原图核结构)。
  黑底辉光支路:alpha 直取亮度(Lmax/255),叠舱底即加色辉光。

### dais-ring.png —— 圆环巨台(环台件,原图 dais)
- AI 生件(watermark:false)——藏蓝环面+同心光环+中心孔,斜视椭圆,
  birefnet 抠底。原图台体:扁环平台非控制台;CSS scaleY(.58) 矫正视角差。

### holo-bowl.png —— 光斗(台↔球连接件,原图核心件)
- AI 生件(watermark:false)——线框半球碗,亮度=alpha 辉光,台孔升起托球底。
  原图连接体非光柱——半球光斗。

### scoop-chair.png —— 矮斗凳(空椅件,原图椅款)
- AI 生件(watermark:false)——米斗壳+铬杆+平圆盘,birefnet 抠底。
  原图椅=矮斗凳非高背扶手椅。

### server-rack.png —— 物理主机柜(主机件,真硬件非人形)
- AI 生件(watermark:false)——藏蓝机柜+青 LED,birefnet 抠底。
  三形分家:器=坐机/服=工位机/机=柜;载 fin_host_metrics cpu%,stale 灰红。

### wall-station.png —— 墙沿坐台工位机(常驻机×8)
- AI 生件(watermark:false)——白瓷机坐窄白台,台上浮青屏(原图墙沿工位)。
  birefnet-general 抠底裁紧。

## 退役件(floor v3→v4 拆件替换)
floor-holo.jpg / bridge-hall.jpg / conf-table.png / robot-chair.png
——整幅 AI 底被 room-shell(原图壳)取代,生成桌/椅被原图抠件取代。
holo-dashboard.png —— 屏框金属件,业主令四屏入墙后退役(屏芯改青玻格区)。
