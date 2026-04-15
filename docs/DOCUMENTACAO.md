# Documentação — API Harém Brasil (backend)

Este documento explica **como o backend está organizado**, **o que acontece quando um pedido HTTP chega** e um **passo a passo** para configurar e executar o projeto localmente ou em servidor.

Para referência rápida em inglês (tabela de rotas, systemd, Nginx), veja também [`README.md`](./README.md). Para visão de produto, segurança e evolução futura do ecossistema, veja [`BACKEND.md`](./BACKEND.md).

---

## 1. O que é este projeto

É a **API HTTP** da plataforma Harém Brasil, escrita em **Go 1.23**. Expõe rotas sob `/api/v1`, usa **PostgreSQL** para dados persistentes e **Redis** para rate limiting (e outras funções que forem acrescentadas). A autenticação baseia-se em **JWT** (Bearer) com papéis (RBAC) verificados em middleware.

O binário único (`harem-api`) suporta dois comandos:

| Comando   | Função |
|-----------|--------|
| `serve`   | Arranca o servidor HTTP. |
| `migrate` | Aplica ficheiros SQL em `migrations/` à base de dados (tabela de controlo interna). |

O ponto de entrada está em `cmd/api/main.go`: carrega variáveis de ambiente (ficheiro `.env` opcional via `godotenv`), configura logging JSON (`slog`) e delega para `serve` ou `migrate`.

---

## 2. Como funciona (fluxo técnico)

### 2.1 Arranque do servidor

1. `main.go` lê `DATABASE_URL`, `REDIS_URL`, `PORT`, `JWT_SECRET` (ou flags em `serve`).
2. `httpapi.New` cria um *pool* **pgx** para PostgreSQL e um cliente **go-redis**, faz *ping* aos dois; se falhar, o processo termina com erro.
3. `setupRouter` monta o router **Chi** com middleware global e grupos de rotas.

### 2.2 Pedido HTTP típico

Ordem aproximada dos middlewares globais (ver `internal/httpapi/server.go`):

1. **Request ID** (Chi).
2. **Logger** de pedidos (`internal/middleware/logger.go`).
3. **Recoverer** (Chi) — evita que *panics* derrubem o processo sem resposta.
4. **CORS** — origens permitidas atualmente incluem `*`; cabeçalhos expostos incluem limites de rate limit.
5. **Rate limit** — baseado em Redis (`internal/middleware/ratelimit.go`).
6. **Content-Type** JSON com charset UTF-8.

Rotas em `/api/v1` aplicam ainda:

- **`MaxBodySize`** — limite de corpo por grupo (por exemplo 1 MiB para auth, 10 MiB para posts/utilizadores).
- **`Auth`** — onde indicado: valida JWT HS256, extrai *claims* (`sub`, `email`, `roles`, etc.) e verifica se o utilizador tem um dos **papéis permitidos** na rota.

Grupos com papéis distintos (resumo):

- **Público (sem JWT no middleware):** registo, login, refresh (ver nota abaixo), logout.
- **`user`, `creator`, `moderator`, `admin`:** perfil (`/me`), feed, fórum, chat, notificações, planos, etc.
- **`creator`, `admin`:** rotas de criador (ex.: candidatura, dashboard).
- **`admin`:** `/api/v1/admin/*` (utilizadores, estatísticas, auditoria).

### 2.3 Saúde do serviço

- **`GET /health`** e **`GET /healthz`** — verificam ligação a PostgreSQL e Redis; em falha respondem com estado *degraded* / *unhealthy* e código HTTP 503 quando aplicável.

### 2.4 Estado atual de alguns endpoints de auth

No código atual, **`POST /api/v1/auth/refresh`** responde **501 Not Implemented** e **`POST /api/v1/auth/logout`** responde **204** sem invalidar sessão no servidor (comportamento pode evoluir). O registo/login que emitem JWT estão implementados de forma coerente com o resto da API.

---

## 3. Estrutura de pastas (resumo)

```
backend/
├── cmd/api/main.go          # CLI: serve | migrate
├── internal/
│   ├── httpapi/             # Servidor Chi, rotas, handlers, JWT helpers
│   ├── middleware/        # Auth, rate limit, tamanho máximo do corpo, logging
│   └── migrate/           # Aplicador de migrações SQL
├── migrations/            # Ficheiros .sql (ex.: 001_initial_schema.sql)
├── .github/workflows/ci.yml
├── go.mod / go.sum
├── README.md
└── BACKEND.md
```

Handlers estão principalmente em `internal/httpapi/handlers_*.go`; a configuração de rotas está centralizada em `internal/httpapi/server.go`.

---

## 4. Passo a passo — ambiente de desenvolvimento

