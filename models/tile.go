package models

import (
	"fmt"
	"strings"
)

type (
	Tile struct {
		Type   TileType   `json:"type"`
		Status TileStatus `json:"status"`

		Label   string `json:"label,omitempty"`
		Message string `json:"message,omitempty"`

		Metrics *TileMetrics `json:"metrics,omitempty"`
		Build   *TileBuild   `json:"build,omitempty"`
		Items   []TileItem   `json:"items,omitempty"`
	}

	// TileItem is a line of a list tile (pull requests, issues, runs ...) or a chip of a summary tile
	TileItem struct {
		Label    string     `json:"label"`
		Detail   string     `json:"detail,omitempty"`
		Status   TileStatus `json:"status,omitempty"`
		Tag      string     `json:"tag,omitempty"`
		Progress *int       `json:"progress,omitempty"` // percentage of the estimated duration, for running items
	}

	TileType   string //PING, PORT, ... (defined in usecase.go for each monitorable)
	TileStatus string
)

const (
	ActionRequiredStatus TileStatus = "ACTION_REQUIRED"
	CanceledStatus       TileStatus = "CANCELED"
	DisabledStatus       TileStatus = "DISABLED"
	FailedStatus         TileStatus = "FAILURE"
	QueuedStatus         TileStatus = "QUEUED"
	RunningStatus        TileStatus = "RUNNING"
	SuccessStatus        TileStatus = "SUCCESS"
	UnknownStatus        TileStatus = "UNKNOWN"
	WarningStatus        TileStatus = "WARNING"
)

const generatorPrefix string = "GENERATE:"

func NewTile(t TileType) *Tile {
	return &Tile{Type: t}
}

func NewGeneratorTileType(t TileType) TileType {
	tileType := fmt.Sprintf("%s%s", generatorPrefix, t)
	return TileType(tileType)
}

func (t TileType) IsGenerator() bool {
	return strings.HasPrefix(string(t), generatorPrefix)
}

func (t TileType) GetGeneratedTileType() TileType {
	return TileType(strings.TrimPrefix(string(t), generatorPrefix))
}
