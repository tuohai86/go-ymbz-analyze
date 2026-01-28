-- 奔驰宝马分析系统 - 数据库初始化脚本
-- 注意：game_rounds, game_winners, bet_distribution 表应该已经存在

-- 创建策略状态表
CREATE TABLE IF NOT EXISTS strategies (
    name VARCHAR(50) PRIMARY KEY COMMENT '策略名称',
    profit BIGINT DEFAULT 0 COMMENT '理论总盈利',
    real_profit BIGINT DEFAULT 0 COMMENT '实盘累计盈利',
    wins INT DEFAULT 0 COMMENT '获胜次数',
    count INT DEFAULT 0 COMMENT '总次数',
    state TINYINT DEFAULT 0 COMMENT '状态(0=观望,1=实盘)',
    v_streak INT DEFAULT 0 COMMENT '虚盘连赢次数',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='策略状态表';

-- 创建策略历史快照表
CREATE TABLE IF NOT EXISTS strategy_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '主键',
    round_id VARCHAR(50) NOT NULL COMMENT '期号',
    strategy_name VARCHAR(50) NOT NULL COMMENT '策略名称',
    predictions JSON COMMENT '预测项',
    profit INT DEFAULT 0 COMMENT '本期盈亏',
    state TINYINT DEFAULT 0 COMMENT '当时状态',
    real_change INT DEFAULT 0 COMMENT '实盘变化',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX idx_round_id (round_id),
    INDEX idx_strategy_name (strategy_name),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='策略历史快照表';

-- 查看现有表
SHOW TABLES;

-- 验证表结构
DESCRIBE strategies;
DESCRIBE strategy_logs;
