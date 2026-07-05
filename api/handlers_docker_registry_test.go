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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casuallc/vigil/dockerregistry"
)

func newRegistryTestServer(t *testing.T) *Server {
	t.Helper()
	mgr, err := dockerregistry.NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("new registry manager: %v", err)
	}
	return &Server{dockerRegistryManager: mgr}
}

func TestDockerRegistryVersionCheck(t *testing.T) {
	s := newRegistryTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Docker-Distribution-Api-Version"); got != "registry/2.0" {
		t.Fatalf("expected api version registry/2.0, got %q", got)
	}
}

func TestDockerRegistryPushPullRoundTrip(t *testing.T) {
	s := newRegistryTestServer(t)
	router := s.Router()
	repo := "myrepo"
	blobData := []byte("hello docker registry blob")
	blobDigest, err := dockerregistry.DigestFromReader(bytes.NewReader(blobData))
	if err != nil {
		t.Fatalf("digest blob: %v", err)
	}

	// 1. Initiate upload.
	initReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v2/%s/blobs/uploads/", repo), nil)
	initRec := httptest.NewRecorder()
	router.ServeHTTP(initRec, initReq)
	if initRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on upload init, got %d: %s", initRec.Code, initRec.Body.String())
	}
	location := initRec.Header().Get("Location")
	parts := strings.Split(location, "/")
	uuid := parts[len(parts)-1]
	if uuid == "" {
		t.Fatalf("expected upload uuid in Location header, got %q", location)
	}

	// 2. Upload blob monolithically via final PUT.
	putURL := fmt.Sprintf("%s?digest=%s", location, blobDigest)
	putReq := httptest.NewRequest(http.MethodPut, putURL, bytes.NewReader(blobData))
	putReq.Header.Set("Content-Type", "application/octet-stream")
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on blob put, got %d: %s", putRec.Code, putRec.Body.String())
	}
	if got := putRec.Header().Get("Docker-Content-Digest"); got != blobDigest {
		t.Fatalf("expected digest %q, got %q", blobDigest, got)
	}

	// 3. HEAD/GET blob.
	headReq := httptest.NewRequest(http.MethodHead, fmt.Sprintf("/v2/%s/blobs/%s", repo, blobDigest), nil)
	headRec := httptest.NewRecorder()
	router.ServeHTTP(headRec, headReq)
	if headRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on blob head, got %d: %s", headRec.Code, headRec.Body.String())
	}
	if headRec.Header().Get("Content-Length") != fmt.Sprintf("%d", len(blobData)) {
		t.Fatalf("expected Content-Length %d, got %q", len(blobData), headRec.Header().Get("Content-Length"))
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/%s/blobs/%s", repo, blobDigest), nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on blob get, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !bytes.Equal(getRec.Body.Bytes(), blobData) {
		t.Fatalf("blob body mismatch")
	}

	// 4. PUT manifest.
	manifest := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
		"config": map[string]interface{}{
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size":    7023,
			"digest":  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		"layers": []map[string]interface{}{
			{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size":      len(blobData),
				"digest":    blobDigest,
			},
		},
	}
	manifestBody, _ := json.Marshal(manifest)
	manifestDigest, _ := dockerregistry.DigestFromReader(bytes.NewReader(manifestBody))

	mpReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/%s/manifests/latest", repo), bytes.NewReader(manifestBody))
	mpReq.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	mpRec := httptest.NewRecorder()
	router.ServeHTTP(mpRec, mpReq)
	if mpRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on manifest put, got %d: %s", mpRec.Code, mpRec.Body.String())
	}
	if got := mpRec.Header().Get("Docker-Content-Digest"); got != manifestDigest {
		t.Fatalf("expected manifest digest %q, got %q", manifestDigest, got)
	}

	// 5. GET manifest by tag and by digest.
	for _, ref := range []string{"latest", manifestDigest} {
		mReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/%s/manifests/%s", repo, ref), nil)
		mRec := httptest.NewRecorder()
		router.ServeHTTP(mRec, mReq)
		if mRec.Code != http.StatusOK {
			t.Fatalf("expected 200 on manifest get %s, got %d: %s", ref, mRec.Code, mRec.Body.String())
		}
		if got := mRec.Header().Get("Docker-Content-Digest"); got != manifestDigest {
			t.Fatalf("manifest digest header mismatch for %s: got %q", ref, got)
		}
		if !bytes.Equal(mRec.Body.Bytes(), manifestBody) {
			t.Fatalf("manifest body mismatch for %s", ref)
		}
	}

	// 6. Catalog and tags.
	catReq := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
	catRec := httptest.NewRecorder()
	router.ServeHTTP(catRec, catReq)
	if catRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on catalog, got %d: %s", catRec.Code, catRec.Body.String())
	}
	var cat dockerregistry.CatalogResponse
	if err := json.Unmarshal(catRec.Body.Bytes(), &cat); err != nil {
		t.Fatalf("unmarshal catalog: %v", err)
	}
	if len(cat.Repositories) != 1 || cat.Repositories[0] != repo {
		t.Fatalf("unexpected catalog: %+v", cat)
	}

	tagsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/%s/tags/list", repo), nil)
	tagsRec := httptest.NewRecorder()
	router.ServeHTTP(tagsRec, tagsReq)
	if tagsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on tags, got %d: %s", tagsRec.Code, tagsRec.Body.String())
	}
	var tags dockerregistry.TagsResponse
	if err := json.Unmarshal(tagsRec.Body.Bytes(), &tags); err != nil {
		t.Fatalf("unmarshal tags: %v", err)
	}
	if tags.Name != repo || len(tags.Tags) != 1 || tags.Tags[0] != "latest" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestDockerRegistryNamespaceRepo(t *testing.T) {
	s := newRegistryTestServer(t)
	router := s.Router()
	repo := "library/nginx"
	blobData := []byte("namespace blob")
	blobDigest, _ := dockerregistry.DigestFromReader(bytes.NewReader(blobData))

	initReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v2/%s/blobs/uploads/", repo), nil)
	initRec := httptest.NewRecorder()
	router.ServeHTTP(initRec, initReq)
	if initRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", initRec.Code, initRec.Body.String())
	}
	location := initRec.Header().Get("Location")

	putReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s?digest=%s", location, blobDigest), bytes.NewReader(blobData))
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", putRec.Code, putRec.Body.String())
	}

	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":0,"digest":"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":14,"digest":"` + blobDigest + `"}]}`)
	mpReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/%s/manifests/v1", repo), bytes.NewReader(manifest))
	mpReq.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	mpRec := httptest.NewRecorder()
	router.ServeHTTP(mpRec, mpReq)
	if mpRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", mpRec.Code, mpRec.Body.String())
	}

	catReq := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
	catRec := httptest.NewRecorder()
	router.ServeHTTP(catRec, catReq)
	var cat dockerregistry.CatalogResponse
	_ = json.Unmarshal(catRec.Body.Bytes(), &cat)
	found := false
	for _, r := range cat.Repositories {
		if r == repo {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected repo %s in catalog, got %+v", repo, cat)
	}

	tagsReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/%s/tags/list", repo), nil)
	tagsRec := httptest.NewRecorder()
	router.ServeHTTP(tagsRec, tagsReq)
	var tags dockerregistry.TagsResponse
	_ = json.Unmarshal(tagsRec.Body.Bytes(), &tags)
	if tags.Name != repo || len(tags.Tags) != 1 || tags.Tags[0] != "v1" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestDockerRegistryChunkedUpload(t *testing.T) {
	s := newRegistryTestServer(t)
	router := s.Router()
	repo := "chunky"
	blobData := []byte("abcdefghij")
	blobDigest, _ := dockerregistry.DigestFromReader(bytes.NewReader(blobData))

	initReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v2/%s/blobs/uploads/", repo), nil)
	initRec := httptest.NewRecorder()
	router.ServeHTTP(initRec, initReq)
	location := initRec.Header().Get("Location")

	// Send first 5 bytes.
	patch1 := httptest.NewRequest(http.MethodPatch, location, bytes.NewReader(blobData[:5]))
	patch1.Header.Set("Content-Range", "bytes 0-4")
	patch1.Header.Set("Content-Type", "application/octet-stream")
	p1Rec := httptest.NewRecorder()
	router.ServeHTTP(p1Rec, patch1)
	if p1Rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on first patch, got %d: %s", p1Rec.Code, p1Rec.Body.String())
	}
	if p1Rec.Header().Get("Range") != "0-4" {
		t.Fatalf("expected Range 0-4, got %q", p1Rec.Header().Get("Range"))
	}

	// Send remaining 5 bytes.
	patch2 := httptest.NewRequest(http.MethodPatch, location, bytes.NewReader(blobData[5:]))
	patch2.Header.Set("Content-Range", "bytes 5-9")
	patch2.Header.Set("Content-Type", "application/octet-stream")
	p2Rec := httptest.NewRecorder()
	router.ServeHTTP(p2Rec, patch2)
	if p2Rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on second patch, got %d: %s", p2Rec.Code, p2Rec.Body.String())
	}

	// Complete upload.
	putReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s?digest=%s", location, blobDigest), nil)
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on complete, got %d: %s", putRec.Code, putRec.Body.String())
	}

	// Verify blob.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/%s/blobs/%s", repo, blobDigest), nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK || !bytes.Equal(getRec.Body.Bytes(), blobData) {
		t.Fatalf("blob mismatch after chunked upload")
	}
}

func TestDockerRegistryManifestMissingBlob(t *testing.T) {
	s := newRegistryTestServer(t)
	router := s.Router()
	repo := "broken"
	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":0,"digest":"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":14,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
	mpReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/%s/manifests/latest", repo), bytes.NewReader(manifest))
	mpReq.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	mpRec := httptest.NewRecorder()
	router.ServeHTTP(mpRec, mpReq)
	// The registry does not validate manifest layer existence by design in v1.
	if mpRec.Code != http.StatusCreated {
		t.Fatalf("expected manifest put to succeed without blob validation, got %d: %s", mpRec.Code, mpRec.Body.String())
	}
}

