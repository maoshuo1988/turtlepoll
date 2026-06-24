package services_test

import (
	"bbs-go/internal/install"
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mlogclub/simple/sqls"
	"github.com/stretchr/testify/require"
)

func TestTearHeatService_SaveToLocalPostgres(t *testing.T) {
	if os.Getenv("TEAR_HEAT_LOCAL_DB_TEST") != "1" {
		t.Skip("set TEAR_HEAT_LOCAL_DB_TEST=1 to write tear heat test data to local PostgreSQL")
	}

	chdirRepoRoot(t)
	install.InitConfig()
	install.InitLogger()
	require.NoError(t, install.InitDB())

	db := sqls.DB()
	require.NotNil(t, db)
	require.NoError(t, db.AutoMigrate(&models.TearHeatSnapshot{}, &models.TearEventHeat{}))

	eventId := time.Now().UnixNano()
	userId := eventId%100000 + 940000
	option := "A"

	require.NoError(t, services.TearHeatService.AddPKHeat(db, eventId, userId, option, services.TearBusinessTypeArena, false, true, false, 0))
	require.NoError(t, services.TearHeatService.AddPKHeat(db, eventId, userId, option, services.TearBusinessTypeArena, false, false, true, 100))
	require.NoError(t, services.TearHeatService.AddPKHeat(db, eventId, userId, option, services.TearBusinessTypeArena, true, false, false, 0))

	var snapshotCount int64
	require.NoError(t, db.Model(&models.TearHeatSnapshot{}).
		Where("event_id = ? AND user_id = ? AND option = ? AND business_type = ?", eventId, userId, option, services.TearBusinessTypeArena).
		Count(&snapshotCount).Error)
	require.EqualValues(t, 3, snapshotCount)

	eventHeat := &models.TearEventHeat{}
	require.NoError(t, db.Take(eventHeat,
		"event_id = ? AND user_id = ? AND option = ? AND business_type = ?",
		eventId, userId, option, services.TearBusinessTypeArena,
	).Error)
	require.EqualValues(t, 1, eventHeat.HLike)
	require.EqualValues(t, 2, eventHeat.HComment)
	require.InDelta(t, 2, eventHeat.HCoin, 0.000001)
	require.InDelta(t, 5, eventHeat.HTotal, 0.000001)

	t.Logf("saved tear heat local PostgreSQL test data: event_id=%d user_id=%d option=%s business_type=%d snapshot_count=%d event_heat_id=%d",
		eventId, userId, option, services.TearBusinessTypeArena, snapshotCount, eventHeat.Id)
}

func chdirRepoRoot(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "bbs-go.yaml")); err == nil {
			require.NoError(t, os.Chdir(dir))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("bbs-go.yaml not found")
		}
		dir = parent
	}
}
