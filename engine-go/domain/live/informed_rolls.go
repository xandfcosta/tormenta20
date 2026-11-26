package live

// AS ROLAGENS INFORMADAS (ver GLOSSARY, **rolagem informada**): o teste de perícia, o disparo do encanto, a magia
// conjurada e o perigo disparado. A mesa as guarda para INFORMAR, e não há o que confirmar nem cancelar — é isso que as separa do
// `PendingAttack`, que tem os verbos do mestre.
//
// Faixa nova entra aqui: o campo em `InformedRolls`, a decisão em `outOfScene`, o `recordIn` do tipo, ao lado
// dele, e, se ela é só do mestre, a linha no `redactForViewer` que a tira do jogador. A faixa da mesa que a desenha é do `serve/web/table`.

// InformedRolls é a ÚLTIMA rolagem de cada tipo — no máximo uma, e a nova substitui a anterior do MESMO tipo.
//
// Elas moram no estado da sessão e não numa notícia que passa porque a mesa REDESENHA: quem chega atrasado, ou
// quem troca de superfície, tem de ver o que acabou de ser rolado. Uma notícia sem estado deixaria a faixa sumir
// no primeiro quadro seguinte.
//
// A struct é EMBUTIDA no `SessionRuntimeState` sem etiqueta, e isso é o formato gravado: os campos sobem para o
// JSON da sessão com os nomes de sempre, e uma sessão gravada antes continua lendo as faixas dela.
type InformedRolls struct {
	LastSkillTest *SkillTestRoll `json:"lastSkillTest,omitempty"`
	LastBlast     *BlastRoll     `json:"lastBlast,omitempty"`
	LastSpellCast *SpellCastRoll `json:"lastSpellCast,omitempty"`
	// LastHazard é só do mestre: o `redactForViewer` o tira do jogador (ver `HazardRoll`).
	LastHazard *HazardRoll `json:"lastHazard,omitempty"`
}

// InformedRoll é uma rolagem que a mesa informa. O método é privado de propósito: só os tipos deste pacote sabem
// em que campo do estado se guardam, e um tipo de fora não teria onde morar.
type InformedRoll interface {
	recordIn(st *SessionRuntimeState) error
}

// RecordRoll guarda a rolagem, substituindo a anterior do mesmo tipo.
//
//	err := live.RecordRoll(st, live.SpellCastRoll{Who: "Arwen", Spell: "Bola de Fogo"})
func RecordRoll(st *SessionRuntimeState, roll InformedRoll) error {
	return roll.recordIn(st)
}

// outOfScene é o que atravessa para o jogador FORA de cena, onde ele não recebe a fila.
//
// O TESTE ATRAVESSA porque a maior parte dos testes de uma mesa acontece fora de cena — a Percepção que abre a
// cena vem antes de haver cena —, e apagá-lo faria a faixa aparecer só para o mestre, que é a metade da mesa que
// menos precisa dela. Medido: sem ele o jogador não via o próprio teste. A MAGIA atravessa pela mesma razão: a
// cura entre as cenas é conjurada fora delas. O DISPARO não: ele é contra uma linha da fila, e a fila não
// atravessa. O PERIGO não atravessa porque nem em cena ele chega ao jogador.
func (r InformedRolls) outOfScene() InformedRolls {
	return InformedRolls{LastSkillTest: r.LastSkillTest, LastSpellCast: r.LastSpellCast}
}
