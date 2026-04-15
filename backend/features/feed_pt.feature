# language: pt
Funcionalidade: Endpoints de Feed
  Como um usuário
  Eu quero ver um feed personalizado de conteúdo
  Para que eu possa descobrir posts de criadores que sigo ou subscrevo

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como usuário "johndoe"

  Cenário: Obter feed principal
    Quando eu enviar uma requisição GET para "/api/v1/feed/home"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
    E a resposta deve conter "has_more"

  Cenário: Obter feed principal com paginação
    Quando eu enviar uma requisição GET para "/api/v1/feed/home?limit=10&cursor=abc123"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E cada post deve ter "visibility" como "public" ou de criadores subscritos

  Cenário: Feed principal inclui posts de criadores seguidos
    Dado que eu sigo um criador com id "creator-456"
    E o criador tem posts publicados
    Quando eu enviar uma requisição GET para "/api/v1/feed/home"
    Então a resposta deve conter posts do criador "creator-456"

  Cenário: Feed principal respeita configurações de visibilidade
    Dado que eu não sou subscrito no criador "creator-789"
    Quando eu enviar uma requisição GET para "/api/v1/feed/home"
    Então a resposta não deve conter posts privados de "creator-789"

  Cenário: Feed principal mostra posts próprios
    Dado que eu tenho posts publicados
    Quando eu enviar uma requisição GET para "/api/v1/feed/home"
    Então a resposta deve conter meus próprios posts
