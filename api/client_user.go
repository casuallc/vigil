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
	"fmt"
	"net/http"
)

// RegisterUser registers a new user
func (c *Client) RegisterUser(username, password, email, role string) error {
	userData := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
		"role":     role,
	}

	resp, err := c.doRequest("POST", "/api/users/register", userData)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return c.errorFromResponse(resp)
	}

	return nil
}

// RegisterUserExtended registers a new user with extended profile fields
func (c *Client) RegisterUserExtended(username, password, email, role, nickname, avatar, region string) error {
	userData := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
		"role":     role,
		"nickname": nickname,
		"avatar":   avatar,
		"region":   region,
	}

	resp, err := c.doRequest("POST", "/api/users/register", userData)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return c.errorFromResponse(resp)
	}

	return nil
}

// UpdateUserExtended updates user information with extended profile fields
func (c *Client) UpdateUserExtended(username, email, role, password, nickname, avatar, region string) error {
	updateData := make(map[string]string)
	if email != "" {
		updateData["email"] = email
	}
	if role != "" {
		updateData["role"] = role
	}
	if password != "" {
		updateData["password"] = password
	}
	if nickname != "" {
		updateData["nickname"] = nickname
	}
	if avatar != "" {
		updateData["avatar"] = avatar
	}
	if region != "" {
		updateData["region"] = region
	}

	if len(updateData) == 0 {
		return fmt.Errorf("no fields to update")
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/users/%s", username), updateData)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// ListUsers lists all users
func (c *Client) ListUsers() ([]User, error) {
	var users []User
	resp, err := c.doRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUser gets user details
func (c *Client) GetUser(username string) (User, error) {
	var user User
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/users/%s", username), nil)
	if err != nil {
		return user, err
	}

	if resp.StatusCode != http.StatusOK {
		return user, c.errorFromResponse(resp)
	}

	if err := c.getJSONResponse(resp, &user); err != nil {
		return user, err
	}

	return user, nil
}

// UpdateUser updates user information
func (c *Client) UpdateUser(username, email, role, password string) error {
	updateData := make(map[string]string)
	if email != "" {
		updateData["email"] = email
	}
	if role != "" {
		updateData["role"] = role
	}
	if password != "" {
		updateData["password"] = password
	}

	if len(updateData) == 0 {
		return fmt.Errorf("no fields to update")
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/users/%s", username), updateData)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(username string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/users/%s", username), nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}

// GetUserConfigs gets user configuration
func (c *Client) GetUserConfigs(username string) (string, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/users/%s/configs", username), nil)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", c.errorFromResponse(resp)
	}

	var result map[string]string
	if err := c.getJSONResponse(resp, &result); err != nil {
		return "", err
	}

	return result["configs"], nil
}

// UpdateUserConfigs updates user configuration
func (c *Client) UpdateUserConfigs(username, configs string) error {
	reqBody := map[string]string{
		"configs": configs,
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/users/%s/configs", username), reqBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}

	return nil
}
