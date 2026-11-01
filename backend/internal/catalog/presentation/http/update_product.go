package http

import (
	"minimart/internal/catalog/application"
	"net/http"
)

// UpdateProduct updates a product's name, price, category, SKU, description, and details.
// Images are managed through the dedicated product image endpoints.
//
//	@Summary		Update a product
//	@Description	Updates a product's name, price, category, SKU, description, and details. Stock is left unchanged. The version is taken from the If-Match header when present, otherwise from the required body field - both must be the value last read from GET. Returns 400 when it is missing or non-positive, and 409 when another writer changed the product since that version was read. Product images are managed through the dedicated image endpoints. Category is required.
//	@Tags			catalog
//	@Accept			json,mpfd
//	@Produce		json
//	@Param			id			path		string	true	"Product ID"
//	@Param			If-Match	header		string	false	"ETag from GET; overrides the body version"
//	@Param			name		formData	string	false	"Product name"
//	@Param			price_cents	formData	integer	false	"Product price in cents"
//	@Param			version		formData	integer	true	"Product version (optimistic lock)"
//	@Param			category_id	formData	string	true	"Product category ID"
//	@Param			sku			formData	string	false	"Product SKU"
//	@Param			description	formData	string	false	"Product description"
//	@Param			details		formData	string	false	"Product details as JSON array of name/value pairs"
//	@Header			200			{string}	ETag	"New product version"
//	@Success		200			{object}	response.Response{data=productResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		404			{object}	response.Response{error=response.ErrorBody}
//	@Failure		409			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products/{id} [patch]
func (h *Handlers) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	req, err := decodeUpdateRequest(r)
	if err != nil {
		writeRequestError(w, err)
		return
	}

	p, err := h.svc.UpdateProduct(r.Context(), application.UpdateProductRequest{
		ID:          r.PathValue("id"),
		Name:        req.Name,
		PriceCents:  req.PriceCents,
		Version:     req.Version,
		CategoryID:  req.CategoryID,
		SKU:         req.SKU,
		Description: req.Description,
		Details:     req.Details,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	h.writeProduct(w, r, http.StatusOK, p)
}
