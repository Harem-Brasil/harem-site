API\_GO\_IMPLEMENTACAO.md 2026-04-13 

Harém Brasil — API REST \+ tempo real: 

implementação em Go 

**Versão:** 1.0 · **Abril de 2026** 

Este documento define o **contrato HTTP versionado**, padrões transversais, webhooks de pagamento e o **contrato WebSocket**, orientados à implementação do **serviço público principal** em **Go**, com persistência em **PostgreSQL**. Complementa a visão de produto com escolhas concretas de stack, estrutura de pacotes e práticas de segurança no ecossistema Go. 

# **1\. Âmbito e princípios**

## **1.1 O que está coberto**

| Domínio | API REST | Tempo real |
| :---- | :---- | :---- |
| Autenticação (e-mail/senha, social, refresh) | Sim | Não |

| Perfis e níveis (Guest, user, creator, mod, admin) | Sim | Não |
| :---- | :---- | :---- |

| Timeline (posts, mídia, curtidas, comentários, feed) | Sim | Opcional (notificações) |
| :---- | :---- | :---- |
| Fórum (categorias, tópicos, respostas, moderação) | Sim | Não |

| Chat (salas, membros, histórico paginado) | Sim | Sim (mensagens em direto) |
| :---- | :---- | :---- |
| Assinaturas e checkout | Sim | Webhooks |
| Área criador (verificação, conteúdo pago, pedidos, ganhos) | Sim | Parcial |

| Notificações | Sim | Push via WS opcional |
| :---- | :---- | :---- |

## **1.2 O que não duplica**

O esquema físico SQL detalhado segue o documento de arquitetura; aqui ficam **contrato HTTP**, **campos expostos**, **whitelist** e regras sem mass assignment nem vazamento de colunas internas. 

## **1.3 Regras transversais (obrigatórias)**

1\. **Prefixo:** https://api.harembrasil.com.br/api/v1 (ajustar host por ambiente). 2\. **JSON:** Content-Type: application/json; charset=utf-8 salvo multipart ou rotas binárias. 3\. **UTF-8:** validar tamanho em **bytes** e em **runes** em texto livre crítico (utf8.RuneCountInString \+ limite de bytes no middleware). 

4\. **IDs:** UUID (string) em path e corpo; não expor IDs sequenciais internos. 

5\. **Queries parametrizadas:** database/sql com placeholders, **pgx** com argumentos posicionais, ou **sqlc** gerado — nunca concatenar input em SQL. 

6\. **Whitelist** em todos os POST/PATCH (structs com tags \+ validação explícita ou decoder custom). 7\. **BOLA:** validar posse ou papel em {user\_id}, {post\_id}, etc. 

8\. **BFLA:** /admin/\* com middleware e política isolada.

9\. **Logging:** slog ou equivalente estruturado; eventos de segurança sem PII (ver secção 11). 10\. **Rate limit:** por IP e por utilizador; cabeçalhos RateLimit-\* / Retry-After conforme política. 

## **1.4 Papel do Go nesta arquitetura**

O binário Go é o candidato natural ao **edge HTTP público**, **WebSocket**, orquestração de **webhooks** (validação rápida \+ enfileiramento) e maior parte das rotas REST. Serviços satélite em outras linguagens (por exemplo .NET) podem consumir filas e expor APIs internas, mas **o contrato público descrito aqui deve permanecer coerente** (mesmos paths, erros e OpenAPI). 

**Stack sugerida (ajustável por ADR):** 

| Camada | Opções comuns |
| :---- | :---- |

| Router HTTP | *chi, echo, fiber* (preferir middleware padrão idiomático). |
| :---- | :---- |

| Validação | *go-playground/validator/v10* \+ structs; ou contrato gerado a partir de OpenAPI (*oapi-codegen*). |
| :---- | :---- |

| PostgreSQL | *jackc/pgx/v5* (+ *pgxpool*), ou *database/sql* \+ *lib/pq*; sqlc para queries tipadas. |
| :---- | :---- |
| JWT | *golang-jwt/jwt/v5* com validação estrita de *iss, aud, exp*, assinatura. |

| WebSocket | *nhooyr.io/websocket* ou *gorilla/websocket* com limites de leitura e contexto cancelável. |
| :---- | :---- |

| Filas / async | NATS, Redis Streams, SQS, ou RabbitMQ — o handler HTTP devolve *200* ao webhook e publica evento idempotente. |
| :---- | :---- |

| Observabilidade | OpenTelemetry (*otel*) \+ *slog*; *pprof* em ambientes não produtivos ou protegidos. |
| :---- | :---- |

# **2\. Convenções HTTP**

## **2.1 Métodos e semântica**

**Método Uso** 

GET Leitura idempotente. 

POST Criação ou ação (checkout, convite). 

