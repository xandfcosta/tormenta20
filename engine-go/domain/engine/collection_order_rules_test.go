package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

// A ORDEM DOS COLETORES É A REGRA, E ELA PRECISA DE REDE PRÓPRIA (ALE-378).
//
// O cabeçalho do `collectionSystems` já avisa que trocar duas linhas ali troca a
// ordem da lista. O que faltava era algo que ACUSASSE.
//
// # O oráculo não acusa, e isso foi medido
//
// Sabotando a ordem, 3 dos 4 pares adjacentes passaram verdes: nenhuma das 18
// fixtures tem TODAS as fontes ao mesmo tempo, então a maior parte dos pares
// nunca se encontra numa ficha. É "um guarda só mede o que ele VISITA" com o
// DADO no lugar da navegação — e o conserto é um caso que põe as nove fontes na
// mesma ficha.
//
// # Por que ele vira URGENTE agora
//
// Hoje existe uma segunda rede, por acidente: o `TestActiveItemsByEcsMatchesTheOracle`
// compara o coletor de ECS com o legado, e reordenar um quebra o outro. Essa
// rede morre junto com o coletor legado. Este caso é o substituto — ele entra
// ANTES, e não depois.
//
// # Ele prende ESPÉCIE, e não id
//
// Prender os ids faria o caso reprovar no dia em que o catálogo ganhasse um
// poder de bárbaro, dizendo "a ordem mudou" quando o que mudou foi o livro. A
// espécie é estável: item, efeito, raça, origem, classe, geral, tormenta,
// condição, mesa.
func TestEveryCollectorLandsInTheDeclaredOrder(t *testing.T) {
	esperada := []string{
		"item", "efeito", "raça", "origem", "classe", "geral", "tormenta", "condição", "mesa",
	}

	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	world := everyCollectorWorld(t, dir)

	vista := []string{}
	for _, item := range world.ActiveItemsFor(everyCollectorCharacter()) {
		kind := collectorKindOf(item)
		if kind == "" {
			t.Fatalf("fonte %q (%q) não foi reconhecida por nenhuma espécie — o caso "+
				"deixou de saber o que está medindo", item.SourceID, item.Source)
		}
		if len(vista) == 0 || vista[len(vista)-1] != kind {
			vista = append(vista, kind)
		}
	}

	if strings.Join(vista, " → ") != strings.Join(esperada, " → ") {
		t.Fatalf("a ordem dos coletores mudou.\n  declarada: %s\n  colhida:   %s\n"+
			"Se foi de propósito, o `collectionSystems` e esta lista mudam JUNTOS — a "+
			"ordem é a regra, e ela não tem outra rede desde que o coletor legado saiu.",
			strings.Join(esperada, " → "), strings.Join(vista, " → "))
	}
}

// O SILÊNCIO É O ÚLTIMO, e ele não aparece na lista acima porque não acrescenta
// fonte nenhuma — ele TIRA.
//
// Prova pelo efeito: calar um termo da concessão da MESA só funciona se o
// silêncio rodar depois dela. Rodando antes, ele não acharia o que calar e a
// concessão sobreviveria inteira.
func TestTheSilenceRunsAfterEverythingItCanSilence(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	book := primeFromDump(t, dir)
	ch := everyCollectorCharacter()

	daMesa := Modifier{Target: ModifierTarget{K: "attack", Scope: "all"}, Amount: 1, BonusType: "untyped"}
	mesa := Amendments{Grants: []CampaignGrant{
		{ID: "g1", Applies: EveryoneIn(), Label: "Bênção da mesa", Modifiers: []Modifier{daMesa}},
	}}

	temAMesa := func(world *Ruleset) bool {
		for _, item := range world.ActiveItemsFor(ch) {
			if item.SourceID == CampaignGrantSource+"g1" {
				return true
			}
		}
		return false
	}
	if !temAMesa(RulesetOf(book, mesa)) {
		t.Fatal("o controle já estava errado: a concessão da mesa não foi colhida")
	}

	calada := mesa
	calada.Silences = []Silence{{Applies: EveryoneIn(), Term: TermID(CampaignGrantSource+"g1", daMesa)}}
	if temAMesa(RulesetOf(book, calada)) {
		t.Error("a concessão da mesa sobreviveu ao silêncio — o sistema que cala rodou ANTES " +
			"de quem pendurou, e ele tem de ser o último")
	}
}

// everyCollectorCharacter é o personagem que aciona as NOVE fontes de uma vez.
//
// Nenhuma fixture do oráculo faz isso, e é por isso que ele não pega troca de
// ordem: a maior parte dos pares adjacentes nunca se encontra numa ficha.
func everyCollectorCharacter() Character {
	wielded := "wielded"
	axe := "machado-taurico"
	return Character{
		ID: 7, Origin: "Batedor", Level: 10,
		Races:   []CharacterRace{{Race: "Anão"}},
		Classes: []CharacterClass{{ClassName: "Bárbaro", Level: 10}},
		Items:   []CharacterItem{{Name: "Machado", CatalogID: &axe, Equipped: &wielded}},
		ActiveEffects: []ActiveEffectRow{{
			CatalogID: "bencao", Scope: "scene",
			Modifiers: `[{"target":{"k":"attack","scope":"all"},"amount":1,"bonusType":"untyped"}]`,
		}},
		ActiveConditions: `["abalado"]`,
		// `vitalidade` é poder GERAL e `pele-corrompida` é da TORMENTA: os dois
		// entram pela mesma coluna e saem por coletores diferentes.
		ClassPowers:          `["class.barbaro.golpe-poderoso","vitalidade","pele-corrompida"]`,
		OriginChoices:        `["poder-sentidos-agucados"]`,
		RaceAttributeChoices: "{}",
	}
}

func everyCollectorWorld(t *testing.T, dir string) *Ruleset {
	t.Helper()
	return RulesetOf(primeFromDump(t, dir), Amendments{Grants: []CampaignGrant{{
		ID: "g1", Applies: EveryoneIn(), Label: "Bênção da mesa",
		Modifiers: []Modifier{{Target: ModifierTarget{K: "attack", Scope: "all"}, Amount: 1, BonusType: "untyped"}},
	}}})
}

// collectorKindOf diz de qual coletor a fonte veio, ou "" para o que ele não
// reconhece — e não reconhecer FALHA o caso, em vez de sair da conta.
func collectorKindOf(item ActiveItem) string {
	switch {
	case strings.HasPrefix(item.SourceID, CampaignGrantSource):
		return "mesa"
	case item.Source == "Condições":
		return "condição"
	case item.Source == "Poderes da Tormenta":
		return "tormenta"
	case strings.HasPrefix(item.SourceID, "class."):
		return "classe"
	case strings.HasPrefix(item.Source, "Raça: "):
		return "raça"
	case strings.HasPrefix(item.Source, "Origem: "):
		return "origem"
	case strings.HasSuffix(item.Source, "(cena)") || strings.HasSuffix(item.Source, "(dia)"):
		return "efeito"
	case item.SourceID == "machado-taurico":
		return "item"
	case item.SourceID == "vitalidade":
		return "geral"
	}
	return ""
}
