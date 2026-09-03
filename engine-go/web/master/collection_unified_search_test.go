package master

import (
	"t20engine/web/bookui"
	"testing"
)

// O guarda dos três catálogos DO PERSONAGEM (ALE-264).

func TestTheUnifiedSearchReachesTheThreeNewOnes(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "allihanna", Aba: ""}, bookui.BookAddress{})
	if v.Achados == 0 {
		t.Fatal("a busca unificada não achou a deusa Allihanna")
	}
	achouODeus := false
	for _, g := range v.Grupos {
		if g.Rotulo == "Deuses" && len(g.Deuses) > 0 {
			achouODeus = true
		}
	}
	if !achouODeus {
		t.Error("o grupo de deuses não entra na busca unificada")
	}
}
