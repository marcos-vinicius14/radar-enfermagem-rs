# Changelog

Todas as alterações notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e este projeto adere ao [Semantic Versioning (SemVer)](https://semver.org/lang/pt-BR/).

---

## [0.3.1] - 2026-09-26

### Corrigido
- **Alinhamento e Responsividade do Rodapé (`.site-footer`):**
  - Reestruturação do grid (`.footer-grid`) com colapso responsivo e empilhamento fluido em tablets (`@media (max-width: 768px)`) e smartphones (`@media (max-width: 640px)`).
  - Padronização do alinhamento vertical entre a coluna da marca institucional e a coluna de links da comunidade.
  - Harmonização de espaçamentos, bordas e contraste elevado (WCAG AA com `--color-text-muted`) no card de aviso legal (`.footer-disclaimer-card`) e barra de copyright.
  - Aumento das áreas de toque (*touch targets*) dos links no mobile para navegação mais ergonômica.
- **Simplificação e Discreção do Aviso de Versão Beta (`.beta-notice`):**
  - Remoção da faixa verde de 100vw e da frase extensa que poluíam o topo da tela e competiam com a barra de busca principal.
  - Redesenho do aviso para uma badge centralizada, sutil e contida (`.beta-card`), respeitando a largura máxima do layout e mantendo a interface limpa e focada na pesquisa de vagas.

---

## [0.3.0] - 2026-09-26

### Adicionado
- **Filtros de Setores com Múltipla Seleção e Toggle:** Badges de especialidades (UTI/CTI, Centro Cirúrgico, Emergência/PA, Pediatria/Neo, Hemodiálise, etc.) agora suportam seleção cumulativa e desseleção ao clicar novamente.
- **Busca SQL Multi-Termos:** Query `SearchJobs` atualizada no PostgreSQL para buscar termos combinados via `ILIKE ANY (unnest(string_to_array($1, ',')))`.
- **Comando `make dev`:** Atalho de DX para orquestrar inicialização do banco PostgreSQL no Docker, execução de migrações pendentes e inicialização do servidor Go em um único comando.
- **Documentação de Arquitetura no README:** Seção detalhando a restrição do `//go:embed` e o papel do pacote `apps/web` no monorepo de binário único.

### Modificado
- **Reorganização Modular dos Coletores (`internal/collector/sources`):** Separação do motor de ingestão (core) das implementações concretas de scraping, agrupadas por plataforma de recrutamento:
  - `sources/gupy/`: Santa Casa de Porto Alegre, Hospital Moinhos de Vento, Hospital São Lucas PUCRS, Unimed Porto Alegre e Doctor Clin.
  - `sources/senior/`: Hospital Mãe de Deus.
  - `sources/vagascom/`: Grupo Fleury / Weinmann.
  - `sources/custom/`: Hospital de Clínicas de Porto Alegre (HCPA) e Rede Saúde Divina.
  - `sources/defaults.go`: Fachada centralizada para registro de coletores padrão (`NewDefaultRegistry`).
- **Constante Global `DefaultUserAgent`:** Centralizada no contrato canônico do coletor em `collector.go` e consumida de forma unificada pelas plataformas externas.

### Corrigido
- **Perda de Estilos e CSS ao Recarregar a Página (F5):**
  - Diferenciação de ETag HTTP com flag `frag:%t` e inclusão do cabeçalho `Vary: HX-Request` para evitar que requisições parciais HTMX sobrescrevam o cache da página completa com `304 Not Modified`.
  - Cabeçalho `Cache-Control: no-store` aplicado explicitamente aos fragmentos HTML de listagem.
  - Isenção do middleware de rate limiting para rotas estáticas (`/static/*`) e probes de saúde (`/health`, `/ready`).
  - Correção de seletor inválido de desativação de elementos no HTMX (`hx-disabled-elt`).
- **Persistência Visual de Badges Ativos:** Ajustada classe CSS dos chips (`bg-emerald-600 text-white`) para refletir o estado selecionado tanto nas interações dinâmicas quanto no Server-Side Rendering (SSR).

---

## [0.2.1] - 2026-09-21

### Adicionado
- **Rotina de Expurgo de Vagas (Pruner):** Remoção automática de vagas fora do escopo assistencial de enfermagem no startup da aplicação e a cada 12 horas.
- **Filtro Semântico Rigoroso:** Refinamento dos critérios de inclusão de vagas para assegurar aderência estrita ao perfil de Enfermagem e Técnico em Enfermagem.

### Corrigido
- Regras de verificação ortográfica e linter no pipeline de CI.

---

## [0.2.0] - 2026-09-21

### Adicionado
- **Governança Open Source:** Licença oficial GNU Affero General Public License v3.0 (AGPLv3) e Guia de Contribuição (`CONTRIBUTING.md`).
- **Diretrizes de Agentes:** Formalização das regras canônicas de desenvolvimento, padrões de commit e fluxos de trabalho em `.agents/rules/` e `AGENTS.md`.
- **Critérios de Fim do Beta:** Documentação dos marcos para lançamento da versão estável 1.0.0 em `docs/roadmap.md`.

---

## [0.1.0] - 2026-09-21

### Adicionado
- **Primeira Versão Oficial (MVP):**
  - Backend em Go 1.23+ com roteamento Chi e logs estruturados `log/slog`.
  - Banco de dados PostgreSQL 18-alpine com identificadores nativos `UUIDv7`, migrações versionadas com `golang-migrate` e queries tipadas via `sqlc`.
  - Pipeline de coleta (scraping defensivo) com clientes HTTP resilientes, rate limiting por domínio, retries exponenciais e deduplicação lógica via hash SHA-256 (`fingerprint`).
  - Coletores para 9 instituições: Santa Casa, Moinhos de Vento, São Lucas PUCRS, HCPA, Unimed Porto Alegre, Mãe de Deus, Divina Providência, Doctor Clin e Grupo Fleury.
  - Frontend Server-Side Rendering (SSR) com templates em Go e fragments dinâmicos via HTMX.
  - Endpoints de API REST v1 para vagas, instituições e cidades.
  - Orquestração de ambiente local com Docker Compose e `Makefile`.
