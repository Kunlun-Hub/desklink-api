package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
)

const (
	recordingStorageLocal = "local"
	recordingStorageFTP   = "ftp"
	recordingStorageNFS   = "nfs"
	recordingStorageSMB   = "smb"
	recordingStorageS3    = "s3"
)

type RecordingStorageConfig struct {
	Backend      string `json:"backend"`
	Path         string `json:"path"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	AccessKey    string `json:"access_key"`
	SecretKey    string `json:"secret_key,omitempty"`
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`
	Prefix       string `json:"prefix"`
	Secure       bool   `json:"secure"`
	HasSecretKey bool   `json:"has_secret_key"`
	HasPassword  bool   `json:"has_password"`
}

type recordingStorageSecrets struct {
	SecretKey string `json:"secret_key"`
	Password  string `json:"password"`
}

type recordingObjectStore interface {
	Archive(context.Context, string, string) error
	Materialize(context.Context, string) (string, func(), error)
	Delete(context.Context, string) error
	Check(context.Context) error
}

func normalizeRecordingStorageConfig(in RecordingStorageConfig) (RecordingStorageConfig, error) {
	in.Backend = strings.ToLower(strings.TrimSpace(in.Backend))
	in.Path = strings.TrimSpace(in.Path)
	in.Endpoint = strings.TrimSpace(in.Endpoint)
	in.Bucket = strings.TrimSpace(in.Bucket)
	in.Region = strings.TrimSpace(in.Region)
	in.AccessKey = strings.TrimSpace(in.AccessKey)
	in.Username = strings.TrimSpace(in.Username)
	in.Prefix = strings.Trim(strings.TrimSpace(in.Prefix), "/")
	if strings.Contains(in.Prefix, "..") {
		return in, errors.New("storage prefix cannot contain '..'")
	}
	switch in.Backend {
	case recordingStorageLocal:
		if in.Path == "" {
			in.Path = strings.TrimSpace(Config.Recording.Path)
			if in.Path == "" {
				in.Path = "runtime/recordings"
			}
		}
	case recordingStorageNFS, recordingStorageSMB:
		if !filepath.IsAbs(in.Path) {
			return in, errors.New("NFS/SMB mount path must be absolute")
		}
	case recordingStorageFTP:
		if in.Endpoint == "" {
			return in, errors.New("FTP endpoint is required")
		}
		if _, _, err := net.SplitHostPort(in.Endpoint); err != nil {
			return in, errors.New("FTP endpoint must include host and port")
		}
	case recordingStorageS3:
		if in.Endpoint == "" || in.Bucket == "" || in.AccessKey == "" {
			return in, errors.New("S3 endpoint, bucket and access key are required")
		}
	default:
		return in, errors.New("unsupported recording storage backend")
	}
	return in, nil
}

