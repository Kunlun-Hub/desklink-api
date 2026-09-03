package model

const (
	RecordingModeOff      = "off"
	RecordingModeAll      = "all"
	RecordingModeSelected = "selected"

	RecordingStatusUploading   = "uploading"
	RecordingStatusTranscoding = "transcoding"
	RecordingStatusComplete    = "complete"
	RecordingStatusFailed      = "failed"
)

type RecordingPolicy struct {
	Id            uint   `gorm:"primaryKey" json:"id"`
	Mode          string `gorm:"size:16;default:'off';not null" json:"mode"`
	RetentionDays int    `gorm:"default:30;not null" json:"retention_days"`
	TimeModel
}

type RecordingPolicyDevice struct {
	Id     uint   `gorm:"primaryKey" json:"id"`
	PeerId string `gorm:"size:64;uniqueIndex;not null" json:"peer_id"`
	TimeModel
}

type SessionRecording struct {
	Id                 uint   `gorm:"primaryKey" json:"id"`
	UploadId           string `gorm:"size:36;uniqueIndex;not null" json:"upload_id"`
	UploadTokenHash    string `gorm:"size:64;not null" json:"-"`
	PeerId             string `gorm:"size:64;index;not null" json:"peer_id"`
	FromPeer           string `gorm:"size:64;index;default:'';not null" json:"from_peer"`
	FromName           string `gorm:"size:255;index;default:'';not null" json:"from_name"`
	SessionId          string `gorm:"size:128;index;default:'';not null" json:"session_id"`
	OriginalName       string `gorm:"size:255;not null" json:"original_name"`
	StorageName        string `gorm:"size:255;uniqueIndex;not null" json:"-"`
	StorageBackend     string `gorm:"size:16;default:'local';not null;index" json:"storage_backend"`
	StorageSettingId   uint   `gorm:"default:0;not null;index" json:"-"`
	PreviewStorageName string `gorm:"size:255;default:'';not null" json:"-"`
	CursorStorageName  string `gorm:"size:255;default:'';not null" json:"-"`
	CursorRenderStatus string `gorm:"size:16;default:'';not null" json:"cursor_render_status"`
	CursorRenderError  string `gorm:"size:512;default:'';not null" json:"cursor_render_error"`
	Container          string `gorm:"size:16;not null" json:"container"`
	Codec              string `gorm:"size:16;default:'';not null" json:"codec"`
	Status             string `gorm:"size:16;index;not null" json:"status"`
	Size               int64  `gorm:"default:0;not null" json:"size"`
	DurationMs         int64  `gorm:"default:0;not null" json:"duration_ms"`
	StartedAt          int64  `gorm:"default:0;not null;index" json:"started_at"`
	CompletedAt        int64  `gorm:"default:0;not null" json:"completed_at"`
	Sha256             string `gorm:"size:64;default:'';not null" json:"sha256"`
	ErrorMessage       string `gorm:"size:512;default:'';not null" json:"error_message"`
	CursorTrack        string `gorm:"type:longtext" json:"-"`
	CursorAvailable    bool   `gorm:"-" json:"cursor_available"`
	TimeModel
}

type RecordingStorageSetting struct {
	Id              uint   `gorm:"primaryKey" json:"id"`
	Backend         string `gorm:"size:16;default:'local';not null;index" json:"backend"`
	Active          bool   `gorm:"default:0;not null;index" json:"active"`
	Path            string `gorm:"size:1024;default:'';not null" json:"path"`
	Endpoint        string `gorm:"size:512;default:'';not null" json:"endpoint"`
	Bucket          string `gorm:"size:255;default:'';not null" json:"bucket"`
	Region          string `gorm:"size:128;default:'';not null" json:"region"`
	AccessKey       string `gorm:"size:255;default:'';not null" json:"access_key"`
	Username        string `gorm:"size:255;default:'';not null" json:"username"`
	Prefix          string `gorm:"size:512;default:'';not null" json:"prefix"`
	Secure          bool   `gorm:"default:0;not null" json:"secure"`
	EncryptedSecret string `gorm:"type:text" json:"-"`
	TimeModel
}

type SessionRecordingList struct {
	Recordings []*SessionRecording `json:"list"`
	Pagination
}
