package api

import "testing"

// O patch que chega do socket é montado por uma LISTA de campos escrita à mão,
// e uma lista assim envelhece: ao entrar o `creatureId` (ALE-137) o cliente
// passou a mandá-lo e o servidor a descartá-lo em silêncio, com tudo
// compilando. Este teste percorre os campos em vez de conferir um.
func TestParseEntryPatchNaoPerdeCampo(t *testing.T) {
	patch := parseEntryPatch(map[string]any{
		"label":       "Chefe bandido",
		"initiative":  float64(17),
		"characterId": float64(3),
		"hpCurrent":   float64(20),
		"hpMax":       float64(30),
		"mpCurrent":   float64(5),
		"mpMax":       float64(10),
		"hpHidden":    true,
		"creatureId":  float64(7),
	})

	faltando := []string{}
	if patch.Label == nil {
		faltando = append(faltando, "label")
	}
	if patch.Initiative == nil {
		faltando = append(faltando, "initiative")
	}
	if patch.CharacterID == nil {
		faltando = append(faltando, "characterId")
	}
	if patch.HpCurrent == nil || patch.HpMax == nil {
		faltando = append(faltando, "hp")
	}
	if patch.MpCurrent == nil || patch.MpMax == nil {
		faltando = append(faltando, "mp")
	}
	if patch.HpHidden == nil {
		faltando = append(faltando, "hpHidden")
	}
	if patch.CreatureID == nil {
		faltando = append(faltando, "creatureId")
	}
	if len(faltando) > 0 {
		t.Fatalf("o parser descartou: %v", faltando)
	}
	if *patch.CreatureID != 7 {
		t.Errorf("creatureId chegou como %d, queria 7", *patch.CreatureID)
	}
}

// Condição em NPC (ALE-122, destravada pela ALE-137). A lista vem do CATÁLOGO
// e não de uma cópia escrita aqui: a cópia anterior desviou do livro — faltava
// `enfeitiçado`, e aplicá-la dava 400 para todo mundo.
func TestParseConditionsFiltraPeloCatalogo(t *testing.T) {
	// Repare no ç: `enfeitiçado` é o ÚNICO dos 35 ids do catálogo com acento —
	// todos os outros são normalizados ("caido", não "caído"). É o mesmo id que
	// já derrubou a aplicação com 400 quando a API tinha a lista à mão, e a
	// grafia irregular é o que faz qualquer cópia errar de novo.
	list := parseConditions([]any{"caido", "inventada", "enfeitiçado", "atordoado"})

	if len(list) != 3 {
		t.Fatalf("passaram %v, queria as três do livro", list)
	}
	for _, id := range list {
		if id == "inventada" {
			t.Fatalf("id fora do catálogo passou: %v", list)
		}
	}
}

// Id desconhecido derruba um item, não a aplicação inteira: no meio do combate
// o mestre perderia as outras condições junto.
func TestParseConditionsNaoDuplicaNemQuebra(t *testing.T) {
	list := parseConditions([]any{"caido", "caido", 42, nil, ""})

	if len(list) != 1 || list[0] != "caido" {
		t.Fatalf("esperava só caido uma vez, veio %v", list)
	}
	if parseConditions(nil) == nil {
		t.Fatal("lista ausente tem de virar vazia, não nil — o cliente itera sem checar")
	}
}

// A condição é estado de COMBATE e mora na linha, como os PV atuais: o bloco de
// criatura descreve o vilão, e ele não volta na semana seguinte ainda caído.
func TestCondicaoEntraESaiDaLinha(t *testing.T) {
	st := emptyRuntimeState()
	id := counter()
	_ = addEntry(st, npc("Ogro", 12), id)

	aplicadas := []string{"caido", "atordoado"}
	if err := updateEntry(st, "e1", entryPatch{Conditions: &aplicadas}); err != nil {
		t.Fatalf("aplicar: %v", err)
	}
	if len(st.Initiative[0].Conditions) != 2 {
		t.Fatalf("aplicou %v", st.Initiative[0].Conditions)
	}

	vazio := []string{}
	_ = updateEntry(st, "e1", entryPatch{Conditions: &vazio})
	if len(st.Initiative[0].Conditions) != 0 {
		t.Fatalf("limpar deixou %v", st.Initiative[0].Conditions)
	}
}
