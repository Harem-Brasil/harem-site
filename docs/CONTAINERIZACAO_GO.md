# Harém Brasil — Containerização **Go** (API, realtime, worker)

**Versão:** 1.0 · **Abril de 2026**

**Âmbito:** exclusivamente **binários Go** — `cmd/api`, `cmd/realtime`, `cmd/worker` (ou nomes equivalentes no repositório), imagens OCI, Compose e probes. Infra partilhada (PostgreSQL, Redis, Nginx) descreve-se aqui na medida em que **interage** com estes contentores.

**Contrato HTTP / health:** rotas `GET /healthz`, `GET /readyz`.

**Documento irmão (visão alargada):** inclui o mesmo desenho de rede e secções operacionais; aprofunda só a **build e runtime Go**, sem track .NET.

---

## 1. Objetivos

| Objetivo | Descrição |
|----------|-----------|
| **Imagens mínimas** | Multi-stage: artefacto estático ou quase estático (`CGO_ENABLED=0`), runtime **distroless** ou Alpine com utilizador não-root. |
| **Um contexto, três entradas** | Mesmo `Dockerfile` (ou `TARGET` / `--target`) para `api`, `realtime` e `worker` — diferença = `ENTRYPOINT` / `CMD` ou binário compilado distinto. |
| **Graceful shutdown** | `SIGTERM` da orquestração → fechar `http.Server` com `Shutdown(ctx)` e drenar WebSockets com timeout. |
| **Paridade** | Mesma imagem (ou mesma pipeline) em CI, staging e produção; variáveis e secrets injetados no arranque. |

---

## 2. Mapa de contentores Go

```mermaid
flowchart LR
  subgraph edge [Borda]
    NG[Nginx]
  end
  subgraph go [Binários Go]
    API[cmd/api]
    RT[cmd/realtime]
    WK[cmd/worker]
  end
  subgraph data [Dependências]
    PG[(PostgreSQL)]
    RD[(Redis)]
  end
  NG --> API
  NG --> RT
  API --> PG
  API --> RD
  RT --> PG
  RT --> RD
  WK --> PG
  WK --> RD
```

| Contentor | Função | Porta típica (interna) | Expor na Internet |
|-----------|--------|-------------------------|-------------------|
| **api** | REST `/api/v1/*` | `8080` | Não — só via Nginx |
| **realtime** | WebSocket / ticket | `8081` | Não — upgrade via Nginx |
| **worker** | Filas, webhooks longos, jobs | — | Não |

Fixar no repo convenção `HTTP_ADDR` (ex.: `:8080`) por serviço; o host público continua a ser `https://api...` sem documentar portas internas no OpenAPI.

---

## 3. Health checks (probes)

Alinhado ao contrato da API:

| Rota | Uso |
|------|-----|
| `GET /healthz` | **Liveness** — processo vivo; **sem** dependência de BD (evita reinícios em falhas transitórias). |
| `GET /readyz` | **Readiness** — PostgreSQL (e Redis se for obrigatório ao arranque) acessíveis; `503` se dependência crítica falhar. |

**realtime:** expor os **mesmos paths** (ou prefixo único documentado, ex. sob `/api/v1/...`) para o Nginx e orquestradores usarem os mesmos probes.

**Docker Compose:**

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/healthz"]
  interval: 30s
  timeout: 5s
  retries: 3
```

Em **distroless** não há `wget`/`curl`: usar `HEALTHCHECK` com binário embebido ou probes HTTP do orquestrador (Kubernetes `httpGet`) em vez de `CMD` dentro da imagem.

---

## 4. Variáveis de ambiente (aplicação Go)

| Variável | Onde | Notas |
|----------|------|--------|
| `APP_ENV` | api, realtime, worker | `development` \| `staging` \| `production` |
| `HTTP_ADDR` | api, realtime | Ex.: `:8080` — escutar `0.0.0.0` dentro da rede Docker |
| `DATABASE_URL` | todos | Utilizador com privilégios mínimos; em prod preferir PgBouncer na URL |
| `REDIS_URL` | todos | TLS se a política exigir |
| `JWT_*` / `JWT_KEY_PATH` | api, realtime | Chave em ficheiro montado quando possível |
| `LOG_LEVEL` | todos | `info` em prod |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | opcional | Tracing métricas — só rede interna |

**Go:** validar env no `main` (falhar rápido com mensagem clara); não logar valores de secrets.

---

## 5. Dockerfile multi-stage (padrão Harém)

### 5.1 Regras

1. **Build:** `golang:<versão>-bookworm` (ou `-alpine` se a equipa preferir, com libc compatível).  
2. **`CGO_ENABLED=0`** quando não for necessário CGO — binário estático, imagem final menor.  
3. **Flags:** `-trimpath -ldflags="-s -w"` para reduzir tamanho e remover paths locais.  
4. **Runtime:** `gcr.io/distroless/static-debian12:nonroot` (recomendado) ou `alpine` + `USER` não-root.  
5. **Não** copiar para a imagem final: `.git`, testes, código-fonte, `go.mod` (salvo decisão explícita de debug).  
6. **`USER nonroot:nonroot`** na imagem final.

### 5.2 Um Dockerfile, três alvos (`TARGET`)

```dockerfile
# syntax=docker/dockerfile:1
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGET=cmd/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./${TARGET}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

