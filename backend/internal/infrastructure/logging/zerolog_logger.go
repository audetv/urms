// backend/internal/infrastructure/logging/zerolog_logger.go
package logging

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/audetv/urms/internal/core/ports"
	"github.com/rs/zerolog"
)

// ZerologLogger реализует ports.Logger используя zerolog
type ZerologLogger struct {
	logger zerolog.Logger
}

// NewZerologLogger создает новый structured logger
func NewZerologLogger(level string, format string) *ZerologLogger {
	// Настраиваем уровень логирования
	zerologLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		zerologLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(zerologLevel)

	// Настраиваем output в зависимости от формата
	var logger zerolog.Logger
	if format == "json" {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// Console output для development
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	return &ZerologLogger{
		logger: logger,
	}
}

// getCallerInfo возвращает информацию о вызывающем коде
func (l *ZerologLogger) getCallerInfo() string {
	// Пропускаем 4 кадра: getCallerInfo -> wrapper -> actual log method -> caller
	pc, file, line, ok := runtime.Caller(4)
	if !ok {
		return "unknown:0"
	}

	// Извлекаем имя функции
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	// Укорачиваем путь файла
	shortFile := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			shortFile = file[i+1:]
			break
		}
	}

	return shortFile + ":" + funcName + ":" + string(rune(line))
}

// getRequestID извлекает correlation ID из context
func (l *ZerologLogger) getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Проверяем стандартный header
	if requestID, ok := ctx.Value("X-Request-ID").(string); ok {
		return requestID
	}

	// Проверяем наш кастомный key
	if requestID, ok := ctx.Value(ports.CorrelationIDKey).(string); ok {
		return requestID
	}

	return ""
}

// Debug логирует сообщение с уровнем DEBUG
func (l *ZerologLogger) Debug(ctx context.Context, msg string, fields ...interface{}) {
	logger := l.logger.Debug().
		Str("caller", l.getCallerInfo()).
		Str("correlation_id", l.getRequestID(ctx))

	l.addFields(logger, fields...)
	logger.Msg(msg)
}

// Info логирует сообщение с уровнем INFO
func (l *ZerologLogger) Info(ctx context.Context, msg string, fields ...interface{}) {
	logger := l.logger.Info().
		Str("caller", l.getCallerInfo()).
		Str("correlation_id", l.getRequestID(ctx))

	l.addFields(logger, fields...)
	logger.Msg(msg)
}

// Warn логирует сообщение с уровнем WARN
func (l *ZerologLogger) Warn(ctx context.Context, msg string, fields ...interface{}) {
	logger := l.logger.Warn().
		Str("caller", l.getCallerInfo()).
		Str("correlation_id", l.getRequestID(ctx))

	l.addFields(logger, fields...)
	logger.Msg(msg)
}

// Error логирует сообщение с уровнем ERROR
func (l *ZerologLogger) Error(ctx context.Context, msg string, fields ...interface{}) {
	logger := l.logger.Error().
		Str("caller", l.getCallerInfo()).
		Str("correlation_id", l.getRequestID(ctx))

	l.addFields(logger, fields...)
	logger.Msg(msg)
}

// addFields добавляет structured fields к логгеру
func (l *ZerologLogger) addFields(logger *zerolog.Event, fields ...interface{}) {
	if len(fields) == 0 {
		return
	}

	// Обрабатываем fields в формате key-value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			logger.Interface(key, fields[i+1])
		}
	}
}

// WithContext создает логгер с обогащенным context
func (l *ZerologLogger) WithContext(ctx context.Context) context.Context {
	return l.logger.WithContext(ctx)
}
