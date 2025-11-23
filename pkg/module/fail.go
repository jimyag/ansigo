package module

import (
	"github.com/jimyag/ansigo/pkg/connection"
)

// FailModule fail 模块实现
// fail 模块用于显式地使任务失败，通常与条件判断配合使用
type FailModule struct{}

// Execute 执行 fail 模块
func (m *FailModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
	result := &Result{
		Failed: true,
	}

	// 获取 msg 参数
	if msgInterface, ok := args["msg"]; ok {
		if msg, ok := msgInterface.(string); ok {
			result.Msg = msg
		} else {
			result.Msg = "Failed as requested"
		}
	} else {
		result.Msg = "Failed as requested"
	}

	return result, nil
}
