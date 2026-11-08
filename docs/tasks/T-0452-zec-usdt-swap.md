# T-0452 探针升件自动起草：观测盲@ZEC-USDT-SWAP

- 状态：已批（DM 裁 2026-11-08——范围节钦定回填，签名面病灶成立候选过基线）
- 领取：常径待派（批准后 DM 常径机派）
- 优先级：P2（起草默认档——DM 裁时可调）
- 收口移交归：决策席
- 立项：起草拍自动件（T-0393——探针升件签名 `观测盲:ZEC-USDT-SWAP`，草案锚 fin_ticket_draft#18）

## 病灶（升件行证据引）

病灶签名：`观测盲:ZEC-USDT-SWAP`（kind=探针类 `观测盲`，key=病灶键 `ZEC-USDT-SWAP`）

末行升件原文：
> 腿行情覆盖失明 ZEC-USDT-SWAP：live/paper 策略腿所需行情键近 10min md_ticks 零新鲜行——归一链订阅面/镜像谓词/写入闸断点待查（探针只报不修）

证据引（ops_event 行 id）：#3451

复核查询式：
```sql
SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='观测盲' AND esc_scope='ZEC-USDT-SWAP' ORDER BY id DESC LIMIT 20
```

## 疑似根域

行情订阅覆盖面（okxws activeSet×md_ticks——T-0392 腿覆盖探针域）

## 范围（DM 钦定回填）

1. 文件路径钦定：internal/data/okxws/（activeSet×md_ticks 归一链——ZEC-USDT-SWAP 腿键覆盖缺）
   internal/portfolio/hunger/（腿覆盖探针域）；
2. 病灶根因定位与根修（签名面病灶本体——不超签名外衍）；
3. 桩测钉死断言（病灶存在断言修前红+同断言修后绿——T-0388 修复类先红桩律）；
4. 生产谱面零误报审计。

## 验收锚（骨架——DM 裁时补足）

① 桩测断言全绿（红/绿双点证据落结果回填）；
② 探针同窗不再报同签名病灶（修复后探针行随停）；
③ check_all 绿戳对口；并入发窗批。

## 红线（继承模板——不改不删）

- 只修本票签名病灶——不超单施工、不碰范围外文件；
- 不部署、不并 master——收口回决策席常径裁；
- 探针只报不修——本票修病灶本体，禁顺手改探针阈/签名面/类册；
- 钱径/凭据域/真单径语义禁碰。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
