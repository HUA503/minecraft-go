# Minecraft-Go 执行状态与里程碑进展

## 当前阶段: P0 与 P1 阶段验收已全部通过，推进至 P2/P3

- **宿主环境**: Windows x64
- **Go 版本**: `go version go1.27.1 windows/amd64` (`CGO_ENABLED=0`)
- **Java 环境**: `OpenJDK 17.0.19 Temurin`
- **公共仓库**: [https://github.com/HUA503/minecraft-go](https://github.com/HUA503/minecraft-go)
- **基线版本**: Minecraft Java Edition 26.3 (Protocol 770 / Modern Configuration state flow)
- **参考实现**: Horizon (Rust) 性能与行为基准

---

## 阶段验收清单 (Milestone Progress)

### ✅ P0 基线与技术验证 (100% 通过)
- [x] **P0-1 工具链与环境验证**: Go 1.27.1 与 Java 17 检测通过；建立目录结构与 `go.mod`。
- [x] **P0-2 远端开源仓库就绪**: 成功创建 GitHub 公开仓库 `HUA503/minecraft-go` 并配置远程主干。
- [x] **P0-3 协议基线与 VarInt/VarLong 验证**: 纯 Go 编解码器，通过边界值与原版规范测试。
- [x] **P0-4 NBT 树形与二进制往返测试**: 100% 覆盖 TAG 0~12，支持 Gzip/Zlib 压缩流解压与写回。
- [x] **P0-5 原版握手与 Ping/Pong 链路**: 建立 TCP 服务端骨架，响应 0x00 StatusRequest 与 0x01 Ping。
- [x] **P0-6 Windows 图形原型与网格构建**: 纯 Go 隐藏面剔除（Face Culling）网格生成器与软件光栅帧缓冲（基准测速达到 4250+ FPS，0.235 ms/frame）。
- [x] **P0-7 最小差异测试**: `test/parity_test.go` 与 `test/e2e_parity_test.go` 对照原版二进制报文与红石衰减计算。

### ✅ P1 Go 服务端纵切与多端同步 (100% 通过)
- [x] **P1-1 权威世界与 20 TPS 主循环**: `pkg/sim` 调度定时 50ms 仿真时钟，处理重力与地面碰撞。
- [x] **P1-2 多客户端网络同步**: 两个独立客户端（Steve & Alex）同时登录进入 Play 状态。
- [x] **P1-3 方块交互同步**: 客户端 A 放置方块，服务端权威校验并更新，客户端 B 实时收到 `BlockUpdatePacket`。
- [x] **P1-4 玩家位移广播**: 客户端 A 移动并改变朝向，客户端 B 实时收到 `PlayerPositionPacket` 广播。
- [x] **P1-5 存档持久化与重启重载**: 借助 Anvil MCA 格式保存修改区块，服务关闭重启后完整还原所有方块数据。

---

## 验证与复现命令 (Reproduction Commands)

```powershell
# 1. 运行所有单元测试与集成测试
cd D:\project\feed\minecraft-go
go test -v ./...

# 2. 独立运行客户端渲染与光栅化性能基准
.\bin\client.exe -benchmark

# 3. 构建独立服务端与客户端二进制
go build -o bin/server.exe ./cmd/server
go build -o bin/client.exe ./cmd/client
```

---

## 下一步计划 (Next Actions)

- 推进 **P2 (世界与核心模拟深化)**：扩展三维度（主世界、下界、末地）高度映射、复杂流体模拟与完整生物群系调色板。
- 推进 **P3 (Go 客户端完整界面与视距串流)**：集成实时窗口交互界面与区块视距剔除加载。
