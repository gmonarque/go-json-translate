package db

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func GetDb() *gorm.DB {
	//Some GORM stuff going on here, rtfm
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	db, err := gorm.Open(sqlite.Open("translator.db?cache=shared"), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
	})

	if err != nil {
		panic("failed to connect database")
	}

	return db
}
