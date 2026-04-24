# language: pt
Funcionalidade: Endpoints de Autenticação
  Como um usuário
  Eu quero me registrar, fazer login e gerenciar minha autenticação
  Para que eu possa acessar recursos protegidos

  Contexto:
    Dado que a API está em execução
    E o banco de dados está conectado

  Cenário: Registrar um novo usuário com sucesso
    Dado que eu tenho um payload de registro válido
      | username | email             | password     |
      | johndoe  | john@example.com  | SecurePass123! |
    Quando eu enviar uma requisição POST para "/api/v1/auth/register" com o payload
    Então o código de status da resposta deve ser 201
    E a resposta deve conter "access_token" não vazio
    E a resposta deve conter "refresh_token" não vazio
    E a resposta deve conter "user" com dados estruturados sem campos sensíveis

  Cenário: Registrar com email duplicado
    Dado que um usuário com email "existing@example.com" já existe
    Quando eu enviar uma requisição POST para "/api/v1/auth/register" com:
      | username | email                | password     |
      | johndoe  | existing@example.com | SecurePass123! |
    Então o código de status da resposta deve ser 409
    E a resposta deve conter "error" com valor "User already exists"

  Cenário: Registrar com email inválido
    Quando eu enviar uma requisição POST para "/api/v1/auth/register" com:
      | username | email           | password     |
      | johndoe  | invalid-email   | SecurePass123! |
    Então o código de status da resposta deve ser 400
    E a resposta deve conter erro de validação para "email"

  Cenário: Login com credenciais válidas
    Dado que um usuário registrado com email "user@example.com" e senha "SecurePass123!"
    Quando eu enviar uma requisição POST para "/api/v1/auth/login" com:
      | email             | password         |
      | user@example.com  | SecurePass123!   |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "access_token" não vazio
    E a resposta deve conter "refresh_token" não vazio
    E a resposta deve conter "user" com dados estruturados sem campos sensíveis

  Cenário: Login com credenciais inválidas
    Quando eu enviar uma requisição POST para "/api/v1/auth/login" com:
      | email             | password      |
      | user@example.com  | WrongPass123! |
    Então o código de status da resposta deve ser 401
    E a resposta deve conter "error"

  Cenário: Atualizar token de acesso
    Dado que eu tenho um refresh token válido
    Quando eu enviar uma requisição POST para "/api/v1/auth/refresh" com:
      | refresh_token          |
      | valid-refresh-token    |
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "access_token" não vazio
    E a resposta deve conter "refresh_token" não vazio

  Cenário: Logout da sessão atual
    Dado que eu estou autenticado como usuário "johndoe"
    Quando eu enviar uma requisição POST para "/api/v1/auth/logout"
    Então o código de status da resposta deve ser 200

  Cenário: Logout de todas as sessões
    Dado que eu estou autenticado como usuário "johndoe"
    Quando eu enviar uma requisição POST para "/api/v1/auth/logout-all"
    Então o código de status da resposta deve ser 200

  Cenário: Solicitar reset de senha
    Dado que um usuário registrado com email "user@example.com"
    Quando eu enviar uma requisição POST para "/api/v1/auth/password/forgot" com:
      | email             |
      | user@example.com  |
    Então o código de status da resposta deve ser 202
    E a resposta deve conter "message" não vazia

  Cenário: Resetar senha com token válido
    Dado que eu tenho um token de reset de senha válido
    Quando eu enviar uma requisição POST para "/api/v1/auth/password/reset" com:
      | token                  | new_password      |
      | valid-reset-token      | NewSecurePass456! |
    Então o código de status da resposta deve ser 200

  Cenário: Verificar email com token válido
    Dado que eu tenho um token de verificação de email válido
    Quando eu enviar uma requisição POST para "/api/v1/auth/email/verify" com:
      | token                  |
      | valid-verify-token     |
    Então o código de status da resposta deve ser 200