func recordingStorageCipher() (cipher.AEAD, error) {
	if strings.TrimSpace(Config.Jwt.Key) == "" {
		return nil, errors.New("JWT key is required to encrypt recording storage credentials")
	}
	key := sha256.Sum256([]byte(Config.Jwt.Key))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptRecordingStorageSecrets(secrets recordingStorageSecrets) (string, error) {
	if secrets.SecretKey == "" && secrets.Password == "" {
		return "", nil
	}
	aead, err := recordingStorageCipher()
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(secrets)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plain, nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func decryptRecordingStorageSecrets(value string) (recordingStorageSecrets, error) {
	if value == "" {
		return recordingStorageSecrets{}, nil
	}
	aead, err := recordingStorageCipher()
	if err != nil {
		return recordingStorageSecrets{}, err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(sealed) < aead.NonceSize() {
		return recordingStorageSecrets{}, errors.New("invalid encrypted recording storage credentials")
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return recordingStorageSecrets{}, errors.New("cannot decrypt recording storage credentials")
	}
	var secrets recordingStorageSecrets
	if err = json.Unmarshal(plain, &secrets); err != nil {
		return recordingStorageSecrets{}, err
	}
	return secrets, nil
}

func recordingStorageConfigFromModel(setting *model.RecordingStorageSetting) (RecordingStorageConfig, error) {
	secrets, err := decryptRecordingStorageSecrets(setting.EncryptedSecret)
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	return RecordingStorageConfig{
		Backend: setting.Backend, Path: setting.Path, Endpoint: setting.Endpoint,
		Bucket: setting.Bucket, Region: setting.Region, AccessKey: setting.AccessKey,
		SecretKey: secrets.SecretKey, Username: setting.Username, Password: secrets.Password,
		Prefix: setting.Prefix, Secure: setting.Secure,
		HasSecretKey: secrets.SecretKey != "", HasPassword: secrets.Password != "",
	}, nil
}

func defaultRecordingStorageConfig() RecordingStorageConfig {
	value := RecordingStorageConfig{Backend: recordingStorageLocal, Path: strings.TrimSpace(Config.Recording.Path)}
	if value.Path == "" {
		value.Path = "runtime/recordings"
	}
	return value
}

func (s *RecordingService) StorageConfig() (RecordingStorageConfig, error) {
	setting := &model.RecordingStorageSetting{}
	err := DB.Where("active = ?", true).First(setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultRecordingStorageConfig(), nil
	}
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	config, err := recordingStorageConfigFromModel(setting)
	config.SecretKey, config.Password = "", ""
	return config, err
}

func (s *RecordingService) activeStorageConfig() (uint, RecordingStorageConfig, error) {
	setting := &model.RecordingStorageSetting{}
	if err := DB.Where("active = ?", true).First(setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, defaultRecordingStorageConfig(), nil
		}
		return 0, RecordingStorageConfig{}, err
	}
	config, err := recordingStorageConfigFromModel(setting)
	return setting.Id, config, err
}

func (s *RecordingService) SaveStorageConfig(in RecordingStorageConfig) (RecordingStorageConfig, error) {
	in, err := normalizeRecordingStorageConfig(in)
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	existing := &model.RecordingStorageSetting{}
	findErr := DB.Where("backend = ?", in.Backend).Order("id desc").First(existing).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return RecordingStorageConfig{}, findErr
	}
	if existing.Id > 0 {
		secrets, secretErr := decryptRecordingStorageSecrets(existing.EncryptedSecret)
		if secretErr != nil {
			return RecordingStorageConfig{}, secretErr
		}
		if in.SecretKey == "" {
			in.SecretKey = secrets.SecretKey
		}
		if in.Password == "" {
			in.Password = secrets.Password
		}
	}
	store, err := s.newRecordingObjectStore(in)
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = store.Check(ctx); err != nil {
		return RecordingStorageConfig{}, fmt.Errorf("storage connection test failed: %w", err)
	}
	encrypted, err := encryptRecordingStorageSecrets(recordingStorageSecrets{SecretKey: in.SecretKey, Password: in.Password})
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	setting := &model.RecordingStorageSetting{
		Backend: in.Backend, Active: true, Path: in.Path,
		Endpoint: in.Endpoint, Bucket: in.Bucket, Region: in.Region,
		AccessKey: in.AccessKey, Username: in.Username, Prefix: in.Prefix,
		Secure: in.Secure, EncryptedSecret: encrypted,
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.RecordingStorageSetting{}).Where("active = ?", true).Update("active", false).Error; err != nil {
			return err
		}
		return tx.Save(setting).Error
	})
	if err != nil {
		return RecordingStorageConfig{}, err
	}
	in.HasSecretKey, in.HasPassword = in.SecretKey != "", in.Password != ""
	in.SecretKey, in.Password = "", ""
	return in, nil
}

func storageObjectName(prefix, name string) (string, error) {
	if name == "" || filepath.Base(name) != name || strings.Contains(name, "..") {
		return "", errors.New("invalid recording storage name")
	}
	if prefix == "" {
		return name, nil
	}
	return path.Join(prefix, name), nil
}

func (s *RecordingService) newRecordingObjectStore(config RecordingStorageConfig) (recordingObjectStore, error) {
	config, err := normalizeRecordingStorageConfig(config)
	if err != nil {
		return nil, err
	}
	switch config.Backend {
	case recordingStorageLocal, recordingStorageNFS, recordingStorageSMB:
		return &filesystemRecordingStore{root: config.Path, prefix: config.Prefix}, nil
	case recordingStorageFTP:
		return &ftpRecordingStore{config: config, tempPath: s.storagePath()}, nil
	case recordingStorageS3:
		client, err := minio.New(config.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
			Secure: config.Secure, Region: config.Region,
		})
		if err != nil {
			return nil, err
		}
		return &s3RecordingStore{client: client, config: config, tempPath: s.storagePath()}, nil
	default:
		return nil, errors.New("unsupported recording storage backend")
	}
}

