# T-0387 skill 自修环——矿工产 skill 补丁草案裁定入册

- 状态：完成（决策席裁决通过 merge 9e31bc93——
  T-0388 repro_hint 并入载荷 union 解；全链绿戳对口）
- 领取：devin-o-worker-t0387 @ 2026-11-06（常径机派施工席；
  提示词 docs/prompts/T-0387-O-Dev施工提示词.md）
- 优先级：P1——自我迭代加固族②；依 T-0385 矿工产出物
- 收口移交归：决策席

## 病灶

skills canon 只能加不能修——mt5-*/okx-* 能力账若与实况
漂移（RPC 改名/端点死/字段变）,错账会带偏施工。Memento
环=发现弱 skill→改进→写回；O 的等价缺口=矿工产出物没
有「skill 补丁」这个去向。

## 范围

1. T-0385 pending 行载荷扩 `target` 域（lesson|skill_patch）
   ——矿工聚类命中 skill 相关签名（坑条目 ID/能力账键/
   RPC 名）时产 skill_patch 型 pending：载荷=目标 skill
   +病灶锚+补丁草案（diff 或节替换文）；
2. 裁定席裁流扩 skill_patch 型：verdict=approved→补丁
   入 skills/<name>/ 落盘（裁定席写径扩 skills/** 白名单
   位——canon 写域按名册钉,机器不直改）；
3. 防漂移闸：补丁入册须附「漂移证据锚」（实测对拍行/
   事件锚——无证据锚的补丁草案拒入,防 LLM 臆改 canon）；
4. kbgen 索引随件重建；
5. 防灌：skill_patch 型日额 ≤2（比 lesson 更严——canon
   是高权文本面）。

## 验收锚

① 桩测：skill 签名聚类→skill_patch pending 落（目标/
   锚/草案齐）→verdict→skills 落盘 diff 断言；
② 无锚草案拒入断言；③ 日额闸；
④ 既有 lesson 型流回归不破（T-0385 域内）；
⑤ check_all 绿戳对口；镜像面件并入发窗。

## 红线

- canon 写域仍裁定席独占——矿工/LLM 不直改 skills/**；
- 补丁必须证据锚随行（臆改拒）；
- 不改 T-0385 pending 表构（载荷扩非表变）；
- 不动真单径。

## 结果回填

- **实现摘要**：
  - `internal/portfolio/hunger/miner_skill.go`——skill_patch 产线：
    skillSigRoster 签名册（keyset 硬钉同 capcheck 例——票面钦三类
    token：坑条目 id=条目 frontmatter skills: 标签反查/能力账键=
    skill 目录名+字段键/RPC 名=能力账表列特异词；泛词 okx/mt5/venue
    不入册防误路由，新签名不入册降级 lesson 静默兜底宁漏不错投）；
    matchSkillSig 簇行 note/esc_class/esc_scope/ticket 四面词界
    匹配（横线当界连缀中、字母粘不中）→命中数最多 skill=目标
    （平手字典序定排）；minerPatchDailyCap=2 越额降级 lesson。
  - `internal/portfolio/hunger/miner.go`——minerPayload 载荷扩
    target 域（lesson|skill_patch——红线守：表构不动载荷扩非表变）
    +skill_patch 三件（target_skill/lesion_anchor=token@面 列/
    patch_draft=节替换草案含目标+病灶+证据锚+定稿规约）；minerPatch
    CapLeft 日额闸（当日 skill_patch 行<2 才产 patch 型）；留账行
    携 target=skill_patch skill=<名> 标。
  - `tools/skillpatch`——裁定席 canon 落件机械闸（skillgen 同例
    判后命令行触发；FIN_PG_DSN 读 pending 行）：机械五闸=pending
    态/target=skill_patch 型对拍/target_skill 双钉（pending 钦目标
    +skills/<name>/SKILL.md 实存名册）/evidence_ids 非空（漂移
    证据锚——无锚拒防 LLM 臆改，票面红线机械位）/载荷三件齐；
    append=新「漂移勘正」节+replace=既有节正文整换（节缺席拒防
    虚构结构）；落盘节自携证据行（ops_event#ids+病灶锚+pending#id
    ——补丁必须证据锚随行机内化）；verdict 收尾 UPDATE 照旧归裁流
    step4 两拍分权。
  - `skills/裁定席/SKILL.md` 裁域扩 skill_patch 型：读面补 target
    域+钦三件读法；eng 词集增 `skill_patch:<name>`（lesson 行裁此
    词=越域）；落面 3b——裁前核锚（空集=dismiss 锚缺）+名册核
    （非实存目录=升不裁）+按 patch_draft+证据行定稿（禁超锚外引）
    +skillpatch 落件+kbgen skills 随件重建+commit；升不裁补
    skill_patch 同则（误路由/件缺席/锚行核不出漂移=stalled 留
    pending）。
  - `docs/数据流.md` 矿工拍段补 T-0387 支线链（链路变更同 commit）。
  - 无迁移——载荷扩非表变，迁移号水位查 fin_0121 不动（next=0122
    未取）；verdict_tick.sh 水印闸零改（target 无关计 pending 行数）。
- **测试证据**：
  - 锚①：`go test ./internal/portfolio/hunger -run MinerSkill` ——
    TestMinerSkillSigProducesPatch（skill 签名簇→skill_patch
    pending 落：target/target_skill/lesion_anchor=token@esc_scope
    面/patch_draft/evidence_ids 钦件断言+留账行 target 标）；
    TestMatchSkillSigScoring（多 token 目标择+定排）。
  - 锚①后半 verdict→skills 落盘 diff：`go test ./tools/skillpatch`
    ——TestApplyAppendLandsDiff（pending→apply→SKILL.md 新节+
    证据锚行+补丁文 diff 断言+节序不扰）；TestApplyReplaceSection
    （既有节正文整换+邻节不动）。
  - 锚②无锚拒入：TestGateMatrix 八案——闸①非 pending/闸② lesson
    型/闸③目标错位+名册缺席/闸④ evidence_ids 空集拒/闸⑤缺件/
    replace 节缺席/mode 非法——逐案断言拒因+文件零触。
  - 锚③日额闸：TestMinerSkillPatchCapTwo——三 skill 簇同窗=2
    patch+1 lesson 降级（字典序定排可复现断言）。
  - 锚④回归：miner 原 13 测全绿+TestMinerNonSkillClusterLesson
    Target（非 skill 簇 target=lesson+三件空）。
  - 名册机钉：TestSkillRosterPinned——册键⊆skills/ 实存目录+
    坑引⊆条目 frontmatter skills: 标签双向对拍全绿；词界
    TestHasTokenBoundaries 9 案。
  - `go build ./...`+`go vet`+`gofmt` 净；check_all 绿戳对口收口 SHA。
- **偏离说明**：
  - bootstrap 降级径起步（rsync gen/dist 快照——本票无 proto 面动，
    快照与支面同宗无漂）。
  - skill_patch 型日额越额簇降级 lesson 产（票面只钦「≤2」闸——
    降级择=信号不丢：事件仍沉淀 lesson 案，patch 稿翌日窗再挖；
    异择「直接不产」会丢信号）。
  - 名册=keyset 硬钉非机生件（capcheck 例——机生册须新 kbgen
    子令+embed 链路，超票面最小面；册红/新 skill 盲区由
    TestSkillRosterPinned 机钉兜住）。
  - 裁定席 skill_patch 全链实证=devin 无头 run 件（SKILL runbook
    +机械闸已落；fin_eng_lesson 未部署窗同 T-0385 边界如实记）——
    落件侧五闸+双模桩测已机器断言。
  - 镜像面件（Go 码+工具）——进阻塞集，发窗归决策席判据。
