# ControlMan

ControlMan 是一个轻量级、高性能的后台进程管理工具，采用 Go 语言编写。它基于 Client-Daemon 架构，使用 Pebble 键值数据库进行元数据持久化，并提供健壮的进程监控和自动重启能力。

## 核心特性

- **高性能持久化**：使用 [Pebble](https://github.com/cockroachdb/pebble) 数据库存储服务元数据，替代传统的 JSON 文件存储，读写更高效且支持事务。
- **进程监控**：基于 `syscall` 信号检测进程存活状态，比文件 PID 锁更可靠。
- **状态管理**：精确维护服务生命周期状态（Running, Stopped, Failed, Restarting 等）。
- **自动重启**：内置监控机制，当服务非预期退出时自动尝试重启。
- **日志管理**：自动捕获并追加标准输出/错误到日志文件。
- **C/S 架构**：通过 Unix Domain Socket 通信，支持多客户端并发操作。

## 安装

### 前置要求

- Go 1.23+
- Linux / macOS (支持 Unix Socket 和 Signal 的环境)

### 编译安装

1. 克隆仓库：
   ```bash
   git clone https://github.com/tangthinker/controlman.git
   cd controlman
   ```

2. 下载依赖并编译安装：
   ```bash
   go mod tidy
   make install
   ```

   `make install` 会将二进制文件安装到 `/usr/local/bin`，并配置 systemd 服务（如果支持）。

## 快速开始

### 1. 启动守护进程

如果是通过 `make install` 安装，守护进程通常由 systemd 管理：

```bash
# 启动服务
sudo systemctl start controlman

# 查看状态
sudo systemctl status controlman
```

或者在当前终端手动启动（用于调试）：

```bash
controlman -daemon
```

### 2. 管理服务

使用 `controlman` 命令行工具与守护进程交互：

*   **添加并启动服务**：
    ```bash
    # 语法: controlman add <名称> "<命令>"
    controlman add myserver "python3 -m http.server 8080"
    ```

*   **查看服务列表**：
    ```bash
    controlman list
    # 输出包含 PID, 状态, 创建时间, 启动时间等信息
    ```

*   **查看日志**：
    ```bash
    controlman logs myserver
    ```

*   **停止服务**：
    ```bash
    controlman stop myserver
    ```

*   **启动服务**：
    ```bash
    controlman start myserver
    ```

*   **重启服务**：
    ```bash
    controlman restart myserver
    ``` 

*   **删除服务**：
    ```bash
    controlman delete myserver
    # 注意：这会同时删除服务的日志文件和数据库记录
    ```

## 数据存储

所有服务相关的数据默认存储在当前用户的 `~/.controlman` 目录下：

*   `data/`：Pebble 数据库目录，存储所有服务的元数据（PID、状态、命令等）。
*   `<service_name>/`：
    *   `service.log`：服务的运行日志文件。
*   `controlman.sock`：守护进程监听的 Unix Socket 文件。

## 开发与构建

```bash
# 整理依赖
go mod tidy

# 编译二进制文件
make build

# 运行测试
make test

# 清理构建文件
make clean
```

## 许可证

MIT License
