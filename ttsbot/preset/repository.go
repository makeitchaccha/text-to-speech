package preset

import (
	"context"
	"errors"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"gorm.io/gorm"
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

func NewPresetIDRepository(db *gorm.DB) PresetIDRepository {
	return &presetIDRepositoryImpl{
		db: db,
	}
}

type presetIDRepositoryImpl struct {
	db *gorm.DB
}

type ScopedPresetID struct {
	gorm.Model
	Scope     Scope        `gorm:"primaryKey"`
	ID        snowflake.ID `gorm:"primaryKey"`
	PresetID  PresetID     `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (r *presetIDRepositoryImpl) Find(ctx context.Context, scope Scope, ID snowflake.ID) (PresetID, error) {
	var scopedPresetID ScopedPresetID
	if err := r.db.WithContext(ctx).Where("scope = ? AND id = ?", scope, ID).First(&scopedPresetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return scopedPresetID.PresetID, nil
}

func (r *presetIDRepositoryImpl) Save(ctx context.Context, scope Scope, ID snowflake.ID, presetID PresetID) error {
	scopedPresetID := ScopedPresetID{
		Scope:    scope,
		ID:       ID,
		PresetID: presetID,
	}

	if err := r.db.WithContext(ctx).Save(&scopedPresetID).Error; err != nil {
		return err
	}
	return nil
}

func (r *presetIDRepositoryImpl) Delete(ctx context.Context, scope Scope, ID snowflake.ID) error {
	if err := r.db.WithContext(ctx).Where("scope = ? AND id = ?", scope, ID).Delete(&ScopedPresetID{}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
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
