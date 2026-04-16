// Jenkins Pipeline para deploy do backend Harem Brasil
// Build multi-arch (amd64/arm64) e deploy amd64 para servidor

pipeline {
  agent any

  options {
    timestamps()
    skipDefaultCheckout(false)
  }

  environment {
    GO111MODULE = 'on'

    // Deployment targets
    TARGET_HOST     = 'web1'  // Ajuste para seu servidor de deploy
    TARGET_DIR      = '/var/www/vhosts/api.harembrasil.com.br'
    SERVICE_NAME    = 'harem-api'
    SSH_CREDENTIALS = 'harem-jenkins-ssh-key'  // Jenkins credential ID (Username with private key)

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

    stage('Build Matrix') {
      matrix {
        axes {
          axis {
            name 'GOARCH'
            values 'amd64', 'arm64'
          }
        }
        stages {
          stage('Build') {
            steps {
              dir('backend') {
                sh label: 'Go build', script: '''
                  set -euo pipefail
                  go version || true
                  export GOOS=linux
                  export CGO_ENABLED=0
                  echo "Building for $GOOS/$GOARCH"
                  out="harem-api-${GOOS}-${GOARCH}"
                  go build -ldflags="-s -w" -o "$out" ./cmd/api
                '''
              }
            }
          }
          stage('Archive') {
            steps {
              sh '''
                set -euo pipefail
                mkdir -p artifacts
                cp backend/harem-api-linux-${GOARCH} artifacts/
              '''
              stash name: "bin-${GOARCH}", includes: "artifacts/harem-api-linux-${GOARCH}"
            }
          }
        }
        post {
          success {
            echo "Built ${GOARCH} successfully"
          }
        }
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
            ../artifacts/harem-api-linux-amd64 migrate
          '''
        }
      }
    }

    stage('Deploy Backend') {
      steps {
        unstash "bin-amd64"
        sshagent(credentials: [env.SSH_CREDENTIALS]) {
          sh label: 'Upload & install binary', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"

# Criar arquivo .env localmente (segurança: não expor segredos na linha de comando)
cat > /tmp/harem-api.env << 'EOF'
PORT=8080
DATABASE_URL=${DATABASE_URL}
REDIS_URL=${REDIS_URL}
JWT_SECRET=${JWT_SECRET}
STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
EOF

# Upload arquivos para /tmp no target
scp "$BIN_LOCAL" ${TARGET_HOST}:/tmp/harem-api
scp /tmp/harem-api.env ${TARGET_HOST}:/tmp/harem-api.env

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
  sudo chown -R grimlock:grimlock ${TARGET_DIR}
"

# Criar/Atualizar serviço systemd
ssh ${TARGET_HOST} "
  set -euo pipefail

  sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null << 'SERVICEFILE'
[Unit]
Description=Harem Brasil API
After=network.target

[Service]
Type=simple
User=grimlock
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
  sudo systemctl status ${SERVICE_NAME} --no-pager
"
          '''
        }
      }
    }
  }

  post {
    success { echo 'Deploy realizado com sucesso!' }
    failure { echo 'Falha no deploy.' }
    always  { echo 'Pipeline finalizado.' }
  }
}
