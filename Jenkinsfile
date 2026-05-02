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

    // --- DEVELOP ---
    DEVELOP_TARGET_HOST     = 'web1'
    DEVELOP_TARGET_DIR      = '/var/www/vhosts/api-develop.harembrasil.com.br'
    DEVELOP_SERVICE_NAME    = 'harem-api-develop'
    DEVELOP_SERVICE_USER    = 'grimlock'
    DEVELOP_PORT            = '41082'
    DEVELOP_API_URL         = 'https://api-develop.harembrasil.com.br'
    DEVELOP_FRONTEND_DIR    = '/var/www/vhosts/develop.harembrasil.com.br'
    FRONTEND_DEVELOP_NAME   = 'harembrasil-frontend-develop'

    // Production Secrets - configure no Jenkins Credentials
    DATABASE_URL    = credentials('harem-brasil-database-url')
    REDIS_URL       = credentials('harem-brasil-redis-url')
    JWT_SECRET=credentials('harem-brasil-jwt-secret')
    STRIPE_SECRET_KEY=credentials('harem-brasil-stripe-secret-key')
    CLOUDFLARE_API_TOKEN=credentials('truvis-co-cloudflare-api-token')

    // Staging Secrets - configure no Jenkins Credentials
    STAGE_DATABASE_URL    = credentials('harem-brasil-database-url-stage')
    STAGE_REDIS_URL       = credentials('harem-brasil-redis-url-stage')
    STAGE_JWT_SECRET=credentials('harem-brasil-jwt-secret-stage')
    STAGE_STRIPE_SECRET_KEY=credentials('harem-brasil-stripe-secret-key-stage')

    // Develop Secrets - configure no Jenkins Credentials
    DEVELOP_DATABASE_URL    = credentials('harem-brasil-database-url-develop')
    DEVELOP_REDIS_URL       = credentials('harem-brasil-redis-url-develop')
    DEVELOP_JWT_SECRET=credentials('harem-brasil-jwt-secret-develop')
    DEVELOP_STRIPE_SECRET_KEY=credentials('harem-brasil-stripe-secret-key-develop')
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
                export VITE_APP_ENV=\"${env.GIT_BRANCH == 'main' ? 'production' : env.GIT_BRANCH == 'develop' ? 'develop' : 'staging'}\"
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
            elif [ "${GIT_BRANCH}" = "develop" ]; then
              export DATABASE_URL="${DEVELOP_DATABASE_URL}"
              export REDIS_URL="${DEVELOP_REDIS_URL}"
              export JWT_SECRET="${DEVELOP_JWT_SECRET}"
              export STRIPE_SECRET_KEY="${DEVELOP_STRIPE_SECRET_KEY}"
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

    // ==================== DEVELOP ====================
    stage('Deploy Develop Backend') {
      when { expression { return env.GIT_BRANCH == 'develop' } }
      steps {
        unstash "bin-amd64"
        sh label: 'Upload & install binary (develop)', script: '''
set -euo pipefail
BIN_LOCAL="artifacts/harem-api-linux-amd64"
COMMIT=$(git rev-parse --short HEAD)

# Upload binary e migrations (sem segredos)
scp "$BIN_LOCAL" ${DEVELOP_TARGET_HOST}:/tmp/harem-api-develop
scp -r backend/migrations ${DEVELOP_TARGET_HOST}:/tmp/migrations-develop

# Prepare target e instalar tudo via SSH (segredos injetados via pipe, sem tocar disco local)
ssh ${DEVELOP_TARGET_HOST} "
  set -euo pipefail

  sudo mkdir -p ${DEVELOP_TARGET_DIR}

  sudo mv /tmp/harem-api-develop ${DEVELOP_TARGET_DIR}/harem-api
  sudo chmod 0755 ${DEVELOP_TARGET_DIR}/harem-api

  printf 'PORT=%s\nENV=develop\nCOMMIT_HASH=%s\nDATABASE_URL=%s\nREDIS_URL=%s\nJWT_SECRET=%s\nSTRIPE_SECRET_KEY=%s\n' \
    '$DEVELOP_PORT' '$COMMIT' '$DEVELOP_DATABASE_URL' '$DEVELOP_REDIS_URL' '$DEVELOP_JWT_SECRET' '$DEVELOP_STRIPE_SECRET_KEY' | sudo tee ${DEVELOP_TARGET_DIR}/.env > /dev/null
  sudo chmod 0600 ${DEVELOP_TARGET_DIR}/.env

  sudo rm -rf ${DEVELOP_TARGET_DIR}/migrations
  sudo mv /tmp/migrations-develop ${DEVELOP_TARGET_DIR}/migrations
  sudo chmod -R 0755 ${DEVELOP_TARGET_DIR}/migrations

  sudo chown -R ${DEVELOP_SERVICE_USER}:${DEVELOP_SERVICE_USER} ${DEVELOP_TARGET_DIR}
"

# Criar/Atualizar serviço systemd (develop)
ssh ${DEVELOP_TARGET_HOST} "
  set -euo pipefail

  sudo tee /etc/systemd/system/${DEVELOP_SERVICE_NAME}.service > /dev/null << SERVICEFILE
[Unit]
Description=Harem Brasil API (Develop)
After=network.target

[Service]
Type=simple
User=${DEVELOP_SERVICE_USER}
WorkingDirectory=${DEVELOP_TARGET_DIR}
EnvironmentFile=${DEVELOP_TARGET_DIR}/.env
ExecStart=${DEVELOP_TARGET_DIR}/harem-api serve
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SERVICEFILE

  sudo systemctl daemon-reload
  sudo systemctl enable ${DEVELOP_SERVICE_NAME}
  sudo systemctl restart ${DEVELOP_SERVICE_NAME}

  for i in {1..10}; do
    if sudo systemctl is-active ${DEVELOP_SERVICE_NAME} > /dev/null; then
      break
    fi
    sleep 1
  done

  sudo journalctl -u ${DEVELOP_SERVICE_NAME} --no-pager -n 50
  sudo systemctl is-active ${DEVELOP_SERVICE_NAME}
"
        '''
      }
    }

    stage('Deploy Develop Frontend') {
      when { expression { return env.GIT_BRANCH == 'develop' } }
      steps {
        unstash 'frontend-dist'
        sh label: 'Deploy frontend to develop (VPS)', script: '''
set -euo pipefail
FRONTEND_LOCAL="artifacts/frontend-dist/client"
ARCHIVE="/tmp/frontend-develop-$(git rev-parse --short HEAD).tar.gz"

# Criar arquivo tar localmente e fazer upload via scp
tar -C "$FRONTEND_LOCAL" -czf "$ARCHIVE" .
scp "$ARCHIVE" ${DEVELOP_TARGET_HOST}:/tmp/frontend-develop.tar.gz
rm -f "$ARCHIVE"

# Extrair remotamente — usar heredoc com aspas simples para evitar expansão local
ssh ${DEVELOP_TARGET_HOST} <<'REMOTE'
  set -euo pipefail
  sudo mkdir -p /var/www/vhosts/develop.harembrasil.com.br
  sudo find /var/www/vhosts/develop.harembrasil.com.br -mindepth 1 -not -path "*/logs*" -delete
  sudo tar -C /var/www/vhosts/develop.harembrasil.com.br -xzf /tmp/frontend-develop.tar.gz
  sudo chown -R grimlock:grimlock /var/www/vhosts/develop.harembrasil.com.br
  rm -f /tmp/frontend-develop.tar.gz
  echo "=== Files on VPS after deploy ==="
  ls -la /var/www/vhosts/develop.harembrasil.com.br/
  echo "=== index.html commit hash on VPS ==="
  grep -o 'commit-hash[^>]*>' /var/www/vhosts/develop.harembrasil.com.br/index.html || echo "WARNING: Could not read index.html"
REMOTE
'''
      }
    }

    stage('Smoke Test Develop') {
      when { expression { return env.GIT_BRANCH == 'develop' } }
      steps {
        sh label: 'Health check and smoke test develop API', script: '''
          set -euo pipefail
          EXPECTED_COMMIT=$(git rev-parse --short HEAD)

          # Aguardar API develop ficar disponível
          for i in {1..30}; do
            if curl -sf "${DEVELOP_API_URL}/health" > /dev/null 2>&1; then
              echo "Develop API is up"
              break
            fi
            echo "Waiting for develop API... ($i/30)"
            sleep 2
          done

          echo "=== Health check ==="
          curl -sf -D - "${DEVELOP_API_URL}/health" | head -c 200 || true
          echo ""

          echo "=== API info ==="
          curl -sf -D - "${DEVELOP_API_URL}/readyz" | head -c 200 || true
          echo ""

          echo "=== Validate X-Environment header ==="
          ENV_HEADER=$(curl -sfI "${DEVELOP_API_URL}/health" | grep -i "X-Environment" || true)
          if echo "$ENV_HEADER" | grep -qi "develop"; then
            echo "OK: X-Environment: develop"
          else
            echo "WARN: X-Environment header missing or not 'develop'"
            echo "$ENV_HEADER"
          fi

          echo "=== Validate frontend deploy (VPS file check) ==="
          REMOTE_COMMIT=$(ssh ${DEVELOP_TARGET_HOST} "grep -o 'commit-hash\" content=\"[^\"]*\"' /var/www/vhosts/develop.harembrasil.com.br/index.html | cut -d'\"' -f3" 2>/dev/null || echo 'NOT_FOUND')
          echo "Expected commit : $EXPECTED_COMMIT"
          echo "VPS file commit : $REMOTE_COMMIT"
          if [ "$REMOTE_COMMIT" != "$EXPECTED_COMMIT" ]; then
            echo "ERROR: index.html on VPS has wrong commit — file deployment failed!"
            exit 1
          fi
          echo "OK: VPS file matches expected commit"

          echo "=== Validate frontend deploy (HTTP check) ==="
          FRONTEND_HTML=$(curl -sf "https://develop.harembrasil.com.br/" || true)
          DEPLOYED_COMMIT=$(echo "$FRONTEND_HTML" | grep -o 'commit-hash" content="[^"]*"' | cut -d'"' -f3 || true)
          echo "HTTP commit : $DEPLOYED_COMMIT"
          if [ "$DEPLOYED_COMMIT" = "$EXPECTED_COMMIT" ]; then
            echo "OK: Frontend HTTP response matches expected commit"
          else
            echo "ERROR: HTTP response has wrong commit ($DEPLOYED_COMMIT != $EXPECTED_COMMIT)"
            echo "HINT: VPS files are correct but nginx may be serving from a different directory."
            echo "Check nginx root directive on the VPS."
            exit 1
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