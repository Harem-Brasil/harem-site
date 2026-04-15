# language: pt
Funcionalidade: Endpoints de Webhooks
  Como um provedor de pagamento
  Eu quero enviar eventos de webhook para a API
  Para que atualizações de status de pagamento sejam processadas

  Contexto:
    Dado que a API está em execução

  Cenário: Receber webhook do Stripe
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/stripe" com um evento Stripe:
      | id                | type              |
      | evt_stripe_123    | checkout.completed |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status" com valor "received"

  Cenário: Receber webhook do PagSeguro
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/pagseguro" com um evento PagSeguro:
      | id                | type              |
      | evt_ps_456        | payment.approved  |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status" com valor "received"

  Cenário: Receber webhook do MercadoPago
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/mercadopago" com um evento MercadoPago:
      | id                | type              |
      | evt_mp_789        | payment.success   |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status" com valor "received"

  Cenário: Receber webhook genérico para provedor válido
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/stripe" com:
      | id                | type              |
      | evt_generic_001   | payment.success   |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "provider" com valor "stripe"

  Cenário: Provedor de webhook desconhecido
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/unknown-provider" com:
      | id                | type              |
      | evt_test_001      | test.event        |
    Então o código de status da resposta deve ser 404

  Cenário: Webhook com JSON inválido
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/stripe" com JSON inválido
    Então o código de status da resposta deve ser 400

  Cenário: Webhook registra evento de forma segura
    Quando eu enviar uma requisição POST para "/api/v1/webhooks/stripe" com:
      | id                | type              |
      | evt_log_001       | payment.success   |
    Então o código de status da resposta deve ser 200
    E o evento deve ser registrado sem dados sensíveis
