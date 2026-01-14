# Docker Compose Deployment

Инструкции по развертыванию NewsFlow с помощью Docker Compose.

## Предварительные требования

- Docker Engine 20.10+
- Docker Compose v2.0+

## Сервисы

- **kafka** - Kafka брокер в режиме KRaft (без Zookeeper)
- **account-service** - Сервис управления Telegram аккаунтами и сбора новостей
- **postgres-subscriptions** - PostgreSQL для Subscription Service
- **postgres-news** - PostgreSQL для News Service
- **kafka-ui** - Web UI для мониторинга Kafka (опционально)

## Быстрый старт

### 1. Настройка переменных окружения

```bash
cd deployments
cp .env.example .env
```

Отредактируйте `.env` и заполните обязательные переменные:

```env
# Получите с https://my.telegram.org/apps
TELEGRAM_API_ID=12345678
TELEGRAM_API_HASH=your_api_hash_here

# Telegram аккаунты (номера телефонов)
TELEGRAM_ACCOUNTS=+79001234567,+79007654321
```

### 2. Запуск всех сервисов

```bash
docker compose up -d
```

### 3. Запуск только account-service

```bash
docker compose up -d account-service
```

Команда автоматически запустит зависимости (kafka).

### 4. Просмотр логов

```bash
# Все сервисы
docker compose logs -f

# Только account-service
docker compose logs -f account-service

# Последние 100 строк
docker compose logs --tail=100 account-service
```

### 5. Остановка сервисов

```bash
# Остановка всех сервисов
docker compose down

# Остановка с удалением volumes (ВНИМАНИЕ: удалит все данные)
docker compose down -v
```

## Конфигурация account-service

### Переменные окружения

| Переменная | Описание | Обязательная | Default |
|------------|----------|--------------|---------|
| `TELEGRAM_API_ID` | Telegram API ID | Да | - |
| `TELEGRAM_API_HASH` | Telegram API Hash | Да | - |
| `TELEGRAM_ACCOUNTS` | Список номеров телефонов | Да | - |
| `TELEGRAM_MIN_REQUIRED_ACCOUNTS` | Минимум аккаунтов для запуска | Нет | 1 |
| `NEWS_POLL_INTERVAL` | Интервал сбора новостей | Нет | 30s |
| `NEWS_COLLECTION_TIMEOUT` | Таймаут сбора | Нет | 5m |
| `LOG_LEVEL` | Уровень логирования | Нет | info |

### Volumes

- `account-sessions` - Хранилище Telegram сессий (персистентное)
- `account-logs` - Логи приложения (опционально)

### Зависимости

- **kafka** - для обмена сообщениями с другими сервисами

### Health Check

Сервис проверяется каждые 30 секунд. Проверка считается успешной, если процесс `account-service` запущен.

## Управление

### Пересборка образа

```bash
docker compose build account-service
docker compose up -d account-service
```

### Просмотр статуса

```bash
docker compose ps
```

### Вход в контейнер

```bash
docker compose exec account-service sh
```

### Просмотр volumes

```bash
docker volume ls | grep account
```

### Резервное копирование sessions

```bash
# Создать backup
docker run --rm -v deployments_account-sessions:/data -v $(pwd):/backup alpine tar czf /backup/sessions-backup.tar.gz -C /data .

# Восстановить backup
docker run --rm -v deployments_account-sessions:/data -v $(pwd):/backup alpine tar xzf /backup/sessions-backup.tar.gz -C /data
```

## Мониторинг

### Kafka UI

После запуска доступен по адресу: http://localhost:8080

Здесь можно просматривать:
- Топики и сообщения
- Consumer groups
- Метрики брокеров

### Logs

```bash
# Real-time logs
docker compose logs -f account-service

# Поиск ошибок
docker compose logs account-service | grep ERROR

# Статистика сбора новостей
docker compose logs account-service | grep "News collection completed"
```

## Troubleshooting

### Сервис не запускается

1. Проверьте логи:
   ```bash
   docker compose logs account-service
   ```

2. Проверьте переменные окружения:
   ```bash
   docker compose config | grep -A 20 account-service
   ```

3. Убедитесь что Kafka запущен:
   ```bash
   docker compose ps kafka
   ```

### Ошибка "No active accounts"

Убедитесь что:
- `TELEGRAM_API_ID` и `TELEGRAM_API_HASH` корректны
- `TELEGRAM_ACCOUNTS` содержит валидные номера телефонов
- Сессии Telegram корректно сохранены в volume

### Ошибка подключения к Kafka

1. Проверьте что Kafka запущен:
   ```bash
   docker compose ps kafka
   docker compose logs kafka
   ```

2. Проверьте сетевое подключение:
   ```bash
   docker compose exec account-service ping -c 3 kafka
   ```

## Production Recommendations

1. **Secrets Management**
   - Не храните `.env` в git
   - Используйте Docker secrets или внешние secret managers
   - Ротируйте credentials регулярно

2. **Monitoring**
   - Настройте мониторинг здоровья сервисов
   - Используйте централизованное логирование
   - Настройте алерты на критические ошибки

3. **Backups**
   - Регулярно делайте backup volume `account-sessions`
   - Храните backups в безопасном месте
   - Тестируйте процедуру восстановления

4. **Resources**
   - Настройте resource limits в docker-compose.yml
   - Мониторьте использование CPU/RAM
   - Масштабируйте по необходимости

5. **Security**
   - Используйте приватные Docker registry
   - Сканируйте образы на уязвимости
   - Обновляйте базовые образы регулярно
