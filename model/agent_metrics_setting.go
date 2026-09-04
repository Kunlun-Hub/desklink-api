package model

type AgentMetricsSetting struct {
	Id              uint `gorm:"primaryKey" json:"id"`
	IntervalSeconds int  `gorm:"default:5;not null" json:"interval_seconds"`
	TimeModel
}
