// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"net/http"

	"io"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/log"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/logutil"
)

func chop(data []byte, maxlen int64) string {
	s := string(data)
	if int64(len(s)) > maxlen {
		s = s[:maxlen] + "..."
	}
	return s
}

func HandleBody(body io.ReadCloser, maxlen int64) (io.ReadCloser, string) {
	if body == nil {
		return nil, ""
	}

	defer body.Close()

	b, err := io.ReadAll(body)
	if err != nil {
		return nil, ""
	}

	return io.NopCloser(bytes.NewReader(b)), chop(b, maxlen)
}

func WithInspection(maxlen int64) client.RequestMiddleware {
	return func(r *http.Request) (*http.Request, error) {
		if IsDebugEnabled() {
			body, bodyString := HandleBody(r.Body, maxlen)
			r.Body = body

			log.Print("Azure request", logutil.Fields{
				"method":  r.Method,
				"request": r.URL.String(),
				"body":    bodyString,
			})
		}
		return r, nil
	}
}

func ByInspecting(maxlen int64) client.ResponseMiddleware {
	return func(req *http.Request, resp *http.Response) (*http.Response, error) {
		if IsDebugEnabled() {
			body, bodyString := HandleBody(resp.Body, maxlen)
			resp.Body = body

			log.Print("Azure response", logutil.Fields{
				"status":          resp.Status,
				"method":          resp.Request.Method,
				"request":         resp.Request.URL.String(),
				"x-ms-request-id": ExtractRequestID(resp),
				"body":            bodyString,
			})
		}

		return resp, nil
	}
}

// ExtractRequestID extracts the Azure server generated request identifier from the
// x-ms-request-id header.
func ExtractRequestID(resp *http.Response) string {
	if resp != nil && resp.Header != nil {
		header := resp.Header[http.CanonicalHeaderKey("x-ms-request-id")]
		if len(header) > 0 {
			return header[0]
		}
	}
	return ""
}
