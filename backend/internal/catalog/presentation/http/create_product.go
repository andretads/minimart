package http

import (
	"minimart/internal/catalog/application"
	"net/http"
)

// CreateProduct creates a product.
// Images are added separately through POST /v1/products/{id}/images.
//
//	@Summary		Create a product
//	@Description	Adds a new product to the catalog. Product images are managed through the dedicated image endpoints.
//	@Tags			catalog
//	@Accept			json,mpfd
//	@Produce		json
//	@Param			name		formData	string	true	"Product name"
//	@Param			price_cents	formData	integer	true	"Product price in cents"
//	@Param			stock		formData	integer	true	"Product stock"
//	@Param			category_id	formData	string	true	"Product category ID"
//	@Param			sku			formData	string	false	"Product SKU"
//	@Param			description	formData	string	false	"Product description"
//	@Param			details		formData	string	false	"Product details as JSON array of name/value pairs"
//	@Header			201			{string}	ETag	"New product version"
//	@Success		201			{object}	response.Response{data=productResponse}
//	@Failure		400			{object}	response.Response{error=response.ErrorBody}
//	@Failure		500			{object}	response.Response{error=response.ErrorBody}
//	@Router			/v1/products [post]
func (h *Handlers) CreateProduct(w http.ResponseWriter, r *http.Request) {
	req, err := decodeCreateRequest(r)
	if err != nil {
		writeRequestError(w, err)
		return
	}

	p, err := h.svc.CreateProduct(r.Context(), application.CreateProductRequest{
		Name:        req.Name,
		PriceCents:  req.PriceCents,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
		SKU:         req.SKU,
		Description: req.Description,
		Details:     req.Details,
	})
	if err != nil {
		respondDomainError(w, err)
		return
	}

	h.writeProduct(w, r, http.StatusCreated, p)
}
