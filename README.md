# ControlMan

ControlMan 是一个简单的服务管理器，用于管理后台服务的生命周期。它包含一个守护进程和一个命令行工具，通过 Unix socket 进行通信。

## 功能特性

- 添加服务：自动启动并监控服务
- 停止服务：强制终止服务进程
- 查看服务日志
- 列出所有服务状态
- 删除服务
- 自动重启：服务崩溃时每5秒自动重启
- 日志管理：自动保存服务日志
- PID 管理：自动记录服务进程 ID

## 安装

### 从源码安装

1. 克隆仓库：
```bash
git clone https://github.com/yourusername/controlman.git
cd controlman
```

2. 编译并安装：
```bash
make install
```

### 卸载

```bash
make uninstall
```

## 使用方法

### 系统服务

ControlMan 守护进程会作为系统服务自动运行。你可以使用以下命令管理守护进程：

```bash
# 查看服务状态
sudo systemctl status controlman

# 启动服务
sudo systemctl start controlman

# 停止服务
sudo systemctl stop controlman

# 重启服务
sudo systemctl restart controlman

# 查看日志
sudo journalctl -u controlman
```

### 命令行工具

使用命令行工具管理服务：

```bash
# 添加新服务
controlman add myservice "python my_script.py"

# 停止服务
controlman stop myservice

# 查看服务日志
controlman logs myservice

# 列出所有服务
controlman list

# 删除服务
controlman delete myservice
```

## 文件结构

所有服务相关的文件都存储在 `~/.controlman` 目录下：

- `~/.controlman/controlman.sock`：守护进程的 Unix socket 文件
- `~/.controlman/<service_name>/`：每个服务的独立目录
  - `config.json`：服务配置文件
  - `service.log`：服务日志文件
  - `service.pid`：服务进程 ID 文件

系统日志文件：
- `/var/log/controlman.log`：守护进程的标准输出日志
- `/var/log/controlman.error.log`：守护进程的错误日志

## 注意事项

1. 服务命令应该是一个完整的命令行，包含所有必要的参数
2. 服务日志会持续追加到对应的日志文件中
3. 如果服务崩溃，守护进程会自动尝试重启
4. 使用 `stop` 命令会强制终止服务进程
5. 删除服务会同时删除所有相关的配置文件和日志
6. 守护进程会以当前用户身份运行，确保有适当的权限 