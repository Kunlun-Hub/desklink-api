package model

type AccessRule struct {
	Id           uint   `gorm:"primaryKey" json:"id"`
	TargetPeerId string `gorm:"size:64;index;not null" json:"target_peer_id"`
	SourceIp     string `gorm:"size:128;index;default:'';not null" json:"source_ip"`
	SourcePeerId string `gorm:"size:64;index;default:'';not null" json:"source_peer_id"`
	SourceUserId uint   `gorm:"default:0;index;not null" json:"source_user_id"`
	Action       string `gorm:"size:8;not null" json:"action"`
	Priority     int    `gorm:"default:0;index;not null" json:"priority"`
	Enabled      bool   `gorm:"default:1;index;not null" json:"enabled"`
	Remark       string `gorm:"size:255;default:'';not null" json:"remark"`
	TimeModel
}

type AccessRuleList struct {
	Rules []*AccessRule `json:"list"`
	Pagination
}
