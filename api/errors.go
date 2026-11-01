package api

// NewError builds the body every error response shares:
// {"error": {"code": "...", "message": "..."}}.
func NewError(code ErrorErrorCode, message string) Error {
	var e Error
	e.Error.Code = code
	e.Error.Message = message
	return e
}

// NewBadRequest is a 400: the request is malformed.
func NewBadRequest(message string) BadRequestJSONResponse {
	return BadRequestJSONResponse(NewError(ErrorErrorCodeBadRequest, message))
}

// NewNotFound is a 404: no such resource.
func NewNotFound(message string) NotFoundJSONResponse {
	return NotFoundJSONResponse(NewError(ErrorErrorCodeNotFound, message))
}

// NewValidationFailed is a 422: well-formed, but it breaks a rule.
func NewValidationFailed(message string) ValidationFailedJSONResponse {
	return ValidationFailedJSONResponse(NewError(ErrorErrorCodeValidationFailed, message))
}
