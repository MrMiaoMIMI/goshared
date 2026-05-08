// Package logger 提供统一的结构化日志能力。
//
// 使用方通常只需要在服务启动时调用 Configure 初始化全局 logger，
// 在业务代码中调用 Info/Warn/Error 等方法，并传入 context.Context 让
// 请求 trace_id 自动写入日志。
package logger
