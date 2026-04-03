package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/shaikzhafir/when-bus/internal/generated"
	log "github.com/shaikzhafir/when-bus/internal/logging"
)

// OpenAPIValidationErrorHandler handles query/header validation errors from oapi-codegen.
// It returns JSON and logs a clear reason (missing param, bad format, etc.).
func OpenAPIValidationErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	status, reason := classifyOpenAPIError(err)
	log.Debug(
		"request validation failed: reason=%s method=%s path=%s query=%q remote_addr=%s detail=%v",
		reason,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		r.RemoteAddr,
		err,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   err.Error(),
		"reason":  reason,
	})
}

func classifyOpenAPIError(err error) (status int, reason string) {
	switch e := err.(type) {
	case *generated.RequiredParamError:
		return http.StatusBadRequest, "missing_required_query_param:" + e.ParamName
	case *generated.InvalidParamFormatError:
		return http.StatusBadRequest, "invalid_query_param_format:" + e.ParamName
	case *generated.RequiredHeaderError:
		return http.StatusBadRequest, "missing_required_header:" + e.ParamName
	case *generated.UnmarshalingParamError:
		return http.StatusBadRequest, "invalid_param_json:" + e.ParamName
	case *generated.TooManyValuesForParamError:
		return http.StatusBadRequest, "too_many_values_for_query_param:" + e.ParamName
	default:
		return http.StatusBadRequest, "invalid_request"
	}
}
