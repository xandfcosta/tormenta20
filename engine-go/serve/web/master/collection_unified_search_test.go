package master

import (
	"t20engine/serve/web/bookui"
	"testing"
)

// O guarda dos três catálogos DO PERSONAGEM (ALE-264).

func TestTheUnifiedSearchReachesTheThreeNewOnes(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "allihanna", Aba: ""}, bookui.BookAddress{})
	if v.Findings == 0 {
		t.Fatal("a busca unificada não achou a deusa Allihanna")
	}
	foundGod := false
	for _, g := range v.Groups {
		if g.Label == "Deuses" && len(g.Gods) > 0 {
			foundGod = true
		}
	}
	if !foundGod {
		t.Error("o grupo de deuses não entra na busca unificada")
	}
}
