package api

import (
	"context"
	"strconv"
	"strings"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/characters"
	"t20engine/serve/web/ui"
	"testing"
)

// corpoDoBotao é uma CÓPIA do helper do `web/characters`, e não um símbolo
// exportado de lá: importar do que está sendo testado faz o teste andar junto
// com o defeito. São nove linhas de parse.
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

// Os guardas da cena de PERSONAGENS: a Defesa saindo da mesma conta da ficha, os
// textos de raça saindo do catálogo embutido, e a gramática do cursor.

func novaCenaDeHerois(t *testing.T) (*Server, AuthUser) {
	t.Helper()
	s := newTestServer(t)
	catalogos, err := engine.PrimeEngineCatalogs([]byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.primeCatalogs(catalogos)
	dono := seedUser(t, s, "jogadora@t20.local")
	u, err := s.queries.GetUserByID(context.Background(), dono)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return s, s.accountRules().authUser(u)
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
	id := seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", "Guerreiro", 5, -4, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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
	ficha, err := s.sheetRules().ComputeSheet(context.Background(), linha)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if v.Heroes[0].Defense != strconv.Itoa(ficha.Defense.Total) {
		t.Errorf("Defesa do palco = %q, a da ficha = %d", v.Heroes[0].Defense, ficha.Defense.Total)
	}
}

// Defesa que o motor não soube dar vira TRAVESSÃO, e nunca zero: zero é um valor
// plausível de Defesa, então mostrá-lo seria mentir com um número redondo. E
// travessão em vez de omitir porque uma coluna que some faz o palco dançar ao
// trocar de herói.
//
// # Por que ele chama o `HeroCardOf` e não a cena
//
// Porque o caminho pela cena passou a ser IMPOSSÍVEL. Ele montava um servidor,
// fazia `primeCatalogs(nil)` e pedia a cena — e desde a ALE-355 o `cmd/api` se
// recusa a subir sem catálogo e o agregado se recusa a montar sem ele, porque o
// poço de PV é derivado. Um caso que arranja um estado que o app não alcança
// mede outra coisa.
//
// O que SOBRA de alcançável é a outra metade da mesma guarda: o `sheet.Compute`
// devolver erro para um agregado que o motor não digere. O travessão é a
// resposta das duas, e é aqui que ele mora.
func TestADefenseTheEngineCannotGiveBecomesAnEmDash(t *testing.T) {
	cartao := characters.HeroCardOf(nil, sheet.CharacterDTO{ID: 1, Name: "Thessa", Level: 5})

	if cartao.Defense != "—" || cartao.DefenseVs != "—" {
		t.Errorf("Defesa = %q / %q, queria travessão nos dois",
			cartao.Defense, cartao.DefenseVs)
	}
	html, err := ui.RenderFragment(t.Context(),
		characters.SceneBody(characters.View{Heroes: []characters.HeroCard{cartao}, Total: 1, HasAny: true}))
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
	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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

// A vaga é POSIÇÃO DE CURSOR e não um link solto: ela declara
// `role=option` e escreve o cursor no foco, como qualquer herói. Um `<a>` no
// fim da fita pareceria igual e as setas o pulariam.
func TestTheCreateSlotIsACursorPositionAndNotALooseLink(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", "Guerreiro", 5, -4, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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

// Os QUATRO campos indexados: nome, classe primária, origem e raças.
//
// O nome e a classe são propositalmente DISJUNTOS, e cada campo é buscado pelo
// termo que só ELE contém: com um personagem chamado "Guerreiro" o caso casaria
// pelo nome e ficaria verde com a classe fora de `searchFields`.
func TestTheCharacterSearchLooksAtTheFourFields(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	id := seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Bárbaro", 5, 0, 0)
	seedRaca(t, s, id, "Anão")

	// "Soldado" é a origem que o `seedCharacterAtLevel` grava.
	for termo, campo := range map[string]string{
		"thalen":  "nome",
		"barbaro": "classe",
		"soldado": "origem",
		"anao":    "raça",
	} {
		v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, termo)
		if err != nil {
			t.Fatalf("carregar %q: %v", termo, err)
		}
		if len(v.Heroes) != 1 {
			t.Errorf("busca por %s (%q) devolveu %d heróis, queria 1", campo, termo, len(v.Heroes))
		}
	}

	nada, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "zzzzzz")
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
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Guerreiro", 5, -4, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", "Arcanista", 4, 4, 4)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "thalen")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 1 || v.Total != 2 {
		t.Errorf("filtrados=%d total=%d, queria 1 de 2", len(v.Heroes), v.Total)
	}
}

