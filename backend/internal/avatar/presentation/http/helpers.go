package http

import (
	"context"
	"errors"
	"mime"
	"minimart/internal/avatar/application"
	"minimart/internal/avatar/domain"
	"minimart/pkg/etag"
	"minimart/pkg/logger"
	"minimart/pkg/response"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const maxMultipartFormSize = 10 << 20

// clientError is an error with a message safe to return to the client.
type clientError struct{ msg string }

func (e clientError) Error() string { return e.msg }

func badRequest(msg string) error { return clientError{msg: msg} }

// isMultipart reports whether the request has a multipart/form-data Content-Type.
func isMultipart(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "multipart/form-data")
}

// parseImageUpload extracts the image file from a multipart request.
// Returns (nil, nil, nil) if no image field is present.
// The caller must invoke the returned close function after consuming the reader.
func parseImageUpload(r *http.Request) (*application.ImageUpload, func(), error) {
	file, header, err := r.FormFile("image")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil, nil
		}
		return nil, nil, badRequest("invalid image upload")
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}
	return &application.ImageUpload{
		Reader:      file,
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
	}, func() { _ = file.Close() }, nil
}

// errorStatuses maps avatar sentinel errors to HTTP statuses, in matching order.
var errorStatuses = []response.ErrorStatus{
	{Err: domain.ErrAvatarNotFound, Status: http.StatusNotFound},
	{Err: domain.ErrEmptyAccountID, Status: http.StatusBadRequest},
	{Err: domain.ErrImageRequired, Status: http.StatusBadRequest, Message: "imagem é obrigatória"},
	{Err: domain.ErrInvalidImageFormat, Status: http.StatusBadRequest},
	{Err: domain.ErrInvalidImageSize, Status: http.StatusBadRequest},
	{Err: domain.ErrImageTooLarge, Status: http.StatusBadRequest},
}

// respondDomainError maps a domain error to an HTTP response.
// It logs unexpected (default 500) errors with request context for debugging.
func respondDomainError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	if !response.KnownDomainError(err, errorStatuses) {
		logger.FromContext(ctx).Error("avatar request failed",
			"operation", "http_internal_error",
			"method", r.Method,
			"endpoint", r.Pattern,
			"error", err,
		)
	}
	response.DomainError(w, err, errorStatuses, "erro interno do servidor")
}

// avatarResponse is the JSON returned to clients.
type avatarResponse struct {
	AccountID   string    `json:"account_id"`
	ImageURL    string    `json:"image_url,omitempty"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toAvatarResponse(v application.AvatarView) avatarResponse {
	return avatarResponse{
		AccountID:   v.Avatar.AccountID,
		ImageURL:    v.URL,
		ContentType: v.Avatar.ContentType,
		SizeBytes:   v.Avatar.SizeBytes,
		Version:     v.Avatar.Version,
		CreatedAt:   v.Avatar.CreatedAt,
		UpdatedAt:   v.Avatar.UpdatedAt,
	}
}

// writeAvatar writes an avatar response with ETag.
func writeAvatar(w http.ResponseWriter, status int, v application.AvatarView) {
	w.Header().Set("ETag", etag.Format(v.Avatar.Version))
	response.WriteJSON(w, status, response.OK(toAvatarResponse(v)))
}
