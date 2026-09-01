package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// Os guardas do painel de MAGIAS (ALE-272, fatia 6).
//
// O que eles prendem é a REGRA — quem prepara, quanto custa, quem alcança qual
// aprimoramento — e a decisão de quem VÊ cada bloco. O catálogo em si é dado
// transcrito e tem validação de schema.

// oArcanista é uma ficha que conjura de verdade: nível 9 abre o 3º círculo.
func oArcanista(t *testing.T) (pilotoFixture, int64) {
	t.Helper()
	f := novoPiloto(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.jogador, Name: "Conjuradora", Origin: "Charlatão", Level: 9,
		HpMax: 40, HpCurrent: 40, MpMax: 40, MpCurrent: 40,
		Strength: 0, Dexterity: 2, Constitution: 2, Intelligence: 4, Wisdom: 1, Charisma: 1,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: `{"Arcanista":{"caminho":"bruxo"}}`,
		PowerChoices: "{}", CreatedAt: plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a conjuradora: %v", err)
	}
	seedClasse(t, f.s, id, "Arcanista", 9)
	return f, id
}

func aTelaDasMagias(t *testing.T, f pilotoFixture, id int64) string {
	t.Helper()
	return f.pede(t, f.jogador, http.MethodGet,
		fmt.Sprintf("/personagens/%d?tab=spells", id), "").Body.String()
}

func aMagia(t *testing.T, f pilotoFixture, id int64, caminho string) int {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/magias/%s?tab=spells", id, caminho)
	return f.pede(t, f.jogador, http.MethodPost, alvo, "").Code
}

// aRecusaDaMagia é a frase da regra que barrou o comando, ou "".
func aRecusaDaMagia(t *testing.T, f pilotoFixture, id int64, caminho string) string {
	t.Helper()
	alvo := fmt.Sprintf("/personagens/%d/magias/%s?tab=spells", id, caminho)
	return aRecusaDaCena(f.pede(t, f.jogador, http.MethodPost, alvo, "").Body.String())
}

func oGrimorioDe(t *testing.T, f pilotoFixture, id int64) map[string]bool {
	t.Helper()
	linhas, err := f.s.queries.ListSpellsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o grimório: %v", err)
	}
	fora := map[string]bool{}
	for _, l := range linhas {
		fora[l.Catalogspellid] = l.Prepared != 0
	}
	return fora
}

// APRENDER, PREPARAR E ESQUECER, nessa ordem.
func TestAMagiaEntraNoGrimorioPreparaESai(t *testing.T) {
	f, id := oArcanista(t)

	if got := aMagia(t, f, id, "aprende/bola-de-fogo"); got != http.StatusOK {
		t.Fatalf("aprender respondeu %d", got)
	}
	if grimorio := oGrimorioDe(t, f, id); !existe(grimorio, "bola-de-fogo") {
		t.Fatal("a magia não entrou no grimório")
	}

	aMagia(t, f, id, "prepara/bola-de-fogo")
	if grimorio := oGrimorioDe(t, f, id); !grimorio["bola-de-fogo"] {
		t.Error("o toque não preparou a magia")
	}
	aMagia(t, f, id, "prepara/bola-de-fogo")
	if grimorio := oGrimorioDe(t, f, id); grimorio["bola-de-fogo"] {
		t.Error("o segundo toque não despreparou: o comando manda o ESTADO em vez da magia")
	}

	aMagia(t, f, id, "esquece/bola-de-fogo")
	if grimorio := oGrimorioDe(t, f, id); existe(grimorio, "bola-de-fogo") {
		t.Error("a magia sobreviveu ao esquecer")
	}
}

func existe(m map[string]bool, chave string) bool {
	_, tem := m[chave]
	return tem
}

// UMA MAGIA INVENTADA NÃO ENTRA no grimório.
func TestUmaMagiaInventadaNaoEntraNoGrimorio(t *testing.T) {
	f, id := oArcanista(t)
	if aRecusaDaMagia(t, f, id, "aprende/bola-de-neve-magica") == "" {
		t.Error("uma magia que não existe foi aprendida sem uma palavra na tela")
	}
	if len(oGrimorioDe(t, f, id)) != 0 {
		t.Error("a recusa gravou assim mesmo")
	}
}

// CONJURAR COBRA O PM, e a conta é a do servidor.
func TestConjurarCobraOPm(t *testing.T) {
	f, id := oArcanista(t)
	aMagia(t, f, id, "aprende/bola-de-fogo")

	antes := oPmDe(t, f, id)
	if got := aMagia(t, f, id, "conjura/bola-de-fogo"); got != http.StatusOK {
		t.Fatalf("conjurar respondeu %d", got)
	}
	depois := oPmDe(t, f, id)
	// Bola de Fogo é de 2º círculo, e o 2º custa 3 PM (Tabela 4-1, livro p170).
	if antes-depois != 3 {
		t.Errorf("a conjuração cobrou %d PM, quer 3 (base do 2º círculo)", antes-depois)
	}
}

func oPmDe(t *testing.T, f pilotoFixture, id int64) int64 {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	return row.Mpcurrent
}

