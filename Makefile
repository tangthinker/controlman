.PHONY: all build install uninstall clean

# 变量定义
BINARY_NAME=controlman
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system

all: build

build:
	go build -o $(BINARY_NAME) cmd/controlman/main.go

install: build
	# 复制静态文件到/root/.controlman/static
	sudo mkdir -p /root/.controlman/static
	sudo cp -r static/* /root/.controlman/static
	# 安装二进制文件
	sudo install -m 755 $(BINARY_NAME) $(INSTALL_DIR)
	# 安装systemd服务文件
	sudo install -m 644 controlman.service $(SERVICE_DIR)
	# 重新加载systemd
	sudo systemctl daemon-reload
	# 启用并启动服务
	sudo systemctl enable controlman
	sudo systemctl start controlman

uninstall:
	# 删除静态文件
	sudo rm -rf /root/.controlman/static
	# 停止并禁用服务
	sudo systemctl stop controlman
	sudo systemctl disable controlman
	# 删除systemd服务文件
	sudo rm -f $(SERVICE_DIR)/controlman.service
	# 删除二进制文件
	sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	# 重新加载systemd
	sudo systemctl daemon-reload

reinstall: uninstall install

clean:
	rm -f $(BINARY_NAME)
