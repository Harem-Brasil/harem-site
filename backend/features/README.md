# Godog BDD Tests

Este diretório contém testes de comportamento (BDD) usando [Godog](https://github.com/cucumber/godog) (Cucumber para Go) para a API.

## Estrutura

- `*_pt.feature` - Arquivos de funcionalidades em Gherkin (português brasileiro)
- `godog_test.go` - Definições dos passos e executor de testes

## Executando Testes

```bash
# Executar todos os testes de funcionalidades
go test ./features/...

# Executar com CLI do godog
godog

# Executar funcionalidade específica
godog features/auth_pt.feature
```

## Arquivos de Funcionalidades

| Arquivo | Descrição | Cenários |
|---------|-----------|----------|
| `health_pt.feature` | Endpoints de health check | 4 |
| `auth_pt.feature` | Autenticação (registro, login, logout, refresh) | 11 |
| `users_pt.feature` | Gerenciamento de usuários | 7 |
| `posts_pt.feature` | CRUD de posts, comentários, curtidas | 10 |
| `feed_pt.feature` | Feed principal | 5 |
| `billing_pt.feature` | Planos, assinaturas, checkout | 10 |
| `creator_pt.feature` | Dashboard de criador, catálogo, pedidos | 9 |
| `webhooks_pt.feature` | Webhooks de provedores de pagamento | 6 |
| `media_pt.feature` | Sessões de upload de mídia | 7 |

## Ambiente de Teste

Os testes requerem:
- Banco de dados PostgreSQL (instância de teste)
- Redis (opcional, para testes de rate limiting)

Defina as variáveis de ambiente:
```bash
export DATABASE_URL="postgres://user:pass@localhost/harem_test"
export REDIS_URL="redis://localhost:6379/1"
export JWT_SECRET="test-secret-min-32-characters-long"
```

## Escrevendo Novos Cenários

1. Adicione arquivo `.feature` com sintaxe Gherkin em português
2. Execute `godog --format=cucumber` para ver definições de passos faltantes
3. Implemente as definições dos passos em `godog_test.go`

## Convenções de Idioma

- `Funcionalidade:` - Nome da funcionalidade
- `Contexto:` - Passos executados antes de cada cenário (equivalente a `Background`)
- `Cenário:` - Caso de teste específico
- `Dado que` - Precondição (Given)
- `Quando` - Ação (When)
- `Então` - Resultado esperado (Then)
- `E` - Passo adicional do mesmo tipo (And)
