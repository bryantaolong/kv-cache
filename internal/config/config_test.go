package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Address != ":6379" {
		t.Errorf("expected address :6379, got %s", cfg.Address)
	}
	if cfg.DataDir != "./data" {
		t.Errorf("expected data-dir ./data, got %s", cfg.DataDir)
	}
	if cfg.NoPersist != false {
		t.Errorf("expected no-persist false, got %v", cfg.NoPersist)
	}
	if cfg.RewriteSize != 64*1024*1024 {
		t.Errorf("expected rewrite-size 67108864, got %d", cfg.RewriteSize)
	}
	if cfg.MaxMemory != 0 {
		t.Errorf("expected max-memory 0, got %d", cfg.MaxMemory)
	}
	if cfg.EvictionPolicy != "noeviction" {
		t.Errorf("expected eviction-policy noeviction, got %s", cfg.EvictionPolicy)
	}
	if cfg.AppendOnlyPolicy != "everysec" {
		t.Errorf("expected append-only-policy everysec, got %s", cfg.AppendOnlyPolicy)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  string
		wantErr bool
	}{
		{"valid noeviction", "noeviction", false},
		{"valid allkeys-lru", "allkeys-lru", false},
		{"valid volatile-lru", "volatile-lru", false},
		{"valid allkeys-random", "allkeys-random", false},
		{"valid volatile-random", "volatile-random", false},
		{"valid uppercase", "ALLKEYS-LRU", false},
		{"invalid policy", "invalid", true},
		{"empty policy", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Address:          ":6379",
				DataDir:          "./data",
				EvictionPolicy:   tt.policy,
				AppendOnlyPolicy: "everysec", // 使用有效值，避免干扰 eviction 测试
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSyncPolicyValidate(t *testing.T) {
	tests := []struct {
		name       string
		syncPolicy string
		wantErr    bool
	}{
		{"valid always", "always", false},
		{"valid everysec", "everysec", false},
		{"valid no", "no", false},
		{"valid uppercase", "ALWAYS", false},
		{"invalid policy", "invalid", true},
		{"empty policy", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Address:          ":6379",
				DataDir:          "./data",
				EvictionPolicy:   "noeviction",
				AppendOnlyPolicy: tt.syncPolicy,
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 测试用例1：有效的配置文件
	t.Run("valid config file", func(t *testing.T) {
		configContent := `
address: ":6380"
data-dir: "./test-data"
no-persist: true
rewrite-size: 134217728
max-memory: 104857600
eviction-policy: "allkeys-lru"
`
		configFile := filepath.Join(tmpDir, "config.yaml")
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		cfg, err := LoadFromFile(configFile)
		if err != nil {
			t.Fatalf("LoadFromFile() error = %v", err)
		}

		if cfg.Address != ":6380" {
			t.Errorf("expected address :6380, got %s", cfg.Address)
		}
		if cfg.DataDir != "./test-data" {
			t.Errorf("expected data-dir ./test-data, got %s", cfg.DataDir)
		}
		if cfg.NoPersist != true {
			t.Errorf("expected no-persist true, got %v", cfg.NoPersist)
		}
		if cfg.RewriteSize != 134217728 {
			t.Errorf("expected rewrite-size 134217728, got %d", cfg.RewriteSize)
		}
		if cfg.MaxMemory != 104857600 {
			t.Errorf("expected max-memory 104857600, got %d", cfg.MaxMemory)
		}
		if cfg.EvictionPolicy != "allkeys-lru" {
			t.Errorf("expected eviction-policy allkeys-lru, got %s", cfg.EvictionPolicy)
		}
		// append-only-policy 使用默认值
		if cfg.AppendOnlyPolicy != "everysec" {
			t.Errorf("expected append-only-policy everysec, got %s", cfg.AppendOnlyPolicy)
		}
	})

	// 测试用例2：不存在的配置文件（应使用默认值）
	t.Run("non-existent config file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tmpDir, "non-existent.yaml")

		cfg, err := LoadFromFile(nonExistentFile)
		if err != nil {
			t.Fatalf("LoadFromFile() with non-existent file should not error, got: %v", err)
		}

		// 验证使用的是默认值
		defaultCfg := DefaultConfig()
		if cfg.Address != defaultCfg.Address {
			t.Errorf("expected default address %s, got %s", defaultCfg.Address, cfg.Address)
		}
		if cfg.DataDir != defaultCfg.DataDir {
			t.Errorf("expected default data-dir %s, got %s", defaultCfg.DataDir, cfg.DataDir)
		}
	})

	// 测试用例3：无效的配置值
	t.Run("invalid config value", func(t *testing.T) {
		configContent := `
eviction-policy: "invalid-policy"
`
		configFile := filepath.Join(tmpDir, "invalid-config.yaml")
		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		_, err := LoadFromFile(configFile)
		if err == nil {
			t.Error("LoadFromFile() with invalid policy should return error")
		}
	})
}

func TestLoaderWithFlags(t *testing.T) {
	loader := NewLoader()

	// 测试绑定命令行参数
	loader.BindFlag("data-dir", "/custom/data")
	loader.BindFlag("max-memory", int64(1024))

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DataDir != "/custom/data" {
		t.Errorf("expected data-dir /custom/data, got %s", cfg.DataDir)
	}
	if cfg.MaxMemory != 1024 {
		t.Errorf("expected max-memory 1024, got %d", cfg.MaxMemory)
	}
}
