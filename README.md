# tribalwarshelp.com cron

Features:

- Adds automatically new TribalWars servers.
- Fetches TribalWars servers data (players, tribes, ODA, ODD, ODS, OD, conquers, configs).
- Saves daily player/tribe stats, player/tribe history, tribe changes, player name changes, server stats.
- Vacuums the database daily from old player/tribe stats, player/tribe history.

## Development

**Required env variables to run this cron** (you can set them directly in your system or create .env.development file):

```
DB_USER=your_db_user
DB_NAME=your_db_name
DB_PORT=5432
DB_HOST=your_db_host
DB_PASSWORD=your_db_pass
```

### Prerequisites

1. Golang
2. PostgreSQL database

### Installing

1. Clone this repo.
2. Navigate to the directory where you have cloned this repo.
3. Set required env variables directly in your system or create .env.development file.
4. go run main.go
