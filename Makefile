.PHONY: all build install uninstall clean

# 变量定义
BINARY_NAME=controlman
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system
USER_HOME=/root

all: build

build:
	go build -o $(BINARY_NAME) cmd/controlman/main.go

install: build
	# 安装二进制文件
	sudo install -m 755 $(BINARY_NAME) $(INSTALL_DIR)
	# 创建配置目录
	sudo mkdir -p $(USER_HOME)/.controlman
	# 安装systemd服务文件
	sudo install -m 644 controlman.service $(SERVICE_DIR)
	# 重新加载systemd
	sudo systemctl daemon-reload
	# 启用并启动服务
	sudo systemctl enable controlman
	sudo systemctl start controlman

uninstall:
	# 停止并禁用服务
	sudo systemctl stop controlman
	sudo systemctl disable controlman
	# 删除systemd服务文件
	sudo rm -f $(SERVICE_DIR)/controlman.service
	# 删除二进制文件
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	# 删除配置目录
	sudo rm -rf $(USER_HOME)/.controlman
	# 重新加载systemd
	sudo systemctl daemon-reload

clean:
	rm -f $(BINARY_NAME) 