package engine

import (
	"regexp"
	"strconv"
	"strings"
)

// OS ITENS DA ORIGEM (p85–95, a linha "Itens" de cada bloco).
//
// "Personagens de 1º nível começam com os itens fornecidos pela sua origem e os
// itens a seguir" (p140) — a origem entra no nascimento junto com o kit.
//
// O catálogo guarda a frase do livro literalmente, e várias delas NÃO são
// concessão: são escolha ("Estojo de disfarces OU gazua"), teto de preço ("Um
// item estrangeiro (até T$ 100)") ou dinheiro ("T$ 2d6"). Escrever a frase como
// nome de item faria a Mochila do acólito ganhar uma linha chamada "Arma
// marcial" com um espaço de carga — um item que não existe.
//
// Este classificador diz de que tipo é cada frase. Quem decide o que fazer com
// cada tipo é quem chama: a forja curta concede as fixas, rola as de dinheiro, e
// deixa as de escolha como linha a resolver na Mochila.

// OriginItemKind é o tipo de uma concessão de origem.
type OriginItemKind string

const (
	// OriginItemFixed é o item concedido de verdade: "Símbolo sagrado".
	OriginItemFixed OriginItemKind = "fixed"
	// OriginItemWeapon é uma arma a escolher dentro de categorias.
	OriginItemWeapon OriginItemKind = "weapon"
	// OriginItemAny é um item qualquer com teto de preço.
	OriginItemAny OriginItemKind = "anyItem"
	// OriginItemOneOf é a escolha entre alternativas escritas.
	OriginItemOneOf OriginItemKind = "oneOf"
	// OriginItemMoney é bolsa e não item: "T$ 2d6 (último salário)".
	OriginItemMoney OriginItemKind = "money"
)

// OriginItemGrant é uma linha "Itens" já classificada. Os campos que não
// pertencem ao tipo ficam vazios — um `struct` só, porque Go não tem união e
// cinco tipos irmãos custariam mais do que a leitura de um `Kind`.
type OriginItemGrant struct {
	Kind OriginItemKind
	// Label é sempre a frase do livro, inteira. É ela que a tela mostra e é ela
	// que vira a linha a resolver quando a escolha não foi feita.
	Label string
	// Name só existe no tipo fixo, e é o nome do item.
	Name string
	// Categories são as categorias de arma aceitas (tipo weapon).
	Categories []string
	// MaxPrice é o teto em tibares (tipo anyItem).
	MaxPrice int
	// Options são as alternativas escritas (tipo oneOf).
	Options []string
	// Dice é o dado da bolsa (tipo money).
	Dice string
}

// originWeaponEntries são as três frases que o livro usa para "escolha uma arma".
var originWeaponEntries = map[string][]string{
	"Arma simples":            {"weapon-simple"},
	"Arma marcial":            {"weapon-martial"},
	"Arma marcial ou exótica": {"weapon-martial", "weapon-exotic"},
}

var (
	originMoneyDice = regexp.MustCompile(`^T\$ (\d+d\d+)`)
	originPriceCap  = regexp.MustCompile(`até T\$ (\d+)`)
	originChoiceEnd = regexp.MustCompile(`\s*\(escolha\)$`)
	originChoiceSep = regexp.MustCompile(`,\s*|\s+ou\s+`)
)

// ParseOriginItem classifica uma frase da linha "Itens" de uma origem.
//
// A ordem dos testes é a do livro e não é livre: o "OU" MAIÚSCULO separa
// alternativas, e o "ou" minúsculo dentro de "Arma marcial ou exótica" é texto
// corrido. Trocar a ordem faria a arma virar escolha entre duas frases soltas.
//
//	ParseOriginItem("Estojo de disfarces OU gazua").Kind // OriginItemOneOf
func ParseOriginItem(entry string) OriginItemGrant {
	if categories, weapon := originWeaponEntries[entry]; weapon {
		return OriginItemGrant{Kind: OriginItemWeapon, Label: entry, Categories: categories}
	}
	if data := originMoneyDice.FindStringSubmatch(entry); data != nil {
		return OriginItemGrant{Kind: OriginItemMoney, Label: entry, Dice: data[1]}
	}
	if ceiling := originPriceCap.FindStringSubmatch(entry); ceiling != nil {
		price, _ := strconv.Atoi(ceiling[1])
		return OriginItemGrant{Kind: OriginItemAny, Label: entry, MaxPrice: price}
	}
	if strings.Contains(entry, " OU ") {
		return OriginItemGrant{
			Kind: OriginItemOneOf, Label: entry, Options: strings.Split(entry, " OU "),
		}
	}
	if originChoiceEnd.MatchString(entry) {
		list := originChoiceEnd.ReplaceAllString(entry, "")
		return OriginItemGrant{
			Kind: OriginItemOneOf, Label: entry, Options: withInitialCapital(originChoiceSep.Split(list, -1)),
		}
	}
	return OriginItemGrant{Kind: OriginItemFixed, Label: entry, Name: entry}
}

// withInitialCapital arruma as alternativas de uma lista escrita em corrido —
// "cão de caça, cavalo, pônei ou trobo" vira quatro nomes próprios de item.
func withInitialCapital(parts []string) []string {
	output := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		runes := []rune(part)
		output = append(output, strings.ToUpper(string(runes[0]))+string(runes[1:]))
	}
	return output
}
