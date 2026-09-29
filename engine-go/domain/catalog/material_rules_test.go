package catalog

import (
	"encoding/json"
	"sort"
	"testing"
)

// OS MATERIAIS ESPECIAIS, pelo que sobrevive SEM o livro na mão (ALE-415).
//
// A conferência contra a p166-167 é do `scripts/audit-materials.py`, que lê o
// PDF e não roda na CI. O que fica aqui é a coerência INTERNA da entrada — e
// ela é mais do que parece, porque o preço de um material é uma MATRIZ e o
// catálogo guarda a matriz junto do número que a tela mostra. Dois lugares
// dizendo o mesmo preço é exatamente o que diverge sozinho.

type specialMaterial struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	// Em ponto flutuante porque o catálogo tem item de T$ 0,1 (uma vela), e uma
	// struct que não o leia derruba a LEITURA INTEIRA do arquivo — o guarda
	// morre num item que ele nem vinha medir.
	Price       float64            `json:"price"`
	PriceByItem map[string]float64 `json:"priceByItem"`
	AppliesTo   []string           `json:"appliesTo"`
	Description string             `json:"description"`
	Modeled     []string           `json:"modeled"`
	Unmodeled   map[string]string  `json:"unmodeled"`
	Modifiers   json.RawMessage    `json:"modifiers"`
}

// familiaDoTipoDeItem traduz a linha da Tabela 3-9 para a família do
// `appliesTo`. As duas armaduras caem na mesma: o livro cobra preços
// diferentes por PESO, e o `appliesTo` classifica por família.
var familiaDoTipoDeItem = map[string]string{
	"weapon": "weapon", "armorLight": "armor", "armorHeavy": "armor",
	"shield": "shield", "esoteric": "esoteric",
}

// O PREÇO DA TELA É O DA MATRIZ, e a família segue a matriz.
//
// O `price` é um número só porque a tela mostra um número só, e a Tabela 3-9
// dá CINCO por material. Ele carrega a coluna de ARMA e a verdade inteira mora
// no `priceByItem` — duas grafias do mesmo preço, e a que diverge sozinha é
// sempre a copiada.
//
// O `appliesTo` sai da mesma matriz: um material vira o que o livro precifica,
// e a madeira Tollon tem travessão nas duas armaduras.
func TestEverySpecialMaterialAgreesWithItsOwnPriceMatrix(t *testing.T) {
	materiais := itemsOfCategory[specialMaterial](t, "material")
	for _, m := range materiais {
		daArma, tem := m.PriceByItem["weapon"]
		if !tem {
			t.Errorf("%s não diz o preço de ARMA, e é ele que o `price` carrega", m.Name)
			continue
		}
		if m.Price != daArma {
			t.Errorf("%s mostra T$ %g na tela e a matriz diz T$ %g para arma — "+
				"o `price` é a coluna de arma do `priceByItem`", m.Name, m.Price, daArma)
		}
		familias := map[string]bool{}
		for tipo := range m.PriceByItem {
			familia, conhecida := familiaDoTipoDeItem[tipo]
			if !conhecida {
				t.Errorf("%s dá preço para %q, e a Tabela 3-9 tem cinco linhas: "+
					"weapon, armorLight, armorHeavy, shield, esoteric", m.Name, tipo)
				continue
			}
			familias[familia] = true
		}
		esperado := []string{}
		for familia := range familias {
			esperado = append(esperado, familia)
		}
		sort.Strings(esperado)
		aceita := append([]string{}, m.AppliesTo...)
		sort.Strings(aceita)
		if len(aceita) != len(esperado) || !igual(aceita, esperado) {
			t.Errorf("%s aceita %v e tem preço para %v — o que o livro não precifica, "+
				"o material não vira", m.Name, aceita, esperado)
		}
	}
	t.Logf("materiais medidos: %d", len(materiais))
}

// TODA METADE DO VERBETE OU É APLICADA OU DIZ POR QUE NÃO.
//
// O denominador é por METADE e não por material, e a diferença é o ponto: o
// mitral aplica a margem de ameaça e não aplica o limite de Destreza da
// armadura pesada. "Tem modifiers?" daria verde sobre os dois.
//
// Uma metade pode estar nas DUAS listas — é o caso do "Armadura e Escudo" do
// mitral, em que metade entra e metade não. Quem cobra que a união fecha com o
// verbete é o auditor, que tem o livro; aqui fica o que não precisa dele.
func TestEverySpecialMaterialEitherAppliesOrSaysWhyNot(t *testing.T) {
	aplicadas, caladas := 0, 0
	for _, m := range itemsOfCategory[specialMaterial](t, "material") {
		temModificador := len(m.Modifiers) > 2 // "[]" é o vazio
		switch {
		case temModificador && len(m.Modeled) == 0:
			t.Errorf("%s tem modificador e não diz QUAL metade do verbete ele aplica",
				m.Name)
		case !temModificador && len(m.Modeled) > 0:
			t.Errorf("%s diz aplicar %v e não tem modificador nenhum", m.Name, m.Modeled)
		}
		if len(m.Modeled) == 0 && len(m.Unmodeled) == 0 {
			t.Errorf("%s não declara metade nenhuma.\n"+
				"Material que o motor não aplica entra com o MOTIVO escrito, metade a "+
				"metade: sem isso, \"transcrevi os materiais\" e \"transcrevi cinco das "+
				"vinte e três metades\" têm a mesma cara.", m.Name)
		}
		for metade, motivo := range m.Unmodeled {
			if motivo == "" {
				t.Errorf("%s/%s está na lista do que não foi modelado sem motivo",
					m.Name, metade)
			}
		}
		if m.Description == "" {
			t.Errorf("%s entrou sem a regra do livro em prosa", m.Name)
		}
		aplicadas += len(m.Modeled)
		caladas += len(m.Unmodeled)
	}
	t.Logf("metades declaradas: %d aplicadas, %d com motivo", aplicadas, caladas)
}

func igual(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
