# Harém Brasil — Demandas, estimativa e tasks (lane **Go**)

**Versão:** 1.0 · **Abril de 2026**

**Stack:** exclusivamente **`go`** — API HTTP pública, **WebSocket** (`realtime`), **workers** canónicos (webhooks, pós-processamento leve), **PostgreSQL** como fonte de verdade do domínio, **OpenAPI** como contrato.

---

## 1. Objetivo

Padronizar como se **levanta escopo**, **parte em tarefas implementáveis**, **estima** e **abre demandas** (Jira, Azure DevOps, GitHub Projects, Linear, etc.) para código **Go**, com rastreio explícito a **API**, **OpenAPI** e **arquitetura**, e **DoD** alinhado a segurança (BOLA/BFLA, SQL parametrizado, rate limit onde aplicável).

---

## 2. IDs e labels sugeridos

| Elemento | Convenção |
|----------|-----------|
| **Épico** | `HB-EPIC-01` … `HB-EPIC-07` (fases MVP) ou nomes equivalentes no tracker. |
| **História / Story** | `HB-101`, `HB-102`, … (sem prefixo NET). |
| **Task** | Subtarefas ou checklist no mesmo card; ou `HB-101a` conforme política da equipa. |
| **Label** | `stack:go` (obrigatório em todo o código Go do produto). |

**Campos recomendados no card:** `Stack: go` · `Doc-Refs` (ex.: `API_GO §6.4`, `openapi §components`) · link para PR.

---

## 3. Tipos de demanda

| Tipo | Quando usar | Estimativa |
|------|-------------|------------|
| **História (Story)** | Funcionalidade com valor de produto (rota, fluxo, módulo) | Pontos ou dias ideais |
| **Técnica** | CI, observabilidade, refactor de pacotes, migração tooling | Dias ou spike primeiro |
| **Spike** | Incerteza (webhook PSP, limites OIDC, desenho WS) | Time-box (ex.: 2 dias); saída = ADR ou atualização de doc |
| **Bug** | Comportamento incorreto vs contrato OpenAPI ou critérios de aceite | Incluir repro e teste de regressão |
| **Chore** | Bump Go minor, dependências, `golangci-lint` config | Baixa incerteza |

---

## 4. Estimativa

### 4.1 Story points (Fibonacci: 1, 2, 3, 5, 8, 13)

- **Ancla:** 1 ponto ≈ ajuste trivial com teste; **8+** deve ser partido ou precedido de **spike**.

### 4.2 Dias ideais por Task

- Somar tasks + **buffer** (secção 4.4).

### 4.3 T-shirt (S / M / L)

- Descoberta rápida; depois refinar para pontos ou dias.

### 4.4 Fatores que **aumentam** estimativa em Go

Marcar no card os que se aplicam (efeito composto típico **+20–40%** por fator relevante):

| Fator | Motivo |
|-------|--------|
| **Nova migração PG + índices** | Review, rollback, dados existentes |
| **BOLA/BFLA em muitos paths** | Testes de autorização por recurso |
| **Integração externa** (PSP, OIDC) | Sandbox, idempotência, webhooks |
| **Concorrência** (chat hub, workers) | `context`, race detector, goroutine leaks |
| **OpenAPI** + exemplos | Contrato sincronizado com handlers |
| **Observabilidade** (métricas, trace OTel) | Primeira instrumentação do módulo |

### 4.5 Buffer de planeamento (tech lead)

- Sobre a soma das tasks do épico: **+15–25%** para integração e revisão de segurança.  
- **Primeiro** gateway de pagamento: considerar **spike** antes da estimativa fechada.

---

## 5. Definition of Ready (DoR)

1. **Objetivo** em uma frase + **fora de âmbito** quando útil.  
2. **`Stack: go`** explícito.  
3. **Contrato:** referência a `docs/API_GO_IMPLEMENTACAO.md` (secção) e/ou paths no `openapi/openapi.yaml`.  
4. **Critérios de aceite** numerados e testáveis.  
5. **Dados:** tabelas PG ou alterações nomeadas (ou spike para definir).  
6. **Segurança:** auth, BOLA (rotas com `{id}`), rate limit em `/auth/*` quando aplicável.  
7. **Dependências:** outro card, secret no CI, sandbox PSP.  
8. **Tamanho:** se > 8 pontos ou > 3 dias ideais → dividir ou spike.

