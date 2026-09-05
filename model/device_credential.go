package model

type DeviceCredential struct {
	Id                 uint   `gorm:"primaryKey" json:"id"`
	PeerId             string `gorm:"size:64;uniqueIndex;not null" json:"peer_id"`
	EncryptedSecret    string `gorm:"type:text;not null" json:"-"`
	TemporaryUpdatedAt int64  `gorm:"default:0;not null" json:"temporary_updated_at"`
	PermanentUpdatedAt int64  `gorm:"default:0;not null" json:"permanent_updated_at"`
	TimeModel
}
