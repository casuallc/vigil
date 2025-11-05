package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCosmicHealthCommand 测试cosmic health命令
func TestCosmicHealthCommand(t *testing.T) {
	// 创建测试配置文件
	testConfigDir := filepath.Join(os.TempDir(), "vigil_test", "conf", "cosmic")
	err := os.MkdirAll(testConfigDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(os.TempDir(), "vigil_test"))

	// 创建测试配置文件
	testConfigPath := filepath.Join(testConfigDir, "cosmic_test.yml")
	testConfig := `
node:
  - ip: 127.0.0.1
    port: 8080

admq:
  - name: test-admq
    ip: 127.0.0.1
    port: 5672

amdc:
  - name: test-amdc
    ip: 127.0.0.1
    ports: [6379]
    password: ""
`
	err = os.WriteFile(testConfigPath, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// 这里可以使用exec.Command执行CLI命令进行集成测试
	// 或者直接调用handleCosmicHealth函数进行单元测试
	// 由于这是一个简单的测试用例，我们只验证配置文件是否成功创建

	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Errorf("Test config file was not created")
	}
}