package storage

import (
	"overseer/common"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSqliteDB(databaseName string, migrate bool) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databaseName), &gorm.Config{})
	if err != nil {
		common.GetLogger("storage.NewSqliteDB").Error("failed to connect to sqlite", "error", err)
		return nil, err
	}

	if !migrate {
		return db, nil
	}

	common.GetLogger("storage.NewSqliteDB").Info("migrating models")
	err = db.AutoMigrate(
		&actor{},
		&user{},
		&game{},
		&gameParticipant{},
		&eventRow{},
		&eventReceipt{},
		&lock{},
		&gameMap{},
		&mapCoordinate{},
	)
	common.GetLogger("storage.NewSqliteDB").Info("migrated models")

	if err != nil {
		common.GetLogger("storage.NewSqliteDB").Error("failed to migrate models", "error", err)
		return nil, err
	}

	return db, nil
}
