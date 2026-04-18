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
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/casuallc/vigil/file"
)

const (
	streamUploadThreshold = 100 << 20 // 100MB
)

// FileUpload uploads a file to the server, automatically choosing multipart or stream based on file size
func (c *Client) FileUpload(sourcePath, targetPath string) error {
	// Get file info to determine upload strategy
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %v", err)
	}

	// For large files, use stream upload
	if fileInfo.Size() > streamUploadThreshold {
		return c.fileStreamUpload(sourcePath, targetPath, "/api/files/stream")
	}

	// For small files, use multipart upload
	return c.fileMultipartUpload(sourcePath, targetPath, "/api/files/upload")
}

// fileStreamUpload uploads a file via raw body stream
func (c *Client) fileStreamUpload(sourcePath, targetPath, endpoint string) error {
	// Open local file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create request with file as body (streaming)
	req, err := http.NewRequest("POST", c.host+endpoint, sourceFile)
	if err != nil {
		return err
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Target-Path", targetPath)

	// If Basic Auth is configured, attach to request
	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// fileMultipartUpload uploads a file via multipart/form-data
func (c *Client) fileMultipartUpload(sourcePath, targetPath, endpoint string) error {
	// Open local file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Create multipart/form-data request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file field
	fileField, err := writer.CreateFormFile("file", filepath.Base(sourcePath))
	if err != nil {
		return err
	}

	// Copy file content to multipart writer
	if _, err := io.Copy(fileField, sourceFile); err != nil {
		return err
	}

	// Add target path field
	if err := writer.WriteField("target_path", targetPath); err != nil {
		return err
	}

	// Finish multipart writer
	contentType := writer.FormDataContentType()
	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", c.host+endpoint, body)
	if err != nil {
		return err
	}

	// Set request headers
	req.Header.Set("Content-Type", contentType)

	// If Basic Auth is configured, attach to request
	if c.basicUser != "" && c.basicPass != "" {
		req.SetBasicAuth(c.basicUser, c.basicPass)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// VMFileUpload uploads a file to a VM (through server), automatically choosing multipart or stream based on file size
func (c *Client) VMFileUpload(vmName, sourcePath, targetPath string) error {
	// Get file info to determine upload strategy
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %v", err)
	}

	// For large files, use stream upload
	if fileInfo.Size() > streamUploadThreshold {
		return c.fileStreamUpload(sourcePath, targetPath, fmt.Sprintf("/api/vms/files/%s/stream", vmName))
	}

	// For small files, use multipart upload
	return c.fileMultipartUpload(sourcePath, targetPath, fmt.Sprintf("/api/vms/files/%s/upload", vmName))
}

// FileDownload downloads a file from the server
func (c *Client) FileDownload(sourcePath, targetPath string) error {
	reqBody := map[string]interface{}{
		"source_path": sourcePath,
	}

	// Send request
	resp, err := c.doRequest("POST", "/api/files/download", reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// Save file content
	if _, err := io.Copy(targetFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save file content: %v", err)
	}

	return nil
}

// VMFileDownload downloads a file from a VM (through server)
func (c *Client) VMFileDownload(vmName, sourcePath, targetPath string) error {
	reqBody := map[string]interface{}{
		"source_path": sourcePath,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/download", vmName), reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	// Save file content
	if _, err := io.Copy(targetFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save file content: %v", err)
	}

	return nil
}

// VMFileList lists files in a VM (through server)
func (c *Client) VMFileList(vmName, path string, maxDepth int) ([]file.Info, error) {
	var files []file.Info

	reqBody := map[string]interface{}{
		"path":      path,
		"max_depth": maxDepth,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/list", vmName), reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &files); err != nil {
		return nil, err
	}

	return files, nil
}

// VMFileDelete deletes a file in a VM
func (c *Client) VMFileDelete(vmName, path string) error {
	reqBody := map[string]interface{}{
		"path": path,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/delete", vmName), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// VMFileMkdir creates a directory on a VM
func (c *Client) VMFileMkdir(vmName, path string, parents bool) error {
	reqBody := map[string]interface{}{
		"path":    path,
		"parents": parents,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/mkdir", vmName), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// VMFileTouch creates a file on a VM
func (c *Client) VMFileTouch(vmName, path string) error {
	reqBody := map[string]interface{}{
		"path": path,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/touch", vmName), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// VMFileRmdir deletes a directory in a VM
func (c *Client) VMFileRmdir(vmName, path string, recursive bool) error {
	reqBody := map[string]interface{}{
		"path":      path,
		"recursive": recursive,
	}

	// Send request with new route format
	resp, err := c.doRequest("POST", fmt.Sprintf("/api/vms/files/%s/rmdir", vmName), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// FileList lists files
func (c *Client) FileList(path string, maxDepth int) ([]file.Info, error) {
	var files []file.Info

	reqBody := map[string]interface{}{
		"path":      path,
		"max_depth": maxDepth,
	}

	resp, err := c.doRequest("POST", "/api/files/list", reqBody)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &files); err != nil {
		return nil, err
	}

	return files, nil
}

// FileDelete deletes a file
func (c *Client) FileDelete(path string) error {
	reqBody := map[string]interface{}{
		"path": path,
	}

	resp, err := c.doRequest("POST", "/api/files/delete", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// FileCopy copies a file
func (c *Client) FileCopy(sourcePath, targetPath string) error {
	reqBody := map[string]interface{}{
		"source_path": sourcePath,
		"target_path": targetPath,
	}

	resp, err := c.doRequest("POST", "/api/files/copy", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// FileMove moves a file
func (c *Client) FileMove(sourcePath, targetPath string) error {
	reqBody := map[string]interface{}{
		"source_path": sourcePath,
		"target_path": targetPath,
	}

	resp, err := c.doRequest("POST", "/api/files/move", reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}
