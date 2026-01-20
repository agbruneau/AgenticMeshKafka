package database

import (
	"testing"
)

func TestNewDBPool_InvalidConfig(t *testing.T) {
	// Test with empty host
	cfg := PoolConfig{
		Host:     "",
		Port:     5432,
		Database: "edalab",
		User:     "edalab",
		Password: "password",
	}

	_, err := NewDBPool(cfg)
	if err == nil {
		t.Error("NewDBPool() should return error for empty host")
	}

	// Test with invalid port
	cfg = PoolConfig{
		Host:     "localhost",
		Port:     0,
		Database: "edalab",
		User:     "edalab",
		Password: "password",
	}

	_, err = NewDBPool(cfg)
	if err == nil {
		t.Error("NewDBPool() should return error for invalid port")
	}

	// Test with empty database
	cfg = PoolConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "",
		User:     "edalab",
		Password: "password",
	}

	_, err = NewDBPool(cfg)
	if err == nil {
		t.Error("NewDBPool() should return error for empty database")
	}

	// Test with empty user
	cfg = PoolConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "edalab",
		User:     "",
		Password: "password",
	}

	_, err = NewDBPool(cfg)
	if err == nil {
		t.Error("NewDBPool() should return error for empty user")
	}
}

func TestPoolConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     PoolConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: PoolConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "edalab",
				User:     "edalab",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "empty host",
			cfg: PoolConfig{
				Host:     "",
				Port:     5432,
				Database: "edalab",
				User:     "edalab",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "port 0",
			cfg: PoolConfig{
				Host:     "localhost",
				Port:     0,
				Database: "edalab",
				User:     "edalab",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "port too high",
			cfg: PoolConfig{
				Host:     "localhost",
				Port:     70000,
				Database: "edalab",
				User:     "edalab",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "empty database",
			cfg: PoolConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "",
				User:     "edalab",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "empty user",
			cfg: PoolConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "edalab",
				User:     "",
				Password: "password",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPoolConfig_ConnectionString(t *testing.T) {
	cfg := PoolConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "edalab",
		User:     "edalab",
		Password: "secret",
	}

	connStr := cfg.ConnectionString()
	expected := "postgres://edalab:secret@localhost:5432/edalab"

	if connStr != expected {
		t.Errorf("ConnectionString() = %q, want %q", connStr, expected)
	}
}

func TestPool_Interface(t *testing.T) {
	// Verify that Pool interface exists with expected methods
	var _ Pool = (*DBPool)(nil)
}
