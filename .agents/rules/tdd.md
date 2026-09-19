# Regra: Prática Obrigatória de TDD e Qualidade de Testes

## 1. Test-Driven Development (TDD)
- **Ciclo Red-Green-Refactor Obrigatório**: Sempre que for criar ou alterar funcionalidades, regras de negócio, utilitários ou endpoints:
  1. **Red**: Escreva primeiro os testes automatizados que descrevem o comportamento esperado e garanta que falhem inicialmente.
  2. **Green**: Implemente o código estritamente necessário para fazer os testes passarem.
  3. **Refactor**: Refatore e limpe o código mantendo todos os testes passando.

## 2. Testes Relevantes (Valor Real vs. Apenas Métricas)
- O objetivo dos testes **não é apenas inflar métricas de cobertura de código**, mas sim garantir confiabilidade real e manutenibilidade.
- Evite testes triviais ou redundantes (ex: testar apenas se um mock foi invocado sem validar o resultado, ou testar getters/setters óbvios).
- Testes devem verificar contratos, comportamentos esperados e efeitos colaterais reais.

## 3. Cobertura de Edge Cases (Além do Caminho Feliz)
- É obrigatório testar cenários adversos e casos de borda:
  - Entradas nulas, vazias, indefinidas, tipos inesperados ou fora dos limites aceitáveis.
  - Erros e falhas em dependências externas (timeouts, exceções, respostas inesperadas de APIs/banco).
  - Comportamentos sob limites máximos/mínimos de dados.
  - Validações de permissão e regras de segurança.

## 4. Malha de Segurança contra Regressões e Bugs
- A suíte de testes deve funcionar como uma rede de segurança sólida contra a maior parte dos bugs conhecidos.
- Sempre que um bug for identificado ou reportado, o primeiro passo deve ser escrever um teste automatizado que reproduza o problema antes de aplicar a correção.
