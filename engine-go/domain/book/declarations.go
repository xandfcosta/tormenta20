package book

import "sync"

// AS DECLARAÇÕES: o poder que se USA antes de agredir e vale até o próximo turno — "se fizer isso,
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

var declarations = sync.OnceValue(func() map[string]Activation {
	byFlag := map[string]Activation{}
	for _, a := range Activations() {
		if flag := DeclaredFlag(a); flag != "" {
			byFlag[flag] = a
		}
	}
	return byFlag
})

// DeclarationFor é a ativação que declara esta flag, para quem escreve o nome e a página dela. A
// Ambidestria tem duas, uma por classe, com o mesmo nome e a mesma flag; vale qualquer uma.
//
//	book.DeclarationFor("disparo-rapido") // a ativação do Disparo Rápido, true
func DeclarationFor(flag string) (Activation, bool) {
	a, found := declarations()[flag]
	return a, found
}

// IsDeclaredFlag diz se a flag se liga por um Usar, e não por interruptor.
//
//	book.IsDeclaredFlag("disparo-rapido") // true
//	book.IsDeclaredFlag("ataque-poderoso") // false: esse é interruptor
func IsDeclaredFlag(flag string) bool {
	_, found := DeclarationFor(flag)
	return found
}
