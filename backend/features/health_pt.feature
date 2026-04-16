# language: pt
Funcionalidade: Endpoints de Health Check
  Como um operador
  Eu quero verificar a saúde da API
  Para que eu possa monitorar a disponibilidade do serviço

  Cenário: Verificar endpoint de liveness
    Dado que a API está em execução
    Quando eu enviar uma requisição GET para "/healthz"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status" com valor "ok"

  Cenário: Verificar endpoint de readiness
    Dado que o banco de dados está conectado
    E o cache está conectado
    Quando eu enviar uma requisição GET para "/readyz"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status" com valor "ready"

  Cenário: Verificar endpoint de versão
    Dado que a API está em execução
    Quando eu enviar uma requisição GET para "/version"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "version"
    E a resposta deve conter "build"

  Cenário: Verificar endpoint de health detalhado
    Dado que a API está em execução
    Quando eu enviar uma requisição GET para "/health"
    Então o código de status da resposta deve ser 200
    E a resposta deve conter "status"
    E a resposta deve conter "timestamp"
