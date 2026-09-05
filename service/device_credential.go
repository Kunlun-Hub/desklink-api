package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const maxManagedPasswordLength = 128

type DeviceCredentialSecrets struct {
	TemporaryPassword string `json:"temporary_password"`
	PermanentPassword string `json:"permanent_password"`
}

type DeviceCredentialStatus struct {
	HasTemporary       bool  `json:"has_temporary"`
	HasPermanent       bool  `json:"has_permanent"`
	TemporaryUpdatedAt int64 `json:"temporary_updated_at"`
	PermanentUpdatedAt int64 `json:"permanent_updated_at"`
}

func deviceCredentialCipher() (cipher.AEAD, error) {
	if Config == nil || strings.TrimSpace(Config.Jwt.Key) == "" {
		return nil, errors.New("JWT key is required to encrypt device credentials")
	}
	key := sha256.Sum256([]byte(Config.Jwt.Key + "\x00device-credentials"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptDeviceCredentialSecrets(secrets DeviceCredentialSecrets) (string, error) {
	aead, err := deviceCredentialCipher()
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
	return base64.RawStdEncoding.EncodeToString(aead.Seal(nonce, nonce, plain, nil)), nil
}

func decryptDeviceCredentialSecrets(value string) (DeviceCredentialSecrets, error) {
	if value == "" {
		return DeviceCredentialSecrets{}, nil
	}
	aead, err := deviceCredentialCipher()
	if err != nil {
		return DeviceCredentialSecrets{}, err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(sealed) < aead.NonceSize() {
		return DeviceCredentialSecrets{}, errors.New("invalid encrypted device credentials")
	}
	plain, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return DeviceCredentialSecrets{}, errors.New("cannot decrypt device credentials")
	}
	var secrets DeviceCredentialSecrets
	if err = json.Unmarshal(plain, &secrets); err != nil {
		return DeviceCredentialSecrets{}, err
	}
	return secrets, nil
}

func UpsertDeviceCredentials(peerID string, temporary, permanent *string) error {
	if temporary == nil && permanent == nil {
		return errors.New("at least one credential is required")
	}
	if temporary != nil && len(*temporary) > maxManagedPasswordLength || permanent != nil && len(*permanent) > maxManagedPasswordLength {
		return errors.New("credential is too long")
	}
	var credential model.DeviceCredential
	result := DB.Where("peer_id = ?", peerID).First(&credential)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	secrets, err := decryptDeviceCredentialSecrets(credential.EncryptedSecret)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	if temporary != nil {
		secrets.TemporaryPassword = *temporary
		credential.TemporaryUpdatedAt = now
	}
	if permanent != nil {
		secrets.PermanentPassword = *permanent
		credential.PermanentUpdatedAt = now
	}
	credential.PeerId = peerID
	credential.EncryptedSecret, err = encryptDeviceCredentialSecrets(secrets)
	if err != nil {
		return err
	}
	return DB.Save(&credential).Error
}

func DeviceCredentials(peerID string) (DeviceCredentialSecrets, DeviceCredentialStatus, error) {
	var credential model.DeviceCredential
	if err := DB.Where("peer_id = ?", peerID).First(&credential).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeviceCredentialSecrets{}, DeviceCredentialStatus{}, nil
		}
		return DeviceCredentialSecrets{}, DeviceCredentialStatus{}, err
	}
	secrets, err := decryptDeviceCredentialSecrets(credential.EncryptedSecret)
	if err != nil {
		return DeviceCredentialSecrets{}, DeviceCredentialStatus{}, err
	}
	return secrets, DeviceCredentialStatus{
		HasTemporary: secrets.TemporaryPassword != "", HasPermanent: secrets.PermanentPassword != "",
		TemporaryUpdatedAt: credential.TemporaryUpdatedAt, PermanentUpdatedAt: credential.PermanentUpdatedAt,
	}, nil
}
