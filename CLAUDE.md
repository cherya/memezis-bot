# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**memezis-bot** is a Telegram bot written in Go that manages meme submissions and publications. The bot connects to an external gRPC-based Memezis service for meme processing and uses Redis for caching, queueing, and ban management.

## Development Commands

### Building and Running
```bash
# Build the bot binary
make build

# Run tests
make test

# Build and run locally
make run

# Clean build artifacts
make clean
```

### Testing
```bash
# Run all tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/bot
```

### Docker
```bash
# Build Docker image
docker build -t memezis-bot .

# The Dockerfile expects production.env to be present
# Container runs: /app/bin/memezisbot --env=production.env
```

### Running with Custom Environment
```bash
# Default uses local.env
./bin/memezisbot

# Specify custom env file
./bin/memezisbot --env=production.env
```

## Architecture

### Core Components

1. **Bot Entry Point** (`cmd/memezis-bot/main.go`)
   - Initializes environment from `.env` files (via `--env` flag)
   - Sets up Redis connection pool
   - Creates gRPC connection to Memezis service with Bearer token auth
   - Instantiates Telegram Bot API client
   - Wires together all components (queue manager, word generator, ban hammer, user cache)

2. **Bot Package** (`internal/bot/`)
   - `MemezisBot` struct is the central coordinator
   - Uses worker pool pattern: 5 message workers + 5 callback workers
   - Rate limiting: 3-second delay between messages (except callbacks)
   - Handles three message types:
     - Private messages (DMs to the bot)
     - Admin channel messages (suggestion/moderation channel)
     - Chat messages (group conversations)
   - Callback queries handle inline keyboard interactions (approve/reject memes)

3. **Memezis gRPC Client**
   - External service for meme processing
   - Uses custom `tokenAuth` struct implementing `credentials.PerRPCCredentials`
   - Bearer token sent in "authorization" header

4. **Queue Manager** (`pkg/queue` from `github.com/cherya/memezis`)
   - Redis-backed job queue named "memezis"
   - Consumer: `ShaurmemesConsumer` processes queued items every 10 seconds

5. **BanHammer** (`internal/banhammer/`)
   - Redis-based user banning system
   - Temporary bans (default 300 seconds) via `Ban()`
   - Permanent bans via `Permaban()`
   - Key pattern: `ban:<user_id>`

6. **DailyWord Generator** (`internal/dailyword/`)
   - Fetches random Wikipedia article titles
   - Caches per-user-per-day in Redis with 24h expiry
   - Filters out biographical/geographic articles
   - Key pattern: `dailyword:<uid>:<DD.MM.YYYY>`

7. **UserCache** (`internal/userchache/`)
   - Stores mapping between post IDs and user info (name and ID)

### Message Flow

```
Telegram Update → updatesFromPoll() → Select on updates channel
                     ↓
         Message or CallbackQuery?
                     ↓
              ┌──────┴──────┐
              │             │
         messages chan   callback chan
              │             │
         5 workers      5 workers
              │             │
        messageWorker  callbackWorker
              │             │
         Route based    Handle inline
         on chat type   keyboard actions
```

### Configuration (`.env` file)

Required environment variables:
- `TG_BOT_TOKEN` - Telegram bot API token
- `SUGGESTION_CHANNEL_ID` - Channel for meme suggestions/moderation
- `PUBLICATION_CHANNEL_ID` - Channel for approved meme publication
- `REDIS_ADDRESS` - Redis server address
- `REDIS_PASSWORD` - Redis password
- `MEMEZIS_ADDRESS` - gRPC service address
- `MEMEZIS_TOKEN` - Bearer token for Memezis service
- `OWNER_ID` - Bot owner's Telegram user ID
- `DEBUG` - Enable debug logging (optional)

### Deployment

GitHub Actions workflow (`.github/workflows/docker-deploy.yml`):
1. **Test**: Build Docker image on push to master
2. **Push**: Tag and push to GitHub Container Registry
3. **Deploy**: Trigger webhook deployment (via `WEBHOOK_URL` and `WEBHOOK_SECRET`)

The webhook likely pulls and restarts the container on the production server.

## Key Design Patterns

- **Worker Pool**: Multiple goroutines process messages/callbacks concurrently
- **Dependency Injection**: All dependencies passed to `NewBot()` constructor
- **Interface-based Design**: `Ban` and `UserCache` interfaces allow easy testing/mocking
- **Rate Limiting**: Time-based ticker prevents Telegram API rate limit violations
- **Idempotency**: `sync.Map` (`callbackAtom`) prevents duplicate callback processing
