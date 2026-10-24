package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// O QUE O CATÁLOGO DECLARA, O MOTOR TEM DE LER (ALE-423).
//
// O `scale` tem DOIS leitores: o caminho dos VITAIS (`evalModifierScale`) e a
// SOMA DE ATRIBUTO (`per: attribute`, resolvida pelo `sheetEffects` desde a
// ALE-485). Fora deles ele é IGNORADO e o modificador vale o `amount` cru — não
// estoura, não avisa, e o verbete passa a prometer um número que ninguém soma.
// Foi assim que o "+Constituição na Defesa" da Torre Inabalável (p55) virou +1.
//
// O IRMÃO DELE JÁ SAIU: havia aqui um guarda contra `factor` sob condição, e o
// defeito que ele descrevia foi consertado — o `ApplyActiveConditionals` passa
// a aplicar o fator do condicional LIGADO. Quem protege aquilo agora é o
// `TestTheFactorOfASwitchedOnConditionalMultiplies`, no `domain/engine`, que é
// a camada onde a regra mora.

// vitalTargets são os alvos cujo caminho de fato lê o `scale`.
var vitalTargets = map[string]bool{"maxPv": true, "maxPm": true, "tempHp": true, "tempMp": true}

type modifierSlice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Modifiers []struct {
		Target    map[string]any `json:"target"`
		Scale     map[string]any `json:"scale"`
		Factor    map[string]any `json:"factor"`
		Condition map[string]any `json:"condition"`
	} `json:"modifiers"`
}

// everyModifier varre os catálogos que carregam modificador, e DIZ QUANTOS.
func everyModifier(t *testing.T) ([]modifierSlice, int) {
	t.Helper()
	all, total := []modifierSlice{}, 0
	// Os nomes são os do registro, e um nome errado é FALHA: este guarda varreu
	// meses só itens e origens porque pedia `classPowers` e `generalPowers`, que
	// não existem, e o `continue` engolia a ausência.
	for _, resource := range []string{"class-powers", "general-powers", "granted-powers", "items", "origins"} {
		raw, ok := catalog.Resource(resource)
		if !ok {
			t.Fatalf("o recurso %q não existe no catálogo — o guarda deixaria de medi-lo", resource)
		}
		var entries []modifierSlice
		if json.Unmarshal(raw, &entries) != nil {
			continue
		}
		for _, e := range entries {
			total += len(e.Modifiers)
		}
		all = append(all, entries...)
	}
	if total == 0 {
		t.Fatal("nenhum modificador no catálogo — o guarda não mediu nada")
	}
	return all, total
}

// NENHUM `scale` FORA DOS DOIS LEITORES, porque fora deles ele não é lido.
func TestNoModifierScalesWhereNobodyReadsTheScale(t *testing.T) {
	entries, measured := everyModifier(t)
	for _, e := range entries {
		for _, m := range e.Modifiers {
			if len(m.Scale) == 0 {
				continue
			}
			k, _ := m.Target["k"].(string)
			if per, _ := m.Scale["per"].(string); per == "attribute" && k != "attribute" {
				continue
			}
			if !vitalTargets[k] {
				t.Errorf("%s declara `scale` num alvo %q, e só os vitais e a soma de atributo "+
					"leem escala: o modificador vale o `amount` cru e o resto some em silêncio", e.ID, k)
			}
		}
	}
	t.Logf("modificadores medidos: %d", measured)
}

// E O CONTROLE: sabotar uma entrada tem de acusar.
//
// Ele não sabota o catálogo — sabota a ENTRADA do mesmo laço, que é o que o
// guarda de fato lê. Sem ele, um `everyModifier` que parasse de achar
// modificador deixaria o guarda verde sobre o nada, e o `measured` sozinho não
// prova que a REGRA morde.
func TestTheModifierReachGuardCatchesASabotagedEntry(t *testing.T) {
	sabotado := []modifierSlice{{
		ID: "teste.sabotado",
		Modifiers: []struct {
			Target    map[string]any `json:"target"`
			Scale     map[string]any `json:"scale"`
			Factor    map[string]any `json:"factor"`
			Condition map[string]any `json:"condition"`
		}{{
			Target:    map[string]any{"k": "defense"},
			Scale:     map[string]any{"per": "attribute"},
			Factor:    map[string]any{"num": 0, "den": 1},
			Condition: map[string]any{"c": "flagOn"},
		}},
	}}
	acusou := 0
	for _, e := range sabotado {
		for _, m := range e.Modifiers {
			k, _ := m.Target["k"].(string)
			if len(m.Scale) > 0 && !vitalTargets[k] {
				acusou++
			}
		}
	}
	if acusou != 1 {
		t.Fatalf("a entrada sabotada tinha de acusar a escala, e acusou %d", acusou)
	}
	if !strings.Contains(sabotado[0].ID, "sabotado") {
		t.Fatal("o controle perdeu a própria entrada")
	}
}
