package book

import (
	"sync"

	"t20engine/domain/engine"
)

// AS DECLARAÇÕES: o que se declara antes de agredir e vale até o próximo turno — "se fizer isso,
// sofre -2 em todos os testes de ataque até o seu próximo turno" (Estilo de Duas Armas, p125).
//
// O estado é o condicional da flag, e por isso a duração não tem código próprio: o `Ends: own-turn` dos
// modificadores o desliga quando a vez volta, e o fim da cena também. Ele não é interruptor porque
// declaração não se desliga à vontade: o que se desfaz é o uso por engano, no "encerrar" da aba Efeitos.

// DeclaredFlag é a flag que usar este poder liga, ou "" quando ele não é uma declaração.
//
//	book.DeclaredFlag(*book.ActivationOf("estilo-de-duas-armas", "")) // "estilo-de-duas-armas"
func DeclaredFlag(spec Activation) string {
	if spec.Kind != "instant" || spec.Grant == nil || spec.Grant.Kind != "switch" {
		return ""
	}
	return spec.Flag
}

// Declaration é uma declaração vista de fora: a flag que ela liga, o nome e a página que a aba Efeitos
// escreve, e o poder cujos modificadores leem a flag.
type Declaration struct {
	Flag    string
	Name    string
	PowerID string
	Page    int
}

// shapeDeclarations são as que não têm ativação própria: o livro não dá verbete à Forma Ágil, e o gesto
// dela mora no card da Forma Selvagem. O -2 está no poder da Forma Selvagem, que é quem o lê.
var shapeDeclarations = []Declaration{{
	Flag: engine.WildAgileBothFlag, Name: "Forma Ágil: as duas armas naturais",
	PowerID: engine.WildTierPowerID(engine.WildTierNormal), Page: 63,
}}

// declarations são todas: as do Usar de um poder e as da forma.
var declarations = sync.OnceValue(func() []Declaration {
	out := []Declaration{}
	for _, a := range Activations() {
		if flag := DeclaredFlag(a); flag != "" {
			out = append(out, Declaration{Flag: flag, Name: a.Name, PowerID: a.ID, Page: a.BookPage})
		}
	}
	return append(out, shapeDeclarations...)
})

// DeclarationFor é a declaração desta flag. A Ambidestria tem duas, uma por classe, com o mesmo nome e a
// mesma flag; vale a primeira.
//
//	book.DeclarationFor("disparo-rapido") // o Disparo Rápido, p125, true
func DeclarationFor(flag string) (Declaration, bool) {
	for _, d := range declarations() {
		if d.Flag == flag {
			return d, true
		}
	}
	return Declaration{}, false
}

// IsDeclaredFlag diz se a flag se liga por uma declaração, e não por interruptor.
//
//	book.IsDeclaredFlag("disparo-rapido") // true
//	book.IsDeclaredFlag("ataque-poderoso") // false: esse é interruptor
func IsDeclaredFlag(flag string) bool {
	_, found := DeclarationFor(flag)
	return found
}