PATCH Atualização parcial com whitelist. 

DELETE Remoção lógica (deleted\_at) salvo purge admin. 

## **2.2 Cabeçalhos comuns**

**Cabeçalho** | **Descrição**

Authorization: Bearer `<access_token>` | Obrigatório nas rotas autenticadas (exceto públicas).

X-Request-Id | Opcional do cliente; se ausente, gerar (uuid ou ULID) e ecoar na resposta.

Idempotency-Key: `<uuid>` | Obrigatório em POST de cobrança, subscrição ou encomenda.

If-Match: `<etag>` | Opcional em PATCH com controlo de concorrência.

## **2.3 Paginação (cursor)**

limit — default 20, máximo 100. 

cursor — string opaca (ex.: base64url de (created\_at, id)). 

{ 

 "data": \[\], 

 "meta": { 

 "next\_cursor": "eyJjIjo…", 

 "has\_more": true 

 } 

} 

## **2.4 Ordenação e filtros**

*snake\_case* nas query keys; rejeitar filtros não documentados (evitar abuso / injection lógica).

## **2.5 Limites de payload (Nginx \+ http.MaxBytesReader)**

**Contexto Limite sugerido** 

JSON genérico 256 KiB 

Criar post 64 KiB texto \+ refs de mídia 

Mensagem chat (REST) 8 KiB 

Multipart metadados 1 MiB (ficheiro via URL pré-assinada) 

**Go:** aplicar MaxBytesReader por rota ou middleware global antes do json.Decoder. 

# **3\. Autenticação e tokens**

**Access JWT** curto (10–15 min); **refresh** opaco com hash em PostgreSQL (ou Redis) e **rotação**. Claims: sub, iss, aud, exp, iat, jti, role ou roles. 

Validar assinatura, exp, iss, aud; opcional denylist por jti. 

**OAuth/OIDC:** Authorization Code \+ PKCE; handlers Go trocam code e emitem par Harém. **Go:** armazenar refresh com **Argon2id** ou **bcrypt** do segredo/opaco; nunca logar tokens.

# **4\. Modelo de erros**

application/problem+json (ou JSON com campo type URI): 

{ 

 "type": "https://api.harembrasil.com.br/problems/validation-error",  "title": "Validation failed", 

 "status": 422, 

 "detail": "One or more fields are invalid.", 

 "instance": "/api/v1/posts", 

 "request\_id": "01JR…", 

 "errors": \[ 

 { "field": "body", "code": "too\_long", "message": "Body exceeds maximum length" } 

 \] 

} 

**Status Uso** 

400 JSON inválido / parâmetros fora do contrato 

401 Token ausente ou inválido 

403 RBAC ou BOLA 

404 Não encontrado ou oculto (anti-enumeração) 

409 Idempotência / If-Match 

422 Validação de domínio 

429 Rate limit 

500 Interno (sem stack na resposta) 

**Go:** helper único WriteProblem(w, status, typeURI, detail, fields...) \+ recover middleware que mapeia panic para 500 genérico. 

# **5\. RBAC**

Papéis: guest, user, creator, moderator, admin. 

Permissões exemplo: post.create, post.create\_paid, forum.moderate, chat.public\_write, billing.manage\_self, creator.verification.submit. 

Matriz MVP (resumo): registo/login público; feed/post conforme papel; moderação moderator; chat membro; checkout autenticado; webhooks com segredo; /admin/\* só admin. 

# **6\. Catálogo de endpoints**

Path relativo a /api/v1. Campos só leitura nunca em PATCH/POST sem regra explícita.

## **6.1 Saúde e metadados**

**Método Path Auth** 

GET /healthz Não 

GET /readyz Não 

GET /version Não 

## **6.2 Autenticação**

**Método Path Rate limit** 

POST /auth/register Estrito 

POST /auth/login Estrito 

POST /auth/refresh Médio 

POST /auth/logout — 

POST /auth/logout-all — 

GET /auth/oauth/{provider}/authorize OIDC 

GET /auth/oauth/{provider}/callback OIDC 

POST /auth/email/verify Estrito 

POST /auth/password/forgot Muito estrito 

POST /auth/password/reset Estrito 

**Registo — whitelist:** email, password, screen\_name, accept\_terms\_version. **Resposta 201:** user mínimo \+ tokens (access\_token, access\_expires\_in, refresh\_token, refresh\_expires\_in). 

## **6.3 /me**

GET / PATCH. **Whitelist PATCH:** screen\_name, bio, locale, notify\_preferences (subchaves permitidas). 

## **6.4 Utilizadores públicos**

*GET /users/{user\_id}, /users/{user\_id}/posts, GET /users/search* (*q, limit, cursor*).

## **6.5 Posts e feed**

