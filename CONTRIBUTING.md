# Guia de Contribuição — Radar Enfermagem RS

> Bem-vindo(a) ao repositório do **Radar Enfermagem RS**!  
> Agradecemos imensamente o seu interesse em contribuir com esta iniciativa de impacto social.

---

## 🧭 1. Princípios e Compromisso Ético

O **Radar Enfermagem RS** é uma plataforma comunitária, cidadã e estritamente **sem fins lucrativos**. O objetivo é monitorar, centralizar e simplificar a busca por vagas de Técnico em Enfermagem e Enfermagem em Porto Alegre e região metropolitana.

Ao contribuir com este projeto, você concorda com nossos compromissos fundamentais:
- **Acesso Gratuito e Universal:** A plataforma nunca terá paywalls, planos pagos para candidatos ou cobrança de qualquer natureza.
- **Privacidade e Proteção de Dados:** Não capturamos, armazenamos ou comercializamos dados pessoais de currículos. Todas as candidaturas são direcionadas para as páginas oficiais das instituições contratantes.
- **Transparência e Respeito às Fontes:** Todas as vagas são exibidas com atribuição clara à instituição e link direto para o anúncio oficial.

---

## 🌳 2. Modelo de Branches (Git Flow)

Adotamos um fluxo de desenvolvimento estruturado para garantir a estabilidade do sistema em produção:

* **`main` (Produção):**
  * Branch estável e protegida contra push direto.
  * Todo código em `main` reflete o ambiente de produção e é versionado por tags SemVer (`vX.Y.Z`).
  * Atualizações só entram na `main` através de Pull Requests aprovados a partir de `develop`.
* **`develop` (Ambiente de Integração):**
  * Branch principal de trabalho e integração de novas funcionalidades.
  * Todo Pull Request deve ter como destino a branch `develop`.
* **Branches de Trabalho:**
  * Crie branches a partir da `develop` utilizando a convenção:
    * `feat/nome-da-funcionalidade` (novas funcionalidades)
    * `fix/descricao-da-correcao` (correções de bugs)
    * `docs/descricao-da-melhoria` (documentação e guias)
    * `refactor/area-refatorada` (refatorações sem alteração de comportamento)
    * `ci/descricao-da-esteira` (melhorias em pipelines e automação)

---

## 📝 3. Padrão de Commits (Conventional Commits)

Utilizamos a convenção do [Conventional Commits](https://www.conventionalcommits.org/pt-br/v1.0.0/) para manter o histórico de alterações legível e automatizável:

```text
tipo(escopo opcional): descrição curta no imperativo

[corpo opcional explicando o porquê da mudança]
```

### Exemplos:
* `feat(collector): adicionar coletor do Hospital Divina Providência`
* `fix(web): corrigir contraste de cores na badge de turno noturno`
* `test(database): adicionar testes de concorrência para inserção em lote`
* `docs(readme): atualizar instruções de execução com docker compose`
* `refactor(scheduler): simplificar lógica de cancelamento do cron`

---

## 🧪 4. Estratégia de Testes: Troféu de Testes & TDD

Seguimos a filosofia do **Troféu de Testes (Testing Trophy)**:

```text
        /   E2E   \        -> Fluxos ponta a ponta
       / INTEGRATION \   -> [CORAÇÃO DA SUÍTE] Banco real PostgreSQL 18
      /     UNIT      \  -> Algoritmos puros e regras matemáticas
     / STATIC ANALYSIS \ -> Compilador Go, go vet e formatação
```

1. **Priorize Testes de Integração:** O coração da nossa suíte são testes reais rodando contra o PostgreSQL 18 em container Docker. Não utilizamos mocks frágeis de banco de dados (`sqlmock`).
2. **Ciclo TDD:**
   - **Red:** Escreva o teste automatizado descrevendo o comportamento esperado e observe-o falhar.
   - **Green:** Implemente o código mínimo necessário para fazer o teste passar.
   - **Refactor:** Limpe e formate o código mantendo todos os testes verdes.
3. **Detecção de Concorrência Obrigatória:** Todos os testes de backend Go devem passar com `-race`:
   ```bash
   make test
   ```

---

## 🇧🇷 5. Padrão de Idioma Obrigatório (pt-BR)

- O projeto é voltado aos profissionais de saúde do Brasil.
- **Todas as mensagens de erro da API, retornos HTTP, validações de domínio e erros sentinela DEVEM ser em Português (pt-BR)** (ex.: `"vaga não encontrada"`, `"o título da vaga é obrigatório"`, `"portal retornou código HTTP 403"`).
- Comentários de código e documentação operacional devem priorizar clareza em português.
- Nomes de símbolos de código (funções, structs, interfaces, variáveis) seguem a convenção idiomática em inglês técnico (`Job`, `Repository`, `Insert`, `Normalize`).

---

## 💻 6. Como Rodar o Projeto Localmente

### Pré-requisitos
- [Go 1.23+](https://golang.org/)
- [Docker](https://www.docker.com/) e Docker Compose
- `make`

### Passo a Passo

```bash
# 1. Clone o repositório
git clone https://github.com/marcos-vinicius14/radar-enfermagem-rs.git
cd radar-enfermagem-rs

# 2. Crie sua branch a partir de develop
git checkout develop
git checkout -b feat/minha-contribuicao

# 3. Configure as variáveis de ambiente
cp .env.example .env

# 4. Inicie o banco de dados PostgreSQL 18
make up

# 5. Aplique as migrações de banco
make migrate-up

# 6. Execute a suíte de testes com detecção de concorrência
make test

# 7. Inicie a aplicação localmente
make run
```
Acesse a aplicação no navegador em: `http://localhost:8080`.

---

## ✅ 7. Checklist Antes de Abrir seu Pull Request

Antes de submeter o seu Pull Request, certifique-se de que:

- [ ] A branch base do PR é a **`develop`** (não a `main`).
- [ ] O código segue o ciclo **TDD** e possui testes cobrindo a nova funcionalidade ou correção.
- [ ] O comando `make test` passa com sucesso e sem detecção de *race conditions*.
- [ ] A formatação estática e linter passam sem avisos:
  ```bash
  cd apps/api && go vet ./... && gofmt -s -w .
  cd apps/web && go vet ./... && gofmt -s -w .
  ```
- [ ] Todas as mensagens de erro e retornos para o usuário estão em **Português (pt-BR)**.
- [ ] O título e a descrição do PR explicam claramente o problema resolvido e as decisões tomadas.

---

Obrigado por ajudar a construir um **Radar Enfermagem RS** cada vez melhor para os profissionais de saúde! 🏥💙
