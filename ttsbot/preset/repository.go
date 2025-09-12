package preset

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/disgoorg/snowflake/v2"
	"github.com/jmoiron/sqlx"
)

type Scope string

const (
	ScopeGuild Scope = "guild"
	ScopeUser  Scope = "user"
)

func (s Scope) String() string {
	return string(s)
}

var (
	ErrNotFound = errors.New("preset ID not found")
)

type PresetIDRepository interface {
	Find(ctx context.Context, scope Scope, ID snowflake.ID) (PresetID, error)
	Save(ctx context.Context, scope Scope, ID snowflake.ID, presetID PresetID) error
	Delete(ctx context.Context, scope Scope, ID snowflake.ID) error
}

func NewPresetIDRepository(db *sqlx.DB) PresetIDRepository {
	return &presetIDRepositoryImpl{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}
}

type presetIDRepositoryImpl struct {
	db   *sqlx.DB
	psql squirrel.StatementBuilderType
}

type ScopedPresetID struct {
	Scope     Scope        `db:"scope"`
	ID        snowflake.ID `db:"id"`
	PresetID  PresetID     `db:"preset_id"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

func (r *presetIDRepositoryImpl) Find(ctx context.Context, scope Scope, ID snowflake.ID) (PresetID, error) {
	query, args, err := r.psql.Select("preset_id").
		From("scoped_preset_ids").
		Where(squirrel.Eq{"scope": scope, "id": ID}).
		ToSql()
	if err != nil {
		return "", err
	}

	var presetID PresetID
	if err := r.db.GetContext(ctx, &presetID, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return presetID, nil
}

func (r *presetIDRepositoryImpl) Save(ctx context.Context, scope Scope, ID snowflake.ID, presetID PresetID) error {
	now := time.Now()
	query, args, err := r.psql.Insert("scoped_preset_ids").
		Columns("scope", "id", "preset_id", "created_at", "updated_at").
		Values(scope, ID, presetID, now, now).
		Suffix("ON CONFLICT(scope, id) DO UPDATE SET preset_id = ?, updated_at = ?", presetID, now).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *presetIDRepositoryImpl) Delete(ctx context.Context, scope Scope, ID snowflake.ID) error {
	query, args, err := r.psql.Delete("scoped_preset_ids").
		Where(squirrel.Eq{"scope": scope, "id": ID}).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

type MockPresetIDRepository struct {
}

func (m *MockPresetIDRepository) Find(ctx context.Context, scope Scope, ID snowflake.ID) (PresetID, error) {
	return "", ErrNotFound
}

func (m *MockPresetIDRepository) Save(ctx context.Context, scope Scope, ID snowflake.ID, presetID PresetID) error {
	return nil
}

func (m *MockPresetIDRepository) Delete(ctx context.Context, scope Scope, ID snowflake.ID) error {
	return nil
}
