# Инструменты разработки

Эта директория содержит вспомогательные утилиты для работы с кодовой базой.

## `skeleton.go`

Генерирует «скелет» Go-файла: оставляет объявления структур, функций и методов, но удаляет тела функций и **все комментарии внутри них**. Комментарии **над** объявлениями (например, над `func` или `type`) сохраняются.

### Установка

Утилита использует только стандартную библиотеку Go — дополнительная установка не требуется.

### Использование

Из корня проекта:

```bash
# Вывести скелет в stdout
go run ./tools ./path/to/file.go

# Сохранить скелет в файл
go run ./tools ./internal/core/domain/email.go > email_skeleton.go
```

Или собрать бинарник для частого использования:

```bash
go build -o skel ./tools/skeleton.go
./skel ./cmd/api/main.go > api_skeleton.go
```

### Пример

Исходный код:
```go
// ProcessEmail обрабатывает входящее письмо
func ProcessEmail(msg string) error {
    // Парсим заголовки
    headers := parseHeaders(msg)
    // Валидируем
    if !isValid(headers) {
        return ErrInvalid
    }
    return nil
}
```

Результат:
```go
// ProcessEmail обрабатывает входящее письмо
func ProcessEmail(msg string) error {}
```

> 💡 Полезно для документирования архитектуры, code review или анализа API без шума от реализации.