package model

// AgentMetricSample stores one client-reported observation for device history.
type AgentMetricSample struct {
	Id           uint    `gorm:"primaryKey" json:"id"`
	PeerId       string  `gorm:"size:64;index;not null" json:"peer_id"`
	Timestamp    int64   `gorm:"index;not null" json:"timestamp"`
	CpuUsage     float64 `gorm:"default:0;not null" json:"cpu_usage"`
	MemoryTotal  uint64  `gorm:"default:0;not null" json:"memory_total"`
	MemoryUsed   uint64  `gorm:"default:0;not null" json:"memory_used"`
	MemoryUsage  float64 `gorm:"default:0;not null" json:"memory_usage"`
	DiskUsage    string  `gorm:"type:longtext" json:"disk_usage"`
	DiskReadBps  uint64  `gorm:"default:0;not null" json:"disk_read_bps"`
	DiskWriteBps uint64  `gorm:"default:0;not null" json:"disk_write_bps"`
}
