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
//     de comparar. O dado é 1d10, e o livro é explícito: "independentemente do
//     resultado do teste de ataque" (p238), o que inclui o 20 natural;
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
	// TargetObjectInMotion não está na Tabela 5-3 — ela é a PROSA da mesma
	// página: "se o objeto estiver em movimento, recebe +5 na Defesa" (p239).
	// Mora aqui porque a FORMA é a da segunda metade da tabela ("o alvo
	// está... modificador na Defesa"), idêntica à da cobertura leve, e um
	// caso especial no código do objeto poria o mesmo mecanismo em dois
	// lugares — com só um deles aparecendo na decomposição que a mesa lê.
	TargetObjectInMotion SpecialSituation = "object-in-motion"
	// AttackerSwitchesTheDamageType é a ESCOLHA de quem ataca, da p236: usar a
	// arma contra a natureza dela — o fio para derrubar, ou o punho para matar.
	//
	// Ela mora aqui pelo mesmo motivo do objeto em movimento: o MECANISMO é o da
	// Tabela 5-3 — desloca o teste de ataque e a mesa precisa ler por quê. A
	// regra é SIMÉTRICA, e por isso é UMA linha e não duas: o livro gasta uma
	// frase inteira dizendo que vale nos dois sentidos, e duas linhas fariam
	// alguém aplicar só a que lembrou.
	//
	// A TROCA DO TIPO não é mecanismo desta tabela e não entrou nela: ela é lida
	// no `ResolveAttackUnder`, ao lado do `card.NonLethal` que já estava lá. Um
	// quinto campo no `situationRule` serviria a uma linha só.
	AttackerSwitchesTheDamageType SpecialSituation = "switched-damage-type"
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
	// A CAMUFLAGEM ROLA 1d10, e não um percentual: "ao fazer um ataque, o
	// atacante rola 1d10 junto com o d20 do teste de ataque; se o resultado
	// desse d10 for 1 ou 2, o ataque erra, independentemente do resultado do
	// teste de ataque" (p238). A total é "1 a 5 no d10" (p239).
	//
	// A Tabela 5-3 escreve "20%" e "50%", e é a PROSA que diz o dado. Escrever
	// o percentual e rolar d100 dá a mesma estatística e o dado errado na mesa
	// — e a mesa rola o dado de verdade.
	TargetUnderLightConcealment: {Label: "camuflagem leve", MissUpTo: 2},
	TargetUnderTotalConcealment: {Label: "camuflagem total", MissUpTo: 5},
	// "Se o objeto estiver em movimento, recebe +5 na Defesa" (p239). Só
	// objeto: criatura em movimento não ganha nada do livro — quem corre
	// gasta ação, e é isso que a p238 cobra dela.
	TargetObjectInMotion: {Label: "objeto em movimento", Defense: 5},
	// "Você pode usar uma arma para causar dano não letal [...], mas sofre uma
	// penalidade de –5 no teste de ataque. [...] Você pode usar esses ataques e
	// armas para causar dano letal, mas sofre a mesma penalidade" (p236).
	AttackerSwitchesTheDamageType: {Label: "trocando o tipo de dano", Attack: -5},
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
