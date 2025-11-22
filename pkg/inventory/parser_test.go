package inventory

import (
	"os"
	"testing"
)

func TestParseINI(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*testing.T, *Inventory)
	}{
		{
			name: "simple host",
			content: `[webservers]
web1 ansible_host=192.168.1.10`,
			wantErr: false,
			check: func(t *testing.T, inv *Inventory) {
				group := inv.Groups["webservers"]
				if group == nil {
					t.Fatal("webservers group not found")
				}
				if len(group.Hosts) != 1 || group.Hosts[0] != "web1" {
					t.Errorf("Expected 1 host named web1, got %v", group.Hosts)
				}
				host := inv.Hosts["web1"]
				if host == nil {
					t.Fatal("web1 host not found")
				}
				if host.Vars["ansible_host"] != "192.168.1.10" {
					t.Errorf("Expected ansible_host=192.168.1.10, got %v", host.Vars["ansible_host"])
				}
			},
		},
		{
			name: "multiple hosts",
			content: `[webservers]
web1 ansible_host=192.168.1.10
web2 ansible_host=192.168.1.11`,
			wantErr: false,
			check: func(t *testing.T, inv *Inventory) {
				group := inv.Groups["webservers"]
				if group == nil {
					t.Fatal("webservers group not found")
				}
				if len(group.Hosts) != 2 {
					t.Errorf("Expected 2 hosts, got %d", len(group.Hosts))
				}
			},
		},
		{
			name: "group variables",
			content: `[webservers]
web1 ansible_host=192.168.1.10

[webservers:vars]
http_port=80
domain=example.com`,
			wantErr: false,
			check: func(t *testing.T, inv *Inventory) {
				group := inv.Groups["webservers"]
				if group == nil {
					t.Fatal("webservers group not found")
				}
				if group.Vars["http_port"] != "80" {
					t.Errorf("Expected http_port=80, got %v", group.Vars["http_port"])
				}
				if group.Vars["domain"] != "example.com" {
					t.Errorf("Expected domain=example.com, got %v", group.Vars["domain"])
				}
			},
		},
		{
			name: "comments and empty lines",
			content: `# This is a comment
[webservers]
# Another comment
web1 ansible_host=192.168.1.10

; Semicolon comment
web2 ansible_host=192.168.1.11`,
			wantErr: false,
			check: func(t *testing.T, inv *Inventory) {
				group := inv.Groups["webservers"]
				if group == nil {
					t.Fatal("webservers group not found")
				}
				if len(group.Hosts) != 2 {
					t.Errorf("Expected 2 hosts, got %d", len(group.Hosts))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpfile, err := os.CreateTemp("", "inventory-*.ini")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			// 解析
			parser := NewINIParser()
			inv, err := parser.Parse(tmpfile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, inv)
			}
		})
	}
}

func TestMergeHostVars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hostname string
		want     map[string]interface{}
	}{
		{
			name: "host vars override group vars",
			content: `[webservers]
web1 ansible_host=192.168.1.10 env=prod

[webservers:vars]
env=dev
http_port=80`,
			hostname: "web1",
			want: map[string]interface{}{
				"ansible_host": "192.168.1.10",
				"env":          "prod",
				"http_port":    "80",
			},
		},
		{
			name: "group vars override all vars",
			content: `[all:vars]
env=dev
domain=example.com

[webservers]
web1 ansible_host=192.168.1.10

[webservers:vars]
env=prod`,
			hostname: "web1",
			want: map[string]interface{}{
				"ansible_host": "192.168.1.10",
				"env":          "prod",
				"domain":       "example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tmpfile, err := os.CreateTemp("", "inventory-*.ini")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			// 解析
			parser := NewINIParser()
			inv, err := parser.Parse(tmpfile.Name())
			if err != nil {
				t.Fatal(err)
			}

			host := inv.Hosts[tt.hostname]
			if host == nil {
				t.Fatalf("Host %s not found", tt.hostname)
			}

			for key, wantVal := range tt.want {
				if gotVal, exists := host.Vars[key]; !exists || gotVal != wantVal {
					t.Errorf("Host.Vars[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
