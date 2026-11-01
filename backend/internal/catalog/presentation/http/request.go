package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"minimart/internal/catalog/application"
	"minimart/internal/catalog/domain"
	"minimart/pkg/etag"
	"minimart/pkg/response"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// clientError is an error with a message safe to return to the client.
type clientError struct{ msg string }

func (e clientError) Error() string { return e.msg }

func badRequest(msg string) error { return clientError{msg: msg} }

const maxMultipartFormSize = 10 << 20

// isMultipart reports whether the request has a multipart/form-data Content-Type.
func isMultipart(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.HasPrefix(ct, "multipart/form-data")
}

// parseFormInt extracts an integer form value, returning a client error if invalid.
func parseFormInt(r *http.Request, key string) (int64, error) {
	v := r.FormValue(key)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, badRequest(key + " must be an integer")
	}
	return n, nil
}

// parseFormIntRequired extracts a required integer form value.
func parseFormIntRequired(r *http.Request, key string) (int64, error) {
	v := r.FormValue(key)
	if v == "" {
		return 0, badRequest(key + " is required")
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, badRequest(key + " must be an integer")
	}
	return n, nil
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

// decodeJSONBody decodes a JSON request body into the provided pointer.
func decodeJSONBody(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return badRequest("corpo JSON inválido")
	}
	return nil
}

// versionFromIfMatch parses the If-Match header and returns the version.
// Returns 0 if the header is absent.
func versionFromIfMatch(r *http.Request) (int64, error) {
	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		return 0, nil
	}
	version, wildcard, ok := etag.Parse(ifMatch)
	if !ok || wildcard {
		return 0, badRequest("cabeçalho If-Match inválido")
	}
	return int64(version), nil
}

// parseMultipart parses the multipart form with a size limit.
func parseMultipart(r *http.Request) error {
	if err := r.ParseMultipartForm(maxMultipartFormSize); err != nil { //nolint:gosec // maxMultipartFormSize is a bounded constant
		return badRequest("formulário multipart inválido")
	}
	return nil
}

// decodeCreateRequest decodes either multipart/form-data or JSON for product creation.
// Images are managed through the dedicated product image endpoints.
func decodeCreateRequest(r *http.Request) (createProductRequest, error) {
	if isMultipart(r) {
		if err := parseMultipart(r); err != nil {
			return createProductRequest{}, err
		}
		priceCents, err := parseFormInt(r, "price_cents")
		if err != nil {
			return createProductRequest{}, err
		}
		stock, err := parseFormInt(r, "stock")
		if err != nil {
			return createProductRequest{}, err
		}
		details := r.FormValue("details")
		var detailsList []domain.ProductDetail
		if details != "" {
			if err := json.Unmarshal([]byte(details), &detailsList); err != nil {
				return createProductRequest{}, fmt.Errorf("decode details: %w", err)
			}
		}
		return createProductRequest{
			Name:        r.FormValue("name"),
			PriceCents:  priceCents,
			Stock:       int(stock),
			CategoryID:  r.FormValue("category_id"),
			SKU:         r.FormValue("sku"),
			Description: r.FormValue("description"),
			Details:     detailsList,
		}, nil
	}
	var req createProductRequest
	if err := decodeJSONBody(r, &req); err != nil {
		return createProductRequest{}, err
	}
	return req, nil
}

// decodeUpdateRequest decodes either multipart/form-data or JSON for product update.
// Images are managed through the dedicated product image endpoints.
func decodeUpdateRequest(r *http.Request) (updateProductRequest, error) {
	if isMultipart(r) {
		if err := parseMultipart(r); err != nil {
			return updateProductRequest{}, err
		}
		priceCents, err := parseFormInt(r, "price_cents")
		if err != nil {
			return updateProductRequest{}, err
		}
		version, err := parseFormIntRequired(r, "version")
		if err != nil {
			return updateProductRequest{}, err
		}
		details := r.FormValue("details")
		var detailsList []domain.ProductDetail
		if details != "" {
			if uerr := json.Unmarshal([]byte(details), &detailsList); uerr != nil {
				return updateProductRequest{}, fmt.Errorf("decode details: %w", uerr)
			}
		}
		ifMatchVersion, err := versionFromIfMatch(r)
		if err != nil {
			return updateProductRequest{}, err
		}
		if ifMatchVersion > 0 {
			version = ifMatchVersion
		}
		return updateProductRequest{
			Name:        r.FormValue("name"),
			PriceCents:  priceCents,
			Version:     int(version),
			CategoryID:  r.FormValue("category_id"),
			SKU:         r.FormValue("sku"),
			Description: r.FormValue("description"),
			Details:     detailsList,
		}, nil
	}
	var req updateProductRequest
	if err := decodeJSONBody(r, &req); err != nil {
		return updateProductRequest{}, err
	}
	ifMatchVersion, err := versionFromIfMatch(r)
	if err != nil {
		return updateProductRequest{}, err
	}
	if ifMatchVersion > 0 {
		req.Version = int(ifMatchVersion)
	}
	return req, nil
}

// writeRequestError maps a parse error to a 400 response.
func writeRequestError(w http.ResponseWriter, err error) {
	if ce, ok := errors.AsType[clientError](err); ok {
		response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, ce.msg))
		return
	}
	response.WriteJSON(w, http.StatusBadRequest, response.Fail(http.StatusBadRequest, "requisição inválida"))
}

// productResponse builds the response DTO from a domain product and its images.
func (h *Handlers) productResponse(ctx context.Context, p domain.Product) (productResponse, error) {
	images, err := h.svc.ListProductImages(ctx, p.ID)
	if err != nil {
		return productResponse{}, err
	}
	return toProductResponse(p, h.imageResponses(ctx, images)), nil
}

func (h *Handlers) imageResponses(ctx context.Context, images []domain.ProductImage) []productImageResponse {
	out := make([]productImageResponse, 0, len(images))
	for _, img := range images {
		out = append(out, toProductImageResponse(img, h.svc.PresignImageURL(ctx, img.ObjectKey)))
	}
	return out
}

// writeProduct writes a product response with ETag and appropriate status envelope.
func (h *Handlers) writeProduct(w http.ResponseWriter, r *http.Request, status int, p domain.Product) {
	resp, err := h.productResponse(r.Context(), p)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	w.Header().Set("ETag", etag.Format(p.Version))
	if status == http.StatusCreated {
		response.WriteJSON(w, status, response.Created(resp))
	} else {
		response.WriteJSON(w, status, response.OK(resp))
	}
}
