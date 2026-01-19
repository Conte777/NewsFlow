package telegram

import (
	"context"

	"github.com/gotd/td/telegram/updates"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TelegramUpdatesState represents the state of Telegram updates for a user
type TelegramUpdatesState struct {
	UserID int64 `gorm:"primaryKey;column:user_id"`
	Pts    int   `gorm:"column:pts;default:0"`
	Qts    int   `gorm:"column:qts;default:0"`
	Date   int   `gorm:"column:date;default:0"`
	Seq    int   `gorm:"column:seq;default:0"`
}

func (TelegramUpdatesState) TableName() string {
	return "telegram_updates_state"
}

// TelegramChannelState represents the pts state for a specific channel
type TelegramChannelState struct {
	UserID     int64 `gorm:"primaryKey;column:user_id"`
	ChannelID  int64 `gorm:"primaryKey;column:channel_id"`
	Pts        int   `gorm:"column:pts;default:0"`
	AccessHash int64 `gorm:"column:access_hash"`
}

func (TelegramChannelState) TableName() string {
	return "telegram_channel_state"
}

// UpdatesStateStorage implements updates.StateStorage interface using PostgreSQL
type UpdatesStateStorage struct {
	db     *gorm.DB
	logger zerolog.Logger
}

// NewUpdatesStateStorage creates a new PostgreSQL-based state storage
func NewUpdatesStateStorage(db *gorm.DB, logger zerolog.Logger) *UpdatesStateStorage {
	return &UpdatesStateStorage{
		db:     db,
		logger: logger.With().Str("component", "updates_state_storage").Logger(),
	}
}

// GetState retrieves the updates state for a user
func (s *UpdatesStateStorage) GetState(ctx context.Context, userID int64) (updates.State, bool, error) {
	var state TelegramUpdatesState

	result := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&state)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			s.logger.Debug().Int64("user_id", userID).Msg("state not found")
			return updates.State{}, false, nil
		}
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to get state")
		return updates.State{}, false, result.Error
	}

	s.logger.Debug().
		Int64("user_id", userID).
		Int("pts", state.Pts).
		Int("qts", state.Qts).
		Int("date", state.Date).
		Int("seq", state.Seq).
		Msg("state retrieved")

	return updates.State{
		Pts:  state.Pts,
		Qts:  state.Qts,
		Date: state.Date,
		Seq:  state.Seq,
	}, true, nil
}

// SetState saves the complete updates state for a user
func (s *UpdatesStateStorage) SetState(ctx context.Context, userID int64, state updates.State) error {
	record := TelegramUpdatesState{
		UserID: userID,
		Pts:    state.Pts,
		Qts:    state.Qts,
		Date:   state.Date,
		Seq:    state.Seq,
	}

	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pts", "qts", "date", "seq"}),
	}).Create(&record)

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set state")
		return result.Error
	}

	s.logger.Debug().
		Int64("user_id", userID).
		Int("pts", state.Pts).
		Int("qts", state.Qts).
		Int("date", state.Date).
		Int("seq", state.Seq).
		Msg("state saved")

	return nil
}

// SetPts updates only the pts value for a user
func (s *UpdatesStateStorage) SetPts(ctx context.Context, userID int64, pts int) error {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pts"}),
	}).Create(&TelegramUpdatesState{UserID: userID, Pts: pts})

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set pts")
		return result.Error
	}
	return nil
}

// SetQts updates only the qts value for a user
func (s *UpdatesStateStorage) SetQts(ctx context.Context, userID int64, qts int) error {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"qts"}),
	}).Create(&TelegramUpdatesState{UserID: userID, Qts: qts})

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set qts")
		return result.Error
	}
	return nil
}

// SetDate updates only the date value for a user
func (s *UpdatesStateStorage) SetDate(ctx context.Context, userID int64, date int) error {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"date"}),
	}).Create(&TelegramUpdatesState{UserID: userID, Date: date})

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set date")
		return result.Error
	}
	return nil
}

// SetSeq updates only the seq value for a user
func (s *UpdatesStateStorage) SetSeq(ctx context.Context, userID int64, seq int) error {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"seq"}),
	}).Create(&TelegramUpdatesState{UserID: userID, Seq: seq})

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set seq")
		return result.Error
	}
	return nil
}

// SetDateSeq updates both date and seq values for a user
func (s *UpdatesStateStorage) SetDateSeq(ctx context.Context, userID int64, date, seq int) error {
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"date", "seq"}),
	}).Create(&TelegramUpdatesState{UserID: userID, Date: date, Seq: seq})

	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to set date and seq")
		return result.Error
	}
	return nil
}

// GetChannelPts retrieves the pts value for a specific channel
func (s *UpdatesStateStorage) GetChannelPts(ctx context.Context, userID, channelID int64) (int, bool, error) {
	var state TelegramChannelState

	result := s.db.WithContext(ctx).
		Where("user_id = ? AND channel_id = ?", userID, channelID).
		First(&state)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return 0, false, nil
		}
		s.logger.Error().Err(result.Error).
			Int64("user_id", userID).
			Int64("channel_id", channelID).
			Msg("failed to get channel pts")
		return 0, false, result.Error
	}

	return state.Pts, true, nil
}

// SetChannelPts saves the pts value for a specific channel
func (s *UpdatesStateStorage) SetChannelPts(ctx context.Context, userID, channelID int64, pts int) error {
	record := TelegramChannelState{
		UserID:    userID,
		ChannelID: channelID,
		Pts:       pts,
	}

	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pts"}),
	}).Create(&record)

	if result.Error != nil {
		s.logger.Error().Err(result.Error).
			Int64("user_id", userID).
			Int64("channel_id", channelID).
			Int("pts", pts).
			Msg("failed to set channel pts")
		return result.Error
	}

	s.logger.Debug().
		Int64("user_id", userID).
		Int64("channel_id", channelID).
		Int("pts", pts).
		Msg("channel pts saved")

	return nil
}

// ForEachChannels iterates over all channels for a user and calls the provided function
func (s *UpdatesStateStorage) ForEachChannels(ctx context.Context, userID int64, f func(ctx context.Context, channelID int64, pts int) error) error {
	var states []TelegramChannelState

	result := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&states)
	if result.Error != nil {
		s.logger.Error().Err(result.Error).Int64("user_id", userID).Msg("failed to get channels")
		return result.Error
	}

	for _, state := range states {
		if err := f(ctx, state.ChannelID, state.Pts); err != nil {
			return err
		}
	}

	return nil
}

// Ensure UpdatesStateStorage implements updates.StateStorage interface
var _ updates.StateStorage = (*UpdatesStateStorage)(nil)