---

## 6. Definition of Done (DoD) — Go

Toda demanda de **código Go** backend cumpre, salvo exceção assinada no card:

| Critério | Detalhe |
|----------|---------|
| **Código** | `gofmt`; `golangci-lint` do repo; sem race óbvio em código concorrente novo (`go test -race` nos pacotes afetados). |
| **Testes** | Table-driven para lógica e handlers críticos; integração com PG quando houver query/migração relevante. |
| **SQL** | Apenas queries **parametrizadas** (pgx/sqlc); migração versionada em `migrations/`. |
| **API** | Rotas e schemas refletidos no OpenAPI **ou** issue filha “documentar OpenAPI” com data. |
| **Segurança** | Validação de input (tipo, tamanho, runes onde texto livre); sem secrets no repo; logs sem PII sensível. |
| **Revisão** | PR com link ao card; checklist **BOLA** para rotas com recurso por ID. |
| **Deploy** | Pipeline verde; variáveis novas documentadas (README ou doc ops). |

**Revisão de PR:** usar as **lentes** da skill `.cursor/skills/GOLANG_SKILL.md` (sénior, tech lead, arquiteto, soluções) conforme o tipo de alteração.

---

## 7. Template — Épico (Go)

```markdown
## Épico: [Nome curto]
**ID:** HB-EPIC-XX
**Stack:** go
**Objetivo de negócio:** [1–2 frases]
**Documentação:** ARQUITETURA §X · API_GO_IMPLEMENTACAO §Y · openapi/openapi.yaml
**Dependências externas:** [ex.: conta Stripe teste, Redis staging]
**Riscos:** [lista]
**Milestone / Release:** [nome]

### Features incluídas
- [ ] HB-FEAT-…
- [ ] HB-FEAT-…

### Métricas de sucesso
- [ex.: p95 login < 500ms em staging; 0 erros 5xx no health check]
```

---

## 8. Template — Demanda (Story / Técnica / Spike)

```markdown
## [HB-XXX] Título imperativo (ex.: Implementar refresh token com rotação)
**Tipo:** Story | Técnica | Spike | Bug
**Stack:** go
**Épico:** HB-EPIC-XX
**Prioridade:** P0–P3

### Contexto
[Porquê agora; utilizador ou sistema afetado]

### Escopo
**Inclui:** …
**Não inclui:** …

### Contrato / API
- Método e path: `POST /api/v1/…`
- Doc: API_GO_IMPLEMENTACAO §…
- OpenAPI: operationId / tag

### Dados (PostgreSQL)
- Tabelas: …
- Migrações: …

### Segurança
- Auth: …
- BOLA/BFLA: …
- Rate limit: …

### Critérios de aceite
1. …
2. …
3. …

### Tasks (checklist)
- [ ] Task 1 — estimativa: …
- [ ] Task 2 — …

### Estimativa
- Story points: [ ]  |  Dias ideais: [ ]
**Notas:** [fatores secção 4.4]
```

---

## 9. Template — Task (subdivisão)

```markdown
## Task: [verbo + objeto]
**Demanda pai:** HB-XXX
**Estimativa:** [horas ou 0.5 dia]

### Entregáveis
- Pacotes: `internal/<domínio>/...`, `cmd/api` ou `cmd/realtime` / `cmd/worker` conforme épico

### Validação local
- `go test ./internal/...`
- `go test -race ./...` (pacotes com concorrência)

### PR
- Link: …
```

---

## 10. Rastreio documentação ↔ demandas Go

| Documento / artefacto | Uso na demanda |
|----------------------|----------------|
| `docs/API_GO_IMPLEMENTACAO.md` | Paths, payloads, erros, RBAC, WebSocket |
| `docs/ARQUITETURA` (ou equivalente no repo) | C4, PG, deploy, Redis |
| `openapi/openapi.yaml` | Contrato canónico; diff no PR |
| `docs/adr/` (quando existir) | Decisões pós-spike |

**Campo no tracker:** `Doc-Refs: API_GO§6.2, openapi#Auth` (ajustar por card).

