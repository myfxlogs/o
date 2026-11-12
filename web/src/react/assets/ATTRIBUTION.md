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

### holo-globe.png —— 全息地球(脑)
- AI 生件(watermark:false)——线框地球+轨道环+底部投影光柱。
  黑底辉光支路:alpha 直取亮度(Lmax/255),叠舱底即加色辉光。

### holo-dashboard.png —— 屏框(四活屏金属框)
- AI 生件(watermark:false),1024×1024——拉丝金属框+细青边辉光,屏内纯空。
  白底洪泛抠边(阈值 212)+`--kill-floor-shadow`。

## 退役件(floor v3→v4 拆件替换)
floor-holo.jpg / bridge-hall.jpg / conf-table.png / robot-chair.png
——整幅 AI 底被 room-shell(原图壳)取代,生成桌/椅被原图抠件取代。
