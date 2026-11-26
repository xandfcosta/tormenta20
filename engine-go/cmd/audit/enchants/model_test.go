package main

import (
	"testing"

	"t20engine/domain/catalog/transcript"
)

// O IMPULSO É REGRA APLICADA: o Anticriatura não tem modificador fixo, e o +4d8 contra o tipo escolhido entrou
// inteiro como gasto de 2 PM no golpe (`Boost`). Contar só os `Modifiers` o punha no balde "sem modificador nem
// motivo", e o auditor reprovava um encanto que o motor aplica.
func TestTheBoostCountsAsAppliedRule(t *testing.T) {
	boostOnly := transcript.Item{Name: "Anticriatura", Boost: &transcript.ItemBoost{PM: 2}}
	if !appliesSomething(boostOnly) {
		t.Error("o encanto só com impulso foi lido como sem regra aplicada")
	}
	if !appliesSomething(transcript.Item{Modifiers: []transcript.Modifier{{Amount: 1}}}) {
		t.Error("o encanto com modificador foi lido como sem regra aplicada")
	}
	if appliesSomething(transcript.Item{Name: "Conjuradora"}) {
		t.Error("o controle falhou: o encanto sem modificador nem impulso foi lido como aplicado")
	}
}
