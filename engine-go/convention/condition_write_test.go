package convention

import "testing"

// TODA ESCRITA DE CONDIÇÃO PASSA PELO FUNIL, e o funil é o `domain/sheet/conditions_write.go`.
//
// O que ele protege é o que uma condição DESFAZ ao entrar: a Inconsciente reverte a forma selvagem
// (p63). Cinco lugares gravavam a coluna das condições — o PV que cai, a ficha, a Mesa duas vezes e a
// API —, e um que gravasse por fora deixaria o druida desmaiado ainda na forma do animal, sem erro
// nenhum. A lista é de PERMITIDOS por nome, como a do funil de PV.
func TestEveryConditionWriteGoesThroughTheFunnel(t *testing.T) {
	const funnelFile = "domain/sheet/conditions_write.go"
	conditionWrites := map[string]bool{"UpdateConditions": true}

	measured, funnelCalls := sweepForTheFunnel(t, conditionWrites, funnelFile, func(at, query string) {
		t.Errorf("%s chama %s fora do funil.\n"+
			"Gravar as condições por aqui pula o que uma delas desfaz ao entrar: a Inconsciente\n"+
			"reverte a forma selvagem (p63). Use o `sheet.SaveConditions`, que grava e reverte.",
			at, query)
	})

	if measured < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", measured)
	}
	if funnelCalls != 1 {
		t.Fatalf("o funil chama %s %d vezes, e devia chamar uma: se ele parou de gravar, "+
			"este guarda estaria verde sobre uma ficha que não guarda condição", "UpdateConditions", funnelCalls)
	}
}
