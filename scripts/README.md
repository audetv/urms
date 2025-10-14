# URMS-OS Validation Scripts

Набор скриптов для автоматической проверки архитектурного соответствия и качества кода.

## 📋 Доступные скрипты

### 🚀 Полная проверка
```bash
./scripts/full_validation.sh
```
**Проверяет:**

- ✅ Архитектурное соответствие (Hexagonal Architecture)
- ✅ Модульные тесты core слоя
- ✅ Компиляцию всех компонентов
- ✅ Форматирование кода
- ✅ Чистоту доменного слоя

**Использование:**

- Перед коммитом кода
- При приемке новых фич
- Для проверки архитектурной целостности

### ⚡ Быстрая проверка
```bash
./scripts/quick_check.sh
```
**Проверяет:**

- ✅ Основные тесты
- ✅ Компиляцию
- ✅ Базовую архитектуру

**Использование:**

- Во время разработки
- Для быстрой проверки изменений
- В CI/CD пайплайнах

### 📐 Архитектурный аудит
```bash
./scripts/architecture_audit.sh
```
**Проверяет:**

- ✅ Core слой не импортирует infrastructure
- ✅ Наличие интерфейсов в core/ports/
- ✅ Чистоту domain слоя (без внешних зависимостей)
- ✅ Компиляцию infrastructure слоя

## 🎯 Критерии успеха
### Архитектурные требования:
- Core слой: Только бизнес-логика, без импортов infrastructure
- Domain слой: Только стандартная библиотека Go
- Ports слой: Полный набор интерфейсов
- Infrastructure слой: Корректно реализует порты

### Тестовые требования:
- Domain тесты: 100% покрытие бизнес-логики
- Services тесты: 100% покрытие use cases
- Компиляция: Все компоненты собираются без ошибок
## 🔧 Интеграция с разработкой
### Pre-commit хук (рекомендуется)
```bash
# .git/hooks/pre-commit
#!/bin/bash
./scripts/quick_check.sh
```
### CI/CD пайплайн
```yaml
# .github/workflows/ci.yml
jobs:
  validation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: ./scripts/full_validation.sh
```
### Локальная разработка      
```bash
# Быстрая проверка во время coding
./scripts/quick_check.sh

# Полная проверка перед PR
./scripts/full_validation.sh
```
## 📊 Интерпретация результатов
### ✅ Успех
🚀 URMS-OS Full Validation Suite
=================================
✅ Architecture compliance  
✅ Domain layer tests  
✅ Services layer tests  
✅ Core layer compilation  
✅ Infrastructure layer compilation  
✅ Test application compilation  
✅ Code formatting  
✅ Domain layer purity  

### ❌ Типичные проблемы и решения:
**"core/ imports infrastructure/"**

- Уберите импорты infrastructure из core/
- Используйте dependency injection

**"No interfaces in core/ports/"**
- Создайте интерфейсы в core/ports/

**"Domain layer has external dependencies"**
- Уберите внешние импорты из domain/
- Используйте абстракции через порты

**"Infrastructure compilation failed"**
- Проверьте реализацию интерфейсов
- Убедитесь в корректности импортов
## 🛠️ Технические детали
### Зависимости
- Go 1.20+
- Bash 4.0+
- Стандартные утилиты Unix (find, grep, etc.)

### Структура скриптов
```text
scripts/
├── full_validation.sh    # Полная проверка
├── quick_check.sh        # Быстрая проверка
├── architecture_audit.sh # Архитектурный аудит
└── README.md            # Эта документация
```
### Добавление новых проверок
1. Добавьте проверку в architecture_audit.sh
2. Обновите full_validation.sh
3. Протестируйте на текущем коде
4. Обновите документацию

## 📞 Поддержка
**При проблемах с скриптами:**

1. Проверьте что находитесь в корне проекта
1. Убедитесь что скрипты исполняемые (chmod +x scripts/*.sh)
1. Проверьте что Go модуль инициализирован в backend/

**Связанные документы:**

- [Архитектурные принципы](../ARCHITECTURE_PRINCIPLES.md)
- [Руководство по разработке](../docs/development/DEVELOPMENT_GUIDE.md)