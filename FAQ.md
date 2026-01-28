# 常见问题解答（FAQ）

## 🔧 安装和配置

### Q1: 如何安装 Go 环境？

**A:** 访问 [golang.org](https://golang.org/dl/) 下载安装包。

Windows:
```bash
# 下载 .msi 安装包，双击安装
# 验证安装
go version
```

Linux:
```bash
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version
```

### Q2: 依赖下载失败怎么办？

**A:** 配置 Go 代理：

```bash
# Windows PowerShell
$env:GOPROXY = "https://goproxy.cn,direct"

# Linux/Mac
export GOPROXY=https://goproxy.cn,direct

# 永久配置
go env -w GOPROXY=https://goproxy.cn,direct
```

然后重新下载：
```bash
go mod download
```

### Q3: 如何创建 MySQL 数据库？

**A:** 登录 MySQL 后执行：

```sql
CREATE DATABASE benz_analysis CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建专用用户（可选）
CREATE USER 'benz_user'@'localhost' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON benz_analysis.* TO 'benz_user'@'localhost';
FLUSH PRIVILEGES;
```

---

## 🚀 运行问题

### Q4: 提示 "数据库连接失败"？

**A:** 检查以下几点：

1. **MySQL 是否运行？**
   ```bash
   # Windows
   net start mysql
   
   # Linux
   sudo systemctl status mysql
   ```

2. **.env 配置是否正确？**
   ```env
   DB_HOST=localhost    # 确认主机地址
   DB_PORT=3306         # 确认端口
   DB_USER=root         # 确认用户名
   DB_PASSWORD=***      # 确认密码
   DB_NAME=benz_analysis  # 确认数据库名
   ```

3. **防火墙是否阻止？**
   ```bash
   # 测试连接
   mysql -h localhost -u root -p
   ```

### Q5: 提示 "端口已被占用"？

**A:** 修改端口或关闭占用程序：

```bash
# Windows 查看端口占用
netstat -ano | findstr :8001

# Linux 查看端口占用
lsof -i :8001

# 修改配置使用其他端口
# 编辑 .env
SERVER_PORT=8002
```

### Q6: 程序启动后没有数据显示？

**A:** 检查数据源：

```sql
-- 确认游戏表是否有数据
SELECT COUNT(*) FROM game_rounds;
SELECT * FROM game_rounds ORDER BY round_id DESC LIMIT 5;

-- 确认获胜项表
SELECT COUNT(*) FROM game_winners;
```

如果没有数据，需要先运行游戏数据采集系统。

### Q7: 策略一直是"观望"状态？

**A:** 这是正常的，需要满足进场条件：

- 虚盘连赢 **2把** 才会进入实盘
- 查看日志确认是否有预测和结算：

```
💰 结算期号: xxx
  📈 🔥 热门(3码) 虚盘获胜 +1500，连赢 1 次  # 需要连赢2次
  🎯 🔥 热门(3码) 预测: [红奔驰 绿宝马 红奥迪] (状态: 观望)
```

---

## 📊 数据和算法

### Q8: 热度评分算法是如何工作的？

**A:** 算法分析最近30期数据，对每个车型计算加权分数：

```
对于每一期（从旧到新）：
  weight = 0.5 + (当前位置 / 总数)  # 越新权重越高
  
  如果该期开出了某车型：
    score[该车型] += 1.0 × weight
```

**示例：**
- 第1期（最旧）：weight = 0.5
- 第15期（中间）：weight = 1.0
- 第30期（最新）：weight = 1.5

最新开出的车型得分更高，体现趋势追踪。

### Q9: 为什么要虚实切换？

**A:** 这是风控机制：

1. **虚盘观望**：测试策略准确性，不实际投入
2. **连赢触发**：证明策略有效，进入实盘
3. **失败止损**：一旦失利立即退出，保护资金
4. **乘胜追击**：实盘获胜则继续

这样可以最大化收益，最小化风险。

### Q10: 如何调整策略参数？

**A:** 编辑 `services/constants.go`：

```go
const (
    ENTRY_CONDITION = 3  // 改为虚盘连赢3把进实盘（更保守）
    EXIT_CONDITION  = 2  // 改为实盘连输2把退虚盘（更激进）
)
```

修改后重启程序生效。

---

## 🔌 API 使用

### Q11: 如何测试 API 接口？

**A:** 使用 curl 或 Postman：

```bash
# 获取状态
curl http://localhost:8001/api/status

# 获取历史（第2页，每页20条）
curl "http://localhost:8001/api/logs?page=2&size=20"

# 获取预测
curl http://localhost:8001/api/predictions
```

### Q12: API 返回的数据格式是什么？

**A:** 所有接口返回 JSON 格式：

```json
// /api/status
{
  "lid": "当前期号",
  "next_lid": "下期期号",
  "leaderboard": [
    {
      "name": "策略名",
      "profit": 实盘盈利,
      "total_profit": 理论总盈利,
      "rate": 胜率,
      "state": 0或1,  // 0=观望, 1=实盘
      "next": ["预测项1", "预测项2"]
    }
  ]
}
```

### Q13: 如何在其他程序中调用API？

**A:** 示例代码：

**Python:**
```python
import requests

response = requests.get('http://localhost:8001/api/status')
data = response.json()
print(f"当前期号: {data['lid']}")
```

**JavaScript:**
```javascript
fetch('http://localhost:8001/api/status')
  .then(res => res.json())
  .then(data => console.log(data));
```

**Go:**
```go
resp, _ := http.Get("http://localhost:8001/api/status")
body, _ := ioutil.ReadAll(resp.Body)
var data map[string]interface{}
json.Unmarshal(body, &data)
```

---

## 🐛 故障排查

### Q14: 程序运行一段时间后崩溃？

**A:** 检查以下几点：

1. **查看日志**：
   ```bash
   # 如果使用 systemd
   journalctl -u benz-sniper -n 100
   
   # 如果直接运行
   # 查看终端输出的错误信息
   ```

2. **检查内存**：
   ```bash
   # Linux
   free -h
   top -p $(pgrep benz-sniper)
   ```

3. **检查数据库连接**：
   ```sql
   SHOW PROCESSLIST;  -- 查看连接数
   ```

### Q15: 策略盈亏计算不准确？

**A:** 确认以下几点：

1. **赔率配置是否正确？**
   检查 `services/constants.go` 中的 `REAL_ODDS`

2. **获胜项是否正确？**
   ```sql
   SELECT * FROM game_winners WHERE round_id = '期号';
   ```

3. **预测是否正确记录？**
   ```sql
   SELECT * FROM strategy_logs WHERE round_id = '期号';
   ```

### Q16: 如何重置所有策略数据？

**A:** 清空策略表：

```sql
-- 备份数据（可选）
CREATE TABLE strategies_backup AS SELECT * FROM strategies;
CREATE TABLE strategy_logs_backup AS SELECT * FROM strategy_logs;

-- 清空数据
TRUNCATE TABLE strategies;
TRUNCATE TABLE strategy_logs;
```

重启程序后会从零开始。

---

## 🚢 部署相关

### Q17: 如何在生产环境运行？

**A:** 使用 systemd 服务（推荐）：

1. 编译程序：
   ```bash
   go build -o benz-sniper main.go
   ```

2. 创建服务文件（参考 DEPLOY.md）

3. 启动服务：
   ```bash
   sudo systemctl start benz-sniper
   sudo systemctl enable benz-sniper
   ```

### Q18: 如何配置 Nginx 反向代理？

**A:** Nginx 配置示例：

```nginx
server {
    listen 80;
    server_name sniper.yourdomain.com;
    
    location / {
        proxy_pass http://127.0.0.1:8001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Q19: 如何使用 Docker 部署？

**A:** 参考 DEPLOY.md 中的 Docker 部署方案，或：

```bash
# 构建镜像
docker build -t benz-sniper .

# 运行容器
docker run -d \
  -p 8001:8001 \
  -e DB_HOST=host.docker.internal \
  -e DB_PASSWORD=password \
  --name benz-sniper \
  benz-sniper
```

---

## 📝 开发相关

### Q20: 如何添加新的策略？

**A:** 步骤：

1. 在 `services/strategy_engine.go` 添加策略函数：
   ```go
   func (e *StrategyEngine) StratMyNew(rounds []models.GameRound) []string {
       // 你的策略逻辑
       return predictions
   }
   ```

2. 在 `services/bot_system.go` 的 `NewBotSystem` 中注册：
   ```go
   bot.strategies["🎯 我的策略"] = &StrategyState{
       Name: "🎯 我的策略",
       Func: engine.StratMyNew,
       Cost: 500,
       // ...
   }
   ```

3. 重新编译运行

### Q21: 如何修改轮询间隔？

**A:** 编辑 `services/bot_system.go` 的 `Loop` 方法：

```go
func (b *BotSystem) Loop() {
    for b.running {
        b.tick()
        time.Sleep(5 * time.Second)  // 改为5秒
    }
}
```

### Q22: 如何启用调试日志？

**A:** 编辑 `database/db.go`：

```go
gormConfig := &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),  // Info 级别
    // 或改为 Silent 关闭日志
    // Logger: logger.Default.LogMode(logger.Silent),
}
```

---

## 💡 性能优化

### Q23: 如何提升系统性能？

**A:** 优化建议：

1. **数据库索引**：
   ```sql
   CREATE INDEX idx_round_created ON game_rounds(round_id, created_at);
   ```

2. **连接池调优**（`database/db.go`）：
   ```go
   sqlDB.SetMaxIdleConns(20)
   sqlDB.SetMaxOpenConns(200)
   ```

3. **使用 Redis 缓存**（可选）

4. **定期清理旧数据**

### Q24: 数据库占用空间太大？

**A:** 定期清理历史数据：

```sql
-- 只保留最近90天的日志
DELETE FROM strategy_logs 
WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 优化表
OPTIMIZE TABLE strategy_logs;
```

---

## 🆘 获取帮助

### Q25: 遇到问题如何寻求帮助？

**A:** 提供以下信息：

1. **系统信息**：
   - 操作系统版本
   - Go 版本：`go version`
   - MySQL 版本：`mysql --version`

2. **错误信息**：
   - 完整的错误日志
   - 复现步骤

3. **配置信息**：
   - .env 配置（隐藏密码）
   - 相关代码修改

4. **环境检查**：
   ```bash
   # 检查数据库连接
   mysql -h localhost -u root -p -e "SELECT 1"
   
   # 检查端口
   netstat -tuln | grep 8001
   
   # 检查Go环境
   go env
   ```

---

## 📚 相关文档

- [README.md](README.md) - 完整功能文档
- [QUICKSTART.md](QUICKSTART.md) - 快速开始
- [DEPLOY.md](DEPLOY.md) - 部署指南
- [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) - 项目总结

还有问题？欢迎提出！👋
