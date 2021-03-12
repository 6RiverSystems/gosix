package ginmiddleware

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type OASErrorHandler func(c *gin.Context, err error)

func WithOASValidation(
	swagger *openapi3.Swagger,
	validateResponse bool,
	errorHandler OASErrorHandler,
	options *openapi3filter.Options,
) gin.HandlerFunc {
	oasRouter := openapi3filter.NewRouter().WithSwagger(swagger)
	if errorHandler == nil {
		errorHandler = DefaultOASErrorHandler
	}
	return func(c *gin.Context) {
		// TODO: prometheus metrics for validation failures

		route, pathParams, err := oasRouter.FindRoute(c.Request.Method, c.Request.URL)
		if err != nil {
			errorHandler(c, err)
			// either aborted or we don't want to validate this request, either way
			// let gin continue what's left of the chain (if anything)
			return
		}

		// Validate request
		requestValidationInput := &openapi3filter.RequestValidationInput{
			Request:    c.Request,
			PathParams: pathParams,
			Route:      route,
			Options:    options,
			// QueryParams will be auto-populated from the request url
		}
		if err := openapi3filter.ValidateRequest(c.Request.Context(), requestValidationInput); err != nil {
			errorHandler(c, err)
			return
		}

		if !validateResponse {
			// skip the rest, don't do the body capture if we don't need to
			return
		}

		// setup body capture for response validation
		// TODO: refactor this to common code
		// don't let the body pass through as we may need to replace it if there are errors
		bodyCapture, realWriter := CaptureResponseBody(c, false)

		c.Next()

		if len(c.Errors) > 0 || c.IsAborted() {
			// don't do response validation on errors, but we do need to pass-through the response body
			if bodyCapture.Len() > 0 {
				_, err = realWriter.Write(bodyCapture.Bytes())
				if err != nil {
					panic(err)
				}
			}
			return
		}

		responseValidationInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestValidationInput,
			Status:                 c.Writer.Status(),
			Header:                 c.Writer.Header(),
			Options:                options,
			// Body is set below
		}
		responseValidationInput.SetBodyBytes(bodyCapture.Bytes())

		if err := openapi3filter.ValidateResponse(c.Request.Context(), responseValidationInput); err != nil {
			// restore the real writer for error emission
			c.Writer = realWriter
			errorHandler(c, err)
			return
		}

		// write the buffered body now that validation passed
		_, err = realWriter.Write(bodyCapture.Bytes())
		if err != nil {
			panic(err)
		}
	}
}

func DefaultOASErrorHandler(c *gin.Context, err error) {
	// nolint:errcheck // return value here is just a wrapped copy of the input
	c.Error(err)
	var (
		routeErr    *openapi3filter.RouteError
		requestErr  *openapi3filter.RequestError
		responseErr *openapi3filter.ResponseError
		parseErr    *openapi3filter.ParseError
	)
	switch {
	case errors.As(err, &routeErr):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": routeErr.Reason})
	case errors.As(err, &requestErr):
		c.AbortWithStatusJSON(requestErr.HTTPStatus(), gin.H{"error": requestErr.Reason, "details": requestErr.Err})
	case errors.As(err, &responseErr):
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": responseErr.Reason, "details": responseErr.Err})
	case errors.As(err, &parseErr):
		// TODO: this may not be right
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "details": err})
	default:
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

func AllowUndefinedRoutes(handler OASErrorHandler) OASErrorHandler {
	return func(c *gin.Context, err error) {
		var re *openapi3filter.RouteError
		if errors.As(err, &re) {
			// TODO: prometheus metric for this
			return
		}
		handler(c, err)
	}
}
