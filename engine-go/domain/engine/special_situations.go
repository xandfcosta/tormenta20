package engine

import "fmt"

// A TABELA 5-3, "Situações Especiais" (p239), no ataque.
//
// Ela tem duas metades — o que o ATACANTE está e o que o ALVO está — e as seis
// linhas que são CONDIÇÃO (caído, cego, desprevenido, ofuscado) não moram aqui:
// elas chegam pela ficha, pela `conditionModifierTable`, e já valiam. O que
// mora aqui é o que NÃO é condição de ninguém — a relação entre o atacante, o
// alvo e o terreno.
//
// Três mecanismos, e só o primeiro o motor já tinha:
//
//   - DESLOCAR um número, no ataque ou na Defesa;
//   - CHANCE DE FALHA, que age DEPOIS de o ataque acertar e desfaz o acerto.
//     Nenhuma outra regra do motor faz isso — as demais mexem no número antes
//     de comparar;
//   - PROIBIR o ataque, que não é Defesa alta: um 20 natural acerta sempre
//     (p221), e por trás de uma parede ele não pode.

// SpecialSituation é uma linha da Tabela 5-3 que vale AGORA, neste ataque.
//
// Ela é string e não booleano numa struct porque quem a produz é o tabuleiro,
// uma linha por vez, e uma struct de oito campos obrigaria todo chamador a
// saber das oito.
type SpecialSituation string

const (
	AttackerOnHigherGround      SpecialSituation = "higher-ground"
	AttackerFlanking            SpecialSituation = "flanking"
	AttackerInvisible           SpecialSituation = "invisible-attacker"
	TargetUnderLightCover       SpecialSituation = "light-cover"
	TargetUnderTotalCover       SpecialSituation = "total-cover"
	TargetUnderLightConcealment SpecialSituation = "light-concealment"
	TargetUnderTotalConcealment SpecialSituation = "total-concealment"
)

// specialSituationTable é a Tabela 5-3 transcrita, e ela é a FONTE do guarda de
// cobertura: quem acrescentar uma linha aqui sem sistema que a leia reprova.
//
// Em Go e não em JSON pela mesma razão que a `conditionModifierTable`: as
// linhas carregam MECANISMO e não só número, e um `missChance` num catálogo
// seria uma chave que só o motor entende. Quem a confere contra o livro é um
// auditor que lê este arquivo.
var specialSituationTable = map[SpecialSituation]situationRule{
	// "Em posição elevada: +2" no ataque.
	AttackerOnHigherGround: {Label: "posição elevada", Attack: 2},
	// "Flanqueando o alvo: +2 (apenas para corpo a corpo)".
	AttackerFlanking: {Label: "flanqueando", Attack: 2, MeleeOnly: true},
	// "Invisível: o alvo sofre −5 na Defesa". A linha é do ATACANTE e o efeito é
	// no ALVO — foi por isso que ela não cabia na tabela de condições.
	AttackerInvisible: {Label: "atacante invisível", Defense: -5},
	// "Sob cobertura leve: +5" na Defesa.
	TargetUnderLightCover: {Label: "cobertura leve", Defense: 5},
	// "Sob cobertura total: o alvo não pode ser atacado".
	TargetUnderTotalCover: {Label: "cobertura total", Forbids: true},
	// "Sob camuflagem leve: 20% de chance de falha"; "total: 50%".
	TargetUnderLightConcealment: {Label: "camuflagem leve", MissUpTo: 20},
	TargetUnderTotalConcealment: {Label: "camuflagem total", MissUpTo: 50},
}

// situationRule é uma linha da tabela. Zero em tudo que a linha não diz.
type situationRule struct {
	Label     string
	Attack    int
	Defense   int
	MissUpTo  int
	Forbids   bool
	MeleeOnly bool
}

// SpecialSituationsOfTheBook são as linhas transcritas, para quem precisa
// percorrê-las — a tela que as oferece e o guarda que as varre.
//
// @example for _, s := range engine.SpecialSituationsOfTheBook() { … }
func SpecialSituationsOfTheBook() []SpecialSituation {
	out := make([]SpecialSituation, 0, len(specialSituationTable))
	for s := range specialSituationTable {
		out = append(out, s)
	}
	return out
}

// SpecialSituationLabel é o nome que a mesa lê.
func SpecialSituationLabel(s SpecialSituation) string {
	return specialSituationTable[s].Label
}

// errTotalCover é a recusa da cobertura total. Erro e não "acertou zero":
// o gesto tem de PARAR antes de rolar nada, e a mesa tem de ler por quê.
var errTotalCover = fmt.Errorf("o alvo está sob cobertura total e não pode ser atacado (p239)")
