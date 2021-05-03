# TWHelp cron

Features:

- Adds automatically new servers.
- Fetches and updates server data (players, tribes, ODA, ODD, ODS, OD, conquers, configs).
- Saves daily player/tribe stats, player/tribe history, tribe changes, player name changes, server stats.
- Clears database from old player/tribe stats, player/tribe history.

## Development

**Required env variables:**

```
DB_USER=your_db_user
DB_NAME=your_db_name
DB_PORT=5432
DB_HOST=your_db_host
DB_PASSWORD=your_db_pass

REDIS_ADDR=redis_addr
REDIS_DB=redis_db
REDIS_USER=redis_user
REDIS_PASSWORD=redis_password

RUN_ON_INIT=true|false
LOG_DB_QUERIES=true|false

WORKER_LIMIT=1
```

### Prerequisites

1. Golang
2. PostgreSQL
3. Redis

### Installing

1. Clone this repo.
2. Navigate to the directory where you have cloned this repo.
3. Set the required env variables directly in your system or create .env.local file.
4. go run main.go
