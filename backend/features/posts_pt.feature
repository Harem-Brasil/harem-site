# language: pt
Funcionalidade: Endpoints de Gerenciamento de Posts
  Como um usuário
  Eu quero criar, ler, atualizar e excluir posts
  Para que eu possa compartilhar conteúdo na plataforma

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como usuário "johndoe"

  Cenário: Criar um novo post
    Quando eu enviar uma requisição POST para "/api/v1/posts" com:
      | title              | content            | visibility |
      | My First Post      | Hello world!       | public     |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "id"
    E a resposta deve conter "title" com valor "My First Post"
    E a resposta deve conter "author_id"

  Cenário: Criar post com visibilidade padrão
    Quando eu enviar uma requisição POST para "/api/v1/posts" com:
      | title              | content            |
      | Another Post       | Some content       |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "visibility" com valor "public"

  Cenário: Obter um post por ID
    Dado que um post com id "post-123" existe
    Quando eu enviar uma requisição GET para "/api/v1/posts/post-123"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "id" com valor "post-123"

  Cenário: Atualizar meu post
    Dado que eu sou dono de um post com id "post-123"
    Quando eu enviar uma requisição PATCH para "/api/v1/posts/post-123" com:
      | title              | content           |
      | Updated Title      | Updated content   |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "title" com valor "Updated Title"

  Cenário: Excluir meu post
    Dado que eu sou dono de um post com id "post-123"
    Quando eu enviar uma requisição DELETE para "/api/v1/posts/post-123"
    Então o código de status da resposta deve ser 204

  Cenário: Listar posts com paginação
    Quando eu enviar uma requisição GET para "/api/v1/posts?limit=20"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "data"
    E a resposta deve conter "next_cursor"
    E a resposta deve conter "has_more"

  Cenário: Adicionar comentário ao post
    Dado que um post com id "post-123" existe
    Quando eu enviar uma requisição POST para "/api/v1/posts/post-123/comments" com:
      | content            |
      | Great post!        |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "id"
    E a resposta deve conter "content" com valor "Great post!"

  Cenário: Curtir um post
    Dado que um post com id "post-123" existe
    Quando eu enviar uma requisição POST para "/api/v1/posts/post-123/like"
    Então o código de status da resposta deve ser 200
    E a resposta deve indicar que o post foi curtido

  Cenário: Descurtir um post
    Dado que um post com id "post-123" existe
    E eu já curti o post
    Quando eu enviar uma requisição DELETE para "/api/v1/posts/post-123/like"
    Então o código de status da resposta deve ser 200
    E a resposta deve indicar que o post foi descurtido
