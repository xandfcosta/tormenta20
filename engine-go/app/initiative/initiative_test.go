package initiative

import (
	"context"
	"errors"
	"testing"

	"t20engine/app"
)

func int64p(v int64) *int64 { return &v }

// O MONSTRO DO BESTIÁRIO CARREGA O PV DELE, que é o motivo inteiro de rastreá-lo
// na fila.
//
// A montagem já descartou `hpCurrent`/`hpMax` em silêncio: o cliente mandava, e
// a linha chegava sem barra de vida (ALE-75). O pedido TIPADO tira essa família
// de defeito do mapa — um campo que o struct não tem não compila —, mas o que se
// prende aqui é o comportamento, não a forma.
func TestAMonsterFromTheBestiaryKeepsItsHp(t *testing.T) {
	row, err := Roster{}.npcEntry(EntryRequest{
		Label: "Goblin 1", Initiative: int64p(13), Kind: "npc",
		HpCurrent: int64p(4), HpMax: int64p(4),
	})
	if err != nil {
		t.Fatalf("montar a linha: %v", err)
	}
	if row.HpCurrent == nil || *row.HpCurrent != 4 || row.HpMax == nil || *row.HpMax != 4 {
		t.Errorf("o PV chegou como %v/%v, e o verbete diz 4/4", row.HpCurrent, row.HpMax)
	}
}

// AUSENTE continua ausente: um NPC pelado — "Voz na escuridão" — não tem vida a
// acompanhar, e zerar desenharia uma barra vazia, que diz que ele já está morto.
func TestABareNpcStaysWithoutHp(t *testing.T) {
	row, err := Roster{}.npcEntry(EntryRequest{Label: "Voz na escuridão", Initiative: int64p(7)})
	if err != nil {
		t.Fatalf("montar a linha: %v", err)
	}
	if row.HpCurrent != nil || row.HpMax != nil {
		t.Errorf("o PV veio %v/%v, e este NPC não tem vida registrada", row.HpCurrent, row.HpMax)
	}
}

func TestAnNpcNeedsALabelAndAnInitiative(t *testing.T) {
	if _, err := (Roster{}).npcEntry(EntryRequest{Initiative: int64p(1)}); err == nil {
		t.Error("um NPC sem rótulo passou")
	}
	if _, err := (Roster{}).npcEntry(EntryRequest{Label: "Goblin"}); err == nil {
		t.Error("uma linha sem iniciativa passou")
	}
	if _, err := (Roster{}).npcEntry(EntryRequest{Label: "   ", Initiative: int64p(1)}); err == nil {
		t.Error("um rótulo só de espaço passou")
	}
}

// O d20 é um d20, e a recusa é da REGRA — quem traduz o número é o transporte.
func TestTheOwnRollHasToBeAD20(t *testing.T) {
	for _, outside := range []int64{0, -1, 21, 100} {
		_, err := (Roster{}).SelfEntry(context.Background(), app.Caller{ID: 1}, 1, 1, outside)
		if err == nil {
			t.Errorf("o d20 %d passou", outside)
			continue
		}
		if !errors.Is(err, app.ErrRefused) {
			t.Errorf("o d20 %d foi recusado como %v, e queria ErrRefused", outside, err)
		}
	}
}

// CONDIÇÃO QUE O CATÁLOGO NÃO CONHECE é descartada em silêncio, e as repetidas
// entram uma vez só: derrubar a aplicação inteira no meio do combate por causa
// de um item seria pior que ignorar o item.
func TestOnlyConditionsTheBookHasGetThrough(t *testing.T) {
	// O `enfeiticado` está no caso de propósito: ele era `enfeitiçado`, com
	// cedilha, e a grafia irregular fazia toda cópia da lista errar NELE. Quem
	// segura a forma hoje é o `catalog.TestNoCatalogIDIsAccented`.
	outside := KnownConditions([]string{"caido", "inventada", "enfeiticado", "atordoado"})
	if len(outside) != 3 {
		t.Fatalf("passaram %v, e as três do livro deviam passar", outside)
	}
	for _, id := range outside {
		if id == "inventada" {
			t.Fatalf("um id fora do catálogo passou: %v", outside)
		}
	}

	repeated := KnownConditions([]string{"caido", "caido", ""})
	if len(repeated) != 1 || repeated[0] != "caido" {
		t.Errorf("as repetidas viraram %v, e a mesma condição entra uma vez só", repeated)
	}
	if empty := KnownConditions(nil); empty == nil || len(empty) != 0 {
		t.Errorf("lista vazia virou %v, e um nulo faria o JSON dizer null em vez de []", empty)
	}
}