// ── os vizinhos que ladeiam o palco ──────────────────────────────────────────

// O que os vizinhos carregam de regra: o nome legível e o caminho de volta.
func TestTheNeighborsFlankTheStageWithAReadableName(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Guerreiro", 5, -4, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", "Arcanista", 4, 4, 4)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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
	// quem vem a seguir, e é para os olhos que ele existe. Procurá-lo no HTML
	// INTEIRO sobrevive à sabotagem — o nome também está no `title`, no `h2` do
	// palco e no rótulo do filme.
	if corpo := corpoDoBotao(t, html, "Próximo: "+segundo); !strings.Contains(corpo, segundo) {
		t.Errorf("o peek de %q não mostra o nome na tela, só em atributo: %q", segundo, corpo)
	}
}

// A vaga de criar tem o ÚLTIMO herói à esquerda: é o caminho de volta para o
// elenco, e sem ele quem anda até o fim do trilho fica sem pista de retorno.
func TestTheCreateSlotShowsTheLastHeroAsTheWayBack(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Guerreiro", 5, -4, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", "Arcanista", 4, 4, 4)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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
// primeiro herói — é garantia de LAYOUT, e layout só existe num navegador: está
// no `characters.spec.ts`. Aqui fica só o que é verdade de dado.
func TestALoneHeroGetsNoInventedNeighbor(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Guerreiro", 5, -4, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
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

// O TRILHO mostra quem está mal, e não o elenco inteiro com cara de saudável.
//
// A escada aqui é a de ESCREVER, e ela diverge da de preencher: na ficha o vital
// é uma FAIXA, aqui ele é um NÚMERO, e o crítico não pode usar `--hp-critical`,
// que dá 4,11:1 como letra pequena. Por isso o caso afirma a tinta de perigo da
// casa e não o vermelho da barra — um guarda que aceitasse `text-hp-critical`
// estaria prendendo o defeito.
//
// Os três heróis vão num elenco SÓ, e de propósito: com um por vez, um cartão
// que ignorasse o herói e lesse o primeiro do elenco passaria nos três.
func TestTheCastPaintsEachHeroByHowBadlyHurtHeIs(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	// O dano sai da FRAÇÃO do poço e não de um número escolhido: o que este caso
	// afirma são as três faixas de tinta, e um dano absoluto mudaria de faixa no
	// dia em que a tabela de classe mudasse — sem ninguém mexer no teste.
	poco := bookPools(t, s, "Guerreiro", 5).PvMax
	seedCharacterAtLevel(t, s, eu.ID, "Inteiro", "Guerreiro", 5, 0, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Machucado", "Guerreiro", 5, poco*60/100, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Morrendo", "Guerreiro", 5, poco*90/100, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 3 {
		t.Fatalf("o elenco veio com %d heróis, e o caso precisa dos três", len(v.Heroes))
	}

	// As tintas escritas à mão: derivá-las de `ui.HpInkTone` faria a asserção
	// andar junto com o defeito.
	querido := map[string]string{
		"Inteiro":   "text-hp-full",                 // 100%
		"Machucado": "text-hp-hurt",                 // 40%
		"Morrendo":  "text-grimorio-crimson-bright", // 10%
	}
	for _, h := range v.Heroes {
		if tinta, ok := querido[h.Name]; ok && h.PVInk != tinta {
			t.Errorf("%s (PV %s) escreve com %q, e o esperado é %q", h.Name, h.PV, h.PVInk, tinta)
		}
	}
}
