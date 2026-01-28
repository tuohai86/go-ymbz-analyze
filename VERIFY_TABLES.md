# 验证表自动创建功能

## ✅ 自动创建表功能已实现

程序启动时会**自动创建**以下表：
- `strategies` - 策略状态表
- `strategy_logs` - 策略历史快照表（含3个索引）

## 🔍 验证步骤

### 第1步：清空测试环境（可选）

如果想测试从零创建表，可以先删除现有表：

```sql
-- 备份数据（如有）
CREATE TABLE strategies_backup AS SELECT * FROM strategies;
CREATE TABLE strategy_logs_backup AS SELECT * FROM strategy_logs;

-- 删除表
DROP TABLE IF EXISTS strategies;
DROP TABLE IF EXISTS strategy_logs;

-- 验证表已删除
SHOW TABLES;
```

### 第2步：启动程序

```bash
go run main.go
```

### 第3步：查看启动日志

成功时会看到以下输出：

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

如果表已存在，会显示：

```
🔄 开始数据库表自动迁移...
  ✓ strategies 表已存在，检查字段...
  ✓ strategy_logs 表已存在，检查字段...
✅ 数据库表迁移完成
```

### 第4步：验证表结构

在 MySQL 中执行：

```sql
-- 查看所有表
SHOW TABLES;

-- 验证 strategies 表结构
DESCRIBE strategies;

-- 验证 strategy_logs 表结构
DESCRIBE strategy_logs;

-- 查看 strategy_logs 的索引
SHOW INDEX FROM strategy_logs;
```

### 第5步：验证表结构详情

**strategies 表应该包含：**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| name | VARCHAR(50) | 策略名称（主键） |
| profit | BIGINT | 理论总盈利 |
| real_profit | BIGINT | 实盘累计盈利 |
| wins | INT | 获胜次数 |
| count | INT | 总次数 |
| state | TINYINT | 状态（0=观望,1=实盘） |
| v_streak | INT | 虚盘连赢次数 |
| updated_at | TIMESTAMP | 更新时间 |

**strategy_logs 表应该包含：**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | BIGINT | 主键（自增） |
| round_id | VARCHAR(50) | 期号（有索引） |
| strategy_name | VARCHAR(50) | 策略名称（有索引） |
| predictions | JSON | 预测项 |
| profit | INT | 本期盈亏 |
| state | TINYINT | 当时状态 |
| real_change | INT | 实盘变化 |
| created_at | TIMESTAMP | 创建时间（有索引） |

**strategy_logs 的索引：**
- `idx_round_id` - 加速按期号查询
- `idx_strategy_name` - 加速按策略查询
- `idx_created_at` - 加速按时间查询

## 🔧 自动迁移功能说明

### GORM AutoMigrate 会自动：

1. ✅ **创建不存在的表**
2. ✅ **添加缺失的字段**
3. ✅ **创建索引**
4. ✅ **更新字段类型**（某些情况）
5. ⚠️ **不会删除已存在的字段**（安全设计）
6. ⚠️ **不会修改已存在的数据**

### 优点：

- 🚀 **开箱即用**：无需手动创建表
- 🔄 **自动更新**：代码更新后表结构自动同步
- 🛡️ **安全可靠**：不会删除数据
- 📝 **代码即文档**：表结构定义在 Go 代码中

## 🧪 测试场景

### 场景1：首次运行（表不存在）

**操作：**
```bash
# 确保表不存在
mysql -u root -p -e "DROP TABLE IF EXISTS benz_analysis.strategies, benz_analysis.strategy_logs"

# 启动程序
go run main.go
```

**预期：**
- ✅ 自动创建两个表
- ✅ 创建所有索引
- ✅ 程序正常运行

### 场景2：表已存在但缺少字段

**操作：**
```sql
-- 手动删除一个字段
ALTER TABLE strategies DROP COLUMN v_streak;
```

**启动程序后：**
- ✅ 自动添加缺失的 v_streak 字段
- ✅ 不影响现有数据

### 场景3：表结构完整

**操作：**
```bash
# 正常启动
go run main.go
```

