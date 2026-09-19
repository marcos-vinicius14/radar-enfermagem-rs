# Regra: Prática Obrigatória de TDD e Estratégia de Troféu de Testes

## 1. Estratégia do Troféu de Testes (Testing Trophy)
Adotamos como filosofia central a estratégia do **Troféu de Testes**, priorizando testes que oferecem o maior retorno sobre investimento (ROI) e maior confiança operacional:

1. **Análise Estática / Compilador**: Linters (`go vet`, `golangci-lint`), tipagem forte do Go e geração segura do `sqlc`.
2. **Testes Unitários (Focados e Enxutos)**: Reservados para lógica pura, algoritmos isolados (ex: cálculo de fingerprint, normalização de texto, regras puras de validação). Evitar testes unitários excessivos com mocks artificiais que apenas validam se uma interface foi chamada.
3. **Testes de Integração (O Coração do Troféu - Foco Prioritário)**: **Foco principal da nossa suíte de testes.** Testam múltiplos componentes trabalhando juntos de verdade: repositórios contra PostgreSQL real (queries sqlc, constraints, índices, transações, integridade referencial), handlers HTTP com middlewares e serviços reais.
4. **Testes End-to-End (E2E)**: Cenários pontuais e críticos de ponta a ponta validando o fluxo completo do sistema.

> **Princípio:** *"Escreva testes. Não muitos. Principalmente de integração."* Testes de integração oferecem muito mais confiança contra regressões e refatorações do que testes unitários acoplados a mocks.

## 2. Test-Driven Development (TDD)
- **Ciclo Red-Green-Refactor Obrigatório**: Aplicado prioritariamente a nível de integração (e unitário para algoritmos puros):
  1. **Red**: Escreva primeiro o teste automatizado (de integração ou unitário) que descreve o comportamento esperado e observe-o falhar.
  2. **Green**: Implemente o código estritamente necessário para fazer o teste passar.
  3. **Refactor**: Refatore e limpe o código mantendo toda a suíte passando.

## 3. Testes Relevantes (Valor Real vs. Apenas Métricas)
- O objetivo dos testes **não é inflar métricas de cobertura de código sintética**, mas sim garantir confiabilidade real e manutenibilidade.
- **Evite mocks desnecessários**: Mocks frágeis que testam implementação em vez de comportamento devem ser evitados. Prefira testar contra instâncias reais de banco (PostgreSQL em container) para validar SQL real, triggers, constraints e conversões de tipo.
- Testes devem verificar contratos, comportamentos observáveis e efeitos colaterais reais.

## 4. Cobertura de Edge Cases (Além do Caminho Feliz)
- É obrigatório testar cenários adversos e casos de borda:
  - Entradas nulas, vazias, strings maliciosas ou fora dos limites aceitáveis.
  - Violações de constraints (`UNIQUE`, chaves estrangeiras, limites de tamanho, ranges inválidos).
  - Comportamentos sob limites de paginação (`LIMIT`, `OFFSET`, ordenação determinística).
  - Erros e falhas em dependências (timeouts, registros não encontrados `ErrNoRows`).

## 5. Malha de Segurança contra Regressões e Bugs
- A suíte de testes de integração deve funcionar como uma rede de segurança sólida contra regressões.
- Sempre que um bug for identificado ou reportado, o primeiro passo deve ser escrever um teste de integração que reproduza o problema antes de aplicar a correção.