// SEM PM, A CONJURAÇÃO É RECUSADA e nada é cobrado.
func TestSemPmAConjuracaoERecusada(t *testing.T) {
	f, id := oArcanista(t)
	aMagia(t, f, id, "aprende/bola-de-fogo")
	if err := f.s.queries.SetMpCurrent(context.Background(), sqlcgen.SetMpCurrentParams{
		MpCurrent: 1, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		t.Fatalf("zerar o PM: %v", err)
	}

	if recusa := aRecusaDaMagia(t, f, id, "conjura/bola-de-fogo"); !strings.Contains(recusa, "faltam PM") {
		t.Errorf("a recusa por PM não chegou à tela: %q", recusa)
	}
	if pm := oPmDe(t, f, id); pm != 1 {
		t.Errorf("a recusa mexeu no PM assim mesmo: sobrou %d", pm)
	}
}

// O PAINEL CHEGA NA TELA, com o grimório e o catálogo.
func TestOPainelDeMagiasDesenhaOGrimorioEOCatalogo(t *testing.T) {
	f, id := oArcanista(t)
	aMagia(t, f, id, "aprende/bola-de-fogo")
	tela := aTelaDasMagias(t, f, id)

	for _, esperado := range []string{"Grimório", "1 aprendida", "Conjurar Bola de Fogo", "Aprender magia"} {
		if !strings.Contains(tela, esperado) {
			t.Errorf("a tela não tem %q", esperado)
		}
	}
	// O CATÁLOGO sai inteiro, menos o que já se sabe.
	if strings.Count(tela, "/magias/aprende/") < 150 {
		t.Errorf("o catálogo desenhou %d opções de aprender, e são ~197",
			strings.Count(tela, "/magias/aprende/"))
	}
	if strings.Contains(tela, "/magias/aprende/bola-de-fogo") {
		t.Error("uma magia JÁ aprendida continua ofertada no catálogo")
	}
}

// QUEM NÃO CONJURA não recebe o botão nem o catálogo do Capítulo 4.
func TestQuemNaoConjuraNaoRecebeOCatalogo(t *testing.T) {
	f, id := oCombatente(t)
	tela := aTelaDasMagias(t, f, id)

	if strings.Contains(tela, `aria-label="Aprender magia"`) {
		t.Error("um guerreiro recebeu o botão de aprender magia")
	}
	if strings.Contains(tela, "/magias/aprende/") {
		t.Error("o catálogo inteiro viajou para quem não pode aprender nada")
	}
	if !strings.Contains(tela, "não conjura por classe") {
		t.Error("a aba não diz por que está vazia")
	}
}

// A MAGIA QUE UM PODER ENSINOU aparece mesmo para quem não conjura por classe.
//
// O Totem Espiritual do bárbaro (livro p42) escolhe um animal e cada animal
// ensina uma magia. Ela não mora no grimório — não se aprende nem se esquece —,
// e sem este bloco o jogador do bárbaro não teria onde ler o efeito dela.
func TestAMagiaConcedidaPorPoderApareceParaQuemNaoConjura(t *testing.T) {
	f := novoPiloto(t)
	id, err := f.s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.jogador, Name: "Totemista", Origin: "Batedor", Level: 3,
		HpMax: 30, HpCurrent: 30, MpMax: 0, MpCurrent: 0,
		Strength: 4, Dexterity: 1, Constitution: 3, Intelligence: 0, Wisdom: 1, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}",
		PowerChoices: `{"class.barbaro.totem-espiritual":["corvo"]}`,
		CreatedAt:    plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o totemista: %v", err)
	}
	seedClasse(t, f.s, id, "Bárbaro", 3)

	tela := aTelaDasMagias(t, f, id)
	if !strings.Contains(tela, "Visão Mística") {
		t.Error("a magia do corvo não chegou à tela")
	}
	if !strings.Contains(tela, "Totem Espiritual") {
		t.Error("a tela não diz de onde a magia veio")
	}
	// CONTROLE: ela não virou uma linha do grimório, que teria os comandos de
	// preparar e esquecer — nenhum dos dois se aplica a uma concedida.
	if strings.Contains(tela, "/magias/esquece/") {
		t.Error("a concedida virou linha do grimório: dá para esquecê-la")
	}
}

// O APRIMORAMENTO FORA DE ALCANCE aparece TRANCADO, e não some.
//
// Some-lo faria a lista parecer menor do que o livro diz que é; e a fronteira
// não é o cadeado — o servidor recusa o mesmo pedido, com guarda próprio em
// `piloto_spell_progression_test.go`.
func TestOAprimoramentoForaDeAlcanceApareceTrancado(t *testing.T) {
	// Nível 5 abre o 2º círculo; a Invisibilidade tem aprimoramento de 3º.
	f := novoPiloto(t)
	id := seedCharacterAtLevel(t, f.s, f.jogador, "Aprendiz", 5, 20, 20, 20, 20)
	seedClasse(t, f.s, id, "Arcanista", 5)
	aMagia(t, f, id, "aprende/invisibilidade")

	tela := aTelaDasMagias(t, f, id)
	if !strings.Contains(tela, "exige o 3º círculo") {
		t.Error("o aprimoramento fora de alcance não diz que está trancado")
	}
	// A Invisibilidade tem TRÊS aprimoramentos e dois exigem círculo (3º e 4º).
	// Contar é o CONTROLE: uma tela que trancasse tudo também conteria a frase
	// acima, e uma que não trancasse nada nunca chegaria aqui.
	if trancados := strings.Count(tela, "exige o"); trancados != 2 {
		t.Errorf("a tela trancou %d aprimoramentos, quer 2 (3º e 4º círculo)", trancados)
	}
}
