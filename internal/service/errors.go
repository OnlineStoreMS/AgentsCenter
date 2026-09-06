package service

import "errors"

var (
	ErrNotFound      = errors.New("资源不存在")
	ErrBadRequest    = errors.New("参数错误")
	ErrAgentAuth     = errors.New("Agent 鉴权失败")
	ErrNoAgentOnline = errors.New("暂无匹配的在线 Agent")
	ErrJobState      = errors.New("任务状态不允许此操作")
)
