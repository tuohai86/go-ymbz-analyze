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
	
	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	log.Println("🔄 开始数据库表自动迁移...")
	
	// 检查游戏相关表
	if !db.Migrator().HasTable(&models.GameRound{}) {
		log.Println("  📝 创建 game_rounds 表...")
	} else {
		log.Println("  ✓ game_rounds 表已存在")
	}
	
	if !db.Migrator().HasTable(&models.GameWinner{}) {
		log.Println("  📝 创建 game_winners 表...")
	} else {
		log.Println("  ✓ game_winners 表已存在")
	}
	
	if !db.Migrator().HasTable(&models.BetDistribution{}) {
		log.Println("  📝 创建 bet_distribution 表...")
	} else {
		log.Println("  ✓ bet_distribution 表已存在")
	}
	
	if !db.Migrator().HasTable(&models.StrategyHistory{}) {
		log.Println("  📝 创建 strategy_history 表...")
	} else {
		log.Println("  ✓ strategy_history 表已存在")
	}
	
	// 执行自动迁移（仅游戏相关表）
	err := db.AutoMigrate(
		&models.GameRound{},
		&models.GameWinner{},
		&models.BetDistribution{},
		&models.StrategyHistory{},
	)
	
	if err != nil {
		log.Printf("❌ 表迁移失败: %v", err)
		return err
	}
	
	log.Println("✅ 数据库表迁移完成")
	log.Println("  - game_rounds 表: 游戏期数")
	log.Println("  - game_winners 表: 获胜项")
	log.Println("  - bet_distribution 表: 投注分布")
	log.Println("  - strategy_history 表: 策略历史记录")
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
