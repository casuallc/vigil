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

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/manifoldco/promptui"
)

// SelectOption 用于携带值和显示标签
type SelectOption struct {
	Value string // 实际返回值
	Label string // 显示文本
}

// SelectConfig 配置
type SelectConfig struct {
	Label     string
	Items     interface{} // 支持 []string 或 []SelectOption
	PageSize  int
	HideHelp  bool
	IsVimMode bool
}

// Select 返回选中项的索引、值（string）和错误
func Select(cfg SelectConfig) (int, string, error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if cfg.PageSize <= 0 {
		cfg.PageSize = 10
	}

	var displayItems []string
	var values []string // 用于返回原始值（当使用 SelectOption 时）

	switch v := cfg.Items.(type) {
	case []string:
		displayItems = v
		values = v // 值和显示一致
	case []SelectOption:
		for _, opt := range v {
			displayItems = append(displayItems, opt.Label)
			values = append(values, opt.Value)
		}
	default:
		return -1, "", fmt.Errorf("unsupported items type: %T, only []string or []SelectOption allowed", cfg.Items)
	}

	if len(displayItems) == 0 {
		return -1, "", fmt.Errorf("items list is empty")
	}

	// 美观模板（避免 {}，使用安全符号）
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ . | bold }}",
		Inactive: "  {{ . }}",
		Selected: " {{ . | bold }}",
	}

	prompt := promptui.Select{
		Label:     cfg.Label,
		Items:     displayItems,
		Size:      cfg.PageSize,
		Templates: templates,
		HideHelp:  cfg.HideHelp,
		IsVimMode: cfg.IsVimMode,
	}

	resultChan := make(chan struct {
		index int
		value string
		err   error
	}, 1)

	go func() {
		index, _, err := prompt.Run() // 第二个返回值是 displayItems[index]
		var selectedValue string
		if err == nil && index >= 0 {
			selectedValue = values[index]
		}
		resultChan <- struct {
			index int
			value string
			err   error
		}{index, selectedValue, err}
	}()

	select {
	case <-sigChan:
		fmt.Fprintln(os.Stderr, "\noption cancelled（Ctrl+C）")
		return -1, "", fmt.Errorf("user cancelled")
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, promptui.ErrInterrupt) {
				return -1, "", fmt.Errorf("user cancelled")
			}
			return -1, "", fmt.Errorf("select failed: %w", result.err)
		}
		return result.index, result.value, nil
	}
}
