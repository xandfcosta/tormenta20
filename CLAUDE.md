## Code style

- Functions: 4-20 lines. Split if longer.
- Files: under 500 lines. Split by responsibility.
- One thing per function, one responsibility per module (SRP).
- Names: specific and unique. Avoid `data`, `handler`, `Manager`.
  Prefer names that return <5 grep hits in the codebase.
- Types: explicit. No `any`, no `Dict`, no untyped functions.
- No code duplication. Extract shared logic into a function/module.
- Early returns over nested ifs. Max 2 levels of indentation.
- Exception messages must include the offending value and expected shape.

## Comments

- Keep your own comments. Don't strip them on refactor — they carry intent and provenance.
- Write WHY, not WHAT. Skip `// increment counter` above `i++`.
- Docstrings on public functions: intent + one usage example.
- Reference issue numbers / commit SHAs when a line exists because of a specific bug or upstream constraint.

## Tests

**The goal is confidence that the app produces the results we want — not
coverage of every small piece of code.** A test earns its place by protecting an
outcome someone would notice breaking. Everything below follows from that.

- **Prefer INTEGRATION.** The default test mounts a real page (or hits a real
  handler through the real router) with the I/O faked at the edge, and asserts
  what the user or the caller actually gets. That band catches composition bugs
  — which is where the defects have actually been — and it is where new coverage
  should go first.
- **Unit-test what carries a RULE**, not what carries plumbing. Modifier
  stacking, PV/PM, rounding, limits, optimistic rollback, wire formats: yes.
  Getters, one-line formatters, a `Set` + `sort` the assertion re-implements,
  and anything the typechecker already guarantees: no.
- **E2E is the smallest possible set.** A Playwright test must justify itself
  with a mechanism only a real browser has — animation timeline, real layout and
  overflow, virtualized lists that measure zero in jsdom, a live stream across two
  servers. "It's a user journey" is NOT a justification: journeys are cheaper
  and steadier as integration tests. E2E is the most expensive and most fragile
  thing in this repo; spend it deliberately.
- **Push each guarantee to the cheapest layer that can hold it.** A server rule
  belongs in a handler test, not in an assertion that a button is missing — UI
  gating is UX, the security boundary is the server. Don't assert the same rule
  in three layers; pick the one that owns it.
- **Bug fixes get a regression test, and it must be proven RED first.** A test
  that was never seen failing is a guess. When a fix and its test land together,
  the commit says how the test was proven to fail.
- **O INSTRUMENTO MENTE COM CARA DE RESULTADO, e o formato é sempre o mesmo:**
  a infraestrutura em volta da medição destrói a medição, e o que sobra parece
  um dado. Quatro casos num dia só, e nenhum deles pareceu erro na hora:
  - Um `finally` que fecha contextos de browser lançou "Failed to find context"
    e **substituiu o erro de verdade** do teste. Limpeza ganha `catch`, sempre:
    *limpeza não pode falar mais alto que o defeito* (ALE-245).
  - A suíte rodada com `| tail -15` deixou 981 bytes de stdout; procurar uma
    linha ali e não achar foi lido como "o evento não aconteceu", quando era
    "o canal não existe" (ALE-238).
  - Uma sonda instalou `MutationObserver` em `document.body` num `addInitScript`,
    onde o `body` ainda é `null` — e a ausência de mutação virou conclusão
    (ALE-199).
  - Um teste de componente não passava o tamanho e comparava o valor do
    servidor com o **default** da SPA, passando verde sobre nada.

  **O controle é barato e é obrigatório: antes de ler AUSÊNCIA como evidência,
  provar que o canal estaria lá se o evento tivesse acontecido.** Procurar no
  mesmo arquivo uma linha que sai SEMPRE; conferir que a sonda vê o caso
  positivo conhecido. Sem isso, "não reproduzi" não é evidência de ausência —
  é ausência de evidência, e as duas se parecem no terminal.

  **E o canal pode morrer DEPOIS de instalado: um observador precisa afirmar
  que o DOCUMENTO em que ele foi instalado ainda é o mesmo.** Navegação
  descarta o documento, e com ele o `MutationObserver` — a lista de mutações
  volta VAZIA, que é a mesma coisa que "nada mudou". O guarda
  `não desanexa a cena` (ALE-238) passa exatamente no PIOR caso: a cena não
  desanexou porque a cena deixou de existir. Não é um teste que falha em
  detectar; é um teste que **afirma o oposto do que aconteceu**, e ele só foi
  descoberto porque um clique estourou antes e denunciou a navegação.

  O mesmo vale para qualquer sonda de vida longa — `addEventListener`,
  `PerformanceObserver`, um `page.on(...)` cujo alvo recarregou. Afirme o
  documento antes de afirmar o silêncio.
