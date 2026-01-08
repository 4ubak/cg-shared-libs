# CG Shared Libraries

Общие библиотеки для всех микросервисов CG Platform.

## Установка

```bash
go get gitlab.com/xakpro/cg-shared-libs@latest
```

## Пакеты

| Пакет | Описание |
|-------|----------|
| `config` | Загрузка конфигураций из YAML и env |
| `logger` | Структурированное логирование (zap) |
| `postgres` | PostgreSQL клиент с пулом соединений |
| `redis` | Redis клиент |
| `kafka` | Kafka producer/consumer |
| `jwt` | JWT токены |
| `grpc` | gRPC server/client helpers |

## Использование

```go
import (
    "gitlab.com/xakpro/cg-shared-libs/logger"
    "gitlab.com/xakpro/cg-shared-libs/postgres"
    "gitlab.com/xakpro/cg-shared-libs/redis"
    "gitlab.com/xakpro/cg-shared-libs/kafka"
)

// Logger
logger.Init(cfg.Logger)
logger.Info("Starting service", zap.String("name", "my-service"))

// PostgreSQL
db, err := postgres.New(ctx, cfg.Postgres)

// Redis  
rdb, err := redis.New(ctx, cfg.Redis)

// Kafka Producer
producer := kafka.NewProducer(cfg.Kafka, "my-topic")
producer.Publish(ctx, "key", event)
```

## Версионирование

Используем семантическое версионирование:
- `v1.x.x` - стабильная версия
- Breaking changes = новая major версия

