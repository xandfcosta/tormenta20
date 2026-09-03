package api

import (
	"context"
	"strconv"
	"strings"
	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/web/characters"
	"t20engine/web/ui"
	"testing"
)

// corpoDoBotao é uma CÓPIA do helper do `web/characters`, e não um símbolo
// exportado de lá. É a decisão que a fatia da porta deixou escrita: a bancada do
// hospedeiro escreve o que ela afirma, porque importar do que está sendo testado
// faz o teste andar junto com o defeito. São nove linhas de parse.
func corpoDoBotao(t *testing.T, html, rotulo string) string {
	t.Helper()
	i := strings.Index(html, `aria-label="`+rotulo+`"`)
	if i < 0 {
		return ""
	}
	resto := html[i:]
	j := strings.Index(resto, "</button>")
	if j < 0 {
		return resto
	}
	return resto[:j]
}

// Os guardas da cena de PERSONAGENS (ALE-239).
//
// O que se protege é o que o SERVIDOR passou a fazer e a SPA pedia por
// requisição: a Defesa saindo da mesma `ComputeSheetV2` da ficha, e os textos
// de raça saindo do catálogo embutido. Mais a gramática do cursor, que é o que
// a ALE-98 estabeleceu e que um porte distraído quebra sem perceber.

func novaCenaDeHerois(t *testing.T) (*Server, AuthUser) {
	t.Helper()
	s := newTestServer(t)
	catalogos, err := engine.PrimeEngineCatalogs([]byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.catalogs = catalogos
	dono := seedUser(t, s, "jogadora@t20.local")
	u, err := s.queries.GetUserByID(context.Background(), dono)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return s, s.authUser(u)
}

func seedRaca(t *testing.T, s *Server, characterID int64, raca string) {
	t.Helper()
	err := s.queries.CreateRace(context.Background(), sqlcgen.CreateRaceParams{
		Characterid: characterID, Race: raca,
	})
	if err != nil {
		t.Fatalf("seed raça %q: %v", raca, err)
	}
}

// ── a Defesa ─────────────────────────────────────────────────────────────────

// A Defesa do palco tem de ser a MESMA que a ficha mostra. Duas contas
// diferentes para o mesmo número é o defeito que ninguém reporta: a pessoa vê
// 18 na lista e 17 na ficha e conclui que o app é aproximado.
func TestTheStageDefenseIsTheSameAsTheSheetOne(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	id := seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 1 {
		t.Fatalf("esperava 1 herói, veio %d", len(v.Heroes))
	}

	linha, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("personagem: %v", err)
	}
	ficha, err := s.ComputeSheet(context.Background(), linha)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if v.Heroes[0].Defense != strconv.Itoa(ficha.Defense.Total) {
		t.Errorf("Defesa do palco = %q, a da ficha = %d", v.Heroes[0].Defense, ficha.Defense.Total)
	}
}

// Sem motor a Defesa vira TRAVESSÃO, e nunca zero: zero é um valor plausível de
// Defesa, então mostrá-lo seria mentir com um número redondo. E travessão em vez
// de omitir porque uma coluna que some faz o palco dançar ao trocar de herói
// (ALE-99) — é o que a SPA faz, e eu tinha escrito "some" antes de comparar as
// duas telas.
func TestWithoutTheEngineTheDefenseBecomesAnEmDash(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)
	s.catalogs = nil

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar sem motor deveria funcionar: %v", err)
	}
	if len(v.Heroes) != 1 {
		t.Fatalf("a cena caiu sem o motor")
	}
	if v.Heroes[0].Defense != "—" {
		t.Errorf("Defesa = %q sem motor, queria travessão", v.Heroes[0].Defense)
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, ">DEF</dt>") {
		t.Error("a coluna da Defesa sumiu — a fileira muda de tamanho e o palco dança")
	}
	if strings.Contains(html, ">0</dd>") {
		t.Error("a Defesa saiu como ZERO, que é um valor plausível e errado")
	}
}

// ── o catálogo que deixou de ir ao navegador ─────────────────────────────────

// A vaga de criar existe COM O ELENCO VAZIO, e é ela que dá o que fazer. Uma
// tela vazia que não oferece o primeiro passo é uma tela que não ajuda.
func TestWithAnEmptyCastTheCreateSlotIsWhatIsLeft(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if v.HasAny {
		t.Fatal("elenco vazio marcado como cheio")
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Forjar um herói") || !strings.Contains(html, "/personagens/nova") {
		t.Error("a tela vazia não oferece a Forja")
	}
}

// A vaga é POSIÇÃO DE CURSOR e não um link solto (ALE-98): ela declara
// `role=option` e escreve o cursor no foco, como qualquer herói. Um `<a>` no
// fim da fita pareceria igual e as setas o pulariam.
func TestTheCreateSlotIsACursorPositionAndNotALooseLink(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, `aria-label="Forjar um novo herói"`) {
		t.Fatal("a vaga sumiu do filme")
	}
	// Ela pousa em `$cursor == 0`, que é a posição dela porque nenhum id é zero.
	if !strings.Contains(html, "$cursor = 0") {
		t.Error("a vaga não escreve o cursor no foco — as setas passariam por cima dela")
	}
}

