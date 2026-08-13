package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Condições — livro p394 ("Lista de Condições").
//
// `TestConditionEffects` já cobre alguns deltas na ficha montada. O que se fixa
// aqui é a TABELA em si (`conditionModifierTable`) contra o texto do livro: os
// valores, quais atributos cada uma atinge, e as duas regras estruturais que a
// lista enuncia no cabeçalho (ALE-105).
//
//	"Condições com os mesmos efeitos não se acumulam; aplique apenas os mais
//	 severos. Por exemplo, um personagem desprevenido e vulnerável sofre −5 na
//	 Defesa, não −7."
//
// Trabalhar na tabela, e não só na ficha, é o que pega uma condição cadastrada
// com o número errado — na ficha ela some no meio da soma.

func condTotal(mods []Modifier, target ModifierTarget) int {
	total := 0
	for _, m := range mods {
		if targetKey(m.Target) == targetKey(target) {
			total += m.Amount
		}
	}
	return total
}

func TestConditionModifierValues(t *testing.T) {
	defense := ModifierTarget{K: "defense"}

	// Os pares que só diferem em severidade — o livro usa a mesma redação com
	// −2 e −5, e trocar um pelo outro é o erro de cadastro mais provável.
	t.Run("os pares brando/severo", func(t *testing.T) {
		pares := []struct {
			brando, severo string
			target         ModifierTarget
			querBrando     int
			querSevero     int
			nota           string
		}{
			{"abalado", "apavorado", ModifierTarget{K: "expertiseAll"}, -2, -5, "medo: perícias"},
			{"fraco", "debilitado", ModifierTarget{K: "expertiseByAttribute", Attribute: "strength"}, -2, -5, "perícias de Força"},
			{"frustrado", "esmorecido", ModifierTarget{K: "expertiseByAttribute", Attribute: "intelligence"}, -2, -5, "perícias de Inteligência"},
			{"vulneravel", "desprevenido", defense, -2, -5, "Defesa"},
		}
		for _, p := range pares {
			if got := condTotal(conditionModifiers(p.brando), p.target); got != p.querBrando {
				t.Errorf("%s (%s) = %d, want %d", p.brando, p.nota, got, p.querBrando)
			}
			if got := condTotal(conditionModifiers(p.severo), p.target); got != p.querSevero {
				t.Errorf("%s (%s) = %d, want %d", p.severo, p.nota, got, p.querSevero)
			}
		}
	})

	// "DEBILITADO. O personagem sofre −5 em testes de Força, Destreza e
	// Constituição E DE PERÍCIAS BASEADAS NESSES ATRIBUTOS" — os três, e só esses.
	// O motor modela a metade das perícias (`expertiseByAttribute`), que é a que
	// a ficha deriva; o teste de atributo cru não é um número da ficha.
	t.Run("Debilitado atinge Força, Destreza e Constituição, e nada mais", func(t *testing.T) {
		mods := conditionModifiers("debilitado")
		for _, attr := range []string{"strength", "dexterity", "constitution"} {
			if got := condTotal(mods, ModifierTarget{K: "expertiseByAttribute", Attribute: attr}); got != -5 {
				t.Errorf("debilitado em %s = %d, want -5", attr, got)
			}
		}
		for _, attr := range []string{"intelligence", "wisdom", "charisma"} {
			if got := condTotal(mods, ModifierTarget{K: "expertiseByAttribute", Attribute: attr}); got != 0 {
				t.Errorf("debilitado NÃO deveria tocar %s, mas deu %d", attr, got)
			}
		}
	})

	// "ESMORECIDO. […] −5 em testes de Inteligência, Sabedoria e Carisma" — o
	// espelho mental do Debilitado. Trocar os dois é o erro simétrico.
	t.Run("Esmorecido atinge os três mentais, e nada mais", func(t *testing.T) {
		mods := conditionModifiers("esmorecido")
		for _, attr := range []string{"intelligence", "wisdom", "charisma"} {
			if got := condTotal(mods, ModifierTarget{K: "expertiseByAttribute", Attribute: attr}); got != -5 {
				t.Errorf("esmorecido em %s = %d, want -5", attr, got)
			}
		}
		if got := condTotal(mods, ModifierTarget{K: "expertiseByAttribute", Attribute: "strength"}); got != 0 {
			t.Errorf("esmorecido NÃO deveria tocar Força, mas deu %d", got)
		}
	})

	// "DESPREVENIDO. O personagem sofre −5 na Defesa E EM REFLEXOS." Duas metades
	// numa condição só: cadastrar só a Defesa passaria despercebido na ficha.
	t.Run("Desprevenido tem DUAS metades: Defesa e Reflexos", func(t *testing.T) {
		mods := conditionModifiers("desprevenido")
		if got := condTotal(mods, defense); got != -5 {
			t.Errorf("Defesa = %d, want -5", got)
		}
		if got := condTotal(mods, ModifierTarget{K: "expertise", Name: "Reflexos"}); got != -5 {
			t.Errorf("Reflexos = %d, want -5", got)
		}
	})
}