**预期：**
- ✅ 检测到表已存在
- ✅ 验证字段完整性
- ✅ 快速启动

## 📊 SQL 验证脚本

完整的验证 SQL：

```sql
USE benz_analysis;

-- 1. 查看所有表
SHOW TABLES;

-- 2. 验证 strategies 表
SELECT 
    TABLE_NAME,
    ENGINE,
    TABLE_ROWS,
    CREATE_TIME
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'benz_analysis'
AND TABLE_NAME = 'strategies';

-- 3. 验证 strategy_logs 表
SELECT 
    TABLE_NAME,
    ENGINE,
    TABLE_ROWS,
    CREATE_TIME
FROM information_schema.TABLES
WHERE TABLE_SCHEMA = 'benz_analysis'
AND TABLE_NAME = 'strategy_logs';

-- 4. 查看所有索引
SELECT 
    TABLE_NAME,
    INDEX_NAME,
    COLUMN_NAME,
    SEQ_IN_INDEX
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = 'benz_analysis'
AND TABLE_NAME IN ('strategies', 'strategy_logs')
ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX;

-- 5. 验证数据类型
SELECT 
    COLUMN_NAME,
    DATA_TYPE,
    COLUMN_TYPE,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'benz_analysis'
AND TABLE_NAME = 'strategies'
ORDER BY ORDINAL_POSITION;

-- 6. 查看创建表的完整 SQL
SHOW CREATE TABLE strategies\G
SHOW CREATE TABLE strategy_logs\G
```

## ⚠️ 注意事项

### 权限要求

程序需要以下 MySQL 权限：

```sql
GRANT CREATE, ALTER, SELECT, INSERT, UPDATE, DELETE, INDEX 
ON benz_analysis.* 
TO 'your_user'@'localhost';

FLUSH PRIVILEGES;
```

### 字符集

确保数据库使用 UTF8MB4：

```sql
-- 查看数据库字符集
SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
FROM information_schema.SCHEMATA
WHERE SCHEMA_NAME = 'benz_analysis';

-- 如果不是 utf8mb4，修改：
ALTER DATABASE benz_analysis 
CHARACTER SET = utf8mb4 
COLLATE = utf8mb4_unicode_ci;
```

## 🐛 故障排查

### 问题1：表创建失败

**错误信息：**
```
❌ 表迁移失败: Error 1142: CREATE command denied to user
```

**解决方法：**
```sql
-- 授予 CREATE 权限
GRANT CREATE ON benz_analysis.* TO 'your_user'@'localhost';
FLUSH PRIVILEGES;
```

### 问题2：索引创建失败

**错误信息：**
```
❌ 表迁移失败: Error 1061: Duplicate key name 'idx_round_id'
```

**解决方法：**
```sql
-- 删除重复索引
DROP INDEX idx_round_id ON strategy_logs;

-- 重启程序自动重建
```

### 问题3：字段类型不匹配

**症状：** 数据插入失败或查询异常

**解决方法：**
```sql
-- 重建表（备份数据后）
DROP TABLE strategies;
DROP TABLE strategy_logs;

-- 重启程序自动创建
```

## ✅ 验证清单

- [ ] 程序启动无错误
- [ ] `SHOW TABLES` 能看到 strategies 和 strategy_logs
- [ ] `DESCRIBE strategies` 显示8个字段
- [ ] `DESCRIBE strategy_logs` 显示8个字段
- [ ] `SHOW INDEX FROM strategy_logs` 显示4个索引（PRIMARY + 3个自定义）
- [ ] 访问 `http://localhost:8001/api/status` 正常返回
- [ ] 日志显示 "✅ 数据库表迁移完成"

全部通过即表示自动创建表功能正常！🎉

## 📝 相关文件

- 模型定义：`models/strategy.go`
- 数据库初始化：`database/db.go`
- SQL脚本：`sql/init.sql`（手动创建时使用）

## 💡 提示

如果希望查看 GORM 执行的完整 SQL，可以启用详细日志：

```go
// 编辑 database/db.go
gormConfig := &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),  // 显示 SQL
}
```

这样可以看到 GORM 实际执行的 CREATE TABLE、CREATE INDEX 等语句。
