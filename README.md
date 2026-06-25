# gator

A command-line RSS feed aggregator built in Go. Subscribe to your favorite blogs and news feeds, let gator fetch new posts in the background, and browse them whenever you like — all from your terminal.

---

## Features

- **User accounts** — Register multiple users and switch between them with a simple login command
- **Feed management** — Add RSS feeds by URL and give them friendly names
- **Feed following** — Follow or unfollow any feed in the system, per user
- **Automated aggregation** — Run a background scraper that fetches new posts on a configurable interval
- **Post browsing** — View the latest posts from feeds you follow, with an adjustable limit
- **PostgreSQL storage** — All users, feeds, and posts are persisted in a relational database

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| [Go](https://go.dev/dl/) 1.22+ | Tested on 1.26.4 |
| [PostgreSQL](https://www.postgresql.org/) | A running instance with a database you own |
| [Goose](https://github.com/pressly/goose) | For running database migrations (`go install github.com/pressly/goose/v3/cmd/goose@latest`) |

---

## Installation

**Option A — build from source:**

```bash
git clone https://github.com/paakodev/blogaggregator.git
cd blogaggregator
go build -o gator .
```

**Option B — install directly:**

```bash
go install github.com/paakodev/blogaggregator@latest
```

---

## Configuration

gator reads its configuration from `~/.gatorconfig.json`. Create that file before running any commands:

```json
{
  "db_url": "postgres://username:password@localhost:5432/blogaggregator?sslmode=disable",
  "current_user_name": ""
}
```

- **`db_url`** — A standard PostgreSQL connection string pointing to your database.
- **`current_user_name`** — Managed automatically by the `register` and `login` commands; you can leave it empty initially.

---

## Database Setup

Apply the migrations using Goose from the repository root:

```bash
goose -dir sql/schema postgres "YOUR_DB_URL" up
```

This creates the `users`, `feeds`, `feed_follows`, and `posts` tables.

---

## Usage

```
gator <command> [arguments]
```

| Command | Arguments | Auth | Description |
|---------|-----------|------|-------------|
| `help` | — | — | List all available commands |
| `register` | `<username>` | — | Create a new user and log in as them |
| `login` | `<username>` | — | Switch to an existing user |
| `users` | — | — | List all users (current user marked with `*`) |
| `reset` | — | — | Delete all data from the database |
| `agg` | `<interval>` | — | Start the feed scraper (e.g. `30s`, `5m`) |
| `addfeed` | `<name> <url>` | logged in | Add a new feed and follow it automatically |
| `feeds` | — | — | List every feed in the system |
| `follow` | `<url>` | logged in | Follow an existing feed |
| `following` | — | logged in | Show feeds you are currently following |
| `unfollow` | `<url>` | logged in | Unfollow a feed |
| `browse` | `[limit]` | logged in | Show latest posts from followed feeds (default: 2) |

---

## Example Workflow

```bash
# 1. Register a user (you only need to do this once)
gator register alice

# 2. Add a feed and automatically follow it
gator addfeed "Go Blog" "https://go.dev/blog/feed.atom"
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"

# 3. Start the aggregator in one terminal (fetches every 30 seconds)
gator agg 30s

# 4. In another terminal, browse your latest posts
gator browse 10
```

---

## Tech Stack

- **[Go](https://go.dev/)** — Core language
- **[PostgreSQL](https://www.postgresql.org/)** — Data persistence
- **[SQLC](https://sqlc.dev/)** — Type-safe SQL query generation
- **[lib/pq](https://github.com/lib/pq)** — PostgreSQL driver for Go
- **[google/uuid](https://github.com/google/uuid)** — UUID generation for primary keys
