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

// "CEGO. O personagem fica DESPREVENIDO e lento […]" e "AGARRADO. O personagem
// fica desprevenido e imóvel" — as duas COMPÕEM o desprevenido inteiro, que é
// "−5 na Defesa E EM REFLEXOS".
//
// O cego tinha só a metade da Defesa (ALE-112): um personagem cego resistia a
// área melhor do que deveria. Derivado do desprevenido em vez de repetido, então
// mexer numa metade sem mexer na outra quebra aqui.
func TestCegoAndAgarradoComposeDesprevenido(t *testing.T) {
	desprevenido := conditionModifiers("desprevenido")
	defense := ModifierTarget{K: "defense"}
	reflexos := ModifierTarget{K: "expertise", Name: "Reflexos"}

	for _, id := range []string{"cego", "agarrado"} {
		mods := conditionModifiers(id)
		for _, alvo := range []struct {
			nome   string
			target ModifierTarget
		}{{"Defesa", defense}, {"Reflexos", reflexos}} {
			got, want := condTotal(mods, alvo.target), condTotal(desprevenido, alvo.target)
			if got != want {
				t.Errorf("%s em %s = %d, want %d (o mesmo que desprevenido)", id, alvo.nome, got, want)
			}
		}
	}

	// E cada uma mantém o que tem de próprio além do desprevenido.
	if got := condTotal(conditionModifiers("cego"), ModifierTarget{K: "expertiseByAttribute", Attribute: "strength"}); got != -5 {
		t.Errorf("cego em perícias de Força = %d, want -5", got)
	}
	if got := condTotal(conditionModifiers("agarrado"), ModifierTarget{K: "attack", Scope: "all"}); got != -2 {
		t.Errorf("agarrado em ataque = %d, want -2", got)
	}
}

// Uma perícia recebe modificador por TRÊS alvos distintos — `expertise` (a
// própria), `expertiseAll` (todas) e `expertiseByAttribute` (as do atributo) — e
// somar os três totais deixa efeitos do mesmo tipo acumularem entre si, o que a
// p226 proíbe e a p394 repete para condições.
//
// O golden do `tanque-exausto-atordoado-nv10` foi quem mostrou: Reflexos levava
// −10, porque o debilitado do exausto entra pelas perícias de Destreza e o
// desprevenido do atordoado entra pelo Reflexos. Duas penalidades de condição
// sobre o MESMO número da ficha (ALE-116).
func TestConditionPenaltiesOnOneSkillDoNotStackAcrossTargets(t *testing.T) {
	// Reflexos é uma perícia de Destreza — o alvo por onde o debilitado entra.
	reflexos := CharacterExpertise{Name: "Reflexos", Attribute: "dexterity"}
	ch := Character{Level: 10, Dexterity: 0, Expertises: []CharacterExpertise{reflexos}}

	sheetFor := func(conds ...string) int {
		mods := []Modifier{}
		for _, id := range conds {
			mods = append(mods, conditionModifiers(id)...)
		}
		vested := "vested"
		e := ComputeItemEffects([]ActiveItem{{Source: "Condições", Equipped: &vested, Modifiers: mods}})
		// Mochila vazia: este caso é sobre condições, e a sobrecarga não entra.
		return expertiseBreakdown(ch, reflexos, e, loadBreakdownOf(ch, 10)).ItemBonus
	}

	if got := sheetFor("desprevenido"); got != -5 {
		t.Fatalf("só desprevenido: %d, want -5", got)
	}
	if got := sheetFor("debilitado"); got != -5 {
		t.Fatalf("só debilitado: %d, want -5", got)
	}
	// As duas juntas continuam −5: "condições com os mesmos efeitos não se
	// acumulam; aplique apenas os mais severos" (p394).
	if got := sheetFor("debilitado", "desprevenido"); got != -5 {
		t.Errorf("debilitado + desprevenido em Reflexos = %d, want -5 — somou os dois alvos", got)
	}
	// E o exemplo do personagem real que expôs o bug.
	if got := sheetFor("exausto", "atordoado"); got != -5 {
		t.Errorf("exausto + atordoado em Reflexos = %d, want -5", got)
	}
}