type filesystemRecordingStore struct {
	root   string
	prefix string
}

func (s *filesystemRecordingStore) objectPath(name string) (string, error) {
	object, err := storageObjectName(s.prefix, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(object)), nil
}

func (s *filesystemRecordingStore) Archive(_ context.Context, source, name string) error {
	destination, err := s.objectPath(name)
	if err != nil {
		return err
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return nil
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary := destination + ".uploading"
	output, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	return os.Rename(temporary, destination)
}

func (s *filesystemRecordingStore) Materialize(_ context.Context, name string) (string, func(), error) {
	value, err := s.objectPath(name)
	return value, func() {}, err
}

func (s *filesystemRecordingStore) Delete(_ context.Context, name string) error {
	value, err := s.objectPath(name)
	if err != nil {
		return err
	}
	if err = os.Remove(value); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *filesystemRecordingStore) Check(_ context.Context) error {
	if err := os.MkdirAll(filepath.Join(s.root, filepath.FromSlash(s.prefix)), 0750); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Join(s.root, filepath.FromSlash(s.prefix)), ".desklink-storage-check-")
	if err != nil {
		return err
	}
	name := file.Name()
	if err = file.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

type ftpRecordingStore struct {
	config   RecordingStorageConfig
	tempPath string
}

func (s *ftpRecordingStore) connect() (*ftp.ServerConn, error) {
	options := []ftp.DialOption{ftp.DialWithTimeout(10 * time.Second)}
	if s.config.Secure {
		host, _, _ := net.SplitHostPort(s.config.Endpoint)
		options = append(options, ftp.DialWithTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}))
	}
	conn, err := ftp.Dial(s.config.Endpoint, options...)
	if err != nil {
		return nil, err
	}
	if err = conn.Login(s.config.Username, s.config.Password); err != nil {
		_ = conn.Quit()
		return nil, err
	}
	return conn, nil
}

func ensureFTPDirectories(conn *ftp.ServerConn, object string) error {
	directory := path.Dir(object)
	if directory == "." || directory == "/" {
		return nil
	}
	current := ""
	for _, part := range strings.Split(strings.Trim(directory, "/"), "/") {
		current = path.Join(current, part)
		if err := conn.MakeDir(current); err != nil {
			if _, changeErr := conn.List(current); changeErr != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ftpRecordingStore) Archive(_ context.Context, source, name string) error {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return err
	}
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	if err = ensureFTPDirectories(conn, object); err != nil {
		return err
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	return conn.Stor(object, file)
}

func materializeRecording(tempPath, name string, write func(io.Writer) error) (string, func(), error) {
	if err := os.MkdirAll(tempPath, 0750); err != nil {
		return "", func() {}, err
	}
	file, err := os.CreateTemp(tempPath, ".recording-read-*")
	if err != nil {
		return "", func() {}, err
	}
	value := file.Name()
	cleanup := func() { _ = os.Remove(value) }
	if err = write(file); err != nil {
		_ = file.Close()
		cleanup()
		return "", func() {}, err
	}
	if err = file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return value, cleanup, nil
}

func (s *ftpRecordingStore) Materialize(_ context.Context, name string) (string, func(), error) {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return "", func() {}, err
	}
	conn, err := s.connect()
	if err != nil {
		return "", func() {}, err
	}
	return materializeRecording(s.tempPath, name, func(output io.Writer) error {
		defer conn.Quit()
		response, err := conn.Retr(object)
		if err != nil {
			return err
		}
		defer response.Close()
		_, err = io.Copy(output, response)
		return err
	})
}

func (s *ftpRecordingStore) Delete(_ context.Context, name string) error {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return err
	}
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	err = conn.Delete(object)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") && !strings.Contains(err.Error(), "550") {
		return err
	}
	return nil
}

func (s *ftpRecordingStore) Check(_ context.Context) error {
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Quit()
	name, err := storageObjectName(s.config.Prefix, ".desklink-storage-check")
	if err != nil {
		return err
	}
	if err = ensureFTPDirectories(conn, name); err != nil {
		return err
	}
	if err = conn.Stor(name, bytes.NewReader([]byte("desklink"))); err != nil {
		return err
	}
	return conn.Delete(name)
}

