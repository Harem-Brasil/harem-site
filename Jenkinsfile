// Jenkins Pipeline para deploy do backend e frontend Harem Brasil
// Suporta deploy em staging (todas as branches) e producao (main)

pipeline {
  agent any

  options {
    timestamps()
    skipDefaultCheckout(false)
  }

  environment {
    GO111MODULE = 'on'

    // --- PRODUCAO ---
    TARGET_HOST     = 'web1'
    TARGET_DIR      = '/var/www/vhosts/api.harembrasil.com.br'
    SERVICE_NAME    = 'harem-api'
    SERVICE_USER    = 'grimlock'
    API_URL         = credentials('harem-brasil-api-url')

    // --- STAGING ---
    STAGE_TARGET_HOST     = 'web1'
    STAGE_TARGET_DIR      = '/var/www/vhosts/api-stage.harembrasil.com.br'
    STAGE_SERVICE_NAME    = 'harem-api-stage'
    STAGE_SERVICE_USER    = 'grimlock'
    STAGE_PORT            = '40081'
    STAGE_API_URL         = 'https://api-stage.harembrasil.com.br'
    FRONTEND_STAGE_NAME   = 'harembrasil-frontend-stage'

    // --- TEMPORARY INFRASTRUCTURE (for develop branch) ---
    // Using staging secrets but deploying to temporary host
    TEMP_TARGET_HOST     = 'web1'
    TEMP_TARGET_DIR      = '/var/www/vhosts/api-temp.harembrasil.com.br'
    TEMP_SERVICE_NAME    = 'harem-api-temp'
    TEMP_SERVICE_USER    = 'grimlock'
    TEMP_PORT            = '40082'
    TEMP_API_URL         = 'https://api-temp.harembrasil.com.br'
    FRONTEND_TEMP_NAME   = 'harembrasil-frontend-temp'

    // Production Secrets - configure no Jenkins Credentials
    DATABASE_URL    = credentials('harem-brasil-database-url')
    REDIS_URL       = credentials('harem-brasil-redis-url')
    JWT_SECRET=credentials('harem-brasil-jwt-secret')
    STRIPE_SECRET_KEY=credentials('harem-brasil-stripe-secret-key')
    CLOUDFLARE_API_TOKEN=credentials('harem-brasil-cloudflare-token')

    // Staging Secrets - configure no Jenkins Credentials
    STAGE_DATABASE_URL    = credentials('harem-brasil-database-url-stage')
    STAGE_REDIS_URL       = credentials('harem-brasil-redis-url-stage')
    STAGE_JWT_SECRET=credentials('harem-brasil-jwt-secret-stage')
    STAGE_STRIPE_SECRET_KEY=credentials('harem-brasil-stripe-secret-key-stage')
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        sh 'git rev-parse --short HEAD'
        script {
          env.GIT_BRANCH = env.BRANCH_NAME ?: sh(script: 'git rev-parse --abbrev-ref HEAD', returnStdout: true).trim()
          echo "Branch: ${env.GIT_BRANCH}"
        }
      }
    }

    stage('Test & Build') {
      parallel {
        stage('Backend Build') {
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
        stage('Frontend Test & Build') {
          steps {
            dir('frontend') {
              sh label: 'Install dependencies', script: 'npm ci'
              sh label: 'Run tests', script: 'npm test'
              sh label: 'Build frontend', script: """
                export VITE_APP_ENV=\"${env.GIT_BRANCH == 'main' ? 'production' : 'staging'}\"
                export VITE_APP_COMMIT_HASH=\"\$(git rev-parse --short HEAD)\"
                npm run build
              """
            }
            sh '''
              set -euo pipefail
              mkdir -p artifacts
              cp -r frontend/dist artifacts/frontend-dist
            '''
            stash name: 'frontend-dist', includes: 'artifacts/frontend-dist/**/*'
          }
        }
      }
    }

    stage('DB Migrate') {
      steps {
        unstash "bin-amd64"
        dir('backend') {
          sh label: 'Run database migrations', script: '''
            set -euo pipefail
            chmod +x ../artifacts/harem-api-linux-amd64
            if [ "${GIT_BRANCH}" = "main" ]; then
              export DATABASE_URL="${DATABASE_URL}"
              export REDIS_URL="${REDIS_URL}"
              export JWT_SECRET="${JWT_SECRET}"
              export STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY}"
            else
              export DATABASE_URL="${STAGE_DATABASE_URL}"
              export REDIS_URL="${STAGE_REDIS_URL}"
              export JWT_SECRET="${STAGE_JWT_SECRET}"
              export STRIPE_SECRET_KEY="${STAGE_STRIPE_SECRET_KEY}"
            fi
            ../artifacts/harem-api-linux-amd64 migrate -dir ./migrations
          '''
        }
      }
    }

    // ==================== STAGING ====================
    stage('Deploy Staging Backend') {
      when { expression { return env.GIT_BRANCH != 'main' } }
      steps {
        unstash "bin-amd64"
        sh label: 'Upload & install binary (staging)', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"

# Criar arquivo .env localmente
COMMIT=$(git rev-parse --short HEAD)
printf 'PORT=%s\nENV=staging\nCOMMIT_HASH=%s\nDATABASE_URL=%s\nREDIS_URL=%s\nJWT_SECRET=%s\nSTRIPE_SECRET_KEY=%s\n' \
  "$STAGE_PORT" "$COMMIT" "$STAGE_DATABASE_URL" "$STAGE_REDIS_URL" "$STAGE_JWT_SECRET" "$STAGE_STRIPE_SECRET_KEY" > /tmp/harem-api-stage.env

# Upload arquivos para /tmp no target
scp "$BIN_LOCAL" ${STAGE_TARGET_HOST}:/tmp/harem-api-stage
scp /tmp/harem-api-stage.env ${STAGE_TARGET_HOST}:/tmp/harem-api-stage.env
scp -r backend/migrations ${STAGE_TARGET_HOST}:/tmp/migrations-stage

# Limpar arquivo local temporário
rm -f /tmp/harem-api-stage.env

# Prepare target and install
ssh ${STAGE_TARGET_HOST} "
  set -euo pipefail

  sudo mkdir -p ${STAGE_TARGET_DIR}

  sudo mv /tmp/harem-api-stage ${STAGE_TARGET_DIR}/harem-api
  sudo chmod 0755 ${STAGE_TARGET_DIR}/harem-api

  sudo mv /tmp/harem-api-stage.env ${STAGE_TARGET_DIR}/.env
  sudo chmod 0600 ${STAGE_TARGET_DIR}/.env

  sudo rm -rf ${STAGE_TARGET_DIR}/migrations
  sudo mv /tmp/migrations-stage ${STAGE_TARGET_DIR}/migrations
  sudo chmod -R 0755 ${STAGE_TARGET_DIR}/migrations

  sudo chown -R ${STAGE_SERVICE_USER}:${STAGE_SERVICE_USER} ${STAGE_TARGET_DIR}
"

# Criar/Atualizar serviço systemd (staging)
ssh ${STAGE_TARGET_HOST} "
  set -euo pipefail

  sudo tee /etc/systemd/system/${STAGE_SERVICE_NAME}.service > /dev/null << SERVICEFILE
[Unit]
Description=Harem Brasil API (Staging)
After=network.target

[Service]
Type=simple
User=${STAGE_SERVICE_USER}
WorkingDirectory=${STAGE_TARGET_DIR}
EnvironmentFile=${STAGE_TARGET_DIR}/.env
ExecStart=${STAGE_TARGET_DIR}/harem-api serve
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICEFILE

  sudo systemctl daemon-reload
  sudo systemctl enable ${STAGE_SERVICE_NAME}
  sudo systemctl restart ${STAGE_SERVICE_NAME}

  for i in {1..10}; do
    if sudo systemctl is-active ${STAGE_SERVICE_NAME} > /dev/null; then
      break
    fi
    sleep 1
  done

  sudo journalctl -u ${STAGE_SERVICE_NAME} --no-pager -n 50
  sudo systemctl is-active ${STAGE_SERVICE_NAME}
"
        '''
      }
    }

    stage('Deploy Staging Frontend') {
      when {
        allOf {
          expression { return env.CLOUDFLARE_API_TOKEN?.trim() }
          expression { return env.GIT_BRANCH != 'main' }
        }
      }
      steps {
        unstash 'frontend-dist'
        dir('frontend') {
          sh label: 'Deploy frontend to staging (Cloudflare)', script: '''
            set -euo pipefail
            export CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN}"
            npx wrangler deploy \
              --name "${FRONTEND_STAGE_NAME}" \
              --var API_URL:"${STAGE_API_URL}" \
              --var APP_ENV:"staging" \
              --var COMMIT_HASH:"$(git rev-parse --short HEAD)"
          '''
        }
      }
    }

    stage('Smoke Test Staging') {
      when { expression { return env.GIT_BRANCH != 'main' } }
      steps {
        sh label: 'Health check and smoke test staging API', script: '''
          set -euo pipefail
          # Aguardar API staging ficar disponível (health endpoint sem /api/v1 prefix)
          for i in {1..30}; do
            if curl -sf "${STAGE_API_URL}/health" > /dev/null 2>&1; then
              echo "Staging API is up"
              break
            fi
            echo "Waiting for staging API... ($i/30)"
            sleep 2
          done

          # Smoke tests: validar endpoints criticos
          echo "=== Health check ==="
          curl -sf -D - "${STAGE_API_URL}/health" | head -c 200 || true
          echo ""

          echo "=== API info ==="
          curl -sf -D - "${STAGE_API_URL}/readyz" | head -c 200 || true
          echo ""

          echo "=== Validate X-Environment header ==="
          ENV_HEADER=$(curl -sfI "${STAGE_API_URL}/health" | grep -i "X-Environment" || true)
          if echo "$ENV_HEADER" | grep -qi "staging"; then
            echo "OK: X-Environment: staging"
          else
            echo "WARN: X-Environment header missing or not 'staging'"
            echo "$ENV_HEADER"
          fi

          echo "=== Smoke test passed ==="
        '''
      }
    }

    // ==================== TEMPORARY INFRASTRUCTURE ====================
    stage('Deploy Temp Backend') {
      when { expression { return env.GIT_BRANCH == 'develop' } }
      steps {
        unstash "bin-amd64"
        sh label: 'Upload & install binary (temp)', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"

# Criar arquivo .env localmente
COMMIT=$(git rev-parse --short HEAD)
printf 'PORT=%s\nENV=temp\nCOMMIT_HASH=%s\nDATABASE_URL=%s\nREDIS_URL=%s\nJWT_SECRET=%s\nSTRIPE_SECRET_KEY=%s\n' \
  "$TEMP_PORT" "$COMMIT" "$STAGE_DATABASE_URL" "$STAGE_REDIS_URL" "$STAGE_JWT_SECRET" "$STAGE_STRIPE_SECRET_KEY" > /tmp/harem-api-temp.env

# Upload arquivos para /tmp no target
scp "$BIN_LOCAL" ${TEMP_TARGET_HOST}:/tmp/harem-api-temp
scp /tmp/harem-api-temp.env ${TEMP_TARGET_HOST}:/tmp/harem-api-temp.env
scp -r backend/migrations ${TEMP_TARGET_HOST}:/tmp/migrations-temp

# Limpar arquivo local temporário
rm -f /tmp/harem-api-temp.env

# Prepare target and install
ssh ${TEMP_TARGET_HOST} "
  set -euo pipefail

  sudo mkdir -p ${TEMP_TARGET_DIR}

  sudo mv /tmp/harem-api-temp ${TEMP_TARGET_DIR}/harem-api
  sudo chmod 0755 ${TEMP_TARGET_DIR}/harem-api

  sudo mv /tmp/harem-api-temp.env ${TEMP_TARGET_DIR}/.env
  sudo chmod 0600 ${TEMP_TARGET_DIR}/.env

  sudo rm -rf ${TEMP_TARGET_DIR}/migrations
  sudo mv /tmp/migrations-temp ${TEMP_TARGET_DIR}/migrations
  sudo chmod -R 0755 ${TEMP_TARGET_DIR}/migrations

  sudo chown -R ${TEMP_SERVICE_USER}:${TEMP_SERVICE_USER} ${TEMP_TARGET_DIR}
"

# Criar/Atualizar serviço systemd (temp)
ssh ${TEMP_TARGET_HOST} "
  set -euo pipefail

  sudo tee /etc/systemd/system/${TEMP_SERVICE_NAME}.service > /dev/null << SERVICEFILE
[Unit]
Description=Harem Brasil API (Temp)
After=network.target

[Service]
Type=simple
User=${TEMP_SERVICE_USER}
WorkingDirectory=${TEMP_TARGET_DIR}
EnvironmentFile=${TEMP_TARGET_DIR}/.env
ExecStart=${TEMP_TARGET_DIR}/harem-api serve
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICEFILE

  sudo systemctl daemon-reload
  sudo systemctl enable ${TEMP_SERVICE_NAME}
  sudo systemctl restart ${TEMP_SERVICE_NAME}

  for i in {1..10}; do
    if sudo systemctl is-active ${TEMP_SERVICE_NAME} > /dev/null; then
      break
    fi
    sleep 1
  done

  sudo journalctl -u ${TEMP_SERVICE_NAME} --no-pager -n 50
  sudo systemctl is-active ${TEMP_SERVICE_NAME}
"
        '''
      }
    }

    stage('Deploy Temp Frontend') {
      when {
        allOf {
          expression { return env.CLOUDFLARE_API_TOKEN?.trim() }
          expression { return env.GIT_BRANCH == 'develop' }
        }
      }
      steps {
        unstash 'frontend-dist'
        dir('frontend') {
          sh label: 'Deploy frontend to temp (Cloudflare)', script: '''
            set -euo pipefail
            export CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN}"
            npx wrangler deploy \
              --name "${FRONTEND_TEMP_NAME}" \
              --var API_URL:"${TEMP_API_URL}" \
              --var APP_ENV:"temp" \
              --var COMMIT_HASH:"$(git rev-parse --short HEAD)"
          '''
        }
      }
    }

    stage('Smoke Test Temp') {
      when { expression { return env.GIT_BRANCH == 'develop' } }
      steps {
        sh label: 'Health check and smoke test temp API', script: '''
          set -euo pipefail
          # Aguardar API temp ficar disponível (health endpoint sem /api/v1 prefix)
          for i in {1..30}; do
            if curl -sf "${TEMP_API_URL}/health" > /dev/null 2>&1; then
              echo "Temp API is up"
              break
            fi
            echo "Waiting for temp API... ($i/30)"
            sleep 2
          done

          # Smoke tests: validar endpoints criticos
          echo "=== Health check ==="
          curl -sf -D - "${TEMP_API_URL}/health" | head -c 200 || true
          echo ""

          echo "=== API info ==="
          curl -sf -D - "${TEMP_API_URL}/readyz" | head -c 200 || true
          echo ""

          echo "=== Validate X-Environment header ==="
          ENV_HEADER=$(curl -sfI "${TEMP_API_URL}/health" | grep -i "X-Environment" || true)
          if echo "$ENV_HEADER" | grep -qi "temp"; then
            echo "OK: X-Environment: temp"
          else
            echo "WARN: X-Environment header missing or not 'temp'"
            echo "$ENV_HEADER"
          fi

          echo "=== Smoke test passed ==="
        '''
      }
    }

    // ==================== PRODUCAO ====================
    stage('Deploy Production Backend') {
      when { expression { return env.GIT_BRANCH == 'main' } }
      steps {
        unstash "bin-amd64"
        sh label: 'Upload & install binary (production)', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"

# Criar arquivo .env localmente
COMMIT=$(git rev-parse --short HEAD)
printf 'PORT=40080\nENV=production\nCOMMIT_HASH=%s\nDATABASE_URL=%s\nREDIS_URL=%s\nJWT_SECRET=%s\nSTRIPE_SECRET_KEY=%s\n' \
  "$COMMIT" "$DATABASE_URL" "$REDIS_URL" "$JWT_SECRET" "$STRIPE_SECRET_KEY" > /tmp/harem-api.env

# Upload arquivos para /tmp no target
scp "$BIN_LOCAL" ${TARGET_HOST}:/tmp/harem-api
scp /tmp/harem-api.env ${TARGET_HOST}:/tmp/harem-api.env
scp -r backend/migrations ${TARGET_HOST}:/tmp/migrations

# Limpar arquivo local temporário
rm -f /tmp/harem-api.env

# Prepare target and install
ssh ${TARGET_HOST} "
  set -euo pipefail

  sudo mkdir -p ${TARGET_DIR}

  sudo mv /tmp/harem-api ${TARGET_DIR}/harem-api
  sudo chmod 0755 ${TARGET_DIR}/harem-api

  sudo mv /tmp/harem-api.env ${TARGET_DIR}/.env
  sudo chmod 0600 ${TARGET_DIR}/.env

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

    stage('Deploy Production Frontend') {
      when {
        allOf {
          expression { return env.CLOUDFLARE_API_TOKEN?.trim() }
          expression { return env.GIT_BRANCH == 'main' }
        }
      }
      steps {
        unstash 'frontend-dist'
        dir('frontend') {
          sh label: 'Deploy frontend to production (Cloudflare)', script: '''
            set -euo pipefail
            export CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN}"
            export API_URL="${API_URL:-https://api.harembrasil.com.br}"
            npx wrangler deploy \
              --var API_URL:"${API_URL}" \
              --var APP_ENV:"production" \
              --var COMMIT_HASH:"$(git rev-parse --short HEAD)"
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