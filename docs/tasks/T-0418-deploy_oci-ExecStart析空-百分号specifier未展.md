# T-0418 deploy_oci ExecStart 解析空——%h 等 systemd specifier 未展开

- 状态：已批（决策席直批 2026-11-07——T-0416 部署实证病灶）
- 优先级：P2——deploy_oci.sh 预检死径（环约闸名存实亡绕不过）
- 收口移交归：决策席

## 病灶（实证 2026-11-07）

`scripts/deploy_oci.sh` L91-92 ExecStart 解析：
`grep '^ExecStart=' | sed | tr ' ' '\n' | sed 's/^-*//' | grep '^/'`
——要求字面 `/` 起首路径；OICI o-collecto 单元实书
`ExecStart=%h/o/collector`（%h=systemd specifier 展开为 unit
属主 $HOME）→ 解析空 → `die "ExecStart 解析空"` 预检即拒，
deploy_oci.sh 对 %h 形单元**结构性不可用**（T-0416 部署实证——
决策席手补同序步骤绕径：环约对账/备份/换件/restart/健康闸
逐项手走，绕行在案）。

## 钦定语义

- specifier 展位=部署径责任：**远端语境展开**——%h→该 ssh
  用户 $HOME（`ssh host 'echo $HOME'` 或 unit File 属主家目），
  %u→属主名，%s→unit 名；未识别 specifier=**保守 die 报原值**
  不猜（与脚本既有 fail-closed 风同调）；
- 解析后须仍是绝对路径（现语义）——EXECPATH 供 scp 目标/
  备份 install/回滚径，非绝对路径即死；
- 顺修（可选并入）：ExecStart 多命令/包装形（bash -c/环境
  前缀）=同保守 die 径不猜。

## 范围

1. `deploy_oci.sh` ExecStart 解析段加 specifier 展步；
2. 桩测：%h/%u 两形 + 未知 specifier 保守拒 + 字面路径回归；
3. 实盒干跑证据：`deploy_oci.sh collecto DRY_RUN=1` 过预检段。

## 验收

- 桩测谱绿；o-collecto %h 单元解析出 `/home/ubuntu/o/collector`；
- 未知 specifier fail-closed die 显报不静默。

## 备注

- 同票不背 T-0416 已绕径的换件责任——deploy_oci 修通后
  下窗起回正径；绕行披露在 ops_event/relay 在案。
