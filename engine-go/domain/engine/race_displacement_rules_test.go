package engine

import (
	"path/filepath"
	"testing"
)

// O DESLOCAMENTO VEM DA RAÇA, E A COLUNA NÃO TEM VOTO (ALE-383).
//
// O livro põe o número no verbete da raça:
//
//   - p20, Anão — "Devagar e Sempre. Seu deslocamento é 6m (em vez de 9m)."
//   - p22, Elfo — "Graça de Glórienn. Seu deslocamento é 12m (em vez de 9m)."
//
// Cada caso monta a coluna `displacement` MENTINDO 9, que é o valor que a seed
// gravou para as duas raças. Se o motor voltar a lê-la, os dois reprovam — e é
// por isso que o caso crava o número errado de propósito em vez de deixá-lo
// zerado: zero passaria por "ninguém preencheu".
func TestDisplacementComesFromTheRaceAndNotTheColumn(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)

	for _, caso := range []struct {
		race   string
		want   int
		pagina string
	}{
		{"Anão", 6, "p20, Devagar e Sempre"},
		{"Elfo", 12, "p22, Graça de Glórienn"},
		{"Humano", 9, "o padrão do livro"},
	} {
		t.Run(caso.race, func(t *testing.T) {
			ch := Character{
				Displacement: 9, // a coluna MENTE para o anão e o elfo
				Races:        []CharacterRace{{Race: caso.race}},
			}
			got := catalogs.ComputeSheet(ch, nil).Displacement
			if got.Base != caso.want {
				t.Fatalf("%s: base %dm, esperava %dm (%s).\n"+
					"Se veio 9 para o anão ou o elfo, o motor voltou a ler a coluna "+
					"`characters.displacement` em vez do catálogo da raça.",
					caso.race, got.Base, caso.want, caso.pagina)
			}
			if got.Total != caso.want {
				t.Fatalf("%s: total %dm, esperava %dm sem modificador nenhum", caso.race, got.Total, caso.want)
			}
		})
	}
}

// Raça que o catálogo não conhece cai no padrão do livro, e não na coluna.
//
// Cair na coluna seria manter o espelho vivo por uma porta dos fundos: bastaria
// uma raça nova ainda não transcrita para o número errado voltar a valer.
func TestUnknownRaceFallsBackToTheBookDefault(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)

	ch := Character{Displacement: 42, Races: []CharacterRace{{Race: "Raça Inventada"}}}
	if got := catalogs.ComputeSheet(ch, nil).Displacement.Base; got != 9 {
		t.Fatalf("raça desconhecida deu base %dm, esperava 9m — se veio 42, a coluna voltou a ter voto", got)
	}
}

// O ANÃO NÃO É REDUZIDO POR ARMADURA NEM POR CARGA (p20).
//
// O livro, no "Devagar e Sempre": *"Seu deslocamento é 6m (em vez de 9m).
// Porém, seu deslocamento não é reduzido por uso de armadura ou excesso de
// carga."*
//
// Este caso existe porque os DOIS defeitos se cancelavam. Com a coluna mentindo
// 9m, o anão de armadura completa mostrava 9 − 3 = 6m — o número certo, pelo
// caminho errado. Derivar sem isentar troca isso por 6 − 3 = 3m, que é PIOR do
// que estava. As duas metades andam juntas ou nenhuma anda.
//
// O CONTROLE é o humano ao lado: ele veste a mesma armadura e TEM de perder os
// 3m. Sem ele, uma isenção aplicada a todo mundo passaria por "o anão está
// certo".
func TestDwarfIgnoresArmorSlowdownAndTheHumanDoesNot(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)

	vested := "vested"
	plateID := "armadura-completa"
	plate := []CharacterItem{{
		Name:      "Armadura completa",
		CatalogID: &plateID,
		Equipped:  &vested,
	}}

	for _, caso := range []struct {
		race string
		want int
		why  string
	}{
		{"Anão", 6, "isento da redução por armadura (p20): 6m e fica 6m"},
		{"Humano", 6, "sem isenção: 9m menos os 3m da armadura completa"},
	} {
		t.Run(caso.race, func(t *testing.T) {
			ch := Character{
				Displacement: 9,
				Races:        []CharacterRace{{Race: caso.race}},
				Items:        plate,
			}
			got := catalogs.ComputeSheet(ch, nil).Displacement
			if got.Total != caso.want {
				t.Fatalf("%s de armadura completa: %dm, esperava %dm — %s.\n"+
					"base=%d bônus=%d", caso.race, got.Total, caso.want, caso.why, got.Base, got.ItemBonus)
			}
		})
	}

	// E a prova de que os dois chegam a 6m por caminhos DIFERENTES: se a
	// isenção não existisse, o anão daria 3m. Bases distintas, total igual.
	anao := catalogs.ComputeSheet(Character{Races: []CharacterRace{{Race: "Anão"}}, Items: plate}, nil).Displacement
	humano := catalogs.ComputeSheet(Character{Races: []CharacterRace{{Race: "Humano"}}, Items: plate}, nil).Displacement
	if anao.Base == humano.Base {
		t.Fatalf("as bases ficaram iguais (%d): o anão devia partir de 6 e o humano de 9", anao.Base)
	}
	if anao.ItemBonus == humano.ItemBonus {
		t.Fatalf("os bônus ficaram iguais (%d): o humano devia levar −3 da armadura e o anão não", anao.ItemBonus)
	}
}
