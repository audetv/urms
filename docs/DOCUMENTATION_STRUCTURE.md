## 🏗️ Полная структура документации проекта
```text
urms/
├── docs/
│   ├── specifications/              📁 Спецификации (стабильные)
│   │   ├── URMS_SPECIFICATION.md
│   │   ├── EMAIL_MODULE_SPEC.md
│   │   ├── ARCHITECTURE_PRINCIPLES.md
│   │   └── API_SPECIFICATION.md
│   ├── development/                 📁 Процесс разработки
│   │   ├── reports/                 📁 Индивидуальные отчеты
│   │   │   ├── 2024-01-15_email_module_phase1a.md
│   │   │   ├── 2025-10-14_email_module_phase1a_refactoring.md
│   │   │   ├── 2025-10-16_email_module_phase1b_completion.md
│   │   │   ├── README.md
│   │   │   └── template.md
│   │   ├── issues/                  📁 Активные проблемы
│   │   │   ├── 2025-10-16_imap_hang_large_mailboxes.md
│   │   │   └── BUG_REPORT_TEMPLATE.md
│   │   ├── plans/                   📁 Планы разработки
│   │   │   ├── PHASE_1B_PLAN.md
│   │   │   ├── PHASE_1C_PLAN.md
│   │   │   └── TEMPLATE_PLAN.md
│   │   ├── decisions/               📁 Архитектурные решения   ✅ NEW
│   │   │   ├── ADR-001-hexagonal-architecture.md               ✅ NEW
│   │   │   ├── ADR-002-imap-timeout-strategy.md                ✅ NEW
│   │   │   ├── ADR_EXPLANATION.md                              ✅ NEW
│   │   │   └── template.md                                     ✅ NEW
│   │   ├── AI_CODING_GUIDELINES.md
│   │   ├── ARCHITECTURE_VALIDATION_RULES.md
│   │   ├── CODE_REVIEW_TEMPLATE.md
│   │   ├── COMMIT_MESSAGE_GEN_INSTRUCTION.md
│   │   ├── COMMIT_MESSAGE_GENERATOR.md
│   │   ├── CURRENT_STATUS.md        # Текущий статус (всегда актуальный)
│   │   ├── ISSUE_MANAGEMENT.md      # Процесс управления проблемами ✅ NEW
│   │   ├── ROADMAP.md               # Дорожная карта
│   │   ├── DECISIONS.md             # Архитектурные решения
│   │   └── DEVELOPMENT_GUIDE.md     # Руководство разработчика
├── backend/                         💻 Исходный код
└── README.md