**Builds:**

```bash
docker build -f deploy/docker/Dockerfile.go --build-arg TARGET=cmd/api -t harem/api:local .
docker build -f deploy/docker/Dockerfile.go --build-arg TARGET=cmd/realtime -t harem/realtime:local .
docker build -f deploy/docker/Dockerfile.go --build-arg TARGET=cmd/worker -t harem/worker:local .
```

Ajustar `EXPOSE` / `HTTP_ADDR` por serviço via `ARG` ou ficheiros Compose distintos.

### 5.3 Contentor `migrate` (job one-shot)

- **Imagem:** mesma base de build + binário `migrate`/`goose`/`atlas` **ou** reutilizar stage de build com `ENTRYPOINT` diferente.  
- **Execução:** antes de escalar `api`/`realtime`; `DATABASE_URL` com role de migração separada da app em produção (política da equipa).

### 5.4 Shutdown gracioso (obrigatório no código)

- Registar `http.Server` e em `SIGTERM`/`SIGINT` chamar `Shutdown(context.WithTimeout(...))`.  
- **realtime:** fechar hub WS com prazo; rejeitar novos upgrades após sinal.  
- **worker:** terminar consumo da fila de forma cooperativa (`context` cancelado).

Isto reduz `502` durante deploys com rolling update.

---

## 6. Runtime Go em contentor

| Tópico | Recomendação |
|--------|----------------|
| **`GOMAXPROCS`** | Por omissão = CPUs visíveis no cgroup; em Kubernetes limitar CPU requests/limits coerentes com latência desejada. |
| **Memória** | Definir `deploy.resources.limits.memory` no Compose/K8s; monitorizar RSS após carga. |
| **`pprof`** | Desligado em prod ou só em porta não exposta / rota protegida; nunca público. |

---

## 7. Rede e Nginx (resumo Go)

- Serviços Go na rede **`harem_internal`** com `postgres`, `redis`, `nginx`.  
- Nginx `upstream` aponta a `api:8080` e `realtime:8081` (exemplo).  
- WebSocket: `Upgrade`, `Connection`, timeouts alinhados ao produto.  
- `client_max_body_size` coerente com limites da API Go.

Detalhe de ficheiros Nginx: documento de arquitetura §6–7.

---

## 8. Docker Compose — serviços Go (fragmento)

```yaml
services:
  api:
    build:
      context: ..
      dockerfile: deploy/docker/Dockerfile.go
      args:
        TARGET: cmd/api
    env_file: .env.example
    environment:
      HTTP_ADDR: ":8080"
      DATABASE_URL: postgres://app:***@pgbouncer:6432/harem?sslmode=disable
    depends_on:
      postgres:
        condition: service_healthy
    networks: [harem_internal]
    # ports: "127.0.0.1:8080:8080"  # só se Nginx estiver no host

  realtime:
    build:
      context: ..
      dockerfile: deploy/docker/Dockerfile.go
      args:
        TARGET: cmd/realtime
    environment:
      HTTP_ADDR: ":8081"
    depends_on: [postgres, redis]
    networks: [harem_internal]

  worker:
    build:
      context: ..
      dockerfile: deploy/docker/Dockerfile.go
      args:
        TARGET: cmd/worker
    environment:
      DATABASE_URL: ...
    depends_on: [postgres, redis]
    networks: [harem_internal]
    restart: on-failure
```

Ordem: `postgres`/`redis` healthy → `migrate` → `api` / `realtime` / `worker` → `nginx`.

---

## 9. CI/CD (imagens Go)

1. `docker buildx build --platform linux/amd64` (e `arm64` se necessário).  
2. Cache BuildKit: `type=gha` ou registry.  
3. Scan **Trivy** / **Grype** na imagem final.  
4. Tags: `:<git-sha>` + `:staging` / `:production`.  
5. Opcional: assinatura **Cosign**.

---

## 10. Segurança (checklist focada em Go)

- [ ] Imagem final **non-root**; sem shell em prod (distroless) salvo exceção justificada.  
- [ ] Sem secrets em `ARG`/`ENV` na imagem; montar em runtime.  
- [ ] `read_only: true` + `tmpfs` para dirs de escrita mínimos, se o runtime permitir.  
- [ ] Logs JSON em stdout sem PII em rotas sensíveis.  
- [ ] Rate limit na app + na borda (Nginx).  
- [ ] Binário compilado com versão Go suportada; atualizar base `golang:*` por CVEs.

---

## 11. Estrutura sugerida no repositório

```
deploy/
  docker/
    Dockerfile.go          # multi-stage + ARG TARGET
  compose/
    docker-compose.yml
    docker-compose.prod.yml
  nginx/
    nginx.conf
```

---

## 12. Referência cruzada

| Conteúdo |
|-----------|
| Contrato, RBAC, limites de corpo |
| VPS Hostinger, PgBouncer, secção .NET opcional |
| Épicos com deploy Compose / CI |
| Diagrama completo |

---