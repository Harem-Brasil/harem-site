# language: pt
Funcionalidade: Endpoints de Upload de Mídia
  Como um usuário
  Eu quero fazer upload de arquivos de mídia
  Para que eu possa anexá-los aos meus posts

  Contexto:
    Dado que a API está em execução
    E eu estou autenticado como usuário "johndoe"

  Cenário: Criar sessão de upload
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions" com:
      | file_name    | content_type | size       |
      | photo.jpg    | image/jpeg   | 1048576    |
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "id"
    E a resposta deve conter "status" com valor "pending"
    E a resposta deve conter "upload_url"

  Cenário: Criar sessão de upload sem nome de arquivo
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions" com:
      | content_type | size       |
      | image/jpeg   | 1048576    |
    Então o código de status da resposta deve ser 400
    E a resposta deve conter erro de validação para "file_name"

  Cenário: Criar sessão de upload com tamanho inválido
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions" com:
      | file_name    | content_type | size |
      | photo.jpg    | image/jpeg   | 0    |
    Então o código de status da resposta deve ser 400
    E a resposta deve conter erro de validação para "size"

  Cenário: Criar sessão de upload com arquivo muito grande
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions" com:
      | file_name    | content_type | size          |
      | video.mp4    | video/mp4    | 1048576000    |
    Então o código de status da resposta deve ser 400
    E a resposta deve conter erro de validação para "size"

  Cenário: Completar sessão de upload
    Dado que eu tenho uma sessão de upload com id "upload-123"
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions/upload-123/complete" com:
      | etag                      |
      | "abc123def456"            |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "id" com valor "upload-123"
    E a resposta deve conter "status" com valor "completed"

  Cenário: Completar upload sem autenticação
    Dado que eu não estou autenticado
    E eu tenho uma sessão de upload com id "upload-123"
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions/upload-123/complete" com:
      | etag                      |
      | "abc123def456"            |
    Então o código de status da resposta deve ser 401

  Cenário: Sessão de upload requer autenticação
    Dado que eu não estou autenticado
    Quando eu enviar uma requisição POST para "/api/v1/media/upload-sessions" com:
      | file_name    | content_type | size       |
      | photo.jpg    | image/jpeg   | 1048576    |
    Então o código de status da resposta deve ser 401
