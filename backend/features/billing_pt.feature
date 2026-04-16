# language: pt
Funcionalidade: Endpoints de Cobrança e Assinaturas
  Como um usuário
  Eu quero visualizar planos e gerenciar minhas assinaturas
  Para que eu possa acessar conteúdo premium

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como usuário "johndoe"

  Cenário: Listar planos disponíveis
    Quando eu enviar uma requisição GET para "/api/v1/billing/plans"
    Então o código de status da resposta deve ser 200
    E a resposta deve ser um array
    E cada plano deve conter "id"
    E cada plano deve conter "name"
    E cada plano deve conter "price"

  Cenário: Criar sessão de checkout
    Dado que um plano com id "plan-premium" existe
    Quando eu enviar uma requisição POST para "/api/v1/billing/checkout" com:
      | plan_id        |
      | plan-premium   |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "checkout_session_id"
    E a resposta deve conter "plan"
    E a resposta deve conter "payment_url"

  Cenário: Checkout com plano inválido
    Quando eu enviar uma requisição POST para "/api/v1/billing/checkout" com:
      | plan_id        |
      | invalid-plan   |
    Então o código de status da resposta deve ser 404
    E a resposta deve conter "error"

  Cenário: Obter minha assinatura
    Dado que eu tenho uma assinatura ativa
    Quando eu enviar uma requisição GET para "/api/v1/billing/subscription"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "id"
    E a resposta deve conter "status" com valor "active"
    E a resposta deve conter "plan"

  Cenário: Obter assinatura quando não subscrito
    Dado que eu não tenho uma assinatura ativa
    Quando eu enviar uma requisição GET para "/api/v1/billing/subscription"
    Então o código de status da resposta deve ser 200
    E a resposta deve ser nula

  Cenário: Cancelar assinatura
    Dado que eu tenho uma assinatura ativa
    Quando eu enviar uma requisição POST para "/api/v1/billing/subscription/cancel"
    Então o código de status da resposta deve ser 204

  Cenário: Retomar assinatura
    Dado que eu tenho uma assinatura cancelada
    Quando eu enviar uma requisição POST para "/api/v1/billing/subscription/resume"
    Então o código de status da resposta deve ser 204

  Cenário: Endpoint legado - Criar assinatura
    Quando eu enviar uma requisição POST para "/api/v1/subscriptions" com:
      | plan_id        |
      | plan-premium   |
    Então o código de status da resposta deve ser 201

  Cenário: Endpoint legado - Obter minha assinatura
    Quando eu enviar uma requisição GET para "/api/v1/subscriptions/me"
    Então o código de status da resposta deve ser 200