// ── a busca ──────────────────────────────────────────────────────────────────

// Os QUATRO campos que a SPA indexa: nome, classe primária, origem e raças.
//
// A primeira versão deste teste chamava a busca de "pela CLASSE" e seedava um
// personagem chamado "Guerreiro" SEM linha de classe nenhuma — ele casava pelo
// NOME, e teria continuado verde com a classe fora do índice. Aqui o nome e a
// classe são propositalmente disjuntos, e cada campo é buscado pelo termo que
// só ELE contém; é isso que faz o teste morrer se algum sair de
// `camposDeBusca`.
func TestTheCharacterSearchLooksAtTheFourFields(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	id := seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedClasse(t, s, id, "Bárbaro", 5)
	seedRaca(t, s, id, "Anão")

	// "Soldado" é a origem que o `seedCharacterAtLevel` grava.
	for termo, campo := range map[string]string{
		"thalen":  "nome",
		"barbaro": "classe",
		"soldado": "origem",
		"anao":    "raça",
	} {
		v, err := characters.New(s).Load(context.Background(), eu.ID, termo)
		if err != nil {
			t.Fatalf("carregar %q: %v", termo, err)
		}
		if len(v.Heroes) != 1 {
			t.Errorf("busca por %s (%q) devolveu %d heróis, queria 1", campo, termo, len(v.Heroes))
		}
	}

	nada, err := characters.New(s).Load(context.Background(), eu.ID, "zzzzzz")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !nada.FilteredAll {
		t.Error("busca sem resultado não foi marcada como tal")
	}
}

// A contagem diz FILTRADOS de TOTAL. Dizer "3 de 3" com sete escondidos pela
// busca esconde justamente o que a pessoa precisa saber para limpar o filtro.
func TestTheCountSaysFilteredOutOfTotal(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "thalen")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 1 || v.Total != 2 {
		t.Errorf("filtrados=%d total=%d, queria 1 de 2", len(v.Heroes), v.Total)
	}
}

// ── os vizinhos que ladeiam o palco ──────────────────────────────────────────

// O peek foi PORTADO na virada, e não reescrito: apagar a tela antiga levaria
// junto os retratos apagados dos vizinhos se ninguém os trouxesse. Aqui se
// afirma o que eles carregam de regra — o nome legível e o caminho de volta.
func TestTheNeighborsFlankTheStageWithAReadableName(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 2 {
		t.Fatalf("esperava 2 heróis, veio %d", len(v.Heroes))
	}
	primeiro, segundo := v.Heroes[0].Name, v.Heroes[1].Name

	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// O primeiro palco olha para a frente, o segundo para trás.
	if !strings.Contains(html, `aria-label="Próximo: `+segundo+`"`) {
		t.Errorf("o palco de %q não mostra %q como próximo", primeiro, segundo)
	}
	if !strings.Contains(html, `aria-label="Anterior: `+primeiro+`"`) {
		t.Errorf("o palco de %q não mostra %q como anterior", segundo, primeiro)
	}
	// O NOME vai no CORPO do botão, e não só no rótulo: duas iniciais não dizem
	// quem vem a seguir, e é para os olhos que ele existe. A primeira versão
	// deste guarda contava o nome no HTML inteiro e sobrevivia à sabotagem —
	// o nome também está no `title`, no `h2` do palco e no rótulo do filme.
	if corpo := corpoDoBotao(t, html, "Próximo: "+segundo); !strings.Contains(corpo, segundo) {
		t.Errorf("o peek de %q não mostra o nome na tela, só em atributo: %q", segundo, corpo)
	}
}

// A vaga de criar tem o ÚLTIMO herói à esquerda: é o caminho de volta para o
// elenco, e sem ele quem anda até o fim do trilho fica sem pista de retorno.
func TestTheCreateSlotShowsTheLastHeroAsTheWayBack(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	ultimo := v.Heroes[len(v.Heroes)-1].Name
	// UMA vez, e é isso que prova que veio da vaga: "Anterior: X" só aparece no
	// palco de quem vem DEPOIS de X, e depois do último herói não há palco de
	// herói nenhum. A única coisa à direita dele no trilho é a vaga.
	if got := strings.Count(html, `aria-label="Anterior: `+ultimo+`"`); got != 1 {
		t.Errorf("o último herói aparece como anterior %d vez(es), queria 1 (a vaga de criar)", got)
	}
}

// Nas PONTAS não há vizinho, e o palco não pode inventar um. Que a CAIXA vazia
// continue ocupando a largura — para o retrato não escorregar ao chegar no
// primeiro herói, família de defeito da ALE-99 — é garantia de LAYOUT, e layout
// só existe num navegador: está no `piloto-datastar.spec.ts`. Aqui fica só o
// que é verdade de dado.
func TestALoneHeroGetsNoInventedNeighbor(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)

	v, err := characters.New(s).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(html, `aria-label="Próximo: `) {
		t.Error("herói único ganhou um próximo que não existe")
	}
}
