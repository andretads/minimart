package http

import (
	"minimart/internal/catalog/domain"
	"minimart/pkg/response"
	"net/http"
)

// AddProductImage uploads an image for an existing product.
//
//	@Summary		Add a product image
//	@Description	Uploads an image and appends it to the product's images. The first image has position 0 and is treated as primary. Adding an image bumps the product version.
//	@Tags			catalog
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		string	true	"Product ID"
//	@Param			image	formData	file	true	"Product image (max 5MB, image/*)"
//	@Success		201		{object}	response.Response{data=productImageResponse}
//	@Failure		400		{object}	response.Response{error=response.ErrorBody}
//	@Failure		404		{object}	response.Response{error=response.ErrorBody}
//	@Failure		500		{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id}/images [post]
func (h *Handlers) AddProductImage(w http.ResponseWriter, r *http.Request) {
	if !isMultipart(r) {
		writeRequestError(w, badRequest("multipart/form-data is required"))
		return
	}
	if err := parseMultipart(r); err != nil {
		writeRequestError(w, err)
		return
	}

	img, closeFn, err := parseImageUpload(r)
	if closeFn != nil {
		defer closeFn()
	}
	if err != nil {
		writeRequestError(w, err)
		return
	}
	if img == nil {
		respondDomainError(w, domain.ErrImageRequired)
		return
	}

	productID := r.PathValue("id")
	saved, err := h.svc.AddProductImage(r.Context(), productID, img)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	resp := toProductImageResponse(saved, h.svc.PresignImageURL(r.Context(), saved.ObjectKey))
	response.WriteJSON(w, http.StatusCreated, response.Created(resp))
}

// ListProductImages returns a product's images ordered by position.
//
//	@Summary		List product images
//	@Description	Returns a product's images ordered by position. The first image is the primary image. URLs are short-lived presigned links.
//	@Tags			catalog
//	@Produce		json
//	@Param			id	path		string	true	"Product ID"
//	@Success		200	{object}	response.Response{data=[]productImageResponse}
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id}/images [get]
func (h *Handlers) ListProductImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.svc.ListProductImages(r.Context(), r.PathValue("id"))
	if err != nil {
		respondDomainError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, response.OK(h.imageResponses(r.Context(), images)))
}

// DeleteProductImage removes an image from a product.
//
//	@Summary		Delete a product image
//	@Description	Removes an image from a product and deletes its stored object. Deleting an image bumps the product version.
//	@Tags			catalog
//	@Param			id			path	string	true	"Product ID"
//	@Param			image_id	path	string	true	"Product image ID"
//	@Success		204
//	@Failure		404	{object}	response.Response{error=response.ErrorBody}
//	@Failure		500	{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id}/images/{image_id} [delete]
func (h *Handlers) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.DeleteProductImage(r.Context(), r.PathValue("id"), r.PathValue("image_id")); err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
