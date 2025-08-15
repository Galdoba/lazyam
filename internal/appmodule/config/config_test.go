package config_test

import (
	"fmt"
	"testing"

	"github.com/Galdoba/lazyam/internal/config"
)

func TestLoad(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Errorf("test load config failed: %v", err)
	}
	data, err := cfg.Marshal()
	if err != nil {
		t.Errorf("test marshalling failed: %v", err)
	}
	if err := cfg.Unmarshal(data); err != nil {
		t.Errorf("test unmarshalling failed: %v", err)
	}
	fmt.Println("current config:")
	fmt.Println(string(data))

	// tests := []struct {
	// 	name    string // description of this test case
	// 	want    *config.Config
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		got, gotErr := config.Load()
	// 		if gotErr != nil {
	// 			if !tt.wantErr {
	// 				t.Errorf("Load() failed: %v", gotErr)
	// 			}
	// 			return
	// 		}
	// 		if tt.wantErr {
	// 			t.Fatal("Load() succeeded unexpectedly")
	// 		}
	// 		// TODO: update the condition below to compare got with tt.want.
	// 		if true {
	// 			t.Errorf("Load() = %v, want %v", got, tt.want)
	// 		}
	// 	})
	// }
}
