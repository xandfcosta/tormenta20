package engine

import (
	"fmt"
	"strings"
)

// A COMPOSIÇÃO: o que um verbete do livro JÁ TRAZ aplicado (p337, p340).
//
// Um item específico — a Avalanche, o Baluarte anão — não é regra nova. Ele é
// um item BASE com sobreposições já postas, e o livro escreve quais em ITÁLICO
// dentro do verbete:
//
//	"Este machado de guerra DE GELO ETERNO CONGELANTE FORMIDÁVEL fornece
//	 redução de fogo 10" (Avalanche, p337)
//
// Cada palavra em itálico é uma entrada que o catálogo já tem, então o verbete
// específico não redeclara nada: ele APONTA. O que sobra da frase — a redução
// de fogo 10 — são os modificadores próprios dele.

// Composition são os ids que o verbete traz. O `Base` é obrigatório; o resto é
// o que aquele item em particular tiver.
type Composition struct {
	Base         string   `json:"base"`
	Material     string   `json:"material,omitempty"`
	Improvements []string `json:"improvements,omitempty"`
	Enchants     []string `json:"enchants,omitempty"`
}

// Overlays são as sobreposições na ordem em que entram na conta — a mesma do
// `ownItemMods`, para um item composto e um item que a pessoa montou na mochila
// produzirem a lista na mesma ordem.
//
// @example composition.Overlays() // ["melhoria-reforcada", "material-adamante", "encanto-defensor"]
func (c Composition) Overlays() []string {
	ids := append([]string{}, c.Improvements...)
	if c.Material != "" {
		ids = append(ids, c.Material)
	}
	return append(ids, c.Enchants...)
}

// OverlayModifiers é o que uma SOBREPOSIÇÃO concede a um hospedeiro: com a
// procedência no `Note` e sem as linhas que a categoria do hospedeiro descarta.
//
// Exportada porque os dois lados da fronteira a chamam — a coleta do motor, que
// monta o item que a pessoa encantou na mochila, e o catálogo do livro, que
// expande o item que já vem composto. Uma segunda cópia divergiria na primeira
// correção, e o `appliesToCategory` já custou uma (ALE-438).
//
// @example engine.OverlayModifiers("armor-heavy", "Adamante", adamante.Modifiers)
func OverlayModifiers(hostCategory, overlayName string, mods []Modifier) []Modifier {
	out := make([]Modifier, 0, len(mods))
	for _, m := range mods {
		// O adamante fornece "RD 2 em armaduras leves e escudos, RD 5 em
		// pesadas" (p166): são duas linhas no catálogo, e aqui fica exatamente
		// a que casa com a peça. Linha sem categoria declarada vale em qualquer
		// hospedeiro, que é o caso de quase todas.
		if len(m.AppliesToCategory) > 0 && !contains(m.AppliesToCategory, hostCategory) {
			continue
		}
		out = append(out, withProvenance(m, overlayName))
	}
	return out
}

// withProvenance põe o nome da sobreposição na nota, que é por onde a ficha
// mostra de ONDE o número veio — a linha diz "Espada longa" na fonte e
// "Formidável: +2 atq" na nota.
func withProvenance(m Modifier, overlayName string) Modifier {
	if overlayName == "" || strings.Contains(m.Note, overlayName) {
		return m
	}
	nm := m
	if m.Note == "" {
		nm.Note = overlayName
		return nm
	}
	nm.Note = overlayName + ": " + m.Note
	return nm
}

// fromPiece marca de que PEÇA cada modificador veio — ver `Modifier.From`.
//
// Só a composição marca. O overlay que a PESSOA aplica na mochila tem a mesma
// colisão de endereço e ela é anterior a esta fatia: mudá-la mudaria endereços
// já GRAVADOS em `character_conditionals` e `campaign_silences`, e isso é fatia
// com migração (a 00019 é o precedente).
func fromPiece(id string, mods []Modifier) []Modifier {
	out := make([]Modifier, 0, len(mods))
	for _, m := range mods {
		m.From = id
		out = append(out, m)
	}
	return out
}

// expandComposedItems troca cada item composto pelo que ele concede INTEIRO.
//
// # Na primagem, e não na leitura
//
// Aqui a lista toda está na mão, e depois daqui nada no motor aprende um
// conceito novo: a carta da arma, o `ownItemMods` e a decomposição veem um item
// normal, com o bloco de arma da base e os modificadores das peças já com
// procedência. O preço de resolver na leitura seria um `if` em cada leitor, e
// quem esquecesse um deles teria metade do item funcionando.
//
// # Erro DERRUBA a primagem, de propósito
//
// Peça que não existe é defeito de catálogo, e um item específico que entra
// pela metade é exatamente a falha silenciosa que o `cmd/api` já recusa em
// outras: "um processo que sobe servindo ficha errada em silêncio é pior que um
// processo que não sobe". O guarda do `domain/catalog` pega isso antes.
func (c *Catalogs) expandComposedItems() error {
	for _, item := range c.itemsByID {
		if item.ComposedOf == nil {
			continue
		}
		base, ok := c.itemsByID[item.ComposedOf.Base]
		if !ok {
			return fmt.Errorf("o item composto %q diz que sua base é %q, e não há "+
				"item com esse id", item.ID, item.ComposedOf.Base)
		}
		// COMPOSIÇÃO SOBRE COMPOSIÇÃO não existe no livro, e aceitá-la aqui
		// deixaria o resultado depender da ORDEM de iteração de um `map`, que
		// em Go é aleatória por projeto.
		if base.ComposedOf != nil {
			return fmt.Errorf("o item composto %q tem por base %q, que também é "+
				"composto — a expansão de um dependeria da ordem da expansão do outro",
				item.ID, base.ID)
		}
		mods := append([]Modifier{}, base.Modifiers...)
		for _, id := range item.ComposedOf.Overlays() {
			piece, ok := c.itemsByID[id]
			if !ok {
				return fmt.Errorf("o item composto %q traz %q, e não há item com "+
					"esse id", item.ID, id)
			}
			mods = append(mods, fromPiece(piece.ID, OverlayModifiers(item.Category, piece.Name, piece.Modifiers))...)
		}
		// OS PRÓPRIOS POR ÚLTIMO, na ordem em que o livro escreve: o verbete
		// diz o que o item é — base e itálicos — e só então o que ele faz de
		// diferente.
		item.Modifiers = append(mods, item.Modifiers...)
		if item.Weapon == nil {
			item.Weapon = base.Weapon
		}
		if item.Equip == "" {
			item.Equip = base.Equip
		}
		if item.Slots == 0 {
			item.Slots = base.Slots
		}
	}
	return nil
}
