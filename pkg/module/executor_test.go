package module

import (
	"testing"
)

func TestExecutor_executeDebug(t *testing.T) {
	executor := NewExecutor()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantMsg string
		wantErr bool
		changed bool
	}{
		{
			name: "debug with msg",
			args: map[string]interface{}{
				"msg": "Hello, World!",
			},
			wantMsg: "Hello, World!",
			wantErr: false,
			changed: false,
		},
		{
			name: "debug with var",
			args: map[string]interface{}{
				"var":      "test_var",
				"test_var": "test_value",
			},
			wantMsg: "test_var: test_value",
			wantErr: false,
			changed: false,
		},
		{
			name:    "debug with no args",
			args:    map[string]interface{}{},
			wantMsg: "Debug output",
			wantErr: false,
			changed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.executeDebug(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeDebug() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result.Changed != tt.changed {
				t.Errorf("executeDebug() Changed = %v, want %v", result.Changed, tt.changed)
			}
			if result.Msg != tt.wantMsg {
				t.Errorf("executeDebug() Msg = %v, want %v", result.Msg, tt.wantMsg)
			}
		})
	}
}

func TestParseModuleArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		want    map[string]string
		wantErr bool
	}{
		{
			name: "simple string args",
			args: map[string]interface{}{
				"src":  "/tmp/file.txt",
				"dest": "/etc/file.txt",
			},
			want: map[string]string{
				"src":  "/tmp/file.txt",
				"dest": "/etc/file.txt",
			},
			wantErr: false,
		},
		{
			name: "mixed types",
			args: map[string]interface{}{
				"path":  "/tmp/test",
				"mode":  "0644",
				"owner": "root",
			},
			want: map[string]string{
				"path":  "/tmp/test",
				"mode":  "0644",
				"owner": "root",
			},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    map[string]interface{}{},
			want:    map[string]string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make(map[string]string)
			for k, v := range tt.args {
				if str, ok := v.(string); ok {
					got[k] = str
				}
			}

			for key, wantVal := range tt.want {
				if gotVal, exists := got[key]; !exists {
					t.Errorf("parseModuleArgs()[%s] missing", key)
				} else if gotVal != wantVal {
					t.Errorf("parseModuleArgs()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestResult_Failed(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   bool
	}{
		{
			name: "success result",
			result: &Result{
				Changed: true,
				Failed:  false,
				Msg:     "Success",
			},
			want: false,
		},
		{
			name: "failed result",
			result: &Result{
				Changed: false,
				Failed:  true,
				Msg:     "Error occurred",
			},
			want: true,
		},
		{
			name: "unchanged result",
			result: &Result{
				Changed: false,
				Failed:  false,
				Msg:     "No changes",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Failed; got != tt.want {
				t.Errorf("Result.Failed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_Changed(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   bool
	}{
		{
			name: "changed result",
			result: &Result{
				Changed: true,
				Failed:  false,
			},
			want: true,
		},
		{
			name: "unchanged result",
			result: &Result{
				Changed: false,
				Failed:  false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Changed; got != tt.want {
				t.Errorf("Result.Changed = %v, want %v", got, tt.want)
			}
		})
	}
}
