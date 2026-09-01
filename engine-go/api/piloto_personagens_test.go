package api

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
)

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

func seedClasse(t *testing.T, s *Server, characterID int64, nome string, nivel int64) {
	t.Helper()
	err := s.queries.CreateClass(context.Background(), sqlcgen.CreateClassParams{
		Characterid: characterID, Classname: nome, Level: nivel,
	})
	if err != nil {
		t.Fatalf("seed classe %q: %v", nome, err)
	}
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
func TestDefesaDoPalcoEhAMesmaDaFicha(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	id := seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Herois) != 1 {
		t.Fatalf("esperava 1 herói, veio %d", len(v.Herois))
	}

	linha, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("personagem: %v", err)
	}
	ficha, err := s.computeSheet(context.Background(), linha)
	if err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if v.Herois[0].Defesa != strconv.Itoa(ficha.Defense.Total) {
		t.Errorf("Defesa do palco = %q, a da ficha = %d", v.Herois[0].Defesa, ficha.Defense.Total)
	}
}

// Sem motor a Defesa vira TRAVESSÃO, e nunca zero: zero é um valor plausível de
// Defesa, então mostrá-lo seria mentir com um número redondo. E travessão em vez
// de omitir porque uma coluna que some faz o palco dançar ao trocar de herói
// (ALE-99) — é o que a SPA faz, e eu tinha escrito "some" antes de comparar as
// duas telas.
func TestSemMotorADefesaViraTravessao(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)
	s.catalogs = nil

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar sem motor deveria funcionar: %v", err)
	}
	if len(v.Herois) != 1 {
		t.Fatalf("a cena caiu sem o motor")
	}
	if v.Herois[0].Defesa != "—" {
		t.Errorf("Defesa = %q sem motor, queria travessão", v.Herois[0].Defesa)
	}
	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
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

// Este é o dividendo da ALE-107 aparecendo: os textos vêm do catálogo EMBUTIDO.
// O guarda é sobre o texto existir, não sobre qual é — o conteúdo é dado
// transcrito e quem o valida é o schema, não um `expect` por verbete.
func TestDossieTrazOsTextosDeRacaDoCatalogoEmbutido(t *testing.T) {
	if len(habilidadesDaRaca("Humano", 8)) == 0 {
		t.Fatal("o catálogo embutido não devolveu habilidade nenhuma para Humano")
	}
	for _, h := range habilidadesDaRaca("Humano", 8) {
		if h.Nome == "" || h.Descricao == "" {
			t.Errorf("habilidade %q sem nome ou sem descrição — o dossiê ficaria com linha vazia", h.ID)
		}
	}
}

// Raça que não está no catálogo não derruba nada: o herói abre sem as linhas de
// sabor. Um personagem antigo com raça renomeada é caso normal, não erro.
func TestRacaDesconhecidaNaoDerrubaODossie(t *testing.T) {
	if got := habilidadesDaRaca("Não Existe", 8); got != nil {
		t.Errorf("raça desconhecida devolveu %v", got)
	}
}

func TestODossieRespeitaOLimite(t *testing.T) {
	if got := len(habilidadesDaRaca("Humano", 1)); got != 1 {
		t.Errorf("limite 1 devolveu %d", got)
	}
}

// ── a gramática do cursor ────────────────────────────────────────────────────

// A vaga de criar existe COM O ELENCO VAZIO, e é ela que dá o que fazer. Uma
// tela vazia que não oferece o primeiro passo é uma tela que não ajuda.
func TestComElencoVazioAVagaDeCriarEhOQueSobra(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if v.TemAlgum {
		t.Fatal("elenco vazio marcado como cheio")
	}
	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Forjar um herói") || !strings.Contains(html, "/piloto/personagens/nova") {
		t.Error("a tela vazia não oferece a Forja")
	}
}