// A p394 define VÁRIAS condições dizendo que o personagem "fica" outra. Quando
// isso acontece, a condição derivada tem de carregar TODOS os números da citada
// — foi por perder metade deles que o cego resistia melhor a área do que deveria
// (ALE-112). Este teste varre os pares direto do texto do livro, então uma
// condição nova que cite outra e esqueça seus números cai aqui.
//
// Cinco delas não tinham modificador NENHUM: um personagem atordoado, paralisado
// ou inconsciente ficava com a Defesa cheia (ALE-115).
func TestConditionsThatImplyAnotherCarryItsNumbers(t *testing.T) {
	pares := []struct {
		condicao, implicada, citacao string
	}{
		{"atordoado", "desprevenido", "O personagem fica desprevenido e não pode fazer ações"},
		{"surpreendido", "desprevenido", "O personagem fica desprevenido e não pode fazer ações"},
		{"cego", "desprevenido", "O personagem fica desprevenido e lento"},
		{"agarrado", "desprevenido", "O personagem fica desprevenido e imóvel"},
		{"paralisado", "indefeso", "Fica imóvel e indefeso"},
		{"inconsciente", "indefeso", "O personagem fica indefeso"},
		{"petrificado", "indefeso", "O personagem fica inconsciente (que é indefeso) e recebe RD 8"},
		{"fatigado", "fraco", "O personagem fica fraco e vulnerável"},
		{"fatigado", "vulneravel", "O personagem fica fraco e vulnerável"},
		{"exausto", "debilitado", "O personagem fica debilitado, lento e vulnerável"},
		{"exausto", "vulneravel", "O personagem fica debilitado, lento e vulnerável"},
		{"enredado", "vulneravel", "O personagem fica lento, vulnerável e sofre −2 em ataque"},
	}
	for _, p := range pares {
		derivada := conditionModifiers(p.condicao)
		for _, m := range conditionModifiers(p.implicada) {
			got := condTotal(derivada, m.Target)
			if got != m.Amount {
				t.Errorf("%s em %s = %d, want %d — %q implica %s inteiro",
					p.condicao, targetKey(m.Target), got, m.Amount, p.citacao, p.implicada)
			}
		}
	}
}

