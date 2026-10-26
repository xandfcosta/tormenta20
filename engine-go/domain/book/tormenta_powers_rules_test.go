package book

import "testing"

// OS PODERES DA TORMENTA SÃO PODERES GERAIS (p136), e o `book` os conhece. O
// recurso é um MAPA por id, e lê-lo como lista devolvia nada em silêncio — a
// aba Poderes e o diálogo de escolhas ficavam sem nenhum (ALE-493).
func TestTheBookKnowsTheTormentaPowers(t *testing.T) {
	count := 0
	for _, p := range GeneralPowers() {
		if p.Kind == "tormenta" {
			count++
		}
	}
	if count == 0 {
		t.Fatal("o book não conhece nenhum poder da Tormenta")
	}
	if p := GeneralPowers()["olhos-vermelhos"]; p.Name != "Olhos Vermelhos" || p.Kind != "tormenta" {
		t.Errorf("olhos-vermelhos = %+v, esperava o poder da Tormenta da p137", p)
	}
}
