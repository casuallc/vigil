/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dockerregistry

import (
	"encoding/json"
	"time"
)

// CatalogResponse is the response body for GET /v2/_catalog.
type CatalogResponse struct {
	Repositories []string `json:"repositories"`
}

// TagsResponse is the response body for GET /v2/{name}/tags/list.
type TagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// RegistryError is a single error in the Docker Registry V2 error envelope.
type RegistryError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

// RegistryErrors is the Docker Registry V2 error envelope.
type RegistryErrors struct {
	Errors []RegistryError `json:"errors"`
}

// UploadState tracks the state of an in-progress blob upload.
type UploadState struct {
	Offset  int64     `json:"offset"`
	Started time.Time `json:"started"`
}

// Common Docker Registry error codes.
const (
	ErrCodeNameUnknown       = "NAME_UNKNOWN"
	ErrCodeManifestUnknown   = "MANIFEST_UNKNOWN"
	ErrCodeBlobUnknown       = "BLOB_UNKNOWN"
	ErrCodeDigestInvalid     = "DIGEST_INVALID"
	ErrCodeBlobUploadInvalid = "BLOB_UPLOAD_INVALID"
	ErrCodeBlobUploadUnknown = "BLOB_UPLOAD_UNKNOWN"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeUnsupported       = "UNSUPPORTED"
	ErrCodeUnknown           = "UNKNOWN"
)

// ErrorEnvelope builds a JSON error envelope.
func ErrorEnvelope(code, message string, detail interface{}) []byte {
	env := RegistryErrors{
		Errors: []RegistryError{
			{Code: code, Message: message, Detail: detail},
		},
	}
	b, _ := json.Marshal(env)
	return b
}
