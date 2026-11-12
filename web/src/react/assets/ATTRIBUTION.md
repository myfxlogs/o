# floor v4 拆件包(2026-11-12 业主令:完全拆解原图,件件独立可调)

## 现役件

### room-shell.jpg —— 空厅底壳(整幅氛围底)
- 源链(原图): https://cdn.publish0x.com/prod/fs/images/45377939e7b372aadd8c5e219f21a1820ca3eb6f599bcd34915c47295059c11b.jpg
- 制程: 原图→sensnova edits 抹全部机器人+地球(plate-v1)→再抹环台+椅+壁前小台(plate-v2)→裁 2400×1300 q88。
  只留: 暗墙+左右两大墙屏阵+抛光地+青光环。权属未证,内部运营面板使用。

### console-ring.png —— 环台带屏(独立件)
- 从 plate-v1 抠(rembg/u2net,1480×660 alpha 45%)——椭圆全息控制台+嵌屏+台缘清碎块。
- 独立于底壳,可挪/换/删。

### eames-chair.png —— 空椅(立项/毛坯座)
- sengen 生件(1024×1024 白底,watermark:false)→rembg/u2net 抠(591×877)。
  米色圆背转椅——与 bot-seat 的椅同款,状态语义:座摆人未坐。

### bot-seat.png —— 坐机带椅(完工/在建座)
- 从原图抠件(rembg/u2net)——白瓷机+米色 Eames 椅一体;
  伸向桌面前臂融桌光被食,显示尺寸内读作手搭膝。

### bot-stand.png —— 立机(常驻机×6)
- stockcake.com AI 库件(futuristic-robot-heart-stockcake.jpg,业主钦定 11-12),
  rembg/u2net 抠底。

### bot-seat-back.png —— 坐机背影(前座朝台件)
- AI 图生图件(watermark:false)——以 bot-seat.png 为参考出背后视角,
  birefnet-general 抠底裁紧。与 bot-seat 同瓷质同款,前 3 座专用(face-b)。

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
