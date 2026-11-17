# Minecraft-Go 架构与技术决策记录 (ADR)

## ADR-001: 核心引擎与协议采用 Pure Go，零外部 CGO 依赖

- **状态**: 已接受
- **背景**: 目标是在 Windows x64 与 Linux x64 上实现零障碍跨平台构建与分发。宿主开发机在 `CGO_ENABLED=0` 环境下，若强依赖 CGO（如 raylib C 绑定）会导致纯 Go 环境下无法直接 `go build` 或 `go test`。
- **决策**:
  1. 协议解析（`pkg/protocol`）、NBT 编解码（`pkg/nbt`）、世界存储（`pkg/world`）、物理与红石模拟（`pkg/sim`, `pkg/redstone`）严格采用 100% Pure Go 实现。
  2. 客户端图形采用分层抽象驱动：提供 `Renderer` 接口，包含高性能纯 Go 渲染管线（软件光栅化与像素帧缓冲，用于自动化无头测试与轻量级图形）与可扩展的 GPU 驱动后端。
- **结果**: 任意平台执行 `go test ./...` 与 `go build ./...` 均无需安装 MinGW/C 编译器即可一键通过。

---

## ADR-002: 单体仓库分层架构与单机/多人同构

- **状态**: 已接受
- **背景**: Minecraft 原版在 1.3 之后将单人模式重构为本地嵌入式服务端（Integrated Server）与客户端的内部网络循环，确保多人与单人的物理和逻辑完全一致。
- **决策**:
  1. 采用单个 Go 仓库，提供 `cmd/server`（专用独立服务端）与 `cmd/client`（支持单机嵌入与远程连接的客户端）。
  2. 模拟权威（Authoritative Simulation）全部收口于 `pkg/sim` 与 `pkg/world`，客户端通过协议层或内存管道与服务端通信，消除单机与联机的行为分歧。

---

## ADR-003: 权威世界 20 TPS 固定 Tick 调度与并发模型

- **状态**: 已接受
- **背景**: Minecraft Java 核心模拟严格以 50ms (20 TPS) 为一个生命周期周期（Tick）。网络数据包、区块异步生成和网格构建如果在多线程随意修改世界状态，会产生数据竞争与状态不一致。
- **决策**:
  1. 模拟循环在主 goroutine 中以有界时间步长（50ms）串行推进世界状态更新（实体移动、红石更新、流体扩散、方块更新）。
  2. 网络读写、区块磁盘 I/O、区块数据压缩与网格构建使用有界工作池（Worker Pool）并发处理，通过有界 channel 向主循环提交变更。
  3. 区块操作携带版本号（Generation/Mutation Version），丢弃过期更新。

---

## ADR-004: 现代 Minecraft Java (26.3 / 1.21+) 协议与区块格式兼容

- **状态**: 已接受
- **背景**: Minecraft 现代版本采用 -64 至 320 的世界高度范围（共 24 个 16x16x16 Section），网络握手在 1.20.2+ 引入了独立的 Configuration 状态阶段。
- **决策**:
  1. 协议状态机完整支持：`Handshake (0)` -> `Status (1)` / `Login (2)` -> `Configuration (3)` -> `Play (4)`。
  2. 区块存储采用 Anvil `.mca` 区域文件格式与 24 个 Section 的 PalettedContainer（位打包方块状态与生物群系调色板）。
  3. 严格遵循 VarInt、VarLong、NBT 结构规范，提供双向往返序列化与反序列化。