---

## 11. Levantamento por fase — backlog inicial (**Stack: go**)

Mapeamento sugerido do roadmap para **épicos** e **demandas exemplo**. Ajustar IDs ao vosso sistema.

### HB-EPIC-01 — Fase 1 (~2 semanas) — Fundação

| Demanda exemplo | Tasks típicas | Notas estimativa |
|-----------------|----------------|------------------|
| Monorepo Go multi-`cmd` (`api`, `realtime`, `worker`) | Scaffold; `internal/httpapi`; router versionado | M |
| CI: test + lint + race (job noturno opcional) | GitHub Actions; cache modules; `golangci-lint` | S–M |
| Primeira migração PostgreSQL + `users` mínima | goose/migrate ou atlas; repo pgx/sql | M |
| OpenAPI skeleton + `GET /version` | Spectral/redocly lint no CI | S |
| Deploy staging (Nginx + systemd ou container) | Playbook; secrets fora do repo | M–L |

### HB-EPIC-02 — Fase 2 (~3 semanas) — Auth

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| `POST /auth/register` + hash Argon2id/bcrypt | Handler, whitelist DTO, testes |
| `POST /auth/login` + JWT access | Claims `iss`, `aud`, `exp`, `sub`, `role` |
| Refresh com rotação + tabela `refresh_tokens` | Migração, revogação, sem log de tokens |
| Middleware RBAC base | Extração de claims vs consulta PG quando necessário |
| OAuth OIDC (1 provider) | Spike opcional primeiro |
| Verificação criador (estados) | API stub sem documento real no MVP |

### HB-EPIC-03 — Fase 3 (~3 semanas) — Timeline

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| CRUD posts + `visibility` | BOLA em `PATCH`/`DELETE` |
| `POST /media/upload-sessions` + storage | URLs pré-assinadas; worker opcional pós-upload |
| `GET /feed/home` cursor + índices | Query estável; benchmark leve |
| Likes / comments | Transações; contagens consistentes |

### HB-EPIC-04 — Fase 4 (~4 semanas) — Chat + realtime

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| Salas REST direct/group | Unicidade `direct`; limites grupo |
| Ticket `POST /realtime/ticket` + upgrade WS | Auth; limites de frame; ping/pong |
| Hub: subscribe por `room_id` + verificação membro PG | Cache Redis opcional |
| Histórico `GET .../messages` | Cursor; rate limit envio WS |

### HB-EPIC-05 — Fase 5 (~3 semanas) — Fórum

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| Categorias e tópicos CRUD | Entidade `ForumReply` vs `posts` do feed |
| Moderação + permissões | `forum.moderate`; auditoria |

### HB-EPIC-06 — Fase 6 (~4 semanas) — Billing

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| Planos + `POST /billing/checkout` | `Idempotency-Key` obrigatório |
| `POST /webhooks/*` receção Go | Corpo cru HMAC; `200` rápido; fila interna; idempotência `event_id` |
| Estado subscrição coerente em PG | Transações; flags premium/VIP |

### HB-EPIC-07 — Fase 7 (~3 semanas) — Criador

| Demanda exemplo | Tasks típicas |
|-----------------|---------------|
| Catálogo itens pagos | Whitelist campos |
| Pedidos + máquina de estados | `requested` → … |
| `GET /creator/earnings/summary` | Agregações; índices |

---

## 12. Grooming / planning — ordem sugerida (Go)

1. Rever épico (objetivo, risco, dependências de infra).  
2. Partir em histórias até caber num sprint.  
3. Aplicar **DoR** (secção 5).  
4. Estimar com checklist **4.4**.  
5. Comprometer capacidade do sprint para **`stack:go`**.  
6. Ao fechar: **DoD** (secção 6) + atualizar OpenAPI se o contrato mudou.

---

## 13. Glossário

| Termo | Significado |
|--------|-------------|
| **Grooming** | Refinar backlog e estimativas |
| **Spike** | Investigação time-boxed |
| **DoR / DoD** | Pronto para entrar / pronto para concluir |
| **BOLA** | Autorização ao nível do objeto (recurso de outro utilizador) |
| **BFLA** | Funções admin isoladas em `/admin/*` com política estrita |

---