// A vaga é POSIÇÃO DE CURSOR e não um link solto (ALE-98): ela declara
// `role=option` e escreve o cursor no foco, como qualquer herói. Um `<a>` no
// fim da fita pareceria igual e as setas o pulariam.
func TestAVagaDeCriarEhPosicaoDeCursorENaoUmLinkSolto(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Guerreiro", 5, 16, 12, 3, 8)

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
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
func TestBuscaDePersonagemOlhaOsQuatroCampos(t *testing.T) {
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
		v, err := s.carregaPersonagens(context.Background(), eu, termo)
		if err != nil {
			t.Fatalf("carregar %q: %v", termo, err)
		}
		if len(v.Herois) != 1 {
			t.Errorf("busca por %s (%q) devolveu %d heróis, queria 1", campo, termo, len(v.Herois))
		}
	}

	nada, err := s.carregaPersonagens(context.Background(), eu, "zzzzzz")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !nada.FiltrouTudo {
		t.Error("busca sem resultado não foi marcada como tal")
	}
}

// A contagem diz FILTRADOS de TOTAL. Dizer "3 de 3" com sete escondidos pela
// busca esconde justamente o que a pessoa precisa saber para limpar o filtro.
func TestAContagemDizFiltradosDeTotal(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := s.carregaPersonagens(context.Background(), eu, "thalen")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Herois) != 1 || v.Total != 2 {
		t.Errorf("filtrados=%d total=%d, queria 1 de 2", len(v.Herois), v.Total)
	}
}

// ── os vizinhos que ladeiam o palco ──────────────────────────────────────────

// O peek foi PORTADO na virada, e não reescrito: apagar a tela antiga levaria
// junto os retratos apagados dos vizinhos se ninguém os trouxesse. Aqui se
// afirma o que eles carregam de regra — o nome legível e o caminho de volta.
func TestOsVizinhosLadeiamOPalcoComNomeLegivel(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Herois) != 2 {
		t.Fatalf("esperava 2 heróis, veio %d", len(v.Herois))
	}
	primeiro, segundo := v.Herois[0].Nome, v.Herois[1].Nome

	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
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

// corpoDoBotao devolve o conteúdo visível de um botão achado pelo `aria-label`.
//
// O `>` que fecha a tag de abertura é o primeiro do trecho porque o templ
// escapa `>` dentro de valor de atributo — então o corte é seguro.
func corpoDoBotao(t *testing.T, html, rotulo string) string {
	t.Helper()
	i := strings.Index(html, `aria-label="`+rotulo+`"`)
	if i < 0 {
		t.Fatalf("nenhum botão com rótulo %q no HTML", rotulo)
	}
	resto := html[i:]
	abre := strings.Index(resto, ">")
	fecha := strings.Index(resto, "</button>")
	if abre < 0 || fecha < 0 || abre > fecha {
		t.Fatalf("botão %q malformado no HTML", rotulo)
	}
	return resto[abre+1 : fecha]
}

// A vaga de criar tem o ÚLTIMO herói à esquerda: é o caminho de volta para o
// elenco, e sem ele quem anda até o fim do trilho fica sem pista de retorno.
func TestAVagaDeCriarMostraOUltimoHeroiComoCaminhoDeVolta(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)
	seedCharacterAtLevel(t, s, eu.ID, "Yrla", 4, 10, 14, 2, 6)

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	ultimo := v.Herois[len(v.Herois)-1].Nome
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
func TestHeroiUnicoNaoGanhaVizinhoInventado(t *testing.T) {
	s, eu := novaCenaDeHerois(t)
	seedCharacterAtLevel(t, s, eu.ID, "Thalen", 5, 16, 12, 3, 8)

	v, err := s.carregaPersonagens(context.Background(), eu, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if peekDe(v.Herois, -1) != nil || peekDe(v.Herois, 1) != nil {
		t.Fatal("peekDe inventou vizinho fora do trilho")
	}
	html, err := renderFragmento(t.Context(), cenaDePersonagens(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(html, `aria-label="Próximo: `) {
		t.Error("herói único ganhou um próximo que não existe")
	}
}
