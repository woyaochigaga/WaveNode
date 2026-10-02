package logger

import (
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// redactingCore 在日志编码前统一处理消息和结构化字段，避免调用方漏掉显式脱敏。
// 它只负责安全边界，不改变底层 core 的级别、采样和输出位置。
type redactingCore struct {
	core zapcore.Core
}

func newRedactingCore(core zapcore.Core) zapcore.Core {
	return &redactingCore{core: core}
}

func (c *redactingCore) Enabled(level zapcore.Level) bool {
	return c.core.Enabled(level)
}

func (c *redactingCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactingCore{core: c.core.With(redactZapFields(fields))}
}

func (c *redactingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return checked
	}
	return checked.AddCore(entry, c)
}

func (c *redactingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	entry.Message = logredact.RedactText(entry.Message)
	return c.core.Write(entry, redactZapFields(fields))
}

func (c *redactingCore) Sync() error {
	return c.core.Sync()
}

// redactZapFields 先使用 Zap 自己的字段编码语义取值，再递归脱敏。
// Namespace 字段必须原样保留，否则后续字段会丢失原有的 JSON 层级。
func redactZapFields(fields []zapcore.Field) []zapcore.Field {
	if len(fields) == 0 {
		return nil
	}

	redacted := make([]zapcore.Field, 0, len(fields))
	for _, field := range fields {
		if field.Type == zapcore.NamespaceType || field.Type == zapcore.SkipType {
			redacted = append(redacted, field)
			continue
		}

		encoder := zapcore.NewMapObjectEncoder()
		field.AddTo(encoder)
		values := logredact.RedactMap(encoder.Fields)
		if field.Key == "" {
			redacted = append(redacted, zap.Inline(redactedInlineFields(values)))
			continue
		}
		value, ok := values[field.Key]
		if !ok {
			// 极少数字段编码器不会写回原键；保留字段比静默丢日志更可控。
			redacted = append(redacted, field)
			continue
		}
		redacted = append(redacted, zap.Any(field.Key, value))
	}
	return redacted
}

type redactedInlineFields map[string]any

func (fields redactedInlineFields) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	for key, value := range fields {
		if err := encoder.AddReflected(key, value); err != nil {
			return err
		}
	}
	return nil
}
