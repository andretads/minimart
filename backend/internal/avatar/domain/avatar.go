package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrAvatarNotFound     = errors.New("avatar: avatar não encontrado")
	ErrEmptyAccountID     = errors.New("avatar: id da conta é obrigatório")
	ErrEmptyObjectKey     = errors.New("avatar: object key is required")
	ErrEmptyBucket        = errors.New("avatar: bucket is required")
	ErrImageRequired      = errors.New("avatar: imagem é obrigatória")
	ErrInvalidImageFormat = errors.New("avatar: formato de imagem inválido, deve ser uma imagem")
	ErrInvalidImageSize   = errors.New("avatar: tamanho de imagem inválido")
	ErrImageTooLarge      = errors.New("avatar: imagem excede o tamanho máximo permitido")
)

// Avatar is an avatar record linked to an account.
// Code outside tests should create Avatar values through NewAvatar, so
// normal application flow is valid by construction.
//
// CreatedAt is stamped once when the avatar is created and never changes.
// UpdatedAt moves whenever the storage layer persists a change.
// Version is the optimistic lock: it starts at 1 and is bumped by the storage
// layer on every persisted write. It is not set by domain logic.
type Avatar struct {
	AccountID   string
	ObjectKey   string
	Bucket      string
	ContentType string
	SizeBytes   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
}

// NewAvatar validates the inputs and returns an Avatar, or a domain error.
// accountID, objectKey, and bucket must be non-empty. contentType must be a
// supported image type. sizeBytes must be positive.
// The avatar starts with CreatedAt and UpdatedAt set to now, Version = 1.
func NewAvatar(accountID, objectKey, bucket, contentType string, sizeBytes int64) (Avatar, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return Avatar{}, ErrEmptyAccountID
	}

	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return Avatar{}, ErrEmptyObjectKey
	}

	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return Avatar{}, ErrEmptyBucket
	}

	if !IsSupportedImageType(contentType) {
		return Avatar{}, ErrInvalidImageFormat
	}

	if sizeBytes <= 0 {
		return Avatar{}, ErrInvalidImageSize
	}

	now := time.Now().UTC()
	return Avatar{
		AccountID:   accountID,
		ObjectKey:   objectKey,
		Bucket:      bucket,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}, nil
}

// IsSupportedImageType reports whether the content type is a supported image format.
func IsSupportedImageType(contentType string) bool {
	return contentType == "image/jpeg" ||
		contentType == "image/png" ||
		contentType == "image/webp" ||
		contentType == "image/gif"
}
