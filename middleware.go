package gonertia

import (
	"bytes"
	"io"
	"net/http"
)

// Middleware returns Inertia middleware handler.
//
// All of your handlers that can be handled by
// the Inertia should be under this middleware.
func (i *Inertia) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set header Vary to "X-Inertia".
		//
		// https://github.com/inertiajs/inertia-laravel/pull/404
		setInertiaVaryInResponse(w)

		// If request is not request by Inertia, we can move next.
		if !IsInertiaRequest(r) {
			next.ServeHTTP(w, r)

			return
		}

		// Now we know that this request was made by Inertia.
		//
		// But there is one problem:
		// http.ResponseWriter has no methods for getting the response status code and response content.
		// So, we have to create our own response writer wrapper, that will contain that info.
		//
		// It's not critical that we will have a byte buffer, because we
		// know that Inertia response always in JSON format and actually not very big.
		w2 := buildInertiaResponseWrapper(w)

		// Now put our response writer wrapper to other handlers.
		next.ServeHTTP(w2, r)

		// Our response writer wrapper does have all needle data! Yuppy!
		//
		// Don't forget to copy all data to the original
		// response writer before end!
		defer func() {
			i.copyHeaders(w, w2)
			i.copyStatusCode(w, w2)
			i.copyBuffer(w, w2)
		}()

		// Determines what to do when the Inertia asset version has changed.
		// By default, we'll initiate a client-side location visit to force an update.
		//
		// https://inertiajs.com/asset-versioning
		if r.Method == http.MethodGet && inertiaVersionFromRequest(r) != i.version {
			i.Location(w2, r, i.url+r.RequestURI)
			return
		}

		// Determines what to do when an Inertia action returned empty response.
		// By default, we will redirect the user back to where he came from.
		if w2.StatusCode() == http.StatusOK && w2.IsEmpty() {
			backURL := i.backURL(r)

			if backURL != "" {
				setInertiaLocationInResponse(w, backURL)
				return
			}
		}

		// The POST, PUT and PATCH requests cannot have the 302 code status.
		// Let's set the status code to the 303 instead.
		//
		// https://inertiajs.com/redirects#303-response-code
		if w2.StatusCode() == http.StatusFound && isSeeOtherRedirectMethod(r.Method) {
			setResponseStatus(w2, http.StatusSeeOther)
		}
	})
}

// copyBuffer copying source bytes buf into the destination bytes buffer.
func (i *Inertia) copyBuffer(dst http.ResponseWriter, src *inertiaResponseWrapper) {
	if _, err := io.Copy(dst, src.buf); err != nil {
		i.logger.Printf("cannot copy inertia response buffer to writer: %s", err)
	}
}

// copyStatusCode copying source status code into the destination status code.
func (i *Inertia) copyStatusCode(dst http.ResponseWriter, src *inertiaResponseWrapper) {
	dst.WriteHeader(src.statusCode)
}

// copyHeaders copying source header into the destination header.
func (i *Inertia) copyHeaders(dst http.ResponseWriter, src *inertiaResponseWrapper) {
	for key, headers := range src.header {
		for _, header := range headers {
			dst.Header().Add(key, header)
		}
	}
}

// inertiaResponseWrapper is the implementation of http.ResponseWriter,
// that have response body buffer and status code that will be returned to the front.
type inertiaResponseWrapper struct {
	statusCode int
	buf        *bytes.Buffer
	header     http.Header
}

var _ http.ResponseWriter = (*inertiaResponseWrapper)(nil)

// StatusCode returns HTTP status code of the response.
func (w *inertiaResponseWrapper) StatusCode() int {
	return w.statusCode
}

// IsEmpty returns true is the response body is empty.
func (w *inertiaResponseWrapper) IsEmpty() bool {
	return w.buf.Len() == 0
}

// Header returns response headers.
func (w *inertiaResponseWrapper) Header() http.Header {
	return w.header
}

// Write pushes some bytes to the response body.
func (w *inertiaResponseWrapper) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// WriteHeader sets the status code of the response.
func (w *inertiaResponseWrapper) WriteHeader(code int) {
	w.statusCode = code
}

// buildInertiaResponseWrapper initializes inertiaResponseWrapper.
func buildInertiaResponseWrapper(w http.ResponseWriter) *inertiaResponseWrapper {
	w2 := &inertiaResponseWrapper{
		statusCode: http.StatusOK,
		buf:        bytes.NewBuffer(nil),
		header:     w.Header(),
	}

	// In some situations, we can pass a http.ResponseWriter,
	// that also implements these interfaces.
	if val, ok := w.(interface{ StatusCode() int }); ok {
		w2.statusCode = val.StatusCode()
	}
	if val, ok := w.(interface{ Header() http.Header }); ok {
		w2.header = val.Header()
	}
	if val, ok := w.(interface{ Buffer() *bytes.Buffer }); ok {
		w2.buf = val.Buffer()
	}

	return w2
}