// "EXAUSTO. O personagem fica debilitado, lento e vulnerável." Uma condição
// COMPOSTA: seus efeitos têm de ser os das que ela cita, não um número próprio.
// Derivado das partes em vez de repetido, então mudar `debilitado` sem mudar
// `exausto` quebra aqui.
func TestExaustoComposesDebilitadoAndVulneravel(t *testing.T) {
	exausto := conditionModifiers("exausto")
	debilitado := conditionModifiers("debilitado")
	vulneravel := conditionModifiers("vulneravel")

	for _, attr := range []string{"strength", "dexterity", "constitution"} {
		target := ModifierTarget{K: "expertiseByAttribute", Attribute: attr}
		if got, want := condTotal(exausto, target), condTotal(debilitado, target); got != want {
			t.Errorf("exausto em %s = %d, want %d (o mesmo que debilitado)", attr, got, want)
		}
	}
	defense := ModifierTarget{K: "defense"}
	if got, want := condTotal(exausto, defense), condTotal(vulneravel, defense); got != want {
		t.Errorf("exausto na Defesa = %d, want %d (o mesmo que vulnerável)", got, want)
	}

	// "Lento" é só deslocamento, que a ficha não deriva — por isso não entra na
	// tabela, e não é omissão.
}

// A regra do cabeçalho da lista: "Condições com os mesmos efeitos não se
// acumulam; aplique apenas os mais severos."
//
// Ela é implementada pelo `bonusType: "condition"` — todas as condições
// compartilham o tipo, então o `resolveStack` guarda a de maior módulo por alvo
// em vez de somar. Este teste ataca a REGRA no ponto onde ela mora.
func TestConditionsApplyOnlyTheMostSevere(t *testing.T) {
	defense := ModifierTarget{K: "defense"}
	t.Run("o exemplo do próprio livro: desprevenido + vulnerável = −5, não −7", func(t *testing.T) {
		contribs := []Contribution{}
		for _, id := range []string{"desprevenido", "vulneravel"} {
			for _, m := range conditionModifiers(id) {
				if targetKey(m.Target) == targetKey(defense) {
					contribs = append(contribs, Contribution{Source: id, BonusType: m.BonusType, Amount: m.Amount})
				}
			}
		}
		if got := resolveStack(contribs).Total; got != -5 {
			t.Errorf("Defesa = %d, want -5 (somou em vez de pegar a pior)", got)
		}
	})

	// A regra só funciona porque TODA condição usa o mesmo bonusType. Uma
	// cadastrada como "untyped" empilharia com as outras em silêncio.
	t.Run("toda condição da tabela usa bonusType condition", func(t *testing.T) {
		for id, mods := range conditionModifierTable {
			for _, m := range mods {
				if m.BonusType != "condition" {
					t.Errorf("%s tem modificador com bonusType %q — empilharia em vez de competir", id, m.BonusType)
				}
			}
		}
	})
}

// GAP CONHECIDO, fixado como está de propósito (ALE-112).
//
// "CEGO. O personagem fica DESPREVENIDO e lento […]" — e Desprevenido é "−5 na
// Defesa E EM REFLEXOS". A tabela dá ao cego a metade da Defesa e NÃO a de
// Reflexos, então um personagem cego resiste a área melhor do que deveria.
//
// O teste afirma o comportamento ATUAL para impedir regressão silenciosa; a
// correção muda número de personagem e precisa dos dois motores + regeneração do
// oráculo, então está registrada, não feita aqui.
func TestCegoIsMissingTheReflexosHalfOfDesprevenido(t *testing.T) {
	cego := conditionModifiers("cego")
	if got := condTotal(cego, ModifierTarget{K: "defense"}); got != -5 {
		t.Errorf("cego na Defesa = %d, want -5", got)
	}
	reflexos := condTotal(cego, ModifierTarget{K: "expertise", Name: "Reflexos"})
	if reflexos != 0 {
		t.Fatalf("cego agora penaliza Reflexos em %d — se isso foi intencional, "+
			"feche a ALE-112 e troque este teste por um que exija -5", reflexos)
	}
}

// Sanidade cruzada: a tabela de modificadores vive no Go, e o catálogo que o
// jogador VÊ é o JSON servido pelo /catalog. Uma condição com modificadores que
// não esteja no catálogo nunca aparece na tela; uma do catálogo sem
// modificadores é só lembrete (Lento, Imóvel), o que é legítimo.
func TestModelledConditionsExistInTheServedCatalog(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(mustWd(t), "..", "catalog", "data", "conditions.json"))
	if err != nil {
		t.Fatalf("ler o catálogo de condições: %v", err)
	}
	var catalogo map[string]struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &catalogo); err != nil {
		t.Fatalf("catálogo de condições ilegível: %v", err)
	}
	for id := range conditionModifierTable {
		if _, ok := catalogo[id]; !ok {
			t.Errorf("condição %q tem modificadores mas não está no catálogo — nunca chega à tela", id)
		}
	}
}
