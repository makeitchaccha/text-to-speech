package preset

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestPresetIDRepository(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)

	// always use the latest schema
	goose.SetBaseFS(nil)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.Up(db.DB, "../../migrations"))

	repo := NewPresetIDRepository(db)
	ctx := context.Background()

	t.Run("Save and Find", func(t *testing.T) {
		scope := ScopeGuild
		scopeID := snowflake.ID(12345)
		presetID := PresetID("test-preset-a")

		err := repo.Save(ctx, scope, scopeID, presetID)
		require.NoError(t, err)

		foundPresetID, err := repo.Find(ctx, scope, scopeID)
		require.NoError(t, err)
		require.Equal(t, presetID, foundPresetID)
	})

	t.Run("Save and Update", func(t *testing.T) {
		scope := ScopeGuild
		scopeID := snowflake.ID(67890)
		presetID1 := PresetID("test-preset-c")
		presetID2 := PresetID("test-preset-d")

		err := repo.Save(ctx, scope, scopeID, presetID1)
		require.NoError(t, err)

		err = repo.Save(ctx, scope, scopeID, presetID2) // Save again with the same key
		require.NoError(t, err)

		foundPresetID, err := repo.Find(ctx, scope, scopeID)
		require.NoError(t, err)
		require.Equal(t, presetID2, foundPresetID) // Should be the updated value
	})

	t.Run("Find Not Found", func(t *testing.T) {
		scope := ScopeUser
		scopeID := snowflake.ID(54321)

		_, err := repo.Find(ctx, scope, scopeID)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Delete", func(t *testing.T) {
		scope := ScopeGuild
		scopeID := snowflake.ID(98765)
		presetID := PresetID("test-preset-b")

		err := repo.Save(ctx, scope, scopeID, presetID)
		require.NoError(t, err)

		err = repo.Delete(ctx, scope, scopeID)
		require.NoError(t, err)

		_, err = repo.Find(ctx, scope, scopeID)
		require.ErrorIs(t, err, ErrNotFound)
	})
}
