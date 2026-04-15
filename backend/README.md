# Harém Brasil API

Go backend API for Harém Brasil platform.

## Architecture

Based on BACKEND.md specification:
- **Router**: Chi
- **Database**: PostgreSQL (pgx)
- **Cache/Sessions**: Redis
- **Auth**: JWT with refresh tokens
- **Port**: 40080 (behind reverse proxy)

## Project Structure

```
backend/
├── cmd/api/main.go           # Entry point
├── internal/
│   ├── httpapi/              # HTTP server & handlers
│   │   ├── server.go         # Router setup
│   │   ├── responses.go      # JSON response helpers
│   │   ├── jwt.go            # Token generation
│   │   └── handlers_*.go     # Route handlers
│   └── middleware/           # HTTP middleware
│       ├── logger.go
│       ├── auth.go
│       ├── ratelimit.go
│       └── maxbody.go
├── migrations/               # SQL migrations
└── go.mod
```

## API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | /api/v1/auth/register | No | Create account |
| POST | /api/v1/auth/login | No | Login |
| POST | /api/v1/auth/refresh | No | Refresh token |
| POST | /api/v1/auth/logout | Yes | Logout |
| GET | /api/v1/me | Yes | Get current user |
| PATCH | /api/v1/me | Yes | Update profile |
| GET | /api/v1/users | Yes | List users |
| GET | /api/v1/users/{id} | Yes | Get user |
| GET | /api/v1/posts | Yes | List posts |
| POST | /api/v1/posts | Yes | Create post |
| GET | /api/v1/posts/{id} | Yes | Get post |
| PATCH | /api/v1/posts/{id} | Yes | Update post |
| DELETE | /api/v1/posts/{id} | Yes | Delete post |
| POST | /api/v1/posts/{id}/like | Yes | Like post |
| GET | /api/v1/forum/categories | Yes | List categories |
| GET | /api/v1/forum/topics | Yes | List topics |
| POST | /api/v1/forum/topics | Yes | Create topic |
| GET | /api/v1/chat/rooms | Yes | List chat rooms |
| POST | /api/v1/chat/rooms | Yes | Create room |
| GET | /api/v1/notifications | Yes | List notifications |
| GET | /api/v1/plans | Yes | List subscription plans |
| POST | /api/v1/subscriptions | Yes | Subscribe |
| POST | /api/v1/creator/apply | Yes (creator) | Apply as creator |
| GET | /api/v1/admin/users | Yes (admin) | Admin user list |
| GET | /api/v1/admin/stats | Yes (admin) | Platform stats |
| GET | /health | No | Health check |

## Setup

### 1. Install dependencies

```bash
cd backend
go mod tidy
```

### 2. Setup PostgreSQL

```bash
# Create database
sudo -u postgres psql -c "CREATE DATABASE harem;"
sudo -u postgres psql -c "CREATE USER harem WITH PASSWORD 'harem';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE harem TO harem;"

# Run migrations
psql -d harem -f migrations/001_initial_schema.sql
```

### 3. Setup Redis

```bash
# Arch Linux
sudo pacman -S redis
sudo systemctl enable --now redis
```

### 4. Environment Variables

```bash
export PORT=40080
export DATABASE_URL="postgres://harem:harem@localhost:5432/harem?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="your-secret-key-min-32-chars-long"
```

### 5. Build & Run

```bash
# Build binary
go build -o harem-api ./cmd/api

# Run migrations
./harem-api migrate

# Start server (development)
./harem-api serve

# Production with flags
./harem-api serve -port=40080 -jwt-secret="your-secret"
```

## CLI Commands

| Command | Description | Flags |
|---------|-------------|-------|
| `serve` | Start API server | `-port`, `-redis`, `-jwt-secret` |
| `migrate` | Run database migrations | `-dir` (default: migrations) |

## Systemd Service

Create `/etc/systemd/system/harem-api.service`:

```ini
[Unit]
Description=Harém Brasil API
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=harem
WorkingDirectory=/opt/harem
ExecStart=/opt/harem/harem-api serve
Restart=always
Environment="PORT=40080"
Environment="DATABASE_URL=postgres://harem:password@localhost/harem"
Environment="REDIS_URL=redis://localhost:6379/0"
Environment="JWT_SECRET=your-secret-key"

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now harem-api
```

## Security Headers (Nginx)

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:40080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # Rate limiting
    limit_req zone=api burst=20 nodelay;
}
```

## Development

```bash
# Run tests
go test ./...

# Format code
go fmt ./...

# Lint
go vet ./...
```
