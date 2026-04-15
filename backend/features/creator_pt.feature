# language: pt
Funcionalidade: Endpoints de Criador
  Como um criador
  Eu quero gerenciar meu perfil, catálogo e pedidos
  Para que eu possa monetizar meu conteúdo

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como criador "creatorsarah"

  Cenário: Aplicar para se tornar criador
    Dado que eu estou autenticado como usuário "regularuser"
    Quando eu enviar uma requisição POST para "/api/v1/creator/apply" com:
      | bio                        | social_links                    |
      | Criador de conteúdo profissional | https://twitter.com/sarah       |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "id"
    E a resposta deve conter "status" com valor "pending"

  Cenário: Obter dashboard do criador
    Quando eu enviar uma requisição GET para "/api/v1/creator/dashboard"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "total_posts"
    E a resposta deve conter "total_likes"
    E a resposta deve conter "total_followers"

  Cenário: Obter ganhos do criador
    Quando eu enviar uma requisição GET para "/api/v1/creator/earnings"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "earnings"
    E a resposta deve conter "total"

  Cenário: Obter catálogo do criador
    Quando eu enviar uma requisição GET para "/api/v1/creator/catalog"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
    E a resposta deve conter "has_more"

  Cenário: Obter catálogo do criador com paginação
    Quando eu enviar uma requisição GET para "/api/v1/creator/catalog?limit=5&cursor=item-abc"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter no máximo 5 itens

  Cenário: Obter pedidos do criador
    Quando eu enviar uma requisição GET para "/api/v1/creator/orders"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
    E a resposta deve conter "has_more"

  Cenário: Obter pedidos do criador com paginação
    Quando eu enviar uma requisição GET para "/api/v1/creator/orders?limit=10&cursor=order-xyz"
    Então o código de status da resposta deve ser 200
    E cada pedido deve conter "buyer_id"
    E cada pedido deve conter "item_id"
    E cada pedido deve conter "status"
    E cada pedido deve conter "amount_cents"

  Cenário: Apenas criadores podem acessar endpoints de criador
    Dado que eu estou autenticado como usuário "regularuser"
    Quando eu enviar uma requisição GET para "/api/v1/creator/dashboard"
    Então o código de status da resposta deve ser 403
