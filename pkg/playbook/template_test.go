package playbook

import (
	"testing"
)

func TestJinja2TemplateEngine_RenderString(t *testing.T) {
	engine := NewJinja2TemplateEngine()
	defer engine.Close() // 确保释放资源

	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "Hello {{ name }}",
			context:  map[string]interface{}{"name": "World"},
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "nested variable",
			template: "Server: {{ config.host }}:{{ config.port }}",
			context: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "localhost",
					"port": 8080,
				},
			},
			want:    "Server: localhost:8080",
			wantErr: false,
		},
		{
			name:     "upper filter",
			template: "{{ name | upper }}",
			context:  map[string]interface{}{"name": "test"},
			want:     "TEST",
			wantErr:  false,
		},
		{
			name:     "lower filter",
			template: "{{ name | lower }}",
			context:  map[string]interface{}{"name": "TEST"},
			want:     "test",
			wantErr:  false,
		},
		{
			name:     "if-else",
			template: "{% if enabled %}ON{% else %}OFF{% endif %}",
			context:  map[string]interface{}{"enabled": true},
			want:     "ON",
			wantErr:  false,
		},
		{
			name:     "for loop",
			template: "{% for item in items %}{{ item }}{% if not loop.last %},{% endif %}{% endfor %}",
			context:  map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			want:     "a,b,c",
			wantErr:  false,
		},
		{
			name:     "array first",
			template: "{{ items | first }}",
			context:  map[string]interface{}{"items": []interface{}{"first", "second"}},
			want:     "first",
			wantErr:  false,
		},
		{
			name:     "array last",
			template: "{{ items | last }}",
			context:  map[string]interface{}{"items": []interface{}{"first", "last"}},
			want:     "last",
			wantErr:  false,
		},
		{
			name:     "array length",
			template: "{{ items | length }}",
			context:  map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			want:     "3",
			wantErr:  false,
		},
		{
			name:     "no template syntax",
			template: "Plain text",
			context:  map[string]interface{}{},
			want:     "Plain text",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RenderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJinja2TemplateEngine_EvaluateCondition(t *testing.T) {
	engine := NewJinja2TemplateEngine()
	defer engine.Close() // 确保释放资源

	tests := []struct {
		name      string
		condition string
		context   map[string]interface{}
		want      bool
		wantErr   bool
	}{
		{
			name:      "simple equality",
			condition: "environment == 'production'",
			context:   map[string]interface{}{"environment": "production"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "simple inequality",
			condition: "environment != 'development'",
			context:   map[string]interface{}{"environment": "production"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "greater than",
			condition: "port > 1024",
			context:   map[string]interface{}{"port": 8080},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "less than",
			condition: "port < 65535",
			context:   map[string]interface{}{"port": 8080},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "and condition - true",
			condition: "environment == 'production' and port == 8080",
			context:   map[string]interface{}{"environment": "production", "port": 8080},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "and condition - false",
			condition: "environment == 'production' and port == 3000",
			context:   map[string]interface{}{"environment": "production", "port": 8080},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "or condition - true",
			condition: "environment == 'production' or environment == 'staging'",
			context:   map[string]interface{}{"environment": "production"},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "not condition - true",
			condition: "not debug",
			context:   map[string]interface{}{"debug": false},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "not condition - false",
			condition: "not debug",
			context:   map[string]interface{}{"debug": true},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "empty condition",
			condition: "",
			context:   map[string]interface{}{},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "nested variable access",
			condition: "config.debug == false",
			context:   map[string]interface{}{"config": map[string]interface{}{"debug": false}},
			want:      true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateCondition(tt.condition, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJinja2TemplateEngine_RenderArgs(t *testing.T) {
	engine := NewJinja2TemplateEngine()
	defer engine.Close() // 确保释放资源

	tests := []struct {
		name    string
		args    map[string]interface{}
		context map[string]interface{}
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "simple string rendering",
			args: map[string]interface{}{
				"msg": "Hello {{ name }}",
			},
			context: map[string]interface{}{"name": "World"},
			want: map[string]interface{}{
				"msg": "Hello World",
			},
			wantErr: false,
		},
		{
			name: "nested map rendering",
			args: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "{{ server }}",
					"port": 8080,
				},
			},
			context: map[string]interface{}{"server": "localhost"},
			want: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "localhost",
					"port": 8080,
				},
			},
			wantErr: false,
		},
		{
			name: "no template syntax",
			args: map[string]interface{}{
				"content": "plain text",
			},
			context: map[string]interface{}{},
			want: map[string]interface{}{
				"content": "plain text",
			},
			wantErr: false,
		},
		{
			name: "array with templates",
			args: map[string]interface{}{
				"items": []interface{}{"{{ item1 }}", "{{ item2 }}"},
			},
			context: map[string]interface{}{"item1": "first", "item2": "second"},
			want: map[string]interface{}{
				"items": []interface{}{"first", "second"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderArgs(tt.args, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// 简单比较字符串类型的值
			for key := range tt.want {
				if wantStr, ok := tt.want[key].(string); ok {
					if gotStr, ok := got[key].(string); ok {
						if gotStr != wantStr {
							t.Errorf("RenderArgs()[%s] = %v, want %v", key, gotStr, wantStr)
						}
					}
				}
			}
		})
	}
}

// TestJinja2TemplateEngine_TildeOperator 测试波浪号操作符（Jinja2 字符串连接）
func TestJinja2TemplateEngine_TildeOperator(t *testing.T) {
	engine := NewJinja2TemplateEngine()
	defer engine.Close()

	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple tilde",
			template: "{{ firstname ~ ' ' ~ lastname }}",
			context:  map[string]interface{}{"firstname": "John", "lastname": "Doe"},
			want:     "John Doe",
			wantErr:  false,
		},
		{
			name:     "tilde with string literals",
			template: "{{ 'hello' ~ ' ' ~ 'world' }}",
			context:  map[string]interface{}{},
			want:     "hello world",
			wantErr:  false,
		},
		{
			name:     "tilde for dynamic variable name",
			template: "{{ 'ansible_' ~ interface }}",
			context:  map[string]interface{}{"interface": "eth0"},
			want:     "ansible_eth0",
			wantErr:  false,
		},
		{
			name:     "tilde with filter",
			template: "{{ (app_name ~ '-' ~ version) | upper }}",
			context:  map[string]interface{}{"app_name": "myapp", "version": "1.0.0"},
			want:     "MYAPP-1.0.0",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RenderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJinja2TemplateEngine_InlineConditional 测试内联条件表达式
func TestJinja2TemplateEngine_InlineConditional(t *testing.T) {
	engine := NewJinja2TemplateEngine()
	defer engine.Close()

	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "inline conditional - true",
			template: "{{ 'enabled' if debug else 'disabled' }}",
			context:  map[string]interface{}{"debug": true},
			want:     "enabled",
			wantErr:  false,
		},
		{
			name:     "inline conditional - false",
			template: "{{ 'enabled' if debug else 'disabled' }}",
			context:  map[string]interface{}{"debug": false},
			want:     "disabled",
			wantErr:  false,
		},
		{
			name:     "inline conditional with variables",
			template: "Status: {{ 'ON' if status == 'active' else 'OFF' }}",
			context:  map[string]interface{}{"status": "active"},
			want:     "Status: ON",
			wantErr:  false,
		},
		{
			name:     "inline conditional with filter",
			template: "{{ ('yes' if enabled else 'no') | upper }}",
			context:  map[string]interface{}{"enabled": true},
			want:     "YES",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.RenderString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RenderString() = %v, want %v", got, tt.want)
			}
		})
	}
}
