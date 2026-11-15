package main

import "testing"

// O GESTO DE RAÇA É GESTO. A ficha liga a ativação ao dono pelo id OU pelo nome (`sheet.OwnsActivation`), porque
// cada catálogo escreve o id com a sua grafia: o Olhar Atordoante é `medusa-olhar-atordoante` no de raças e
// `race.medusa.olhar-atordoante` no de ativações. O auditor que junta só pelo id mede outra coisa que o app, e
// conta como "só nota" o que a aba Ações oferece.
func TestARaceAbilityWithAGestureIsCountedAsGesture(t *testing.T) {
	for _, e := range entries("raça") {
		if e.label != "Medusa: Olhar Atordoante" {
			continue
		}
		if got, _ := bucket(e); got != "gesto" {
			t.Errorf("o Olhar Atordoante, que tem gesto na aba Ações, caiu em %q", got)
		}
		return
	}
	t.Fatal("o perfil de raça não tem o Olhar Atordoante")
}

// O PODER ÚNICO DE ORIGEM DIZ O QUE APLICA, como a raça e a classe: a Vida Rústica põe a recuperação nunca abaixo
// do nível no motor e deixa a imunidade aos ingeríveis com o mestre. Sem o `modeled`, o auditor não tem como
// saber que a nota fala só da metade, e reprova o verbete que está certo.
func TestAnOriginPowerSaysWhatItApplies(t *testing.T) {
	for _, e := range entries("origem") {
		if e.label != "Selvagem: Vida Rústica" {
			continue
		}
		if got, failure := bucket(e); got != "em parte" || failure != "" {
			t.Errorf("a Vida Rústica caiu em %q com a falha %q; want em parte, sem falha", got, failure)
		}
		return
	}
	t.Fatal("o perfil de origem não tem a Vida Rústica")
}
