package engine

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// O ATAQUE ANUNCIA A CONDIÇÃO QUE O LIVRO PREVÊ (p335-336, ALE-441).
//
// É o mesmo desenho da MANOBRA (p234, ALE-423): o resultado diz o que o livro
// reserva e NÃO aplica nada. Quem põe a condição na criatura é o mestre, no
// gesto da fila que já existe.
//
// # E há uma diferença com a manobra, que é deliberada
//
// Lá a condição viaja SEMPRE, porque quem decide o vencedor do teste oposto é o
// mestre — o motor não tem como saber se ela vale. Aqui ele sabe: o acerto é a
// comparação do d20 com a Defesa, e ele a faz. Então o anúncio sai quando o
// GATILHO aconteceu, e não antes: anunciar atordoamento num ataque que errou
// mandaria o mestre arbitrar sobre coisa nenhuma.
func TestTheAttackAnnouncesTheConditionTheBookForesees(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)
	weapon.Announces = []AnnouncedCondition{
		// "Uma criatura viva atingida fica sangrando" (p336).
		{Condition: "sangrando", Note: "a perda de PV é cumulativa"},
		// "Quando você faz um acerto crítico, a vítima fica atordoada por uma
		// rodada (apenas uma vez por cena; Fort CD For ou Des evita)" (p336).
		{Condition: "atordoado", OnCritical: true, Note: "por uma rodada; Fort evita"},
	}
	target := AttackTarget{Defense: 15}

	acerto, err := ResolveAttack(weapon, target, 18, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if nomes := announcedNames(acerto); len(nomes) != 1 || nomes[0] != "sangrando" {
		t.Errorf("o acerto normal anuncia %v, e só a Sanguinária age fora do crítico", nomes)
	}

	critico, err := ResolveAttack(weapon, target, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if nomes := announcedNames(critico); len(nomes) != 2 {
		t.Errorf("o crítico anuncia %v, e as duas condições valem nele", nomes)
	}

	erro, err := ResolveAttack(weapon, target, 2, fixedDice(t))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if nomes := announcedNames(erro); len(nomes) != 0 {
		t.Errorf("o ataque que ERROU anuncia %v: não há o que o mestre arbitrar", nomes)
	}
}

// E O ANÚNCIO NÃO APLICA NADA, que é a linha inteira desta família.
//
// A prova é por AUSÊNCIA, e ela é feita por reflexão pela mesma razão que o
// `TestTheManeuverNeverDeclaresAWinner`: uma asserção sobre um campo deixa de
// compilar quando o campo sai, e caso que não compila é caso que alguém apaga.
func TestTheAnnouncedConditionCarriesNoVerdict(t *testing.T) {
	proibidos := map[string]string{
		"Applied":  "aplicar a condição é do mestre",
		"Imposed":  "impor é o verbo que a ALE-423 tirou do motor",
		"Resisted": "o teste de resistência é rolado na mesa",
		"Rounds":   "contar a duração é da mesa",
	}
	campos := map[string]bool{"Condition": true, "OnCritical": true, "Note": true}
	tipo := anuncioType()
	for i := 0; i < tipo.NumField(); i++ {
		nome := tipo.Field(i).Name
		if motivo, proibido := proibidos[nome]; proibido {
			t.Errorf("o anúncio ganhou o campo %s, e %s", nome, motivo)
		}
		if !campos[nome] {
			t.Errorf("o anúncio ganhou o campo %s e este guarda não sabe o que ele é: "+
				"campo novo num tipo que existe para INFORMAR precisa dizer que não "+
				"decide nada", nome)
		}
	}
}

func announcedNames(out AttackOutcome) []string {
	nomes := []string{}
	for _, a := range out.Announces {
		nomes = append(nomes, a.Condition)
	}
	return nomes
}

func anuncioType() reflect.Type { return reflect.TypeOf(AnnouncedCondition{}) }

// O ANÚNCIO ATRAVESSA O CATÁLOGO, e não só o tipo em Go (ALE-441).
//
// É a metade que o `TestTheEnchantReachesTheWeaponCard` ensinou a não confiar:
// montar o `Modifier` em Go prova a composição POR CIMA da fronteira JSON, e
// foi assim que seis encantos entraram inertes com a suíte verde (ALE-412).
func TestTheAnnouncedConditionCrossesTheCatalog(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	cards := BookRuleset(catalogs).ComputeWeaponCards(Character{
		Items: []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: "[]",
			Enchants: []string{"encanto-sanguinaria", "encanto-trovejante"},
		}},
	}, map[string]bool{})
	if len(cards) == 0 {
		t.Fatal("a espada longa empunhada não virou carta")
	}

	porCondicao := map[string]AnnouncedCondition{}
	for _, a := range cards[0].Announces {
		porCondicao[a.Condition] = a
	}
	if len(porCondicao) != 2 {
		t.Fatalf("a carta anuncia %+v, e os dois encantos reservam uma condição cada",
			cards[0].Announces)
	}
	// A SANGUINÁRIA age em todo acerto (p336), a TROVEJANTE só no crítico.
	if porCondicao["sangrando"].OnCritical {
		t.Error("a Sanguinária ficou marcada como só-de-crítico, e a p336 diz " +
			"\"uma criatura viva atingida\"")
	}
	if !porCondicao["atordoado"].OnCritical {
		t.Error("a Trovejante não ficou marcada como só-de-crítico, e a p336 diz " +
			"\"quando você faz um acerto crítico\"")
	}
	// E A NOTA DO LIVRO VIAJA: é ela que dá ao mestre a duração e o teste que
	// evita, sem os quais o anúncio vira um nome solto.
	if porCondicao["atordoado"].Note == "" {
		t.Error("o atordoamento da Trovejante chegou sem a nota do livro (duração, " +
			"teste que evita, limite por cena)")
	}
}

