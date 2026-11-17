# Minecraft-Go

[![Go Version](https://img.shields.io/badge/go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Parity Target](https://img.shields.io/badge/Minecraft-Java%2026.3%20Parity-brightgreen)](https://www.minecraft.net/en-us/article/minecraft-java-edition-26-3)
[![Reference: Horizon](https://img.shields.io/badge/Reference-Horizon%20(Rust)-orange)](https://playhorizon.dev)

A clean-room, high-performance implementation of **Minecraft: Java Edition (Release 26.3)** written entirely in **Go**, featuring both an authoritative dedicated server (`cmd/server`) and client (`cmd/client`) targeting gameplay and protocol parity with Minecraft Java Edition and the Horizon engine.

---

## 🌟 核心特性 (Key Features)

- **Pure Go & 零 CGO 编译约束**: 可以在任意标准 Go 环境（Windows / Linux / macOS）直接 `go build` 和 `go test`。
- **Java 26.3 现代协议支持**:
  - 完整协议状态机：`Handshake` -> `Status` / `Login` -> `Configuration` -> `Play`
  - 7-bit LEB128 VarInt / VarLong 高性能编解码
  - 动态包解析分发与 Keep-Alive 保活机制
- **完整 NBT (Named Binary Tag) 规范**:
  - 支持全套 12 种 TAG 类型（TAG_Byte, TAG_Short, TAG_Int, TAG_Long, TAG_Float, TAG_Double, TAG_ByteArray, TAG_String, TAG_List, TAG_Compound, TAG_IntArray, TAG_LongArray）
  - 支持标准 Gzip / Zlib 压缩往返流读写
- **现代区块与世界存储 (Anvil MCA & 24 Sections)**:
  - Y 轴坐标支持 `-64` 到 `320`（现代 384 格世界高度）
  - 单区段 16×16×16 的调色板位紧凑容器 (`PalettedContainer`)
  - 兼容 Anvil `.mca` 区域头检索与压缩块读写
- **权威 20 TPS 模拟主循环 (`pkg/sim`)**:
  - 定时 50ms 固定 Tick 调度
  - 实体 AABB 空间包围盒与体素光线投射碰撞检测
  - 方块破坏/放置的权威校验与多玩家观察视野广播
- **红石逻辑模拟器 (`pkg/redstone`)**:
  - 15 级红石能量曼哈顿衰减算法
  - 中继器延迟调度、比较器比较/减法模式、火把反相
- **轻量级体素渲染与面剔除网格生成器 (`pkg/render`)**:
  - 隐藏面剔除（Face Culling）算法，剔除被实心相邻方块遮挡的面
  - 软件光栅化测试管线，支持自动化图形验证与无头 CI 运行

---

## 🚀 快速开始 (Quick Start)

### 编译服务端与客户端

```bash
# 编译专用独立服务端
go build -o bin/server.exe ./cmd/server

# 编译客户端
go build -o bin/client.exe ./cmd/client
```

### 运行单元测试与原版差异化回归测试

```bash
go test -v ./...
```

---

## 📖 架构设计与状态

- [架构决策记录 (ADR)](docs/DECISIONS.md)
- [版本兼容性矩阵 (Compatibility)](docs/COMPATIBILITY.md)
- [实施状态报告 (Status)](docs/STATUS.md)

---

## 📄 开源许可

本项目采用 [MIT License](LICENSE)。Minecraft 是 Mojang Synergies AB 的注册商标。本项目为独立开源干净室重新实现。
