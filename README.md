# 奔驰宝马分析系统 - Go版本

基于Gin + GORM + MySQL的奔驰宝马游戏策略分析系统，实现了虚实切换的智能下注策略。

## 功能特点

- ✅ **双策略系统**：热门3码策略 + 均衡4码策略
- ✅ **虚实切换**：虚盘连赢2把进实盘，实盘连输1把退虚盘
- ✅ **热度评分**：基于时间加权的智能热度分析算法
- ✅ **实时轮询**：每2秒自动查询数据库最新期数
- ✅ **状态持久化**：策略状态保存到数据库，重启不丢失
- ✅ **自动建表**：程序启动时自动创建 strategies 和 strategy_logs 表
- ✅ **RESTful API**：提供完整的HTTP接口

## 项目结构

```
.
├── main.go                    # 主入口
├── config/
│   └── config.go             # 配置管理
├── models/
│   ├── game.go               # 游戏数据模型
│   └── strategy.go           # 策略数据模型
├── database/
│   └── db.go                 # 数据库连接
├── services/
│   ├── constants.go          # 常量定义
│   ├── strategy_engine.go    # 策略引擎
│   └── bot_system.go         # 业务系统
├── handlers/
│   └── api.go                # HTTP处理器
├── go.mod                    # 依赖管理
└── .env                      # 环境变量配置
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- MySQL 5.7+
- 已存在的游戏数据表（game_rounds, game_winners, bet_distribution）

### 2. 配置环境变量

复制 `.env.example` 为 `.env`，并修改配置：

```bash
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=benz_analysis
SERVER_PORT=8001
```

### 3. 安装依赖

```bash
go mod download
```

### 4. 运行程序

```bash
go run main.go
```

或编译后运行：

```bash
go build -o benz-sniper
./benz-sniper
```

### 5. 访问服务

- 前端页面：`http://localhost:8001`
- API文档：见下方API说明

## 数据库表结构

### 系统自动创建的表

程序启动时会自动创建以下表：

**strategies 表**（策略状态）
```sql
CREATE TABLE strategies (
    name VARCHAR(50) PRIMARY KEY,
    profit BIGINT DEFAULT 0,
    real_profit BIGINT DEFAULT 0,
    wins INT DEFAULT 0,
    count INT DEFAULT 0,
    state TINYINT DEFAULT 0,
    v_streak INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

**strategy_logs 表**（策略历史快照）
```sql
CREATE TABLE strategy_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    round_id VARCHAR(50) NOT NULL,
    strategy_name VARCHAR(50) NOT NULL,
    predictions JSON,
    profit INT DEFAULT 0,
    state TINYINT DEFAULT 0,
    real_change INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_round_id (round_id),
    INDEX idx_strategy_name (strategy_name),
    INDEX idx_created_at (created_at)
);
```

> 💡 **注意**：这两个表会在程序**首次启动时自动创建**，无需手动创建。如需验证，请参考 [VERIFY_TABLES.md](VERIFY_TABLES.md)

### 需要预先存在的表

需要确保MySQL中已存在以下表（由游戏系统创建）：

- `game_rounds`：游戏期数表
- `game_winners`：获胜项表
- `bet_distribution`：投注分布表

## API 接口

### 1. 获取状态和排行榜

**请求**
```
GET /api/status
```

**响应**
```json
{
  "lid": "当前期号",
  "next_lid": "下期期号",
  "last_res": "上期结果",
  "time_passed": 15,
  "countdown": 19,
  "leaderboard": [
    {
      "name": "🔥 热门(3码)",
      "profit": 2500,
      "total_profit": 3200,
      "rate": 65,
      "state": 1,
      "next": ["红奔驰", "绿宝马", "红奥迪"]
    }
  ],
  "logs": [...]
}
```

### 2. 分页获取历史记录

**请求**
```
GET /api/logs?page=1&size=50
```

**响应**
```json
{
  "total": 1000,
  "page": 1,
  "size": 50,
  "total_pages": 20,
  "logs": [...]
}
```

### 3. 获取实盘策略预测

**请求**
```
GET /api/predictions
```

**响应**
```json
{
  "round": "下期期号",
  "predictions": {
    "红奔驰": 100,
    "绿宝马": 100,
    "红奥迪": 100
  }
}
```

## 策略说明

### 1. 热门3码策略（成本300元）

- 分析最近30期数据
- 计算每个车型的热度分数（时间加权）
- 选择热度最高的3个车型下注

### 2. 均衡4码策略（成本400元）

- 分析最近30期数据
- 从大车（奔驰、宝马）中选热度最高的1个
- 从小车（奥迪、大众）中选热度最高的3个
- 组合成4码下注

### 虚实切换机制

**状态定义：**
- `state = 0`：观望模式（虚盘，不实际下注）
- `state = 1`：实盘模式（真实下注）

**切换规则：**
- 进场条件：虚盘连赢2把 → 进入实盘
- 退场条件：实盘连输1把 → 退回虚盘
- 实盘获胜 → 继续实盘（乘胜追击）

## 核心算法

### 热度评分算法

```go
weight = 0.5 + (当前位置 / 总数)
score[车型] += 出现次数 × weight
```

越近期的数据权重越高（范围：0.5 ~ 1.5），有效捕捉趋势变化。

## 开发说明

### 添加新策略

在 `services/bot_system.go` 的 `NewBotSystem` 函数中添加：

```go
bot.strategies["新策略名称"] = &StrategyState{
    Name:       "新策略名称",
    Func:       engine.NewStrategyFunc,  // 在 strategy_engine.go 中实现
    Cost:       300,
    // ...其他字段
}
```

### 修改进出场规则

在 `services/constants.go` 中修改：

```go
const (
    ENTRY_CONDITION = 2  // 虚盘连赢N把进实盘
    EXIT_CONDITION  = 1  // 实盘连输N把退虚盘
)
```

## 日志说明

程序运行时会输出详细日志：

```
✅ 配置加载完成: localhost:3306/benz_analysis
✅ 数据库连接成功
✅ 数据库表迁移完成
✅ 策略状态加载完成
🚀 V52.0 狙击手版启动（虚实切换）
📱 狙击手地址: http://192.168.1.100:8001
🚀 服务器启动在端口: 8001

💰 结算期号: 20240128001
  ✅ 🔥 热门(3码) 实盘获胜 +2500，继续实盘
  📈 ⚖️ 均衡(4码) 虚盘获胜 +1800，连赢 2 次
  🚀 ⚖️ 均衡(4码) 虚盘连赢达标，进入实盘！
  🎯 🔥 热门(3码) 预测: [红奔驰 绿宝马 红奥迪] (状态: 实盘)
  🎯 ⚖️ 均衡(4码) 预测: [黄奔驰 红大众 绿奥迪 黄奥迪] (状态: 实盘)
```

## 性能优化

- 使用 `sync.RWMutex` 保证并发安全
- 数据库连接池配置（最大100连接）
- GORM 预加载和批量查询优化
- Gin Release 模式运行

## 注意事项

1. 确保 MySQL 中已存在游戏数据表
2. 数据库用户需要有创建表的权限
3. 建议使用反向代理（如Nginx）部署生产环境
4. 定期备份 strategies 和 strategy_logs 表数据

## License

MIT License