// A DRENANTE ANUNCIA OS DOIS EFEITOS DO CRÍTICO, como uma frase só (p335, ALE-461).
//
//	"Quando você faz um acerto crítico em uma criatura viva, a criatura fica
//	fraca e você ganha 2d10 pontos de vida temporários."
//
// A condição já era anunciada; os 2d10 eram a metade calada da ALE-459. Eles
// viajam na MESMA nota porque no livro são a mesma frase, com o mesmo gatilho —
// e quem julga o "criatura viva" dos dois é o mestre, que o livro não ajuda: a
// expressão aparece quinze vezes e não é definida em nenhuma (decisão do dono).
//
// O caso atravessa o despejo de propósito, pela razão do de cima: é o catálogo
// de verdade que a faixa lê.
func TestTheDrainingCriticalAnnouncesTheTempHpAlongWithTheCondition(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	ptr := func(s string) *string { return &s }

	cards := BookRuleset(catalogs).ComputeWeaponCards(Character{
		Items: []CharacterItem{{
			CatalogID: ptr("espada-longa"), Name: "Espada longa",
			Equipped: ptr("wielded"), Improvements: "[]",
			Enchants: []string{"encanto-drenante"},
		}},
	}, map[string]bool{})
	if len(cards) == 0 || len(cards[0].Announces) != 1 {
		t.Fatalf("a Drenante tinha de anunciar UMA coisa, e a carta traz %+v", cards)
	}
	fraco := cards[0].Announces[0]
	if fraco.Condition != "fraco" || !fraco.OnCritical {
		t.Fatalf("a Drenante anuncia %+v, e a p335 dá fraco no crítico", fraco)
	}
	// AS DUAS METADES DA FRASE, cada uma escrita à mão a partir da página.
	for _, quer := range []string{"criatura viva", "2d10 PV temporários", "quem atacou"} {
		if !strings.Contains(fraco.Note, quer) {
			t.Errorf("a nota do crítico da Drenante não diz %q — o mestre lê a faixa e não "+
				"fica sabendo do efeito. Veio: %q", quer, fraco.Note)
		}
	}
}
