package http

import (
	"minimart/pkg/response"
	"net/http"
)

// GetProduct returns a single product by ID.
//
//	@Summary		Get product
//	@Description	Returns a single product by its ID, including its images with short-lived presigned URLs.
//	@Tags			showcase
//	@Produce		json
//	@Param			id	path		string	true	"Product ID"
//	@Success		200	{object}	response.Response{data=productResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/showcase/products/{id} [get]
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetProduct(r.Context(), id)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	images := make([]productImageResponse, 0, len(p.Images))
	for _, img := range p.Images {
		url := h.svc.PresignImageURL(r.Context(), img.ObjectKey)
		images = append(images, toProductImageResponse(img, url))
	}

	response.WriteJSON(w, http.StatusOK, response.OK(toProductDetailResponse(p, images)))
}
