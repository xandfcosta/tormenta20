package api

import (
	"context"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// O MOTOR E A FICHA RESPONDEM A MESMA COISA A "ESTA FICHA TEM O PODER?".
//
// São duas perguntas que precisam ser uma só. O motor responde para as regras da
// mesa ("tem o poder X", `Ruleset.OwnsPower`); a ficha responde para a aba Poderes,
// o Usar, a postura e o bônus cumulativo (`sheet.OwnsPower`, a lista da aba). Quando
// elas discordam, a mesma ficha ganha o bônus da regra da mesa e não consegue usar
// o poder, ou o contrário, sem erro em lugar nenhum.
//
// A comparação é por PROCEDÊNCIA, porque o motor só responde pelas que uma regra da
// mesa pode exigir: classe (pelo nível NA classe), geral, o que a origem deixou
// escolher, e a devoção. Raça, benefício de origem e encanto de item são da ficha só.
//
// # Por que sobre a SEMENTE
//
// Pela razão do `TestTheBatchPoolsMatchTheOneByOnePools`: quem separa as respostas
// é o DADO — o multiclasse, o poder escolhido acima do nível, a origem que escolhe
// um poder —, e o elenco da semente já tem esses casos.
func TestTheEngineAndTheSheetAgreeOnEveryProvenanceOfAPower(t *testing.T) {
	s := newTestServer(t)
	applySeedFile(t, s)
	ctx := context.Background()

	ids := []string{}
	for id := range book.ClassPowers() {
		ids = append(ids, id)
	}
	for id := range book.GeneralPowers() {
		ids = append(ids, id)
	}
	measured := 0
	for _, row := range allCharacters(t, s) {
		dto, err := s.sheetRules().LoadCharacter(ctx, row)
		if err != nil {
			t.Fatalf("carregar a ficha %d: %v", row.ID, err)
		}
		ec, err := sheet.EngineCharacterFrom(dto)
		if err != nil {
			t.Fatalf("converter a ficha %d: %v", row.ID, err)
		}
		for _, id := range ids {
			engineSays, sheetSays := dto.Ruleset.OwnsPower(ec, id), sheet.OwnsPower(dto, id)
			if engineSays {
				measured++
			}
			if engineSays == sheetSays {
				continue
			}
			t.Errorf("%s (ficha %d), poder %s: o motor diz %v e a ficha diz %v — as regras da mesa "+
				"e a aba Poderes respondem diferente a quem tem o poder", row.Name, row.ID, id, engineSays, sheetSays)
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma ficha da semente tem poder de classe ou geral — o guarda não mediu nada")
	}
}
