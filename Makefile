.PHONY: all build install uninstall clean

# 变量定义
BINARY_NAME=controlman
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system
USER_HOME=$(shell echo $$HOME)
LOG_DIR=/var/log

all: build

build:
	go build -o $(BINARY_NAME) cmd/controlman/main.go

install: build
	# 安装二进制文件
	sudo install -m 755 $(BINARY_NAME) $(INSTALL_DIR)
	# 创建配置目录
	mkdir -p $(USER_HOME)/.controlman
	# 创建日志目录并设置权限
	sudo mkdir -p $(LOG_DIR)
	sudo touch $(LOG_DIR)/controlman.log $(LOG_DIR)/controlman.error.log
	sudo chown $(USER):$(USER) $(LOG_DIR)/controlman.log $(LOG_DIR)/controlman.error.log
	sudo chmod 644 $(LOG_DIR)/controlman.log $(LOG_DIR)/controlman.error.log
	# 安装systemd服务文件
	sudo install -m 644 controlman.service $(SERVICE_DIR)
	# 重新加载systemd
	sudo systemctl daemon-reload
	# 启用并启动服务
	sudo systemctl enable controlman@$(USER)
	sudo systemctl start controlman@$(USER)

uninstall:
	# 停止并禁用服务
	sudo systemctl stop controlman@$(USER)
	sudo systemctl disable controlman@$(USER)
	# 删除systemd服务文件
	sudo rm -f $(SERVICE_DIR)/controlman.service
	# 删除二进制文件
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	# 删除日志文件
	sudo rm -f $(LOG_DIR)/controlman.log $(LOG_DIR)/controlman.error.log
	# 重新加载systemd
	sudo systemctl daemon-reload

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(USER_HOME)/.controlman 