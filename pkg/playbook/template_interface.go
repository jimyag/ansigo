package playbook

// TemplateEngineInterface 模板引擎接口
type TemplateEngineInterface interface {
	// RenderString 渲染单个字符串
	RenderString(template string, context map[string]interface{}) (string, error)

	// RenderArgs 渲染模块参数
	RenderArgs(args map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error)

	// EvaluateCondition 评估 when 条件
	EvaluateCondition(condition string, context map[string]interface{}) (bool, error)
}

// NewDefaultTemplateEngine 创建默认的模板引擎（Jinja2）
func NewDefaultTemplateEngine() TemplateEngineInterface {
	engine := NewJinja2TemplateEngine()
	engine.RegisterCustomFilters()
	return engine
}
