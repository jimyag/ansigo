package playbook

// TemplateEngineInterface 模板引擎接口
type TemplateEngineInterface interface {
	// RenderString 渲染单个字符串
	RenderString(template string, context map[string]interface{}) (string, error)

	// RenderValue 渲染并返回原始值（可能是列表、字典等）
	RenderValue(template string, context map[string]interface{}) (interface{}, error)

	// RenderArgs 渲染模块参数
	RenderArgs(args map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error)

	// EvaluateCondition 评估 when 条件
	EvaluateCondition(condition string, context map[string]interface{}) (bool, error)

	// Close 关闭模板引擎并释放资源
	Close() error
}

// NewDefaultTemplateEngine 创建默认的模板引擎（Jinja2）
func NewDefaultTemplateEngine() TemplateEngineInterface {
	engine := NewJinja2TemplateEngine()
	engine.RegisterCustomFilters()
	return engine
}
