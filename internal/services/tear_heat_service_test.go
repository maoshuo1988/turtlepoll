package services

import (
	"bbs-go/internal/models/models"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

func newTearHeatTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
		},
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.TearHeatSnapshot{}, &models.TearEventHeat{}))
	return db
}

func TestTearHeatService_AddPKHeatWritesSnapshotAndEventHeat(t *testing.T) {
	db := newTearHeatTestDB(t)
	eventId := time.Now().UnixNano()
	userId := eventId%100000 + 910000
	t.Cleanup(func() {
		db.Unscoped().Where("event_id = ?", eventId).Delete(&models.TearHeatSnapshot{})
		db.Unscoped().Where("event_id = ?", eventId).Delete(&models.TearEventHeat{})
	})

	require.NoError(t, TearHeatService.AddPKHeat(db, eventId, userId, "A", TearBusinessTypeArena, false, true, false, 0))
	require.NoError(t, TearHeatService.AddPKHeat(db, eventId, userId, "A", TearBusinessTypeArena, false, false, true, 100))

	var snapshotCount int64
	require.NoError(t, db.Model(&models.TearHeatSnapshot{}).
		Where("event_id = ? AND user_id = ? AND option = ? AND business_type = ?", eventId, userId, "A", TearBusinessTypeArena).
		Count(&snapshotCount).Error)
	require.EqualValues(t, 2, snapshotCount)

	eventHeat := &models.TearEventHeat{}
	require.NoError(t, db.Take(eventHeat,
		"event_id = ? AND user_id = ? AND option = ? AND business_type = ?",
		eventId, userId, "A", TearBusinessTypeArena,
	).Error)
	require.EqualValues(t, 0, eventHeat.HLike)
	require.EqualValues(t, 2, eventHeat.HComment)
	require.InDelta(t, 2, eventHeat.HCoin, 0.000001)
	require.InDelta(t, 4, eventHeat.HTotal, 0.000001)
}

func TestTearEventHeatService_SetAndAddHeat(t *testing.T) {
	db := newTearHeatTestDB(t)
	eventId := time.Now().UnixNano()
	userId := eventId%100000 + 920000
	t.Cleanup(func() {
		db.Unscoped().Where("event_id = ?", eventId).Delete(&models.TearEventHeat{})
	})

	require.NoError(t, TearEventHeatService.SetHeat(db, eventId, userId, "B", TearBusinessTypeDark, 1, 2, 3))
	require.NoError(t, TearEventHeatService.AddHeat(db, eventId, userId, "B", TearBusinessTypeDark, 1, 0, 2))

	eventHeat := &models.TearEventHeat{}
	require.NoError(t, db.Take(eventHeat,
		"event_id = ? AND user_id = ? AND option = ? AND business_type = ?",
		eventId, userId, "B", TearBusinessTypeDark,
	).Error)
	require.EqualValues(t, 2, eventHeat.HLike)
	require.EqualValues(t, 2, eventHeat.HComment)
	require.EqualValues(t, 5, eventHeat.HCoin)
	require.EqualValues(t, 9, eventHeat.HTotal)
}
