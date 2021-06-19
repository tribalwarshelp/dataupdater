# tribalwarshelp.com cron

- Adds automatically new servers.
- Fetches and updates server data (players, tribes, ODA, ODD, ODS, OD, conquers, configs).
- Saves daily player/tribe stats, player/tribe history, tribe changes, player name changes, server stats.
- Clears database from old player/tribe stats, player/tribe history.

## Development

### Prerequisites

1. Golang
2. PostgreSQL
3. Redis

### Installation
**Required ENV variables:**

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

1. Clone this repo.
```
git clone git@github.com:tribalwarshelp/cron.git
```
2. Open the folder with this project in a terminal.
3. Set the required env variables directly in your system or create .env.local file.
4. Run the app.
```
go run main.go
```

## License

Distributed under the MIT License. See ``LICENSE`` for more information.

## Contact

Dawid Wysokiński - [contact@dwysokinski.me](mailto:contact@dwysokinski.me)