- **Um guarda só mede o que ele VISITA.** Cobertura de contraste, de tipografia
  e de leiaute é função de onde o teste NAVEGA, não de quantas asserções ele
  tem. Dois defeitos de contraste sobreviveram anos com o guarda no ar porque
  ele nunca abria um popover nem entrava no livro de campanhas (ALE-237); a
  mesma forma reapareceu na tipografia (ALE-252).

  **Mas "põe a cena na lista" só resolve enquanto as cenas forem contáveis, e
  vale saber a diferença.** Um guarda que mede a folha do grimório cobre 43
  telas por AMOSTRAGEM — ele mede uma e vale para todas porque todas passam
  pelos mesmos componentes. No dia em que uma tela escreve as classes à mão, o
  regime vira ENUMERAÇÃO: uma entrada por cena, para sempre, e a que alguém
  esquecer nasce sem medição — em silêncio, que é a marca desta família.
  Enumerar é remendo; **o que restaura a amostragem é a tela nova passar pelos
  componentes da casa.** Escolher o remendo dá sensação de conserto e deixa o
  buraco aberto (ALE-252).
- **Delete tests that cost more than they protect**: assertions on class names
  and DOM shape nobody promised, tests that re-derive the expected value by
  running the implementation, tests over code that is dead or gated out of the
  bundle. Green tests over unused code are the worst kind of debt — they charge
  maintenance and protect nothing.
- Data transcribed from the book (catalogs) is validated by SCHEMA at the dump,
  not by an `expect` per field repeating the same number. Pin the *exception*
  (the trap in the table), never the whole table.
- Mock external I/O (API, DB, filesystem) with named fake classes, not inline
  stubs. Tests must be F.I.R.S.T: fast, independent, repeatable,
  self-validating, timely.
- Write the test first when you can state the expected result first; when
  chasing a bug, reproduce it in the browser or a handler first, then encode it.
- **Cutting order: write the replacement, watch it go green, THEN delete.**
  Deleting first opens a blind window.

## Dependencies

- Inject dependencies through constructor/parameter, not global/import.
- Wrap third-party libs behind a thin interface owned by this project.

## Structure

- Follow the framework's convention (Rails, Django, Next.js, etc.).
- Prefer small focused modules over god files.
- Predictable paths: controller/model/view, src/lib/test, etc.

## Formatting

- Use the language default formatter (`cargo fmt`, `gofmt`, `prettier`, `black`, `rubocop -A`). Don't discuss style beyond that.

## Logging

- Structured JSON when logging for debugging / observability.
- Plain text only for user-facing CLI output.

## Linguagem ubíqua

- **[GLOSSARIO.md](GLOSSARIO.md) — uma palavra por conceito, e um conceito por
  palavra.** Leia antes de nomear qualquer coisa que o usuário vá ler ou que vá
  virar identificador. Ele tem a coluna dos termos PROIBIDOS (é "campanha", não
  "campanha"), as colisões abertas que não se consertam por palpite, e a regra da
  costura pt-BR/inglês. Termo novo: escreva a linha do glossário ANTES do código.

## Reference
- Use [Tormenta 20 book](/t20-book.pdf) as reference for rules

## Package guides
- Frontend has its own adapted rules: [frontend/CLAUDE.md](frontend/CLAUDE.md)
  (FSD layers, thin TanStack routes, styling tokens, vitest/tsgo/eslint).
  When working under `frontend/`, follow it in addition to this file.
- O backend `engine-go/` (Go) tem guia próprio: [engine-go/CLAUDE.md](engine-go/CLAUDE.md)
  — regenerar oráculo é ato deliberado, citação de página conferida, o gerador de
  tipos da fronteira, validação de schema dos catálogos. HTTP API on :3001, the rules engine, and
  the same engine compiled to WASM for the browser. One process serves the SPA,
  the API and the event stream in production (`STATIC_DIR`) — there is no nginx and no
  compose. The NestJS backend was removed once nothing consumed it.
