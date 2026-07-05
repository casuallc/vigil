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

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/casuallc/vigil/dockerregistry"
	"github.com/gorilla/mux"
)

const registryAPIVersion = "registry/2.0"

func (s *Server) handleDockerRegistryVersionCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

func (s *Server) handleDockerRegistryCatalog(w http.ResponseWriter, r *http.Request) {
	repos, err := s.dockerRegistryManager.Repositories()
	if err != nil {
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	writeJSON(w, http.StatusOK, dockerregistry.CatalogResponse{Repositories: repos})
}

func (s *Server) handleDockerRegistryTagsList(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	tags, err := s.dockerRegistryManager.Tags(name)
	if err != nil {
		if isRegistryNotFound(err) {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeNameUnknown, "repository not found")
			return
		}
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	writeJSON(w, http.StatusOK, dockerregistry.TagsResponse{Name: name, Tags: tags})
}

func (s *Server) handleDockerRegistryManifestHead(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	reference := mux.Vars(r)["reference"]
	mediaType, body, digest, err := s.dockerRegistryManager.GetManifest(name, reference)
	if err != nil {
		writeManifestError(w, err)
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDockerRegistryManifestGet(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	reference := mux.Vars(r)["reference"]
	mediaType, body, digest, err := s.dockerRegistryManager.GetManifest(name, reference)
	if err != nil {
		writeManifestError(w, err)
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (s *Server) handleDockerRegistryManifestPut(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	reference := mux.Vars(r)["reference"]
	mediaType := r.Header.Get("Content-Type")
	if mediaType == "" {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeRegistryError(w, http.StatusBadRequest, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}
	defer r.Body.Close()

	digest, err := s.dockerRegistryManager.PutManifest(name, reference, mediaType, body)
	if err != nil {
		if isRegistryNotFound(err) {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeNameUnknown, "repository not found")
			return
		}
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}

	location := fmt.Sprintf("/v2/%s/manifests/%s", name, digest)
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDockerRegistryManifestDelete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	reference := mux.Vars(r)["reference"]
	if err := s.dockerRegistryManager.DeleteManifest(name, reference); err != nil {
		writeManifestError(w, err)
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDockerRegistryBlobUploadInit(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	uuid, err := s.dockerRegistryManager.CreateUpload(name)
	if err != nil {
		if isRegistryNotFound(err) {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeNameUnknown, "repository not found")
			return
		}
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}
	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Location", location)
	w.Header().Set("Range", "0-0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDockerRegistryBlobUploadPatch(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	uuid := mux.Vars(r)["uuid"]

	state, err := s.dockerRegistryManager.ReadUploadState(uuid)
	if err != nil {
		writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUploadUnknown, "upload not found")
		return
	}

	offset := state.Offset
	if cr := r.Header.Get("Content-Range"); cr != "" {
		start, _, err := parseContentRange(cr)
		if err != nil {
			writeRegistryError(w, http.StatusRequestedRangeNotSatisfiable, dockerregistry.ErrCodeBlobUploadInvalid, err.Error())
			return
		}
		offset = start
	}

	n, err := s.dockerRegistryManager.WriteUploadChunk(uuid, r.Body, offset)
	if err != nil {
		if errors.Is(err, dockerregistry.ErrBlobUploadInvalid) {
			writeRegistryError(w, http.StatusRequestedRangeNotSatisfiable, dockerregistry.ErrCodeBlobUploadInvalid, err.Error())
			return
		}
		if errors.Is(err, dockerregistry.ErrBlobUploadUnknown) {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUploadUnknown, "upload not found")
			return
		}
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}

	newState, _ := s.dockerRegistryManager.ReadUploadState(uuid)
	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, uuid)
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Location", location)
	w.Header().Set("Range", fmt.Sprintf("0-%d", newState.Offset-1))
	w.Header().Set("Content-Length", strconv.FormatInt(n, 10))
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleDockerRegistryBlobUploadPut(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	uuid := mux.Vars(r)["uuid"]
	digest := r.URL.Query().Get("digest")
	if digest == "" {
		writeRegistryError(w, http.StatusBadRequest, dockerregistry.ErrCodeDigestInvalid, "missing digest query parameter")
		return
	}

	// Support monolithic upload where the body is provided in the final PUT.
	if r.ContentLength > 0 || r.Body != http.NoBody {
		state, err := s.dockerRegistryManager.ReadUploadState(uuid)
		if err != nil {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUploadUnknown, "upload not found")
			return
		}
		if _, err := s.dockerRegistryManager.WriteUploadChunk(uuid, r.Body, state.Offset); err != nil {
			writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
			return
		}
	}
	defer r.Body.Close()

	actualDigest, err := s.dockerRegistryManager.CompleteUpload(uuid, digest)
	if err != nil {
		if errors.Is(err, dockerregistry.ErrDigestInvalid) {
			writeRegistryError(w, http.StatusBadRequest, dockerregistry.ErrCodeDigestInvalid, err.Error())
			return
		}
		if errors.Is(err, dockerregistry.ErrBlobUploadUnknown) {
			writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUploadUnknown, "upload not found")
			return
		}
		writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", name, actualDigest)
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", actualDigest)
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDockerRegistryBlobUploadDelete(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["uuid"]
	if err := s.dockerRegistryManager.DeleteUpload(uuid); err != nil {
		writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUploadUnknown, "upload not found")
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDockerRegistryBlobHead(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	digest := mux.Vars(r)["digest"]
	size, err := s.dockerRegistryManager.StatBlob(name, digest)
	if err != nil {
		writeBlobError(w, err)
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDockerRegistryBlobGet(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	digest := mux.Vars(r)["digest"]
	reader, size, err := s.dockerRegistryManager.OpenBlob(name, digest)
	if err != nil {
		writeBlobError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func (s *Server) handleDockerRegistryBlobDelete(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	digest := mux.Vars(r)["digest"]
	if err := s.dockerRegistryManager.DeleteBlob(name, digest); err != nil {
		writeBlobError(w, err)
		return
	}
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.WriteHeader(http.StatusAccepted)
}

// Helpers

func writeRegistryError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Docker-Distribution-Api-Version", registryAPIVersion)
	w.WriteHeader(status)
	_, _ = w.Write(dockerregistry.ErrorEnvelope(code, message, nil))
}

func writeManifestError(w http.ResponseWriter, err error) {
	if errors.Is(err, dockerregistry.ErrManifestUnknown) {
		writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeManifestUnknown, "manifest unknown")
		return
	}
	if errors.Is(err, dockerregistry.ErrNameUnknown) {
		writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeNameUnknown, "repository not found")
		return
	}
	writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
}

func writeBlobError(w http.ResponseWriter, err error) {
	if errors.Is(err, dockerregistry.ErrBlobUnknown) {
		writeRegistryError(w, http.StatusNotFound, dockerregistry.ErrCodeBlobUnknown, "blob unknown")
		return
	}
	writeRegistryError(w, http.StatusInternalServerError, dockerregistry.ErrCodeUnknown, err.Error())
}

func isRegistryNotFound(err error) bool {
	return errors.Is(err, dockerregistry.ErrNameUnknown) || errors.Is(err, dockerregistry.ErrManifestUnknown) || errors.Is(err, dockerregistry.ErrBlobUnknown)
}

// parseContentRange parses a "bytes <start>-<end>" Content-Range header.
func parseContentRange(s string) (start, end int64, err error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "bytes ") {
		return 0, 0, fmt.Errorf("invalid content range")
	}
	s = strings.TrimPrefix(s, "bytes ")
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid content range")
	}
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid content range start")
	}
	end, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid content range end")
	}
	if start < 0 || end < start {
		return 0, 0, fmt.Errorf("invalid content range")
	}
	return start, end, nil
}
