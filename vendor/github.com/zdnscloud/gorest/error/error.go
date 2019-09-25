package error

var (
	Unauthorized     = ErrorCode{"Unauthorized", 401}
	PermissionDenied = ErrorCode{"PermissionDenied", 403}
	NotFound         = ErrorCode{"NotFound", 404}
	MethodNotAllowed = ErrorCode{"MethodNotAllow", 405}
	Conflict         = ErrorCode{"Conflict", 409}

	DuplicateResource  = ErrorCode{"DuplicateResource", 422}
	DeleteParent       = ErrorCode{"DeleteParent", 422}
	InvalidFormat      = ErrorCode{"InvalidFormat", 422}
	NotNullable        = ErrorCode{"NotNullable", 422}
	NotUnique          = ErrorCode{"NotUnique", 422}
	MinLimitExceeded   = ErrorCode{"MinLimitExceeded", 422}
	MaxLimitExceeded   = ErrorCode{"MaxLimitExceeded", 422}
	MinLengthExceeded  = ErrorCode{"MinLengthExceeded", 422}
	MaxLengthExceeded  = ErrorCode{"MaxLengthExceeded", 422}
	InvalidOption      = ErrorCode{"InvalidOption", 422}
	InvalidCharacters  = ErrorCode{"InvalidCharacters", 422}
	MissingRequired    = ErrorCode{"MissingRequired", 422}
	InvalidCSRFToken   = ErrorCode{"InvalidCSRFToken", 422}
	InvalidAction      = ErrorCode{"InvalidAction", 422}
	InvalidBodyContent = ErrorCode{"InvalidBodyContent", 422}
	InvalidType        = ErrorCode{"InvalidType", 422}

	ServerError        = ErrorCode{"ServerError", 500}
	ClusterUnavailable = ErrorCode{"ClusterUnavailable", 503}
)

type ErrorCode struct {
	Code   string `json:"code,omitempty"`
	Status int    `json:"status,omitempty"`
}

type APIError struct {
	ErrorCode `json:",inline"`
	Type      string `json:"type,omitempty"`
	Message   string `json:"message,omitempty"`
}

func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		ErrorCode: code,
		Type:      "error",
		Message:   message,
	}
}

func (e *APIError) Error() string {
	return e.Message
}
