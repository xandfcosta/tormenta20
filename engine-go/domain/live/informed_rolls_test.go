package live

import (
	"encoding/json"
	"testing"
)

// Cada rolagem informada guarda a ÚLTIMA do tipo dela: a magia conjurada depois de um teste não apaga o teste,
// e as duas faixas ficam na mesa até a próxima de cada uma.
func TestEachInformedRollReplacesOnlyTheLastOfItsKind(t *testing.T) {
	st := cenaEmCurso()
	id := counter()
	_ = AddEntry(st, npc("Ogro", 20), id)
	_ = AddEntry(st, charEntry("Arcanista", 15, 7), id)

	rolls := []InformedRoll{
		SkillTestRoll{Who: "Arwen", Skill: "Percepção", Total: 12},
		SkillTestRoll{Who: "Arwen", Skill: "Furtividade", Total: 18},
		SpellCastRoll{Who: "Arwen", Spell: "Curar Ferimentos"},
		BlastRoll{Who: "Arwen", Name: "Bola de fogo", AttackerEntryID: "e2", TargetEntryID: "e1"},
	}
	for _, roll := range rolls {
		if err := RecordRoll(st, roll); err != nil {
			t.Fatalf("guardar %+v: %v", roll, err)
		}
	}
	if st.LastSkillTest == nil || st.LastSkillTest.Skill != "Furtividade" {
		t.Errorf("o segundo teste devia substituir o primeiro, e a mesa tem %+v", st.LastSkillTest)
	}
	if st.LastSpellCast == nil || st.LastSpellCast.Spell != "Curar Ferimentos" {
		t.Errorf("a magia conjurada sumiu da mesa: %+v", st.LastSpellCast)
	}
	if st.LastBlast == nil || st.LastBlast.Name != "Bola de fogo" {
		t.Errorf("o disparo sumiu da mesa: %+v", st.LastBlast)
	}
}

// O disparo é contra uma linha da fila, e guardá-lo contra quem não está nela é recusa, que deixa a mesa como
// estava.
func TestTheBlastAgainstWhoIsNotInTheQueueIsRefused(t *testing.T) {
	st := cenaEmCurso()
	_ = AddEntry(st, npc("Ogro", 20), counter())

	err := RecordRoll(st, BlastRoll{Who: "Arwen", Name: "Bola de fogo", AttackerEntryID: "e1", TargetEntryID: "e9"})
	if err == nil || st.LastBlast != nil {
		t.Errorf("o disparo contra quem não está na fila passou: erro %v, mesa %+v", err, st.LastBlast)
	}
}

// FORA DE CENA o jogador não recebe a fila, e o que atravessa é o que não depende dela: o teste (a Percepção que
// abre a cena vem antes da cena) e a magia (a cura entre as cenas). O disparo é contra uma linha da fila, e
// atravessar sem ela daria ao jogador "→ Ogro" sem Ogro nenhum.
func TestOffSceneTheTestAndTheSpellReachThePlayerAndTheBlastDoesNot(t *testing.T) {
	st := cenaEmCurso()
	id := counter()
	_ = AddEntry(st, npc("Ogro", 20), id)
	_ = AddEntry(st, charEntry("Arcanista", 15, 7), id)
	for _, roll := range []InformedRoll{
		SkillTestRoll{Who: "Arwen", Skill: "Percepção"},
		SpellCastRoll{Who: "Arwen", Spell: "Curar Ferimentos"},
		BlastRoll{Who: "Arwen", Name: "Bola de fogo", AttackerEntryID: "e2", TargetEntryID: "e1"},
	} {
		if err := RecordRoll(st, roll); err != nil {
			t.Fatalf("guardar %+v: %v", roll, err)
		}
	}

	inScene := RedactForPlayers(st)
	if inScene.LastSkillTest == nil || inScene.LastSpellCast == nil || inScene.LastBlast == nil {
		t.Fatalf("em cena o jogador perdeu uma faixa: %+v", inScene.InformedRolls)
	}

	EndScene(st)

	outside := RedactForPlayers(st)
	if outside.LastSkillTest == nil {
		t.Error("fora de cena o jogador não vê o teste rolado")
	}
	if outside.LastSpellCast == nil {
		t.Error("fora de cena o jogador não vê a magia conjurada")
	}
	if outside.LastBlast != nil {
		t.Errorf("fora de cena o disparo chegou ao jogador sem a fila: %+v", outside.LastBlast)
	}
}

// O FORMATO GRAVADO não muda com a struct embutida: a sessão gravada com as faixas no topo do JSON continua
// lendo as faixas dela, e a gravada agora as escreve no mesmo lugar.
func TestTheInformedRollsKeepTheirPlaceInTheSavedSession(t *testing.T) {
	saved := `{"initiative":[],"round":0,"turnIndex":-1,"lastSkillTest":{"who":"Arwen","skill":"Percepção","roll":14,"modifier":7,"total":21,"byUserId":3},"lastSpellCast":{"who":"Arwen","spell":"Curar Ferimentos","range":"toque","characterId":2,"byUserId":3}}`
	var st SessionRuntimeState
	if err := json.Unmarshal([]byte(saved), &st); err != nil {
		t.Fatalf("ler a sessão gravada: %v", err)
	}
	if st.LastSkillTest == nil || st.LastSkillTest.Total != 21 || st.LastSpellCast == nil {
		t.Fatalf("a sessão gravada perdeu as faixas: %+v", st.InformedRolls)
	}
	written, err := json.Marshal(&st)
	if err != nil {
		t.Fatalf("gravar: %v", err)
	}
	var top map[string]json.RawMessage
	_ = json.Unmarshal(written, &top)
	for _, key := range []string{"lastSkillTest", "lastSpellCast"} {
		if _, ok := top[key]; !ok {
			t.Errorf("a sessão gravada não tem %q no topo: %s", key, written)
		}
	}
	if _, nested := top["InformedRolls"]; nested {
		t.Errorf("as faixas foram gravadas dentro de um objeto aninhado: %s", written)
	}
}
