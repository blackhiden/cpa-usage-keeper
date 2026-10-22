package test

import (
	"database/sql"
	"testing"
	"time"

	"cpa-usage-keeper/internal/entities"
)

func TestLocalRankingRestoreMigrationRecreatesThePostV2Schema(t *testing.T) {
	db := openUnmigratedTestDatabase(t)
	if err := db.Exec(`CREATE TABLE cpa_api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT,
		display_key TEXT,
		key_alias TEXT,
		is_deleted NUMERIC,
		last_synced_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create legacy CPA API key table: %v", err)
	}
	now := time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)
	if err := db.Exec(
		"INSERT INTO cpa_api_keys (id, api_key, display_key, key_alias, is_deleted, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		7, "sk-existing", "sk-***existing", "Existing", false, now, now,
	).Error; err != nil {
		t.Fatalf("seed existing CPA API key: %v", err)
	}

	runOnlyMigration(t, db, "20261022_restore_local_ranking")
	if !db.Migrator().HasTable(&entities.LocalRankingPeriodStat{}) {
		t.Fatal("expected local ranking period stats table")
	}
	if !db.Migrator().HasColumn("cpa_api_keys", "local_ranking_avatar_id") {
		t.Fatal("expected local ranking avatar column")
	}
	var row struct {
		APIKey               string
		KeyAlias             string
		LocalRankingAvatarID sql.NullInt64
	}
	if err := db.Table("cpa_api_keys").Where("id = ?", 7).First(&row).Error; err != nil {
		t.Fatalf("reload migrated CPA API key: %v", err)
	}
	if row.APIKey != "sk-existing" || row.KeyAlias != "Existing" || row.LocalRankingAvatarID.Valid {
		t.Fatalf("restore migration changed the existing key: %+v", row)
	}
}
