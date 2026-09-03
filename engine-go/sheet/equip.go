package sheet

import (
	"fmt"
	"slices"
	"strings"

	"t20engine/engine"
)

// AS REGRAS DE EQUIPAR: o eixo do item, os dois tetos do livro e a conta de
// espaços.
//
// Portadas do `t20-data` e dos validadores do backend Nest — o eixo vinha do
// `equip-axis.helpers`, os tetos do `rules/equip` e o múltiplo de espaço do
// `characters.helpers`. A prosa estava em INGLÊS e foi traduzida ao mudar de
// pacote (ALE-278): comentário é o que uma pessoa lê, e a regra de idioma vale
// para ele.
//
// Elas moram no `sheet` e não num pacote novo porque é aqui que a ficha já
// mora: o `CharacterDTO` e o `Compute` estão ao lado, e os três imports que
// estas regras precisam — `engine`, `sqlcgen`, `plataforma` — já eram os
// permitidos deste pacote. Um `character` só para elas colidiria com o `sheet`
// sem comprar nada.
//
// Os DOIS TETOS são do livro (p141): no máximo duas mãos ocupadas e quatro
// vestidos. Não há casa de corpo em T20 — nada de elmo, botas e anel numa
// boneca —, e é por isso que a regra conta mãos e vestidos em vez de posições.

const (
	handsLimit  = 2
	VestedLimit = 4
)

// homebrewVestedOK são os esotéricos que podem ser VESTIDOS apesar de o eixo
// cru dizer só empunhado. Espelha o `items/homebrew` do `t20-data`.
var homebrewVestedOK = map[string]bool{"medalhao-de-prata": true}

func handsFor(slot string) int {
	switch slot {
	case "wielded":
		return 1
	case "wielded2":
		return 2
	default:
		return 0
	}
}

// allowedEquipStates diz quais estados um item pode ocupar, a partir do eixo dele.
func allowedEquipStates(equip string) []string {
	switch equip {
	case "vested":
		return []string{"vested"}
	case "wielded":
		return []string{"wielded", "wielded2"}
	default:
		return nil
	}
}

// EquipAxisError cobra que o `equipped` caiba no eixo do item do catálogo.
//
// Devolve DUAS mensagens — a do topo e a do campo — e as duas vêm vazias quando
// a troca é válida, quando o item é inventado ou desconhecido, e quando a
// permissão de vestir do homebrew se aplica. Duas porque quem mostra decide:
// o formulário põe a do campo ao lado da linha, a API JSON responde a do topo.
func EquipAxisError(catalog *engine.CatalogItem, equipped string) (topMsg, fieldMsg string) {
	if catalog == nil {
		return "", ""
	}
	if equipped == "vested" && homebrewVestedOK[catalog.ID] {
		return "", ""
	}
	allowed := allowedEquipStates(catalog.Equip)
	if contains(allowed, equipped) {
		return "", ""
	}
	expected := "null (item is not equippable)"
	field := fmt.Sprintf("%q não é equipável", catalog.Name)
	if len(allowed) > 0 {
		expected = "null | " + quoteJoin(allowed)
		field = fmt.Sprintf("%q só aceita %s", catalog.Name, strings.Join(allowed, " ou "))
	}
	top := fmt.Sprintf("equipped '%s' is invalid for %q (equip axis '%s') — expected %s",
		equipped, catalog.Name, catalog.Equip, expected)
	return top, field
}

// EquipLimitError aplica os dois tetos do livro sobre os OUTROS itens já
// equipados. Devolve "" quando cabe.
func EquipLimitError(otherEquipped []string, incoming string) string {
	vested, hands := 0, 0
	for _, s := range otherEquipped {
		if s == "vested" {
			vested++
		}
		hands += handsFor(s)
	}
	if incoming == "vested" {
		vested++
	}
	if vested > VestedLimit {
		return fmt.Sprintf("Limite de %d itens vestidos atingido", VestedLimit)
	}
	if hands+handsFor(incoming) > handsLimit {
		return fmt.Sprintf("Limite de %d mãos atingido", handsLimit)
	}
	return ""
}

// SlotsNotMultiple diz se o espaço NÃO é um múltiplo finito de 0,5.
//
// A carga conta de meio em meio (uma adaga ocupa 1, um bálsamo 0,5), então um
// item de 0,3 espaço não é do livro — e um infinito quebraria a soma inteira.
func SlotsNotMultiple(slots float64) bool {
	doubled := slots * 2
	return doubled != float64(int64(doubled))
}

// contains é a única checagem de pertencimento do arquivo. Ela sobrevive como
// invólucro de uma linha porque os chamadores a nomeiam; o corpo é o
// `slices.Contains` desde que a dependência entrou.
func contains(xs []string, x string) bool {
	return slices.Contains(xs, x)
}

func quoteJoin(xs []string) string {
	q := make([]string, len(xs))
	for i, x := range xs {
		q[i] = "'" + x + "'"
	}
	return strings.Join(q, " | ")
}
