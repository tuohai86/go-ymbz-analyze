# 快速开始指南

## 5分钟上手

### 第1步：配置环境变量

创建 `.env` 文件（复制 `.env.example`）：

```bash
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=你的MySQL密码
DB_NAME=benz_analysis
SERVER_PORT=8001
```

### 第2步：确认数据库表

确保MySQL中已存在以下表（由游戏系统创建）：
- ✅ `game_rounds` - 游戏期数表
- ✅ `game_winners` - 获胜项表  
- ✅ `bet_distribution` - 投注分布表

> 💡 **自动创建表**：`strategies` 和 `strategy_logs` 表会在程序启动时**自动创建**，无需手动操作！

### 第3步：安装依赖

```bash
go mod download
```

### 第4步：运行程序

```bash
go run main.go
```

看到以下输出说明启动成功：

```
✅ 配置加载完成: localhost:3306/benz_analysis
✅ 数据库连接成功
🔄 开始数据库表自动迁移...
  📝 创建 strategies 表...
  📝 创建 strategy_logs 表...
✅ 数据库表迁移完成
  - strategies 表: 策略状态存储
  - strategy_logs 表: 策略历史快照（含索引）
✅ 策略状态加载完成
🚀 V52.0 狙击手版启动（虚实切换）
📱 狙击手地址: http://192.168.1.100:8001
🚀 服务器启动在端口: 8001
```

> 💡 如果表已存在，会显示 "✓ 表已存在，检查字段..." 而不是 "📝 创建表..."

### 第5步：访问系统

浏览器打开：`http://localhost:8001`

## API测试

### 测试状态接口

```bash
curl http://localhost:8001/api/status
```

### 测试历史记录

```bash
curl http://localhost:8001/api/logs?page=1&size=10
```

### 测试预测接口

```bash
curl http://localhost:8001/api/predictions
```

## 常见问题

### Q1: 数据库连接失败？

**A:** 检查以下几点：
1. MySQL服务是否启动
2. `.env` 中的数据库配置是否正确
3. 数据库是否已创建：`CREATE DATABASE benz_analysis;`

### Q2: 没有数据显示？

**A:** 确认 `game_rounds` 表中有数据。可以执行：

```sql
SELECT COUNT(*) FROM game_rounds;
SELECT * FROM game_rounds ORDER BY round_id DESC LIMIT 5;
```

### Q3: 端口被占用？

**A:** 修改 `.env` 中的 `SERVER_PORT` 为其他端口，如 8002。

### Q4: 编译失败？

**A:** 确保Go版本 >= 1.21，执行：

```bash
go version
go mod tidy
```

## 验证系统运行

系统正常运行时，会看到类似以下日志：

```
💰 结算期号: 20240128123
  ✅ 🔥 热门(3码) 实盘获胜 +2500，继续实盘
  📈 ⚖️ 均衡(4码) 虚盘获胜 +1800，连赢 2 次
  🎯 🔥 热门(3码) 预测: [红奔驰 绿宝马 红奥迪] (状态: 实盘)
  🎯 ⚖️ 均衡(4码) 预测: [黄奔驰 红大众 绿奥迪 黄奥迪] (状态: 观望)
```

## 下一步

- 📖 阅读 [README.md](README.md) 了解详细功能
- 🚀 查看 [DEPLOY.md](DEPLOY.md) 学习部署方法
- 🔧 修改 `services/constants.go` 调整策略参数

## 目录结构说明

```
.
├── main.go                     # 主入口（从这里开始看代码）
├── config/config.go           # 配置管理（环境变量）
├── models/                    # 数据模型
│   ├── game.go               # 游戏表模型
│   └── strategy.go           # 策略表模型
├── database/db.go            # 数据库连接
├── services/                 # 核心业务逻辑
│   ├── constants.go         # 常量定义（赔率、车型等）
│   ├── strategy_engine.go   # 策略引擎（热度评分算法）
│   └── bot_system.go        # 主系统（状态机、结算、预测）
├── handlers/api.go          # HTTP接口
├── sql/init.sql            # 数据库初始化脚本
├── .env                    # 配置文件（需手动创建）
└── README.md              # 完整文档
```

## 核心代码位置

想理解核心逻辑？看这几个文件：

1. **策略算法**：`services/strategy_engine.go`
   - `GetHeatScores()` - 热度评分算法
   - `StratHot3()` - 热门3码策略
   - `StratBalanced4()` - 均衡4码策略

2. **状态机**：`services/bot_system.go`
   - `settle()` - 结算逻辑（虚实切换在这里）
   - `predict()` - 预测逻辑

3. **API接口**：`handlers/api.go`
   - `GetStatus()` - 获取实时状态
   - `GetPredictions()` - 获取下注建议

祝使用愉快！🎉
