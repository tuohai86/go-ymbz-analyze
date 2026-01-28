package database

import (
	"benz-sniper/config"
	"benz-sniper/models"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.Config) error {
	dsn := cfg.GetDSN()
	
	// 配置 GORM（禁用缓存，确保每次查询都是新鲜的）
	gormConfig := &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Warn),
		SkipDefaultTransaction: true,  // 跳过默认事务，提高性能
		PrepareStmt:            false, // 禁用预编译语句缓存
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		log.Printf("❌ 数据库连接失败: %v", err)
		return err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("✅ 数据库连接成功")

	// 自动迁移策略相关表
	if err := AutoMigrate(db); err != nil {
		log.Printf("⚠️ 数据库迁移失败: %v", err)
		return err
	}

	return nil
}

// AutoMigrate 自动迁移表结构（仅迁移游戏相关表）
func AutoMigrate(db *gorm.DB) error {
	// 执行自动迁移（仅游戏相关表）
	err := db.AutoMigrate(
		&models.GameRound{},
		&models.GameWinner{},
		&models.BetDistribution{},
		&models.StrategyHistory{},
		&models.SystemConfig{},
		&models.UserBet{},
	)
	
	if err != nil {
		log.Printf("❌ 数据库表迁移失败: %v", err)
		return err
	}
	
	log.Println("✅ 数据库表迁移完成")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
