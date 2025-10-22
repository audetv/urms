.
├── cmd
│   ├── api
│   │   └── main.go
│   ├── migrate
│   │   └── main.go
│   └── test-imap
│       └── main.go
├── docker-compose.db.yml
├── go.mod
├── go.sum
├── internal
│   ├── config
│   │   └── config.go
│   ├── core
│   │   ├── architecture_compliance_test.go
│   │   ├── domain
│   │   │   ├── email_errors.go
│   │   │   ├── email.go
│   │   │   ├── email_headers.go
│   │   │   ├── email_headers_test.go
│   │   │   ├── email_test.go
│   │   │   ├── entities.go
│   │   │   ├── future_models.go
│   │   │   ├── id_generator.go
│   │   │   ├── task.go
│   │   │   ├── task_test.go
│   │   │   ├── types.go
│   │   │   └── valueobjects.go
│   │   ├── ports
│   │   │   ├── background.go
│   │   │   ├── common.go
│   │   │   ├── email_contract_test.go
│   │   │   ├── email_gateway.go
│   │   │   ├── email_repository_contract_test.go
│   │   │   ├── errors.go
│   │   │   ├── health.go
│   │   │   ├── message_processor.go
│   │   │   ├── message_processor_test.go
│   │   │   ├── migration_gateway.go
│   │   │   ├── repositories.go
│   │   │   └── services.go
│   │   └── services
│   │       ├── background_manager.go
│   │       ├── customer_service.go
│   │       ├── customer_service_test.go
│   │       ├── dummy_processor.go
│   │       ├── email_service.go
│   │       ├── email_service_test.go
│   │       ├── task_service.go
│   │       ├── task_service_test.go
│   │       └── test_utils.go
│   └── infrastructure
│       ├── common
│       │   └── id
│       │       └── uuid_generator.go
│       ├── email
│       │   ├── address_normalizer.go
│       │   ├── basic_test.go
│       │   ├── contract_test.go
│       │   ├── email_gateway_health_adapter.go
│       │   ├── email_poller_task.go
│       │   ├── errors.go
│       │   ├── header_filter.go
│       │   ├── header_filter_test.go
│       │   ├── imap
│       │   │   ├── client.go
│       │   │   ├── config.go
│       │   │   └── utils.go
│       │   ├── imap_adapter.go
│       │   ├── imap_poller.go
│       │   ├── imap_search_test.go
│       │   ├── integration_test.go
│       │   ├── message_processor.go
│       │   ├── message_processor_integration_test.go
│       │   ├── mime_parser.go
│       │   ├── retry_manager.go
│       │   └── utils.go
│       ├── health
│       │   └── aggregator.go
│       ├── http
│       │   ├── dto
│       │   │   ├── requests.go
│       │   │   └── responses.go
│       │   ├── handlers
│       │   │   ├── customer_handler.go
│       │   │   ├── health_handler.go
│       │   │   └── task_handler.go
│       │   ├── health_handler.go
│       │   └── middleware
│       │       ├── context.go
│       │       ├── cors.go
│       │       ├── error_handler.go
│       │       ├── logging.go
│       │       ├── recovery.go
│       │       └── setup.go
│       ├── logging
│       │   ├── test_logger.go
│       │   └── zerolog_logger.go
│       └── persistence
│           ├── email
│           │   ├── factory.go
│           │   ├── inmemory
│           │   │   └── inmemory_repo.go
│           │   └── postgres
│           │       ├── models.go
│           │       ├── postgres_health_check.go
│           │       ├── postgres_repository.go
│           │       ├── postgres_repository_test.go
│           │       └── postgres_schema.sql
│           ├── migrations
│           │   ├── factory.go
│           │   ├── postgres
│           │   │   ├── 001_create_email_tables.sql
│           │   │   └── 002_add_email_indexes.sql
│           │   ├── postgres_migrator.go
│           │   ├── postgres_transaction_manager.go
│           │   └── sql_analyzer.go
│           └── task
│               └── inmemory
│                   ├── customer_repository.go
│                   ├── task_repository.go
│                   └── user_repository.go
├── Makefile
├── PROJECT_STRUCTURE.md
├── README.md
└── README_MIGRATIONS.md

30 directories, 97 files
