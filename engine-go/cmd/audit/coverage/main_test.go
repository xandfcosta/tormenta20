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

// O CAMPO QUE O MOTOR LÊ É EFEITO, mesmo sem modificador. O ataque pago (`ExtraStrike`), o aprimoramento
// (`Boost`), o que se soma a ele (`BoostAddOn`) e o dado da arma favorita (`FavoredWeaponDie`) mudam a ficha e a
// mesa por um campo do verbete. Sem contá-los, só a nota `modeled` escrita à mão sustenta o "inteiro" — e um
// campo que saísse deixaria a nota dizendo um efeito que não existe mais, sem nada reprovar. Por isso o caso
// tira a nota: o balde tem de vir do CAMPO.
func TestAClassPowerFieldTheEngineReadsCountsAsEffect(t *testing.T) {
	byField := map[string]string{
		"ExtraStrike":      "Lutador: Golpe Relâmpago",
		"Boost":            "Guerreiro: Ataque Especial",
		"BoostAddOn":       "Guerreiro: Romper Resistências",
		"FavoredWeaponDie": "Paladino: Arma Sagrada",
	}
	found := map[string]bool{}
	for _, e := range entries("classe") {
		for field, label := range byField {
			if e.label != label || found[field] {
				continue
			}
			found[field] = true
			e.modeled = false
			if got, failure := bucket(e); got != "inteiro" || failure != "" {
				t.Errorf("%s (pelo %s), sem a nota, caiu em %q com a falha %q; want inteiro, sem falha", label, field, got, failure)
			}
		}
	}
	for field, label := range byField {
		if !found[field] {
			t.Errorf("o perfil de classe não tem %s, o caso do %s", label, field)
		}
	}
}
