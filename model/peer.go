package model

type Peer struct {
	RowId          uint    `json:"row_id" gorm:"primaryKey;"`
	Id             string  `json:"id"  gorm:"default:'';not null;index"`
	Cpu            string  `json:"cpu"  gorm:"default:'';not null;"`
	Hostname       string  `json:"hostname"  gorm:"default:'';not null;"`
	Memory         string  `json:"memory"  gorm:"default:'';not null;"`
	Os             string  `json:"os"  gorm:"default:'';not null;"`
	Username       string  `json:"username"  gorm:"default:'';not null;"`
	Uuid           string  `json:"uuid"  gorm:"default:'';not null;index"`
	Version        string  `json:"version"  gorm:"default:'';not null;"`
	UserId         uint    `json:"user_id"  gorm:"default:0;not null;index"`
	User           *User   `json:"user,omitempty"`
	LastOnlineTime int64   `json:"last_online_time"  gorm:"default:0;not null;"`
	LastOnlineIp   string  `json:"last_online_ip"  gorm:"default:'';not null;"`
	Online         bool    `json:"online" gorm:"-"`
	GroupId        uint    `json:"group_id"  gorm:"default:0;not null;index"`
	Alias          string  `json:"alias" gorm:"default:'';not null;index"`
	CpuUsage       float64 `json:"cpu_usage" gorm:"default:0;not null"`
	MemoryTotal    uint64  `json:"memory_total" gorm:"default:0;not null"`
	MemoryUsed     uint64  `json:"memory_used" gorm:"default:0;not null"`
	MemoryUsage    float64 `json:"memory_usage" gorm:"default:0;not null"`
	DiskUsage      string  `json:"disk_usage" gorm:"type:longtext"`
	DiskReadBps    uint64  `json:"disk_read_bps" gorm:"default:0;not null"`
	DiskWriteBps   uint64  `json:"disk_write_bps" gorm:"default:0;not null"`
	MetricsAt      int64   `json:"metrics_at" gorm:"default:0;not null"`
	TimeModel
}

type PeerList struct {
	Peers []*Peer `json:"list"`
	Pagination
}
