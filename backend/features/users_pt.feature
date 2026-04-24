# language: pt
Funcionalidade: Endpoints de Gerenciamento de Usuários
  Como um usuário
  Eu quero visualizar e gerenciar perfis de usuários
  Para que eu possa interagir com outros usuários na plataforma

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como usuário "johndoe"

  Cenário: Obter meu perfil
    Quando eu enviar uma requisição GET para "/api/v1/users/me"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "id"
    E a resposta deve conter "screen_name" com valor "johndoe"
    E a resposta deve conter "email"

  Cenário: Atualizar meu perfil
    Quando eu enviar uma requisição PATCH para "/api/v1/users/me" com:
      | display_name | bio                |
      | John Doe     | Hello world!       |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "display_name" com valor "John Doe"
    E a resposta deve conter "bio" com valor "Hello world!"

  Cenário: Excluir minha conta
    Quando eu enviar uma requisição DELETE para "/api/v1/users/me"
    Então o código de status da resposta deve ser 204

  Cenário: Obter outro usuário por ID
    Dado que um usuário com id "user-123" existe
    Quando eu enviar uma requisição GET para "/api/v1/users/user-123"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "id" com valor "user-123"

  Cenário: Listar usuários com paginação
    Quando eu enviar uma requisição GET para "/api/v1/users?limit=10"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
    E a resposta deve conter "has_more"

  Cenário: Pesquisar usuários por screen_name
    Quando eu enviar uma requisição GET para "/api/v1/users/search?q=john&limit=10"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E cada usuário nos resultados deve ter screen_name contendo "john"

  Cenário: Obter posts do usuário
    Dado que um usuário com id "creator-123" existe
    E o usuário tem posts publicados
    Quando eu enviar uma requisição GET para "/api/v1/users/creator-123/posts"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E cada post deve ter "author_id" com valor "creator-123"

  Cenário: Obter posts do usuário com paginação
    Dado que um usuário com id "creator-123" existe
    Quando eu enviar uma requisição GET para "/api/v1/users/creator-123/posts?limit=5&cursor=abc123"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
