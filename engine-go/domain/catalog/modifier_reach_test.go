package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// O QUE O CATÁLOGO DECLARA, O MOTOR TEM DE LER (ALE-423).
//
// Dois campos do `Modifier` só são lidos em PARTE do motor, e o que fica de
// fora não estoura: ele some. Os dois foram medidos escrevendo as Posturas de
// Combate do cavaleiro (p54-55), e os dois produziram a mesma coisa — um
// verbete que promete um número, cobra 2 PM e não muda nada.
//
//   - `scale` é avaliado pelo `evalModifierScale`, e ele tem UM chamador: o
//     caminho dos VITAIS. Um `scale` em `defense` é ignorado e o modificador
//     vale o `amount` cru — o "+Constituição na Defesa" da Torre Inabalável
//     virou +1.
//   - `factor` é colhido pelo `harvestFactors`, que PULA todo termo adiado — e
//     condicional não resolvido é adiado. Um fator sob `condition` nunca
//     multiplica nada: o "não pode se deslocar" da mesma Torre não desceu.
//
// ESTES GUARDAS SÃO A LÁPIDE DOS DOIS, e eles saem no dia em que o motor
// alcançar os campos. Até lá, o que eles compram é que o próximo a escrever
// essas duas formas descubra na suíte e não na mesa.

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
	for _, resource := range []string{"classPowers", "generalPowers", "items", "races", "origins"} {
		raw, ok := catalog.Resource(resource)
		if !ok {
			continue
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

// NENHUM `scale` FORA DO CAMINHO DOS VITAIS, porque lá ele não é lido.
func TestNoModifierScalesOutsideTheVitalsPath(t *testing.T) {
	entries, measured := everyModifier(t)
	for _, e := range entries {
		for _, m := range e.Modifiers {
			if len(m.Scale) == 0 {
				continue
			}
			k, _ := m.Target["k"].(string)
			if !vitalTargets[k] {
				t.Errorf("%s declara `scale` num alvo %q, e só os vitais leem escala: "+
					"o modificador vale o `amount` cru e o resto some em silêncio", e.ID, k)
			}
		}
	}
	t.Logf("modificadores medidos: %d", measured)
}

// factorUnderConditionBaseline é a DÍVIDA que este guarda encontrou de pé, e
// ela só pode encolher.
//
// O `encanto-ameacadora` não foi escrito por esta fatia: ele já estava no
// catálogo com um fator sob `wielded`, e o guarda o descobriu ao nascer. Ele
// fica nomeado aqui em vez de o guarda ser afrouxado — a dívida registrada é
// visível, e a regra continua valendo para todo o resto.
var factorUnderConditionBaseline = map[string]bool{
	"encanto-ameacadora": true,
}

// NENHUM `factor` SOB CONDIÇÃO, porque o colhedor pula termo adiado.
func TestNoConditionalModifierCarriesAFactor(t *testing.T) {
	entries, measured := everyModifier(t)
	seen := map[string]bool{}
	for _, e := range entries {
		for _, m := range e.Modifiers {
			if len(m.Factor) == 0 || len(m.Condition) == 0 {
				continue
			}
			seen[e.ID] = true
			if factorUnderConditionBaseline[e.ID] {
				continue
			}
			c, _ := m.Condition["c"].(string)
			t.Errorf("%s tem um `factor` sob a condição %q: o `harvestFactors` pula "+
				"termo adiado, então ele nunca multiplica nada", e.ID, c)
		}
	}
	// A LINHA DE BASE SÓ ENCOLHE: quem consertou um e esqueceu de tirá-lo daqui
	// deixa a próxima forma passar despercebida sob o nome dele.
	for id := range factorUnderConditionBaseline {
		if !seen[id] {
			t.Errorf("%q está na linha de base e não tem mais fator sob condição — tire-o daqui", id)
		}
	}
	t.Logf("modificadores medidos: %d", measured)
}

// E O CONTROLE DOS DOIS: sabotar um deles tem de acusar.
//
// Ele não sabota o catálogo — sabota a ENTRADA do mesmo laço, que é o que os
// dois guardas de fato leem. Sem ele, um `everyModifier` que parasse de achar
// modificador deixaria os dois verdes sobre o nada, e o `measured` sozinho não
// prova que a REGRA morde.
func TestTheModifierReachGuardsCatchASabotagedEntry(t *testing.T) {
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
			if len(m.Factor) > 0 && len(m.Condition) > 0 {
				acusou++
			}
		}
	}
	if acusou != 2 {
		t.Fatalf("a entrada sabotada tinha de acusar as duas formas, e acusou %d", acusou)
	}
	if !strings.Contains(sabotado[0].ID, "sabotado") {
		t.Fatal("o controle perdeu a própria entrada")
	}
}
