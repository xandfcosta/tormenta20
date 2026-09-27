package catalog_test

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"testing"

	"t20engine/domain/catalog"
)

// AS ESCALAS QUE O MOTOR SABE LER. Uma lista de PERMITIDOS, e ela existe por
// causa da forma do `evalModifierScale`: o `switch` dele tem `attribute` no
// `default`, então um `per` que ele não conhece cai lá, procura
// `attrTotals[""]`, acha zero — e o modificador some da ficha SEM ERRO NENHUM.
//
// Um `per` digitado errado é, portanto, indistinguível de um bônus que o livro
// não dá. Este caso é o que torna a diferença visível, e ele falha com o valor
// ofensor.
var knownScales = map[string]bool{
	"flat": true, "level": true, "levelStep": true, "attribute": true,
	"patamar": true,
}

func TestNoCatalogScaleIsOneTheEngineCannotRead(t *testing.T) {
	measured := 0
	for _, resource := range catalog.Resources() {
		raw, ok := catalog.Resource(resource)
		if !ok {
			continue
		}
		var tree any
		if json.Unmarshal(raw, &tree) != nil {
			continue
		}
		walkScales(tree, func(per string) {
			measured++
			if !knownScales[per] {
				t.Errorf("%s: a escala `per: %q` não é nenhuma das que o "+
					"`evalModifierScale` conhece (%s). Ela não estoura: cai no ramo de "+
					"atributo, multiplica por zero, e o bônus some da ficha calado",
					resource, per, strings.Join(sortedKeys(knownScales), ", "))
			}
		})
	}
	// O CONTROLE: a varredura encontrou escalas. Zero reprovadas sobre zero
	// escalas lidas tem no terminal a mesma cara de um catálogo inteiro válido.
	if measured < 10 {
		t.Fatalf("só %d escalas varridas, e eram 17 quando isto foi escrito — a "+
			"varredura não está alcançando o catálogo", measured)
	}
}

func walkScales(node any, visit func(string)) {
	switch value := node.(type) {
	case map[string]any:
		if scale, ok := value["scale"].(map[string]any); ok {
			if per, ok := scale["per"].(string); ok {
				visit(per)
			}
		}
		for _, child := range value {
			walkScales(child, visit)
		}
	case []any:
		for _, child := range value {
			walkScales(child, visit)
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// O NÚMERO IMPRESSO NA DESCRIÇÃO DO PODER ÚNICO OU VIRA MODIFICADOR, OU ESTÁ
// NOMEADO AQUI.
//
// Este é o defeito da ALE-397 com outro dono: o texto do poder aparece na ficha
// e o número não muda. Depois da ALE-405 os textos são os do livro, então um
// bônus impresso sem modificador é uma promessa que a ficha não cumpre.
//
// A lista abaixo é de PERMITIDOS — o que reprova é um poder DESCONHECIDO
// imprimir um bônus sem modificador. Uma lista de proibidos subcontaria em
// silêncio, e esta nem teria denominador para denunciar a subcontagem.
var uniquePowersWithoutAModifier = map[string]string{
	"Água no Feijão": "nega a penalidade de –5 de fabricar porção adicional (p90), e o " +
		"motor não aplica essa penalidade. Um +5 valeria SEMPRE para um –5 que vale " +
		"raramente. Mesmo desenho do poder geral `Disparo Preciso`, que também nega um " +
		"–5 e não tem modificador",
	"Pão e Circo": "nega a penalidade de –5 do dano não letal (p90); veja `Água no Feijão`",
	"Gororoba":    "nega a penalidade de –5 do prato especial adicional (p94); veja `Água no Feijão`",
	"Vendedor de Carcaças": "dá +5 num teste que o livro NÃO define — `extrair recursos` " +
		"aparece uma vez no PDF inteiro, no texto deste poder (p92). Sem teste nomeado " +
		"não há alvo",
	"Desejo de Liberdade": "dá +5 em testes CONTRA a manobra agarrar (p89), e o alvo " +
		"`maneuver` significa a rolagem OFENSIVA — é o que o `Derrubar Aprimorado` usa. " +
		"Pendurar o +5 ali daria o bônus para ATACAR com agarrar também. Precisa de uma " +
		"direção no alvo, que é decisão separada (ALE-406)",
}

var printedBonus = regexp.MustCompile(`[+−–-]\s?\d+`)

func TestEveryBonusAnOriginUniquePowerPrintsBecomesAModifier(t *testing.T) {
	raw, ok := catalog.Resource("origins")
	if !ok {
		t.Fatal("catálogo de origens ausente")
	}
	var origins []struct {
		ID          string `json:"id"`
		UniquePower struct {
			Name        string           `json:"name"`
			Description string           `json:"description"`
			Modifiers   []map[string]any `json:"modifiers"`
		} `json:"poderUnico"`
	}
	if err := json.Unmarshal(raw, &origins); err != nil {
		t.Fatalf("origens: %v", err)
	}

	printed, excused := 0, map[string]bool{}
	for _, origin := range origins {
		power := origin.UniquePower
		if !printedBonus.MatchString(power.Description) {
			continue
		}
		printed++
		// A lista diz "imprime bônus e NÃO tem modificador". As duas metades
		// cobram: quem está nela e ganhou modificador sai dela, ou a lista vira
		// uma afirmação falsa que ninguém relê.
		if _, named := uniquePowersWithoutAModifier[power.Name]; named {
			excused[power.Name] = true
			if len(power.Modifiers) > 0 {
				t.Errorf("%s: %q ganhou modificador e continua na lista de exceções — "+
					"tire a linha dele", origin.ID, power.Name)
			}
			continue
		}
		if len(power.Modifiers) > 0 {
			continue
		}
		t.Errorf("%s: o poder único %q imprime um bônus e não tem modificador, então a "+
			"ficha mostra o número e não o aplica.\nDescrição: %q\nOu escreva o "+
			"modificador com a página conferida, ou acrescente o nome a "+
			"`uniquePowersWithoutAModifier` DIZENDO por que não há alvo",
			origin.ID, power.Name, power.Description)
	}
	for name := range uniquePowersWithoutAModifier {
		if !excused[name] {
			t.Errorf("%q está na lista de exceções e nenhum poder único com bônus "+
				"impresso tem esse nome — a lista envelheceu", name)
		}
	}
	// O DENOMINADOR: quantos poderes IMPRIMEM bônus. Sem ele, um regex que
	// parasse de casar deixaria o caso verde sobre nada.
	if printed != 11 {
		t.Errorf("%d poderes únicos imprimem bônus, e eram 11 quando isto foi escrito. "+
			"Se a conta mudou de propósito, mude este número junto", printed)
	}
}
