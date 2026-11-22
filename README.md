# Шаблон сервиса AI-ассистентов

В репозитории четыре Go-сервиса. Gateway принимает HTTP-запросы и WebSocket-соединения. Auth отвечает за учётные записи и токены. AI обслуживает ассистентов, диалоги и каналы. Database предоставляет gRPC API поверх PostgreSQL.

## Запуск через Docker Compose

В корне проекта выполните в PowerShell:

```powershell
foreach ($service in 'database-service', 'auth-service', 'ai-service', 'api-gateway') {
    if (!(Test-Path "$service/.env")) { Copy-Item "$service/.env.example" "$service/.env" }
}
```

Задайте одинаковый непустой `ACCESS_SECRET` в auth и gateway и отдельный `REFRESH_SECRET` в auth. В AI укажите `OPENAI_API_KEY` для ответов модели. Для Telegram задайте `TELEGRAM_WEBHOOK_SECRET` и публичный HTTPS-адрес gateway в `WEBHOOK_BASE_URL`. Для Twilio тот же адрес задаётся в `BASE_URL`. Номер и учётные данные Twilio передаются при регистрации ассистента.

```sh
docker compose up --build
```

Gateway доступен на `http://localhost:8085`, проверка состояния — `GET /health`. Compose собирает приложения из Go-исходников и создаёт отдельный том для базы шаблона.

Для dev-конфигурации используется один `.env` в корне. Поместите в него `ACCESS_SECRET`, `REFRESH_SECRET` и нужные переменные AI из примера. Команда `docker compose -f docker-compose.dev.yml up --build` поднимает gateway на порту 8080.

## Возможности

| Сценарий | Точка входа через gateway |
| --- | --- |
| Регистрация, вход, обновление и завершение сессии | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` |
| Создание ассистента для API, Telegram или Twilio | `POST /ai/webhooks/{api,telegram,twilio}/register` |
| Переписка по API | `POST /webhooks/telegram` с API-токеном ассистента и полем формы `user_message` |
| Настройки ассистента и список чатов | `/ai/assistants/…`, `/ai/chats/…` |
| История диалога через WebSocket | `/ai/websocket/get_conversation?chat=<chat_id>` |
| Список и блокировка контактов для SMS | `GET` и `POST /ai/blocked-customers/` |
| Статистика по ассистенту | `GET /ai/analytics?assistant_id=<id>` |

Для кабинета передавайте access-токен в `Authorization: Bearer …`. При регистрации Telegram-ассистента AI устанавливает callback. При регистрации Twilio-ассистента AI возвращает URL, который нужно указать как обработчик входящих SMS у номера Twilio.

Статистика строится по чатам в PostgreSQL. Она показывает число начатых диалогов, число помеченных завершёнными (`is_end`) и распределение начатых диалогов по дням. Настройки поведения LLM берутся из конфигурации конкретного ассистента.

## База данных и проверки

При первом запуске database-service создаёт таблицы пользователей, refresh-токенов, ассистентов, конфигураций Twilio, чатов, сообщений и заблокированных контактов. Для очистки копии старой базы подготовлен отдельный [SQL-скрипт](database-service/migrations/manual/remove_retired_features.sql). Он удаляет данные прежних интеграций и запускается вручную. Штатный запуск применяет только SQL-файлы верхнего уровня `database-service/migrations`.

У каждого сервиса свой `go.mod`. Для проверки выполните `go test ./...` из каталога каждого сервиса. Инструкция по генерации gRPC-контракта находится в [database-service/proto](database-service/proto/README.md).
