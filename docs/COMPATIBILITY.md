# Minecraft Java Edition 26.3 兼容性矩阵与基准

本文件记录 Go 语言重写版（minecraft-go）与 **Minecraft Java Edition 26.3** 以及 **Horizon (Rust)** 的功能对齐与兼容状态。

| 模块 / 特性 | 原版 Java 26.3 | Horizon (Rust) | Go 实现状态 | 验证方法与备注 |
|---|---|---|---|---|
| **网络协议 - 握手与状态机** | 支持 Handshake, Status, Login, Config, Play | 完全对齐 | ✅ 已通过 | 单元测试 + 自动化端到端网络模拟 |
| **网络协议 - VarInt / VarLong** | 7-bit LEB128 变长整数编码 | 100% 对齐 | ✅ 已通过 | 针对边界值（0, 1, 127, 128, 255, 2147483647 等）自动化 Fuzz 对照测试 |
| **网络协议 - SLP 状态查询与 Ping** | 返回 JSON 描述符与延迟 Pong (0x01) | 完全对齐 | ✅ 已通过 | 客户端发起 StatusRequest (0x00) 并比对 PingPong 往返 |
| **网络协议 - 登录与配置阶段** | LoginSuccess, Config Ack, KnownPacks, FinishConfig | 完全对齐 | ✅ 已通过 | 模拟登录链路，完成客户端进入 Play 模式 |
| **网络协议 - 游戏包通信 (Play)** | ChunkData, PlayerPosition, BlockUpdate, KeepAlive, Chat | 完全对齐 | ✅ 已通过 | 双向收发解析，支持坐标同步与方块变更广播 |
| **NBT 格式编解码** | TAG 0~12 全类型，包含压缩 (Gzip/Zlib) | 完全对齐 | ✅ 已通过 | 针对 NBT 二进制规范的往返无损序列化测试 |
| **区块结构与世界坐标** | Y: -64 ~ 320, 24 个 16x16x16 Section | 完全对齐 | ✅ 已通过 | 24 Section 结构体，支持局部/全局坐标映射 |
| **调色板容器 (PalettedContainer)** | SingleValue / Indirect / Direct 位打包 | 完全对齐 | ✅ 已通过 | 支持位紧凑打包算法与动态扩容调色板 |
| **物理与碰撞 (AABB & Voxel)** | 玩家 AABB (0.6x1.8x0.6), 重力, 阶梯步进 | 基础对齐 | ✅ 已通过 | 物理循环与射线碰撞检测（Raycast & Box Collision） |
| **方块交互与权威同步** | 破坏与放置, 状态变更广播 | 完全对齐 | ✅ 已通过 | 多客户端观察视野下，玩家 A 放置方块，玩家 B 即时收到 BlockUpdate |
| **红石基础引擎** | 红石线 (15~0 衰减), 中继器延时, 比较器, 火把反相 | 核心对齐 | ✅ 已通过 | 逐 Tick 延迟队列调度与信号传播算法验证 |
| **客户端渲染与网格** | 视锥剔除, 面剔除 (Face Culling), 贪婪网格 | 现代管线 | ✅ 已通过 | 纯 Go 软件光栅化与网格构建器（无暴露面剔除算法） |
| **持久化存储 (Anvil MCA)** | 1024 区块头 + 压缩数据块 | 完全对齐 | ✅ 已通过 | 支持读取与写回标准 MCA 格式区块数据 |
