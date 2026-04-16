// Jenkins Pipeline para deploy do backend Harem Brasil

pipeline {
  agent any

  options {
    timestamps()
    skipDefaultCheckout(false)
  }

  environment {
    GO111MODULE = 'on'

    // Deployment targets
    TARGET_HOST     = 'web1'
    TARGET_DIR      = '/var/www/vhosts/api.harembrasil.com.br'
    SERVICE_NAME    = 'harem-api'
    SERVICE_USER    = 'grimlock'

    // Secrets - configure no Jenkins Credentials
    DATABASE_URL    = credentials('harem-brasil-database-url')
    REDIS_URL       = credentials('harem-brasil-redis-url')
    JWT_SECRET      = credentials('harem-brasil-jwt-secret')
    STRIPE_SECRET_KEY = credentials('harem-brasil-stripe-secret-key')
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        sh 'git rev-parse --short HEAD'
      }
    }

    stage('Build') {
      steps {
        dir('backend') {
          sh label: 'Go build', script: '''
            set -euo pipefail
            go version || true
            export GOOS=linux
            export GOARCH=amd64
            export CGO_ENABLED=0
            echo "Building for $GOOS/$GOARCH"
            go build -ldflags="-s -w" -o harem-api-linux-amd64 ./cmd/api
          '''
        }
        sh '''
          set -euo pipefail
          mkdir -p artifacts
          cp backend/harem-api-linux-amd64 artifacts/
        '''
        stash name: 'bin-amd64', includes: 'artifacts/harem-api-linux-amd64'
      }
    }

    stage('DB Migrate') {
      when { expression { return env.DATABASE_URL?.trim() } }
      steps {
        unstash "bin-amd64"
        dir('backend') {
          sh label: 'Run database migrations', script: '''
            set -euo pipefail
            chmod +x ../artifacts/harem-api-linux-amd64
            export DATABASE_URL="${DATABASE_URL}"
            export REDIS_URL="${REDIS_URL}"
            export JWT_SECRET="${JWT_SECRET}"
            export STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY}"
            ../artifacts/harem-api-linux-amd64 migrate -dir ./migrations
          '''
        }
      }
    }

    stage('Deploy Backend') {
      steps {
        unstash "bin-amd64"
        sh label: 'Upload & install binary', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"

# Criar arquivo .env localmente usando printf para evitar re-interpretação de caracteres especiais
printf 'PORT=40080\nDATABASE_URL=%s\nREDIS_URL=%s\nJWT_SECRET=%s\nSTRIPE_SECRET_KEY=%s\n' \
  "$DATABASE_URL" "$REDIS_URL" "$JWT_SECRET" "$STRIPE_SECRET_KEY" > /tmp/harem-api.env

# Upload arquivos para /tmp no target
scp "$BIN_LOCAL" ${TARGET_HOST}:/tmp/harem-api
scp /tmp/harem-api.env ${TARGET_HOST}:/tmp/harem-api.env
scp -r backend/migrations ${TARGET_HOST}:/tmp/migrations

# Limpar arquivo local temporário
rm -f /tmp/harem-api.env

# Prepare target and install
ssh ${TARGET_HOST} "
  set -euo pipefail

  # Criar diretório
  sudo mkdir -p ${TARGET_DIR}

  # Mover binário
  sudo mv /tmp/harem-api ${TARGET_DIR}/harem-api
  sudo chmod 0755 ${TARGET_DIR}/harem-api

  # Mover arquivo de ambiente
  sudo mv /tmp/harem-api.env ${TARGET_DIR}/.env
  sudo chmod 0600 ${TARGET_DIR}/.env

  # Mover diretório de migrações
  sudo rm -rf ${TARGET_DIR}/migrations
  sudo mv /tmp/migrations ${TARGET_DIR}/migrations
  sudo chmod -R 0755 ${TARGET_DIR}/migrations

  sudo chown -R ${SERVICE_USER}:${SERVICE_USER} ${TARGET_DIR}
"

# Criar/Atualizar serviço systemd
ssh ${TARGET_HOST} "
  set -euo pipefail

  sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null << SERVICEFILE
[Unit]
Description=Harem Brasil API
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${TARGET_DIR}
EnvironmentFile=${TARGET_DIR}/.env
ExecStart=${TARGET_DIR}/harem-api serve
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICEFILE

  sudo systemctl daemon-reload
  sudo systemctl enable ${SERVICE_NAME}
  sudo systemctl restart ${SERVICE_NAME}

  # Aguardar o serviço estabilizar com retentativas
  for i in {1..10}; do
    if sudo systemctl is-active ${SERVICE_NAME} > /dev/null; then
      break
    fi
    sleep 1
  done

  sudo journalctl -u ${SERVICE_NAME} --no-pager -n 50
  sudo systemctl is-active ${SERVICE_NAME}
"
          '''
      }
    }
  }

  post {
    success { echo 'Deploy realizado com sucesso!' }
    failure { echo 'Falha no deploy.' }
    always  { echo 'Pipeline finalizado.' }
  }
}
