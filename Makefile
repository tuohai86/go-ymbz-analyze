# Makefile for Benz Sniper

.PHONY: run build clean install dev

# 运行项目
run:
	go run main.go

# 编译项目
build:
	go build -o benz-sniper main.go

# 编译（Windows）
build-windows:
	GOOS=windows GOARCH=amd64 go build -o benz-sniper.exe main.go

# 编译（Linux）
build-linux:
	GOOS=linux GOARCH=amd64 go build -o benz-sniper main.go

# 安装依赖
install:
	go mod download

# 开发模式（热重载需要安装 air）
dev:
	air

# 清理编译文件
clean:
	rm -f benz-sniper benz-sniper.exe

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
vet:
	go vet ./...

# 运行测试
test:
	go test -v ./...

# 查看依赖
deps:
	go list -m all

# 更新依赖
update:
	go get -u ./...
	go mod tidy
