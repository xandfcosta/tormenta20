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

// Os guardas da cena de PERSONAGENS: a Defesa saindo da mesma conta da ficha, a
// busca, a forja e a escada do PV no cartão.

func novaCenaDeHerois(t *testing.T) (*Server, AuthUser) {
	t.Helper()
	s := newTestServer(t)
	catalogs, err := engine.PrimeEngineCatalogs([]byte(`{"items":[]}`))
	if err != nil {
		t.Fatalf("preparar catálogo: %v", err)
	}
	s.primeCatalogs(catalogs)
	owner := seedUser(t, s, "jogadora@t20.local")
	u, err := s.queries.GetUserByID(context.Background(), owner)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return s, s.accountRules().authUser(u)
}

func seedRaca(t *testing.T, s *Server, characterID int64, race string) {
	t.Helper()
	err := s.queries.CreateRace(context.Background(), sqlcgen.CreateRaceParams{
		Characterid: characterID, Race: race,
	})
	if err != nil {
		t.Fatalf("seed raça %q: %v", race, err)
	}
}

// ── a Defesa ─────────────────────────────────────────────────────────────────

// A Defesa do cartão tem de ser a MESMA que a ficha mostra. Duas contas
// diferentes para o mesmo número é o defeito que ninguém reporta: a pessoa vê
// 18 na lista e 17 na ficha e conclui que o app é aproximado.
func TestTheCardDefenseIsTheSameAsTheSheetOne(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	id := seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", "Guerreiro", 5, -4, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 1 {
		t.Fatalf("esperava 1 herói, veio %d", len(v.Heroes))
	}

	row, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("personagem: %v", err)
	}
	sheet, err := s.sheetRules().ComputeSheet(context.Background(), row)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if v.Heroes[0].Defense != strconv.Itoa(sheet.Defense.Total) {
		t.Errorf("Defesa do cartão = %q, a da ficha = %d", v.Heroes[0].Defense, sheet.Defense.Total)
	}
}

// Defesa que o motor não soube dar vira TRAVESSÃO, e nunca zero: zero é um valor
// plausível de Defesa, então mostrá-lo seria mentir com um número redondo. E
// travessão em vez de omitir porque um bloco que some desalinha os cartões.
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
	card := characters.HeroCardOf(nil, sheet.CharacterDTO{ID: 1, Name: "Thessa", Level: 5})

	if card.Defense != "—" || card.DefenseVs != "—" {
		t.Errorf("Defesa = %q / %q, queria travessão nos dois",
			card.Defense, card.DefenseVs)
	}
	html, err := ui.RenderFragment(t.Context(),
		characters.SceneBody(characters.View{Heroes: []characters.HeroCard{card}, HasAny: true}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, ">DEF</dt>") {
		t.Error("o bloco da Defesa sumiu — o cartão perde o alinhamento com os outros")
	}
	if strings.Contains(html, ">0</dd>") {
		t.Error("a Defesa saiu como ZERO, que é um valor plausível e errado")
	}
}

// ── a forja ──────────────────────────────────────────────────────────────────

// Com o ELENCO VAZIO, o vazio é que leva à forja. Uma tela vazia que não
// oferece o primeiro passo é uma tela que não ajuda.
func TestWithAnEmptyCastTheEmptyStateLeadsToTheForge(t *testing.T) {
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

// Com heróis, a forja é o + do cabeçalho — e ele é um link de verdade, que o
// clique do meio e o "abrir em nova aba" alcançam.
func TestWithHeroesTheHeaderPlusLeadsToTheForge(t *testing.T) {
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
	if !strings.Contains(html, `<a href="/personagens/nova" class="ds-button" data-icon aria-label="Forjar um herói"`) {
		t.Error("o cabeçalho não traz o + que leva à forja")
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
	for term, field := range map[string]string{
		"thalen":  "nome",
		"barbaro": "classe",
		"soldado": "origem",
		"anao":    "raça",
	} {
		v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, term)
		if err != nil {
			t.Fatalf("carregar %q: %v", term, err)
		}
		if len(v.Heroes) != 1 {
			t.Errorf("busca por %s (%q) devolveu %d heróis, queria 1", field, term, len(v.Heroes))
		}
	}

	nothing, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "zzzzzz")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !nothing.FilteredAll {
		t.Error("busca sem resultado não foi marcada como tal")
	}
}

// A busca sem resultado diz O QUE se procurou, e o campo continua lá para
// corrigir o termo — sumir com ele obrigaria a limpar para tentar de novo.
func TestASearchWithoutHeroSaysWhatWasSoughtAndKeepsTheField(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", "Guerreiro", 5, -4, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "minotauro")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Nenhum herói casa com &#34;minotauro&#34;") {
		t.Error("a busca sem resultado não diz o termo que procurou")
	}
	if !strings.Contains(html, `aria-label="Buscar personagem"`) {
		t.Error("a busca sem resultado tirou o campo da tela")
	}
}

// O ELENCO mostra quem está mal, e não todos com cara de saudável: cada cartão
// leva o degrau da escada do PV, e a folha pinta o número por ele.
//
// Os três heróis vão num elenco SÓ, e de propósito: com um por vez, um cartão
// que ignorasse o herói e lesse o primeiro do elenco passaria nos três.
func TestTheCastPaintsEachHeroByHowBadlyHurtTheyAre(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	// O dano sai da FRAÇÃO do poço e não de um número escolhido: o que este caso
	// afirma são as três faixas de tinta, e um dano absoluto mudaria de faixa no
	// dia em que a tabela de classe mudasse — sem ninguém mexer no teste.
	pool := bookPools(t, s, "Guerreiro", 5).PvMax
	seedCharacterAtLevel(t, s, eu.ID, "Inteiro", "Guerreiro", 5, 0, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Machucado", "Guerreiro", 5, pool*60/100, 5)
	seedCharacterAtLevel(t, s, eu.ID, "Morrendo", "Guerreiro", 5, pool*90/100, 5)

	v, err := characters.New(s.sceneCore()).Load(context.Background(), eu.ID, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Heroes) != 3 {
		t.Fatalf("o elenco veio com %d heróis, e o caso precisa dos três", len(v.Heroes))
	}

	html, err := ui.RenderFragment(t.Context(), characters.SceneBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// Os degraus escritos à mão: derivá-los de `ui.HpTone` faria a asserção
	// andar junto com o defeito. O caso olha o HTML de CADA cartão, recortado
	// pelo nome, para um cartão que lesse o herói errado não passar.
	wanted := map[string]string{
		"Inteiro":   "full",     // 100%
		"Machucado": "hurt",     // 40%
		"Morrendo":  "critical", // 10%
	}
	for name, tone := range wanted {
		start := strings.Index(html, "Abrir ficha de "+name)
		if start < 0 {
			t.Fatalf("o cartão de %s não saiu", name)
		}
		card := html[start:]
		if end := strings.Index(card, "</article>"); end >= 0 {
			card = card[:end]
		}
		if !strings.Contains(card, `data-tone="`+tone+`"`) {
			t.Errorf("o PV de %s não leva o degrau %q", name, tone)
		}
	}
}
