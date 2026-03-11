package main

import (
	"fmt"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogSink 实现了 klog/v2 的 LogSink 接口
type ZapLogSink struct {
	logger *zap.Logger
	name   string
	keys   []interface{}
}

// 确保 ZapLogSink 实现了 logr.LogSink 接口 (klog v2 基于 logr)
var _ logr.LogSink = &ZapLogSink{}

// Init 初始化 (klog 启动时调用)
func (z *ZapLogSink) Init(info logr.RuntimeInfo) {
	// 通常不需要做什么，zap logger 已经初始化好了
}

// Enabled 检查是否启用指定级别的日志
// klog 的 verbosity (V level) 对应 zap 的 level
// klog V(0) = Info, V(1) = Debug, V(2+) = More Debug
func (z *ZapLogSink) Enabled(level int) bool {
	// 将 klog 的 level 转换为 zap 的 level 进行检查
	// klog: 0=Info, 1=Debug, 2+=More Debug
	// Zap: -1=Info, 0=Debug, 1+=More Debug (默认配置下)
	// 这里简单处理：如果 klog level > 0，我们视为 Debug 或更低优先级
	var zapLevel zapcore.Level
	if level == 0 {
		zapLevel = zapcore.InfoLevel
	} else {
		zapLevel = zapcore.DebugLevel
	}

	return z.logger.Core().Enabled(zapLevel)
}

// Info 记录信息日志
func (z *ZapLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	z.logWithFields(zapcore.InfoLevel, msg, keysAndValues...)
}

// Error 记录错误日志
func (z *ZapLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	// klog 的 Error 通常带有 err 对象
	fields := make([]zap.Field, 0, len(keysAndValues)/2+1)
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	fields = append(fields, z.mapKeysToFields(keysAndValues)...)

	z.logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// WithValues 返回一个新的 Sink，包含额外的键值对
func (z *ZapLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	newKeys := make([]interface{}, len(z.keys)+len(keysAndValues))
	copy(newKeys, z.keys)
	copy(newKeys[len(z.keys):], keysAndValues)

	return &ZapLogSink{
		logger: z.logger,
		name:   z.name,
		keys:   newKeys,
	}
}

// WithName 返回一个新的 Sink，包含名称
func (z *ZapLogSink) WithName(name string) logr.LogSink {
	newName := name
	if z.name != "" {
		newName = z.name + "." + name
	}
	return &ZapLogSink{
		logger: z.logger,
		name:   newName,
		keys:   z.keys,
	}
}

// 辅助函数：将 klog 的 key-value 对转换为 zap 的 Field
func (z *ZapLogSink) mapKeysToFields(keysAndValues []interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			// 奇数个参数，忽略最后一个或作为错误处理
			break
		}
		key, ok := keysAndValues[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keysAndValues[i])
		}
		fields = append(fields, zap.Any(key, keysAndValues[i+1]))
	}

	// 如果有预设的 name，加进去
	if z.name != "" {
		fields = append(fields, zap.String("logger", z.name))
	}

	// 如果有预设的 keys，加进去
	for i := 0; i < len(z.keys); i += 2 {
		if i+1 >= len(z.keys) {
			break
		}
		key, ok := z.keys[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", z.keys[i])
		}
		fields = append(fields, zap.Any(key, z.keys[i+1]))
	}

	return fields
}

func (z *ZapLogSink) logWithFields(level zapcore.Level, msg string, keysAndValues ...interface{}) {
	fields := z.mapKeysToFields(keysAndValues)
	ce := z.logger.Check(level, msg)
	if ce != nil {
		ce.Write(fields...)
	}
}
