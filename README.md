# Personal Watcher

Personal Watcher is a Go service for monitoring HTTP endpoints.

It periodically checks configured URLs, stores check history in PostgreSQL,
detects health state changes, and sends Telegram notifications when an endpoint
changes from healthy to unhealthy or back.

## Features

- Create, update, list and delete monitored endpoints
- Configurable expected HTTP status
- Configurable check interval for each endpoint
- Enable and disable watches
- Concurrent checks using a worker pool
- Check history stored in PostgreSQL
- Paginated check history API
- Telegram notifications on health state changes
- Telegram bot commands for managing watches
- Graceful shutdown using context cancellation

## How it works

Each watch contains:

- URL
- expected HTTP status
- check interval
- enabled/disabled state

The scheduler periodically queries PostgreSQL for watches that are due for a
check and sends them to a fixed-size worker pool.

Each worker:

1. Gets the previous health state.
2. Performs an HTTP request.
3. Compares the returned status code with the expected status.
4. Stores the check result in PostgreSQL.
5. Sends a Telegram notification if the health state changed.

## HTTP API

### Create a watch

```http
POST /watches
```

Request:

```json
{
  "url": "https://example.com",
  "expected_status": 200,
  "interval_seconds": 60
}
```

### List watches

```http
GET /watches
```

### Get a watch

```http
GET /watches/{id}
```

### Update a watch

```http
PATCH /watches/{id}
```

Example:

```json
{
  "expected_status": 200,
  "interval_seconds": 120,
  "enabled": true
}
```

All fields are optional, but at least one field must be provided.

### Delete a watch

```http
DELETE /watches/{id}
```

### Get check history

```http
GET /watches/{id}/checks
```

Optional query parameters:

- `limit` — number of results, from 1 to 100; default: 20
- `offset` — pagination offset; default: 0

Example:

```http
GET /watches/1/checks?limit=20&offset=0
```

## Telegram Bot

The Telegram bot supports:

- `/list`
- `/add <url> <expected_status> <interval_seconds>`
- `/delete <id>`
- `/enable <id>`
- `/disable <id>`
- `/help`

The bot only accepts commands from the configured Telegram chat.

## Configuration

The application uses environment variables:

- `DATABASE_URL` — PostgreSQL connection string
- `TELEGRAM_BOT_TOKEN` — Telegram bot token
- `TELEGRAM_CHAT_ID` — allowed Telegram chat ID
- `PORT` — HTTP server port; defaults to `8080`

## Tech Stack

- Go
- PostgreSQL
- pgx
- net/http
- Telegram Bot API

## Current scope

Personal Watcher currently monitors endpoint health based on the expected HTTP
status code.

Possible future improvements include:

- response body checks
- text matching
- JSON value checks
- response change detection
- additional notification channels