package playbook

import (
	"os"
	"testing"

	"github.com/jimyag/ansigo/pkg/inventory"
)

func TestVariableManager_GetContext(t *testing.T) {
	// 创建测试 inventory manager
	invMgr := inventory.NewManager()

	// 创建临时 inventory 文件
	tmpfile, err := os.CreateTemp("", "inventory-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `[webservers]
web1 ansible_host=192.168.1.10 env=prod

[webservers:vars]
http_port=8080`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// 加载 inventory
	if err := invMgr.Load(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		playVars map[string]interface{}
		regVars  map[string]map[string]interface{}
		hostname string
		want     map[string]interface{}
	}{
		{
			name:     "basic context with inventory vars",
			playVars: map[string]interface{}{},
			regVars:  map[string]map[string]interface{}{},
			hostname: "web1",
			want: map[string]interface{}{
				"ansible_host":       "web1", // Special variable is set to hostname
				"inventory_hostname": "web1",
			},
		},
		{
			name: "play vars override inventory vars",
			playVars: map[string]interface{}{
				"env": "staging",
			},
			regVars:  map[string]map[string]interface{}{},
			hostname: "web1",
			want: map[string]interface{}{
				"env":                "staging",
				"inventory_hostname": "web1",
			},
		},
		{
			name: "registered vars highest priority",
			playVars: map[string]interface{}{
				"env": "staging",
			},
			regVars: map[string]map[string]interface{}{
				"web1": {
					"env":         "test",
					"test_result": "success",
				},
			},
			hostname: "web1",
			want: map[string]interface{}{
				"env":                "test",
				"test_result":        "success",
				"inventory_hostname": "web1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewVariableManager(invMgr)
			vm.SetPlayVars(tt.playVars)
			vm.registeredVars = tt.regVars

			got := vm.GetContext(tt.hostname)

			for key, wantVal := range tt.want {
				if gotVal, exists := got[key]; !exists {
					t.Errorf("GetContext()[%s] missing, want %v", key, wantVal)
				} else if gotVal != wantVal {
					t.Errorf("GetContext()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestVariableManager_SetHostVar(t *testing.T) {
	invMgr := inventory.NewManager()

	// 创建临时 inventory 文件
	tmpfile, err := os.CreateTemp("", "inventory-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `[webservers]
web1 ansible_host=192.168.1.10`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	if err := invMgr.Load(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	vm := NewVariableManager(invMgr)

	tests := []struct {
		name     string
		hostname string
		varName  string
		value    interface{}
		wantVal  interface{}
	}{
		{
			name:     "set simple string",
			hostname: "web1",
			varName:  "test_output",
			value:    "success",
			wantVal:  "success",
		},
		{
			name:     "set integer",
			hostname: "web1",
			varName:  "return_code",
			value:    0,
			wantVal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm.SetHostVar(tt.hostname, tt.varName, tt.value)

			context := vm.GetContext(tt.hostname)
			if gotVal, exists := context[tt.varName]; !exists {
				t.Errorf("SetHostVar() did not set %s", tt.varName)
			} else if gotVal != tt.wantVal {
				t.Errorf("SetHostVar() = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestVariableManager_SetPlayVars(t *testing.T) {
	invMgr := inventory.NewManager()

	// 创建临时 inventory 文件
	tmpfile, err := os.CreateTemp("", "inventory-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `[webservers]
web1 ansible_host=192.168.1.10`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	if err := invMgr.Load(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	vm := NewVariableManager(invMgr)

	tests := []struct {
		name     string
		playVars map[string]interface{}
		checkKey string
		wantVal  interface{}
	}{
		{
			name: "set simple play vars",
			playVars: map[string]interface{}{
				"app_name":    "myapp",
				"app_version": "1.0.0",
			},
			checkKey: "app_name",
			wantVal:  "myapp",
		},
		{
			name: "set integer play var",
			playVars: map[string]interface{}{
				"port": 8080,
			},
			checkKey: "port",
			wantVal:  8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm.SetPlayVars(tt.playVars)

			context := vm.GetContext("web1")
			if gotVal, exists := context[tt.checkKey]; !exists {
				t.Errorf("SetPlayVars() did not set %s", tt.checkKey)
			} else if gotVal != tt.wantVal {
				t.Errorf("SetPlayVars()[%s] = %v, want %v", tt.checkKey, gotVal, tt.wantVal)
			}
		})
	}
}

func TestVariableManager_GetHostVar(t *testing.T) {
	invMgr := inventory.NewManager()

	// 创建临时 inventory 文件
	tmpfile, err := os.CreateTemp("", "inventory-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `[webservers]
web1 ansible_host=192.168.1.10 env=prod`

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	if err := invMgr.Load(tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	vm := NewVariableManager(invMgr)
	vm.SetPlayVars(map[string]interface{}{
		"play_var": "play_value",
	})
	vm.SetHostVar("web1", "reg_var", "reg_value")

	tests := []struct {
		name     string
		hostname string
		varName  string
		want     interface{}
		wantOk   bool
	}{
		{
			name:     "get play var",
			hostname: "web1",
			varName:  "play_var",
			want:     "play_value",
			wantOk:   true,
		},
		{
			name:     "get registered var",
			hostname: "web1",
			varName:  "reg_var",
			want:     "reg_value",
			wantOk:   true,
		},
		{
			name:     "get non-existent var",
			hostname: "web1",
			varName:  "nonexistent",
			want:     nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := vm.GetHostVar(tt.hostname, tt.varName)
			if ok != tt.wantOk {
				t.Errorf("GetHostVar() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("GetHostVar() = %v, want %v", got, tt.want)
			}
		})
	}
}
