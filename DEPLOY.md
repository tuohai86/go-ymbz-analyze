# 部署指南

## 重要说明

本项目使用 Go embed 功能将静态文件（index.html 和 assets）嵌入到二进制文件中，**无需额外复制静态文件**，非常适合 Zeabur、Docker、CI/CD 等自动化部署环境。

## 开发环境部署

### 1. 准备工作

```bash
# 克隆或进入项目目录
cd f:\奔驰宝马\分析端

# 安装 Go 依赖
go mod download
```

### 2. 配置数据库

创建 `.env` 文件：

```bash
cp .env.example .env
```

编辑 `.env`，填入正确的数据库配置：

```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=benz_analysis
SERVER_PORT=8001
```

### 3. 初始化数据库

系统会自动创建所需的表，但如果需要手动创建：

```bash
mysql -u root -p benz_analysis < sql/init.sql
```

### 4. 运行程序

```bash
# 开发模式
go run main.go

# 或使用 Makefile
make run
```

## 生产环境部署

### 方案一：直接编译运行

```bash
# 编译（静态文件会自动嵌入）
go build -o benz-sniper main.go

# ⚠️ 注意：编译后的二进制文件已包含所有静态资源
# 无需复制 index.html 或 assets 目录

# 创建系统服务 (Linux)
sudo vim /etc/systemd/system/benz-sniper.service
```

服务配置文件内容：

```ini
[Unit]
Description=Benz Sniper Analysis System
After=network.target mysql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/benz-sniper
Environment="DB_HOST=localhost"
Environment="DB_PORT=3306"
Environment="DB_USER=benz_user"
Environment="DB_PASSWORD=secure_password"
Environment="DB_NAME=benz_analysis"
Environment="SERVER_PORT=8001"
ExecStart=/opt/benz-sniper/benz-sniper
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable benz-sniper
sudo systemctl start benz-sniper
sudo systemctl status benz-sniper
```

### 方案二：使用 Docker

创建 `Dockerfile`：

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o benz-sniper main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
# 只需复制二进制文件，静态资源已嵌入
COPY --from=builder /app/benz-sniper .

EXPOSE 8001
CMD ["./benz-sniper"]
```

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  benz-sniper:
    build: .
    ports:
      - "8001:8001"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=benz_user
      - DB_PASSWORD=secure_password
      - DB_NAME=benz_analysis
      - SERVER_PORT=8001
    depends_on:
      - mysql
    restart: always

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_password
      MYSQL_DATABASE: benz_analysis
      MYSQL_USER: benz_user
      MYSQL_PASSWORD: secure_password
    volumes:
      - mysql_data:/var/lib/mysql
      - ./sql/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "3306:3306"

volumes:
  mysql_data:
```

启动：

```bash
docker-compose up -d
```

### 方案三：Zeabur 部署（推荐）

Zeabur 会自动检测 Go 项目并构建，无需额外配置：

1. **连接 Git 仓库**
   - 将代码推送到 GitHub/GitLab
   - 在 Zeabur 中连接仓库

2. **配置环境变量**
   ```
   DB_HOST=your-database-host
   DB_PORT=3306
   DB_USER=your-username
   DB_PASSWORD=your-password
   DB_NAME=benz_analysis
   SERVER_PORT=8080
   ```

3. **部署**
   - Zeabur 会自动：
     - 检测 Go 项目
     - 运行 `go build`
     - 将静态文件嵌入二进制
     - 启动服务

4. **优势**
   - ✅ 自动 CI/CD
   - ✅ 静态文件自动嵌入
   - ✅ 无需 Dockerfile
   - ✅ 自动扩展
   - ✅ HTTPS 自动配置

### 方案四：使用 Nginx 反向代理

Nginx 配置示例：

```nginx
server {
    listen 80;
    server_name sniper.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 性能优化建议

### 1. 数据库优化

```sql
-- 为常用查询添加索引
CREATE INDEX idx_round_id_created ON game_rounds(round_id, created_at);
CREATE INDEX idx_winner_round ON game_winners(round_id, winner_name);

-- 定期清理旧数据（保留最近1000期）
DELETE FROM strategy_logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
```

### 2. 应用优化

在 `database/db.go` 中调整连接池：

```go
sqlDB.SetMaxIdleConns(20)      // 增加空闲连接数
sqlDB.SetMaxOpenConns(200)     // 增加最大连接数
sqlDB.SetConnMaxLifetime(time.Hour)
```

### 3. 系统资源

推荐配置：
- CPU: 2核+
- 内存: 2GB+
- 磁盘: 20GB+（含数据库）

## 监控和日志

### 查看日志

```bash
# systemd 服务日志
sudo journalctl -u benz-sniper -f

# Docker 日志
docker-compose logs -f benz-sniper
```

### 监控指标

- 数据库连接数
- API 响应时间
- 策略胜率变化
- 内存使用情况

可使用 Prometheus + Grafana 进行监控。

## 备份策略

### 数据库备份

```bash
# 每日自动备份
0 2 * * * mysqldump -u root -p benz_analysis strategies strategy_logs > /backup/benz_$(date +\%Y\%m\%d).sql
```

### 应用备份

```bash
# 备份可执行文件和配置（静态文件已嵌入二进制）
tar -czf benz-sniper-backup.tar.gz benz-sniper .env
```

## 故障排查

### 1. 数据库连接失败

- 检查数据库是否运行
- 验证连接信息是否正确
- 检查防火墙规则

### 2. 程序无法启动

- 检查端口是否被占用：`netstat -tuln | grep 8001`
- 查看错误日志
- 验证 .env 文件是否存在

### 3. 数据不更新

- 检查 game_rounds 表是否有新数据
- 查看程序日志确认轮询是否正常
- 验证数据库权限

## 安全建议

1. **数据库安全**
   - 使用强密码
   - 限制远程访问
   - 定期更新 MySQL

2. **应用安全**
   - 使用 HTTPS（配置 SSL 证书）
   - 配置防火墙规则
   - 定期更新依赖

3. **访问控制**
   - 可在 Nginx 层添加基础认证
   - 限制 API 访问频率
   - 记录访问日志

## 更新升级

```bash
# 停止服务
sudo systemctl stop benz-sniper

# 备份当前版本
cp benz-sniper benz-sniper.backup

# 编译新版本
go build -o benz-sniper main.go

# 启动服务
sudo systemctl start benz-sniper
```

## 支持

如遇问题，请查看：
- README.md - 功能说明
- 日志文件 - 错误信息
- 数据库状态 - 数据完整性