type s3RecordingStore struct {
	client   *minio.Client
	config   RecordingStorageConfig
	tempPath string
}

func (s *s3RecordingStore) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.config.Bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.config.Bucket, minio.MakeBucketOptions{Region: s.config.Region})
}

func (s *s3RecordingStore) Archive(ctx context.Context, source, name string) error {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return err
	}
	if err = s.ensureBucket(ctx); err != nil {
		return err
	}
	_, err = s.client.FPutObject(ctx, s.config.Bucket, object, source, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (s *s3RecordingStore) Materialize(ctx context.Context, name string) (string, func(), error) {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return "", func() {}, err
	}
	return materializeRecording(s.tempPath, name, func(output io.Writer) error {
		value, err := s.client.GetObject(ctx, s.config.Bucket, object, minio.GetObjectOptions{})
		if err != nil {
			return err
		}
		defer value.Close()
		_, err = io.Copy(output, value)
		return err
	})
}

func (s *s3RecordingStore) Delete(ctx context.Context, name string) error {
	object, err := storageObjectName(s.config.Prefix, name)
	if err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.config.Bucket, object, minio.RemoveObjectOptions{})
}

func (s *s3RecordingStore) Check(ctx context.Context) error {
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}
	name, err := storageObjectName(s.config.Prefix, ".desklink-storage-check")
	if err != nil {
		return err
	}
	if _, err = s.client.PutObject(ctx, s.config.Bucket, name, bytes.NewReader([]byte("desklink")), 8, minio.PutObjectOptions{}); err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.config.Bucket, name, minio.RemoveObjectOptions{})
}

func (s *RecordingService) objectStoreForRecording(recording *model.SessionRecording) (recordingObjectStore, RecordingStorageConfig, error) {
	config := defaultRecordingStorageConfig()
	var err error
	if recording.StorageSettingId > 0 {
		setting := &model.RecordingStorageSetting{}
		if err = DB.First(setting, recording.StorageSettingId).Error; err != nil {
			return nil, RecordingStorageConfig{}, err
		}
		config, err = recordingStorageConfigFromModel(setting)
	} else if recording.StorageBackend != "" && recording.StorageBackend != recordingStorageLocal {
		setting := &model.RecordingStorageSetting{}
		if err = DB.Where("backend = ?", recording.StorageBackend).Order("id asc").First(setting).Error; err != nil {
			return nil, RecordingStorageConfig{}, err
		}
		config, err = recordingStorageConfigFromModel(setting)
	}
	if err != nil {
		return nil, RecordingStorageConfig{}, err
	}
	store, err := s.newRecordingObjectStore(config)
	return store, config, err
}

func (s *RecordingService) archiveRecordingObject(recording *model.SessionRecording, name, source string) error {
	store, _, err := s.objectStoreForRecording(recording)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	return store.Archive(ctx, source, name)
}

func (s *RecordingService) removeStagedRecordingObject(recording *model.SessionRecording, name string) {
	_, config, err := s.objectStoreForRecording(recording)
	if err != nil {
		return
	}
	if config.Backend == recordingStorageLocal || config.Backend == recordingStorageNFS || config.Backend == recordingStorageSMB {
		store := &filesystemRecordingStore{root: config.Path, prefix: config.Prefix}
		storedPath, pathErr := store.objectPath(name)
		if pathErr == nil && filepath.Clean(storedPath) == filepath.Clean(filepath.Join(s.storagePath(), name)) {
			return
		}
	}
	_ = os.Remove(filepath.Join(s.storagePath(), name))
}

func (s *RecordingService) MaterializeRecordingObject(recording *model.SessionRecording, preview bool) (string, func(), error) {
	name := recording.StorageName
	if preview && recording.PreviewStorageName != "" {
		name = recording.PreviewStorageName
	}
	store, _, err := s.objectStoreForRecording(recording)
	if err != nil {
		return "", func() {}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	value, cleanup, err := store.Materialize(ctx, name)
	if err != nil {
		cancel()
		return "", func() {}, err
	}
	return value, func() {
		cleanup()
		cancel()
	}, nil
}

func (s *RecordingService) deleteRecordingObject(recording *model.SessionRecording, name string) error {
	store, _, err := s.objectStoreForRecording(recording)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return store.Delete(ctx, name)
}
