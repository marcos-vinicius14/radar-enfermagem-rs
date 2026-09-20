# Guia Operacional — Deploy Contínuo em Produção com Coolify (VPS)

> Manual de configuração e operação da esteira de entrega contínua (CD) do **Radar Enfermagem RS** em VPS própria utilizando **Coolify**, disparada automaticamente por Tags e Releases do GitHub.

---

## 🧭 1. Visão Geral da Arquitetura

O ciclo de vida de uma release segue o fluxo canônico de Git Flow com entrega contínua automatizada:

```text
feature branch
   ↓
Pull Request (develop / main)
   ↓
CI no GitHub Actions (Lint, Testes Unitários, Integração PostgreSQL 18, Build)
   ↓
Merge na branch main
   ↓
make release VERSION=vX.Y.Z
   ↓
Publicação da GitHub Release com Notas de Versão
   ↓
GitHub Actions (deploy-prod.yaml) é acionado
   ↓
Disparo do Webhook do Coolify via HTTPS
   ↓
Coolify na VPS faz Git Pull e sobe os containers (compose.prod.yaml)
```

---

## ⚙️ 2. Configuração da Aplicação no Coolify

### 2.1 Criar o Recurso Docker Compose
1. Acesse o painel web do seu **Coolify** na VPS (`https://seu-coolify-domain.com`).
2. Clique no seu Projeto / Ambiente (ex: `Production`).
3. Clique em **+ New Resource** e selecione **Docker Compose**.
4. Conecte o repositório GitHub `marcos-vinicius14/radar-enfermagem-rs`.
5. Em **Branch**, selecione `main`.
6. Em **Docker Compose Location**, aponte para:
   ```text
   deployments/compose.prod.yaml
   ```

### 2.2 Configurar Variáveis de Ambiente
Na aba **Environment Variables** da aplicação no Coolify, cadastre:

| Variável | Descrição | Exemplo |
| :--- | :--- | :--- |
| `DB_USER` | Usuário do PostgreSQL de produção | `postgres` |
| `DB_PASSWORD` | Senha forte gerada para o banco | `uma_senha_muito_segura_123` |
| `DB_NAME` | Nome do banco de dados | `radar_enfermagem` |
| `COLLECTOR_RUN_ON_STARTUP` | Executa coleta inicial ao iniciar | `true` |

*(O container da API aplicará as migrações automaticamente e efetuará a primeira coleta de vagas na inicialização).*

### 2.3 Configurar Domínio e SSL
1. Na aba de configurações da aplicação no Coolify, defina o domínio público do serviço (ex: `https://vagas.radarenfermagem.com.br`).
2. O Coolify (via Traefik ou Caddy integrado) emitirá e renovará o certificado Let's Encrypt SSL automaticamente.

---

## 🔐 3. Configuração dos Segredos no GitHub

Para que o GitHub Actions tenha autorização para notificar o Coolify:

1. No painel do Coolify, acesse sua aplicação e clique na aba **Webhooks**.
2. Copie a URL do **Deploy Webhook** (formato: `https://coolify.seudominio.com/api/v1/deploy?uuid=...`).
3. No repositório GitHub do projeto, navegue até:
   **Settings** → **Secrets and variables** → **Actions** → **New repository secret**.
4. Crie o seguinte segredo:
   * **Nome:** `COOLIFY_WEBHOOK_URL`
   * **Valor:** Cole a URL copiada do webhook.
5. *(Opcional)* Se a sua instalação do Coolify exigir token de API no header Authorization:
   * Crie o segredo `COOLIFY_TOKEN` com o token gerado em `Security -> API Tokens` (com permissão de `deploy`).

---

## 🚀 4. Como Publicar uma Nova Versão em Produção

Após realizar o merge do PR aprovado na branch `main`:

### Opção 1: Via Linha de Comando (Recomendado)
Execute na raiz do projeto:

```bash
# 1. Garanta que está com a branch main atualizada
git checkout main
git pull origin main

# 2. Execute o comando de release informando a versão SemVer
make release VERSION=v1.0.0
```

O comando irá automaticamente:
1. Validar se a sintaxe da versão é SemVer (`vX.Y.Z`).
2. Executar toda a suíte de testes locais (`make lint` e `make test`).
3. Criar a tag git correspondente.
4. Publicar a **GitHub Release** com changelog gerado automaticamente.
5. Disparar o workflow de deploy do GitHub Actions.

---

### Opção 2: Via Interface Web do GitHub
1. Acesse o repositório no GitHub e vá até **Releases** → **Draft a new release**.
2. Clique em **Choose a tag**, digite `v1.0.0` e selecione *Create new tag: v1.0.0 on main*.
3. Clique em **Generate release notes** para preencher o changelog.
4. Clique em **Publish release**.

---

### Opção 3: Re-deploy Manual via GitHub Actions
Se precisar re-executar o deploy de uma versão existente sem criar uma nova tag:
1. Acesse a aba **Actions** no GitHub.
2. Selecione o workflow **Deploy Produção (Coolify)**.
3. Clique em **Run workflow**, informe a tag desejada (opcional) e dispare.

---

## 📊 5. Monitoramento e Rollback

### Acompanhar o Deploy
* No GitHub Actions, cada execução de deploy gera um resumo visual (`Step Summary`) com a versão, commit, autor e status do disparo.
* No painel do Coolify, você pode acompanhar os logs de compilação em tempo real na aba **Deployments**.

### Rollback Imediato em Caso de Emergência
Se uma versão implantada apresentar algum problema inesperado em produção:
1. Acesse o painel do **Coolify**.
2. Vá até a aba **Deployments** da aplicação.
3. Localize o último deploy bem-sucedido anterior e clique em **Redeploy**.
4. O Coolify restaurará a versão estável em poucos segundos sem necessidade de novo commit.