CRUD /posts, /posts/{post\_id}, likes, comments, GET /feed/home.   
**POST /posts whitelist:** *body, visibility, media\[\]* com *upload\_id, kind, alt\_text*.

## **6.6 Fórum**

Categorias, tópicos, respostas; no código usar ForumReply / forum\_posts para não colidir com posts do feed. 

## **6.7 Chat (REST)**

Salas, membros, GET .../messages (cursor, limit, before). Envio em tempo real via WebSocket (secção 8). 

## **6.8 Assinaturas**

GET /billing/plans, POST /billing/checkout (**Idempotency-Key**), GET /billing/subscription, POST /billing/subscription/cancel, POST /billing/subscription/resume.

Estado final coerente após webhook \+ BD. 

## **6.9 Webhooks**

POST /webhooks/stripe|pagseguro|mercadopago — corpo cru para HMAC; 200 rápido \+ fila; idempotência por event\_id; logs sem payload completo. 

## **6.10 Criador**

Verificação, catálogo, pedidos, ganhos; whitelist de item: title, description, price\_cents, currency, visibility, media. 

## **6.11 Notificações**

GET /notifications, POST .../read, .../read-all. 

## **6.12 Uploads**

POST /media/upload-sessions; opcional POST .../{upload\_id}/complete para validação server side. 

## **6.13 Admin**

Isolado, só admin, auditoria; nunca password\_hash, tokens ou documentos em base64 na resposta. 

# **7\. Esquemas JSON (OpenAPI)**

Componentes: UserPublic, UserPrivate, Post, ForumTopic, ForumPost, ChatRoom, ChatMessage, Subscription, Plan, CheckoutSession, ProblemDetail. 

# **8\. WebSocket (tempo real)**

URL: preferir POST /realtime/ticket (ticket curto) \+ upgrade com cookie ou header, em vez de token longo na query. 

Envelope:

{ 

 "v": 1, 

 "id": "uuid-msg-client", 

 "type": "chat.send|chat.ack|ping", 

 "payload": {} 

} 

Eventos servidor → cliente: chat.message, chat.typing, notification, subscription.updated. Ao subscrever room\_id, verificar membro em PostgreSQL (cache curto Redis opcional). Rate limit por utilizador e por sala. 

**Go:** goroutines por conexão com context da request; heartbeat e Pong handler; limite de tamanho de frame. 

# **9\. OpenAPI e código Go**

1\. Manter openapi/openapi.yaml como fonte de verdade. 

2\. CI: spectral / redocly lint. 

3\. Opcional: gerar tipos e servidor com oapi-codegen (strict server). 

4\. Sem geração: structs \+ validator/v10 espelhando o schema. 

# **10\. Estrutura de pacotes (internal/)**

**Pacote Responsabilidade** 

internal/httpapi Router, registo de rotas v1, montagem de middleware. 

internal/auth JWT, refresh, OIDC, política de senha. 

internal/user Perfis, pesquisa. 

internal/feed Posts, likes, comments, feed. 

internal/forum Categorias, tópicos, respostas. 

internal/chat REST \+ autorização de salas; ligação ao hub WS. 

internal/billing Planos, checkout, leitura de subscrição. 

internal/webhook Assinaturas HMAC, deduplicação, publicação na fila. 

internal/creator Verificação, catálogo, pedidos, ganhos. 

internal/notify Notificações, fan-out WS. 

internal/admin Handlers isolados \+ auditoria. 

internal/middleware Request ID, auth, rate limit, recover, logging, MaxBytesReader. internal/realtime Hub WebSocket, tickets, quotas. 

**Handler idiomático:** validar → autorizar → transação (ou query) → resposta mínima.

# **11\. Checklist de segurança (Go)**

Limites de corpo no reverse proxy e no MaxBytesReader. 

CORS restrito (sem \* com credenciais). 

Cookies: SameSite, Secure, CSRF se aplicável. 

Segredos de webhook configuráveis e rotação. 

Testes de integração BOLA (GET /posts/{id}, PATCH /me alheio). 

GOMAXPROCS / timeouts de servidor alinhados ao deployment. 

Dependências: govulncheck no CI. 

# **12\. Prefixos rápidos**

**Prefixo Auth** 

/api/v1/auth/\* Parcial 

/api/v1/me Sim 

/api/v1/users/\* Misto 

/api/v1/posts, /feed Misto 

/api/v1/forum/\* Misto 

/api/v1/chat/\* Sim 

/api/v1/billing/\* Misto 

/api/v1/creator/\* Sim 

/api/v1/notifications Sim 

/api/v1/media/\* Sim 

/api/v1/webhooks/\* Servidor 

/api/v1/admin/\* Admin 

*Documento vivo: ao fechar cada epic, alinhar paths, enums e exemplos com a implementação Go e com a política de produto/legal.*
