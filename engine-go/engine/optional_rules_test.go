package engine

import (
	"path/filepath"
	"testing"
)

// As regras opcionais (ALE-221), do lado do motor.
//
// p141, no quadro "Carga: Bastidores": "O mestre pode ignorar essa regra, desde
// que os jogadores não abusem. Nada de sair por aí com 50 essências de mana!"
//
// A frase decide as duas metades que estes testes separam: a mesa pode desligar
// a CONSEQUÊNCIA, e continua precisando ver o NÚMERO — não dá para vigiar abuso
// sem enxergar quantos espaços estão ocupados.

// O que a mesa desliga é a consequência, e ela desaparece da ficha INTEIRA e não
// só do rótulo da mochila. Roda por `ComputeSheetV2` pela mesma razão do teste
// irmão da sobrecarga: o defeito que ele mira não é a conta, é ela chegar (ou
// deixar de chegar) ao deslocamento e às perícias.
func TestLoadTurnedOffPenalizesNeitherDisplacementNorExpertises(t *testing.T) {
	catalogs := primeFromDump(t, filepath.Clean(filepath.Join(mustWd(t), "..", "parity")))
	pericias := []CharacterExpertise{{Name: "Furtividade", Attribute: "dexterity"}}
	// Força 0 ⇒ limite 10; onze espaços ultrapassam com folga.
	sobrecarregado := func(ignorada bool) ComputedSheetV2 {
		ch := Character{
			Level: 1, Displacement: 9, Expertises: pericias,
			Items:        []CharacterItem{{Name: "Barril", Quantity: 1, Slots: 11}},
			IgnoredRules: IgnoredRules{Carga: ignorada},
		}
		return catalogs.ComputeSheetV2(ch, map[string]bool{})
	}

	comRegra, semRegra := sobrecarregado(false), sobrecarregado(true)
	if comRegra.Displacement.Total != 6 {
		t.Fatalf("com a regra ligada o deslocamento é %d, want 6 — o controle já estava errado", comRegra.Displacement.Total)
	}
	if semRegra.Displacement.Total != 9 {
		t.Errorf("mesa sem a regra de carga: deslocamento %d, want 9 (a penalidade não existe)", semRegra.Displacement.Total)
	}
	if got := periciaTotal(t, semRegra, "Furtividade") - periciaTotal(t, comRegra, "Furtividade"); got != 5 {
		t.Errorf("mesa sem a regra: Furtividade %+d contra a mesa com a regra, want +5 (os −5 da p141 não são aplicados)", got)
	}
}

// A metade que sobrevive: os espaços continuam contados. O livro condiciona o
// desligamento a "desde que os jogadores não abusem", e quem vigia abuso precisa
// do número na tela.
func TestLoadTurnedOffStillCountsTheSlots(t *testing.T) {
	ch := Character{
		Tibar:        2000,
		Items:        []CharacterItem{{Name: "Barril", Quantity: 1, Slots: 11}},
		IgnoredRules: IgnoredRules{Carga: true},
	}
	got := loadBreakdownOf(ch, 10)

	if got.Used != 13 {
		t.Errorf("carga usada %v, want 13 (11 do barril + 2 milheiros) — desligar a regra não desliga a CONTA", got.Used)
	}
	if got.Limit != 10 || got.Max != 20 {
		t.Errorf("limite/teto = %d/%d, want 10/20 — a mochila continua sabendo dizer onde ficaria a linha", got.Limit, got.Max)
	}
	if got.Enforced {
		t.Error("enforced=true numa mesa que desligou a carga")
	}
	// `Overloaded` é a CONDIÇÃO do livro, e ela não existe numa mesa que não usa a
	// regra. Ficar `true` aqui faria toda tela que lê só este campo pintar um
	// alerta que a mesa dispensou.
	if got.Overloaded || got.OverMax {
		t.Errorf("sobrecarregado=%v acimaDoTeto=%v, want false/false — a condição não existe sem a regra", got.Overloaded, got.OverMax)
	}
	if got.ArmorPenalty != 0 || got.DisplacementPenalty != 0 {
		t.Errorf("penalidades %d/%dm, want 0/0", got.ArmorPenalty, got.DisplacementPenalty)
	}
}

// O valor zero da struct significa TUDO EM VIGOR, e isso é a proteção do
// desenho: um `Character{}` literal — num teste, num fixture, no oráculo —
// calcula com as regras do livro sem ninguém lembrar de preencher nada.
func TestZeroMeansEveryRuleInForce(t *testing.T) {
	got := loadBreakdownOf(Character{Items: []CharacterItem{{Quantity: 1, Slots: 11}}}, 10)

	if !got.Enforced {
		t.Error("Character{} sem regras declaradas calculou com a carga DESLIGADA — o padrão vazou")
	}
	if !got.Overloaded || got.DisplacementPenalty != -3 {
		t.Errorf("Character{} com 11 espaços num limite de 10: sobrecarregado=%v, deslocamento %dm — want true/−3", got.Overloaded, got.DisplacementPenalty)
	}
}

// Identificador que este binário não conhece é IGNORADO na leitura, e é o lado
// seguro: a lista vive no banco e sobrevive a um rollback do servidor, então uma
// regra do futuro não pode derrubar a ficha nem — pior — afrouxar uma regra por
// acidente. Ela simplesmente continua em vigor.
func TestAnUnknownRuleOnReadTurnsNothingOff(t *testing.T) {
	got := IgnoredRulesFrom([]string{"carga", "custo-de-vida-do-futuro"})

	if !got.Carga {
		t.Error("a regra conhecida não foi lida por causa da desconhecida ao lado")
	}
	if IgnoredRulesFrom([]string{"custo-de-vida-do-futuro"}) != (IgnoredRules{}) {
		t.Error("um identificador desconhecido desligou alguma coisa")
	}
	if IsKnownRule("custo-de-vida-do-futuro") {
		t.Error("IsKnownRule aceitou um identificador que o motor não implementa")
	}
}