### Pré-requisitos

- **Go 1.23** (alinhado com `go.mod` e CI).
- **PostgreSQL** (15+ recomendado; CI usa 15).
- **Redis** (7+ recomendado; CI usa 7).

### Passo 1 — Clonar e entrar na pasta do backend

Na raiz do repositório `harem-site`, a API vive em `backend/`. Todos os comandos `go` devem correr com o diretório de trabalho em `backend` (ou ajustar caminhos).

### Passo 2 — Dependências Go

```bash
cd backend
go mod download
```

Opcional: `go mod tidy` para alinhar `go.sum`.

### Passo 3 — Base de dados PostgreSQL

Crie uma base e um utilizador com permissões (exemplo genérico; adapte palavras-passe):

```sql
CREATE DATABASE harem;
CREATE USER harem WITH PASSWORD 'sua_senha_segura';
GRANT ALL PRIVILEGES ON DATABASE harem TO harem;
```

Em alguns sistemas é necessário conceder permissões no *schema* `public` após a primeira ligação; siga a política da sua instalação.

### Passo 4 — Redis

Inicie uma instância Redis acessível (por defeito `localhost:6379`). Sem Redis, o servidor **não arranca** porque o *ping* inicial falha.

### Passo 5 — Variáveis de ambiente

Defina pelo menos:

| Variável       | Descrição |
|----------------|-----------|
| `DATABASE_URL` | URL PostgreSQL (formato `postgres://...`). |
| `REDIS_URL`    | URL Redis (ex.: `redis://localhost:6379/0`). |
| `PORT`         | Porta HTTP (padrão no código: `40080` se não definido noutro sítio). |
| `JWT_SECRET`   | Segredo HMAC; **mínimo 32 caracteres** em produção. Se vazio em desenvolvimento, o binário usa um segredo por defeito e regista *warning* nos logs. |

Pode colocar estas variáveis num ficheiro **`.env`** na pasta `backend` (o `main` tenta `godotenv.Load()` automaticamente).

### Passo 6 — Migrações

Com `DATABASE_URL` apontando para a base correta:

```bash
go run ./cmd/api migrate
```

Ou, após compilar: `./harem-api migrate`

Isto aplica os ficheiros em `migrations/` pela ordem definida pelo migrador interno.

### Passo 7 — Subir a API

```bash
go run ./cmd/api serve
```

Ou com flags explícitas:

```bash
go run ./cmd/api serve -port=40080 -redis=redis://localhost:6379/0 -jwt-secret="pelo-menos-32-caracteres-aqui"
```

Teste: `GET http://localhost:40080/health` (ajuste host/porta conforme o seu `PORT`).

### Passo 8 — Testes e qualidade

```bash
go test ./...
go vet ./...
go fmt ./...
```

O repositório inclui **GitHub Actions** (`.github/workflows/ci.yml`): testes com PostgreSQL e Redis como *services*, `go test -race -coverprofile`, compilação do binário, execução de `migrate` e job **golangci-lint**.

---

## 5. Passo a passo — pedido autenticado (exemplo)

1. **Registo:** `POST /api/v1/auth/register` com JSON válido (corpo limitado a 1 MiB no grupo auth).
2. **Login:** `POST /api/v1/auth/login` — resposta inclui `access_token` (e eventualmente refresh, conforme implementação).
3. **Chamadas protegidas:** cabeçalho `Authorization: Bearer <access_token>`.
4. Rotas como `GET /api/v1/me` exigem JWT com um papel entre `user`, `creator`, `moderator`, `admin` (conforme definido no router).

Respostas de erro seguem o estilo JSON usado em `internal/httpapi/responses.go` (mensagens para o cliente sem expor detalhes internos desnecessários).

---

## 6. Produção (síntese)

- Colocar **Nginx** (ou outro *reverse proxy*) à frente, TLS, cabeçalhos `X-Forwarded-*`, limites de corpo e *rate limit* na borda conforme `README.md`.
- Serviço **systemd** com `ExecStart` a apontar para o binário `serve` e `Environment=` para segredos (nunca commitar `.env` com segredos reais).
- Garantir `JWT_SECRET` forte e rotação de segredos conforme política da equipa.

---

## 7. Onde aprofundar

| Tema | Onde |
|------|------|
| Lista de endpoints e exemplos de deploy | `README.md` |
| Arquitetura alvo, RBAC, WebSockets, billing, roadmap | `BACKEND.md` |
| Rotas e limites de corpo por grupo | `internal/httpapi/server.go` |
| Regras de JWT e papéis | `internal/middleware/auth.go` |

Se algo neste guia divergir do código, prevalece o **código-fonte** e os testes automatizados.
