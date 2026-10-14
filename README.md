# FinKoper CRM Казахстан

Мини-CRM для двух сотрудников: учёт клиентов, помесячного количества, оплат и ручных продлений FinKoper и WhatsApp. Backend написан на Go, интерфейс — React/TypeScript/Tailwind, хранилище — PostgreSQL. Собранный frontend встраивается в один Go-бинарник.

## Возможности

- Два равноправных аккаунта с идентификацией исполнителя в истории.
- Клиенты, архивирование без удаления финансовых записей, optimistic locking.
- Раздельные подписки FinKoper и WhatsApp.
- Оплата и продление — независимые транзакции. Оплата покрывает 1–120 расчётных месяцев через отдельные allocations и никогда не меняет дату подписки.
- Связь продления с оплатой и три статуса за выбранный месяц.
- Идемпотентность платежей и продлений, предупреждение о похожей оплате.
- Отмена платежа и продления с причиной. Отмена продления восстанавливает предыдущую дату; более новые операции нужно отменять первыми.
- Массовый ввод количества клиентов, автосохранение, копирование предыдущего месяца, подтверждение и аудируемые корректировки.
- Единая хронология оплат/продлений и неизменяемый аудит.
- HttpOnly-сессии, SameSite=Strict, CSRF, Argon2id и ограничение попыток входа.

## Локальный запуск

Требуются Go 1.23+, Node.js 20+ и PostgreSQL 15+.

1. Скопируйте `.env.example` в `.env` и заполните значения. PowerShell не загружает `.env` автоматически; можно использовать Docker/IDE env-file либо установить переменные в текущей сессии.
2. Примените миграции:

   ```bash
   go run ./cmd/migrate
   ```

3. Создайте или безопасно обновите два начальных аккаунта:

   ```bash
   go run ./cmd/init-users
   ```

4. Соберите интерфейс и запустите приложение:

   ```bash
   cd frontend
   npm ci
   npm run build
   cd ..
   go run ./cmd/crm
   ```

Откройте `http://localhost:8080`. Для разработки frontend можно запустить `npm run dev`; Vite проксирует `/api` на порт 8080.

## Переменные окружения

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | URL отдельной PostgreSQL, включая `sslmode=require` для облачной БД |
| `PORT` | HTTP-порт, обычно `8080` |
| `APP_ENV` | `production` включает флаг Secure у cookie |
| `SESSION_SECRET` | случайная строка минимум 32 символа |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | данные первоначального администратора |
| `PARTNER_EMAIL`, `PARTNER_PASSWORD` | данные первоначального партнёра |

Пароли нужны команде `init-users`; после инициализации в базе находятся только Argon2id-хеши. В production используйте HTTPS, иначе браузер не отправит Secure-cookie.

## Миграции

`migrations/001_init.sql` создаёт типы, таблицы `users`, `sessions`, `clients`, `subscriptions`, `monthly_client_counts`, `monthly_confirmations`, `payments`, `payment_allocations`, `renewals`, `audit_logs`, поисковые индексы и триггеры, запрещающие удаление финансовой истории и изменение аудита. Команда `cmd/migrate` ведёт таблицу `schema_migrations` и применяет SQL-файлы по порядку в отдельных транзакциях.

## Проверка

```bash
go test ./...
cd frontend && npm run typecheck && npm run build
```

Тесты проверяют Argon2id, раздельные услуги, годовое покрытие и календарное прибавление месяцев, включая високосный февраль. Ограничения, внешние ключи, уникальность и запрет удаления дополнительно обеспечиваются PostgreSQL.

## Ubuntu, systemd и Nginx

```bash
cd frontend && npm ci && npm run build && cd ..
go build -o bin/kzcrm ./cmd/crm
sudo useradd --system --home /opt/kzcrm --shell /usr/sbin/nologin kzcrm
sudo mkdir -p /opt/kzcrm/migrations
sudo cp bin/kzcrm /opt/kzcrm/
sudo cp -r migrations/* /opt/kzcrm/migrations/
sudo cp .env /opt/kzcrm/.env
sudo chown -R kzcrm:kzcrm /opt/kzcrm
cd /opt/kzcrm && sudo -u kzcrm env $(cat .env | xargs) /path/to/source/migrate
```

На сервере команду миграции удобнее собрать отдельно: `go build -o bin/migrate ./cmd/migrate`, скопировать её в `/opt/kzcrm` и выполнить `sudo -u kzcrm ./migrate`. Аналогично один раз выполните собранный `init-users`.

Скопируйте `deploy/kzcrm.service` в `/etc/systemd/system/`, затем:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kzcrm
```

Скопируйте `deploy/nginx.conf` в `/etc/nginx/sites-available/kzcrm`, замените домен, включите сайт и настройте TLS через certbot. После этого проверьте `nginx -t` и перезагрузите Nginx.

## Эксплуатационные настройки

Перед публикацией нужно указать домен в Nginx, выпустить TLS-сертификат, заполнить `.env`, применить миграции и выполнить `init-users`. Интеграции с основной системой FinKoper нет: получение оплаты и фактическое продление отмечаются сотрудниками вручную.
