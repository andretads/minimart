// Package response defines the shared JSON envelope every HTTP handler in the
// modular monolith returns, plus the helpers that build and write it.
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response represents a standard API response.
type Response struct {
	Time    time.Time  `json:"time"`
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
	Meta    *Meta      `json:"meta,omitempty"`
} //	@name	Response

type ErrorBody struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
} //	@name	ErrorBody

type Meta struct {
	Page       int    `json:"page,omitzero"`
	PageSize   int    `json:"page_size,omitzero"`
	Total      int    `json:"total,omitzero"`
	TotalPages int    `json:"total_pages,omitzero"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    *bool  `json:"has_more,omitempty"`
} // @name Meta

// WriteJSON encodes body as JSON, setting the Content-Type and status code.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func OK(data any) Response {
	return Response{
		Time:    time.Now(),
		Success: true,
		Data:    data,
	}
}

func Created(data any) Response {
	return Response{
		Time:    time.Now(),
		Success: true,
		Data:    data,
	}
}

func NoContent() Response {
	return Response{
		Time:    time.Now(),
		Success: true,
	}
}

func Fail(code int, message string) Response {
	return Response{
		Time:    time.Now(),
		Success: false,
		Error: &ErrorBody{
			Code:    code,
			Message: message,
		},
	}
}

func Paginated(data any, meta Meta) Response {
	return Response{
		Time:    time.Now(),
		Success: true,
		Data:    data,
		Meta:    &meta,
	}
}
