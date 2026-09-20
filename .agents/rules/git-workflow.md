# Regra: Fluxo Git e Política Estrita de Commits e Pushes

> Manual canônico de governança de código, ciclo de trabalho Git e controle de alterações para agentes de IA e desenvolvedores do projeto **Radar Enfermagem RS**.

---

## 1. Regra de Ouro: Autorização Formal Prévia Obrigatória

> [!CAUTION]
> **É TERMINANTEMENTE PROIBIDO que qualquer agente de IA execute comandos `git commit`, `git push`, criação ou mesclagem de Pull Requests de forma autônoma sem antes receber autorização formal, explícita e inequívoca do usuário.**

- **Nenhum commit surpresa:** O agente não deve agrupar tarefas de código com commits automáticos.
- **Nenhum push automático:** Todo push para branches remotas (`origin`) requer permissão expressa.
- **Nenhum merge autônomo:** Nenhuma mesclagem de PR na branch `main` ou `develop` pode ser realizada sem o comando ou consentimento direto do usuário.

---

## 2. Protocolo de Trabalho do Agente

Ao trabalhar em qualquer funcionalidade, correção, refatoração ou documentação, o agente deve seguir rigorosamente as etapas:

1. **Desenvolvimento e Testes:**
   - Realizar as alterações necessárias nos arquivos do projeto.
   - Escrever testes automatizados (TDD / Troféu de Testes) e validar (`make test`, `go vet`, `gofmt`).
2. **Apresentação de Resultados (Stop & Ask):**
   - Executar `git status` e exibir de forma transparente a lista de arquivos criados/modificados/deletados.
   - Exibir o resumo do que foi feito e os testes que foram validados.
   - Apresentar a sugestão de mensagem de commit (seguindo o padrão *Conventional Commits*).
   - **PARAR e perguntar formalmente ao usuário:**
     > *"As alterações foram validadas e os testes passaram. Você autoriza a realização do commit com a mensagem `...` e o push para a branch `...`?"*
3. **Execução Condicional:**
   - Somente após o usuário responder afirmativamente (ex.: *"pode commitar"*, *"autorizo"*, *"faça o commit e push"*), o agente poderá executar os comandos Git.
   - Caso o usuário aponte divergências ou solicite ajustes, o agente continuará refinando o código no working tree local, repetindo a etapa 2.

---

## 3. Estratégia de Branches (Git Flow)

Adotamos o fluxo de branches estruturado:

```text
  main (Protegida)   <=================== PR de Release / Promoção (com CI verde)
      ^
      |
  develop (Integração) <----------------- PRs de Features e Fixes (com CI verde)
      ^
      |--- feat/nome-da-feature
      |--- fix/nome-do-bug
      |--- chore/ajustes-de-infra
```

### 3.1 Diretrizes de Branches
- **`main` (Produção):**
  - Contém exclusivamente código pronto e validado em produção.
  - Possui regra de **Branch Protection ativa**: commits diretos e force pushes são rejeitados pelo GitHub.
  - Atualizações chegam à `main` exclusivamente via Pull Request vindo da `develop`.
- **`develop` (Desenvolvimento e Integração Contínua):**
  - Branch padrão de integração onde todas as branches de trabalho convergem.
- **Branches de Trabalho (`feat/*`, `fix/*`, `chore/*`, `docs/*`):**
  - Devem ser criadas sempre a partir da `develop`.
  - Nomenclatura contextual e descritiva (ex: `fix/filtro-enfermagem-pucrs`, `feat/scheduler-coleta`).

---

## 4. Padrão de Mensagens de Commit (Conventional Commits)

Todas as mensagens de commit devem seguir rigorosamente o padrão *Conventional Commits*:

```text
<tipo>(<escopo>): <descrição em português ou inglês contextual>

[corpo opcional detalhando o motivo]
```

### Tipos Permitidos:
- `feat`: Nova funcionalidade para o usuário ou API.
- `fix`: Correção de bug ou regressão.
- `test`: Adição ou modificação de testes automatizados.
- `refactor`: Alteração de código que não adiciona feature nem corrige bug.
- `chore`: Atualização de dependências, automações, scripts de build, configurações.
- `docs`: Documentação, README, manuais, guias.
- `style`: Formatação, espaçamento, sem alteração de lógica.

---

## 5. Checklist de Verificação Antes de Solicitar Autorização

Antes de perguntar ao usuário se ele autoriza o commit:
- [ ] O código foi compilado e testado localmente (`make test`)?
- [ ] A análise estática passou sem avisos (`go vet ./...`, `gofmt -s -w .`)?
- [ ] Nenhuma credencial, arquivo temporário, log ou chave secreta foi adicionada ao working tree?
- [ ] O `git status` reflete estritamente os arquivos relevantes à tarefa solicitada?