// O Indefeso é o único que SUBSTITUI o número da condição que cita, em vez de
// somar: "O personagem fica desprevenido, MAS sofre −10 na Defesa". O "mas" é a
// palavra que muda tudo — −10, e não −5 nem −15.
func TestIndefesoReplacesTheDesprevenidoDefense(t *testing.T) {
	if got := condTotal(conditionModifiers("indefeso"), ModifierTarget{K: "defense"}); got != -10 {
		t.Errorf("Defesa do indefeso = %d, want -10", got)
	}
	// "falha automaticamente em testes de Reflexos" não é um número, então não
	// vira modificador — fica como lembrete na ficha (ALE-115).
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

// "CAÍDO. O personagem sofre −5 na Defesa contra ataques corpo a corpo e recebe
// +5 na Defesa contra ataques à distância (CUMULATIVOS COM OUTRAS CONDIÇÕES).
// Além disso, sofre −5 em ataques corpo a corpo" (p394).
//
// Duas coisas incomuns numa linha só: a Defesa é DIRECIONAL, e o parêntese abre
// uma exceção explícita à regra de não-acúmulo do cabeçalho da lista. As duas
// caem no mesmo mecanismo — escopo no alvo — porque chaves diferentes não
// competem no `resolveStack` (ALE-115).
func TestCaidoDefenseIsDirectionalAndCumulative(t *testing.T) {
	defenseFor := func(conds ...string) DefenseBreakdown {
		mods := []Modifier{}
		for _, id := range conds {
			mods = append(mods, conditionModifiers(id)...)
		}
		vested := "vested"
		e := ComputeItemEffects([]ActiveItem{{Source: "Condições", Equipped: &vested, Modifiers: mods}})
		e.Flags = map[string]bool{}
		return defenseBreakdown(Character{Level: 1, Dexterity: 0}, e)
	}

	t.Run("as duas direções saem do mesmo caído", func(t *testing.T) {
		d := defenseFor("caido")
		if d.Total != 10 {
			t.Errorf("Defesa geral = %d, want 10 — o caído não deveria mexer nela", d.Total)
		}
		if d.VsMelee != 5 {
			t.Errorf("contra corpo a corpo = %d, want 5 (10 − 5)", d.VsMelee)
		}
		if d.VsRanged != 15 {
			t.Errorf("contra à distância = %d, want 15 (10 + 5)", d.VsRanged)
		}
	})

	// O parêntese do livro: a Defesa do caído SOMA com a de outra condição, em vez
	// de competir com ela como as condições normalmente fazem. Um caído e
	// desprevenido leva −5 do desprevenido E −5 do caído contra corpo a corpo.
	t.Run("cumulativo com outra condição, ao contrário do resto da lista", func(t *testing.T) {
		d := defenseFor("caido", "desprevenido")
		if d.Total != 5 {
			t.Errorf("Defesa geral = %d, want 5 (o −5 do desprevenido)", d.Total)
		}
		if d.VsMelee != 0 {
			t.Errorf("contra corpo a corpo = %d, want 0 (10 −5 desprevenido −5 caído)", d.VsMelee)
		}
		if d.VsRanged != 10 {
			t.Errorf("contra à distância = %d, want 10 (10 −5 desprevenido +5 caído)", d.VsRanged)
		}
	})

	// E as duas condições NÃO-direcionais continuam competindo entre si, que é a
	// regra geral do cabeçalho — o escopo do caído não abriu buraco nela.
	t.Run("as não-direcionais continuam competindo", func(t *testing.T) {
		if d := defenseFor("desprevenido", "vulneravel"); d.Total != 5 {
			t.Errorf("desprevenido + vulnerável = %d, want 5 (a pior, não a soma)", d.Total)
		}
	})

	// "−5 em ataques corpo a corpo" — no motor, ataque corpo a corpo é teste de Luta.
	t.Run("o −5 em ataques corpo a corpo é Luta", func(t *testing.T) {
		if got := condTotal(conditionModifiers("caido"), ModifierTarget{K: "expertise", Name: "Luta"}); got != -5 {
			t.Errorf("Luta = %d, want -5", got)
		}
	})
}

// "INDEFESO. O personagem fica desprevenido, mas sofre −10 na Defesa, FALHA
// AUTOMATICAMENTE EM TESTES DE REFLEXOS e pode sofrer golpes de misericórdia."
//
// A falha automática não é um número: virá-la em −5 inventaria uma regra mais
// branda que a do livro, e deixá-la de fora fazia um personagem inconsciente
// rolar Reflexos normalmente. Vai como FLAG, que é o mecanismo do motor para
// efeito booleano — a ficha mostra "falha automática" na linha em vez de um
// total (ALE-115).
func TestIndefesoAutoFailsReflexos(t *testing.T) {
	vested := "vested"
	flagsFor := func(ids ...string) map[string]bool {
		mods := []Modifier{}
		for _, id := range ids {
			mods = append(mods, conditionModifiers(id)...)
		}
		return ComputeItemEffects([]ActiveItem{{Source: "Condições", Equipped: &vested, Modifiers: mods}}).Flags
	}

	if !flagsFor("indefeso")[autoFailReflexosFlag] {
		t.Error("indefeso deveria marcar a falha automática em Reflexos")
	}

	// As condições que a p394 define COMO indefeso herdam a falha junto — foi por
	// não compor que elas ficaram sem efeito nenhum (ALE-115).
	for _, id := range []string{"paralisado", "inconsciente", "petrificado"} {
		if !flagsFor(id)[autoFailReflexosFlag] {
			t.Errorf("%s fica indefeso pelo livro, então falha em Reflexos", id)
		}
	}

	// O desprevenido sozinho NÃO falha automaticamente — ele sofre −5, e é o
	// "mas" do indefeso que troca a penalidade pela falha.
	if flagsFor("desprevenido")[autoFailReflexosFlag] {
		t.Error("desprevenido sofre −5 em Reflexos, não falha automática")
	}
}
