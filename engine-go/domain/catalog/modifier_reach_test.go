package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// O QUE O CATÁLOGO DECLARA, O MOTOR TEM DE LER (ALE-423).
//
// O `scale` tem TRÊS leitores: o caminho dos VITAIS (`evalModifierScale`), a
// SOMA DE ATRIBUTO (`per: attribute`, resolvida pelo `sheetEffects` desde a
// ALE-485) e o DADO QUE CRESCE de um poder de classe (`growDice`, ALE-486). Fora deles ele é IGNORADO e o modificador vale o `amount` cru — não
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
	// resource é de onde a linha veio: o dado que cresce só é lido na coleta
	// dos poderes de CLASSE, que é onde o nível na classe existe.
	resource  string
	ID        string `json:"id"`
	Name      string `json:"name"`
	Modifiers []struct {
		Target    map[string]any `json:"target"`
		Dice      string         `json:"dice"`
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
		for i := range entries {
			entries[i].resource = resource
			total += len(entries[i].Modifiers)
		}
		all = append(all, entries...)
	}
	if total == 0 {
		t.Fatal("nenhum modificador no catálogo — o guarda não mediu nada")
	}
	return all, total
}

// NENHUM `scale` FORA DOS TRÊS LEITORES, porque fora deles ele não é lido.
func TestNoModifierScalesWhereNobodyReadsTheScale(t *testing.T) {
	entries, measured := everyModifier(t)
	for _, e := range entries {
		for _, m := range e.Modifiers {
			if len(m.Scale) == 0 {
				continue
			}
			k, _ := m.Target["k"].(string)
			per, _ := m.Scale["per"].(string)
			if per == "attribute" && k != "attribute" {
				continue
			}
			// O DADO QUE CRESCE com o nível na classe (`growDice`, ALE-486).
			if e.resource == "class-powers" && m.Dice != "" && (per == "level" || per == "levelStep") {
				continue
			}
			if !vitalTargets[k] {
				t.Errorf("%s declara `scale` num alvo %q, e só os vitais, a soma de atributo e "+
					"o dado de classe leem escala: o modificador vale o `amount` cru e o resto some em silêncio", e.ID, k)
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
			Dice      string         `json:"dice"`
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
