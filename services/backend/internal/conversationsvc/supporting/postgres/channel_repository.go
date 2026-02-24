package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/73ai/infralayer/services/backend/internal/conversationsvc/domain"
)

func (db *BackendDB) AddChannel(ctx context.Context, teamID, channelID, channelName string) error {
	var dbChannelName sql.NullString
	if channelName != "" {
		dbChannelName = sql.NullString{String: channelName, Valid: true}
	}

	err := db.Querier.AddChannel(ctx, AddChannelParams{
		TeamID:      teamID,
		ChannelID:   channelID,
		ChannelName: dbChannelName,
	})
	if err != nil {
		return fmt.Errorf("failed to add channel: %w", err)
	}

	return nil
}

func (db *BackendDB) SetChannelMonitoring(ctx context.Context, teamID, channelID string, isMonitored bool) error {
	err := db.Querier.SetChannelMonitoring(ctx, SetChannelMonitoringParams{
		TeamID:      teamID,
		ChannelID:   channelID,
		IsMonitored: isMonitored,
	})
	if err != nil {
		return fmt.Errorf("failed to set channel monitoring: %w", err)
	}

	return nil
}

func (db *BackendDB) GetMonitoredChannels(ctx context.Context, teamID string) ([]domain.Channel, error) {
	dbChannels, err := db.Querier.GetMonitoredChannels(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitored channels: %w", err)
	}

	channels := make([]domain.Channel, len(dbChannels))
	for i, dbChannel := range dbChannels {
		channels[i] = domain.Channel{
			ChannelID:   dbChannel.ChannelID,
			TeamID:      dbChannel.TeamID,
			ChannelName: dbChannel.ChannelName.String,
			IsMonitored: dbChannel.IsMonitored,
			CreatedAt:   dbChannel.CreatedAt,
		}
	}

	return channels, nil
}

func (db *BackendDB) IsChannelMonitored(ctx context.Context, teamID, channelID string) (bool, error) {
	isMonitored, err := db.Querier.IsChannelMonitored(ctx, IsChannelMonitoredParams{
		TeamID:    teamID,
		ChannelID: channelID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("failed to check if channel is monitored: %w", err)
	}

	return isMonitored, nil
}

var _ domain.ChannelRepository = (*BackendDB)(nil)
