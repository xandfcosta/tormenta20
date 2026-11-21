package book

import (
	"encoding/json"
	"strconv"
	"sync"

	"t20engine/domain/engine"
)

// AS REGRAS DE ATIVAR UM PODER: se ele pode ser usado agora, qual limite o
// prende, e quanto custa entrar numa postura de degraus.
//
// Elas moravam na CENA da ficha, e o `activations.go` ao lado dizia por escrito
// que esta era "a metade que a cena decide". Deixou de ser: quem decide usar um
// poder é o caso de uso (`character.Plays`), e a cena continua lendo as mesmas
// funções para desenhar o botão desligado (ALE-351). Duas cópias — uma para
// decidir, outra para desenhar — dariam um botão aceso sobre um gesto recusado.
//
// # O que é COBRADO e o que é só crachá
//
// "1/cena" e "1/dia" são cobrados — eles têm contador no banco. Um "3/dia" e um
// "1/rodada" saem como crachá e nada mais, e isso é decisão registrada, não
// esquecimento: a mesa conta rodadas, a ficha não.
//
// O REGISTRO em si mora no `activations.go`, ao lado: ler o catálogo é uma
// coisa, decidir a partir dele é outra, e por isso são dois arquivos.

// ── o LIMITE de usos ─────────────────────────────────────────────────────────

// ChargedScope é "scene", "day" ou "" — e "" quer dizer que o limite existe
// no livro e a ficha NÃO o cobra.
//
// A DECISÃO de cobrar mora no `engine.UsageLimit`, junto com a janela: aqui só
// se traduz para a string que a coluna de contadores usa. Enquanto as duas
// coisas eram um `switch` só, "a ficha não conta rodadas" era uma ausência no
// meio de um `case` — e ausência não se lê (ALE-365).
func ChargedScope(spec Activation) string {
	limit, err := engine.ParseUsageLimit(unquoted(spec.Uses))
	if err != nil || !limit.ChargedBySheet() {
		return ""
	}
	return string(limit.Window)
}

// unquoted tira as aspas do JSON cru do `uses`. Ele é cru porque o catálogo
// escreve naturezas diferentes ali — ver o `activations.go`.
func unquoted(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return ""
	}
	return text
}

// CostIsVariable diz que o custo é NEGOCIADO com a mesa, e não um número.
func CostIsVariable(spec Activation) bool {
	return ActivationPm(spec) < 0
}

// ActivationPm é o custo em PM, ou -1 quando ele é variável.
//
// Menos um e não zero: zero é um custo LEGÍTIMO (a maioria das passivas), e
// confundir os dois é o que faria um poder de graça ser tratado como negociado
// com a mesa.
func ActivationPm(spec Activation) int {
	var number int
	if json.Unmarshal(spec.PmCost, &number) == nil {
		return number
	}
	return -1
}

// ── a DECISÃO de usar ────────────────────────────────────────────────────────

// UseDecision responde se o poder pode ser usado AGORA, e por que não.
//
// A ORDEM das recusas importa: a razão mostrada é a PRIMEIRA que barra, então
// "requer Fúria" aparece antes de "PM insuficiente" num poder que precisa das
// duas coisas — e é a que a pessoa pode resolver primeiro.
func UseDecision(spec Activation, context UseContext) (bool, string) {
	if CostIsVariable(spec) {
		return false, "custo variável"
	}
	if context.OnAlly && !spec.AllyTarget {
		return false, "não tem um aliado como alvo"
	}
	if spec.RequiresFlag != "" && !context.Flags[spec.RequiresFlag] {
		return false, "requer " + spec.RequiresFlag
	}
	switch ChargedScope(spec) {
	case "scene":
		if context.UsedThisScene >= 1 {
			return false, "limite por cena atingido"
		}
	case "day":
		if context.UsedToday >= 1 {
			return false, "limite por dia atingido"
		}
	}
	if UsePrice(spec, context.Cost, Use{OnAlly: context.OnAlly}) > context.CurrentPM {
		return false, "PM insuficiente"
	}
	return true, ""
}

// UseContext é o que a decisão precisa saber da ficha AGORA.
type UseContext struct {
	CurrentPM int
	// Cost são as regras do custo de habilidade da ficha (o Alquebrado, p394; a metade do Artista Completo, p45):
	// a decisão confere o PM contra o preço que se cobra, e não contra o do catálogo.
	Cost engine.AbilityCostRules
	// OnAlly diz que o alvo deste uso é um aliado: o preço leva o corte da Caridade (p83), e a habilidade que não
	// pode ter um aliado como alvo recusa.
	OnAlly        bool
	UsedThisScene int
	UsedToday     int
	Flags         map[string]bool
}

// ── a POSTURA de degraus ─────────────────────────────────────────────────────

// LevelSteps são os degraus EXTRAS que o nível na classe concede.
//
// O nível é o da CLASSE e não o do personagem (p40): um bárbaro 5/ladino 5 tem
// a Fúria de um bárbaro de nível 5, e não a de um personagem de nível 10.
func LevelSteps(scale ActivationScale, classLevel int) int {
	return engine.StepsAt(scale.FirstStepLevel, scale.StepEveryLevels, classLevel)
}

// MaxSteps é quantos degraus o poder deixa pagar.
//
// Dois modos. A escala que SOBE POR NÍVEL (a Inspiração, a Marca da Presa)
// abre degraus nos níveis dela. A que não sobe é o CUSTO VARIÁVEL, e o teto é
// o da p224 — o nível na classe que fornece a habilidade, ou o do personagem —
// apertado pelo que o verbete escreve (o atributo, um número). "Você sempre
// pode usar a habilidade em seu custo mínimo": o teto nunca fica abaixo dele.
//
//	MaxSteps(ActivationScale{BasePm: 1, StepPm: 1, CapAttribute: "charisma"}, 4, map[string]int{"charisma": 3}) // 2: até 3 PM
func MaxSteps(scale ActivationScale, classLevel int, attributes map[string]int) int {
	if scale.FirstStepLevel > 0 {
		return LevelSteps(scale, classLevel)
	}
	capPM := classLevel
	if scale.CapAttribute != "" {
		capPM = min(capPM, attributes[scale.CapAttribute])
	}
	if scale.CapPm > 0 {
		capPM = min(capPM, scale.CapPm)
	}
	if scale.StepPm <= 0 {
		return 0
	}
	return max(0, (capPM-scale.BasePm)/scale.StepPm)
}

// ScaledCost é o que entrar na postura, ou usar o poder, custa com os degraus
// escolhidos.
func ScaledCost(spec Activation, steps int) int {
	if spec.Scaling == nil {
		return ActivationPm(spec)
	}
	return spec.Scaling.BasePm + steps*spec.Scaling.StepPm
}

// ScaledDecision responde se dá para pagar esses degraus: dentro do teto do
// nível, e com PM para o PREÇO DO USO (ver `UsePrice`).
func ScaledDecision(spec Activation, use Use, max, currentPM int, rules engine.AbilityCostRules) (bool, string) {
	if use.Steps < 0 || use.Steps > max {
		return false, "o teto permite até " + writtenDegrees(max)
	}
	if UsePrice(spec, rules, use) > currentPM {
		return false, "PM insuficiente"
	}
	return true, ""
}

// ScaledGrantModifiers são os modificadores que a concessão grava com os
// degraus pagos.
//
// O degrau que TROCA o dado (`StepDice`) põe a linha da tabela no lugar do dado
// de base: a Marca da Presa com dois degraus é 1d12, e não 1d4 com dois dados a
// mais (p50).
//
//	ScaledGrantModifiers(*ActivationOf("class.cacador.marca-da-presa-1d4", ""), 1) // dano 1d8
func ScaledGrantModifiers(spec Activation, steps int) []engine.Modifier {
	if spec.Grant == nil {
		return nil
	}
	out := make([]engine.Modifier, 0, len(spec.Grant.Modifiers))
	for _, m := range spec.Grant.Modifiers {
		if steps > 0 && m.Dice != "" && spec.Scaling != nil && steps <= len(spec.Scaling.StepDice) {
			m.Dice = spec.Scaling.StepDice[steps-1]
		}
		out = append(out, m)
	}
	return out
}

// GrantModifiersWithRiders é a concessão com os degraus pagos e o que os
// ACRÉSCIMOS que a ficha tem somam a ela (p51, ALE-509): o Espreitar põe +1
// em perícia e mais 1 por degrau, o Ponto Fraco +2 na margem. `doubled` é a
// caixa do Inimigo — a presa é do tipo escolhido —, e dobra os dados e os
// números de tudo; ele só vale quando um acréscimo dobra, e quem recusa a
// caixa sem o poder é o caso de uso.
//
//	GrantModifiersWithRiders(marca, 1, []Activation{espreitar}, false) // dano 1d8, perícia +2
func GrantModifiersWithRiders(spec Activation, steps int, riders []Activation, doubled bool) []engine.Modifier {
	out := ScaledGrantModifiers(spec, steps)
	for _, r := range riders {
		if r.Rider == nil || r.Rider.On != spec.ID {
			continue
		}
		for _, m := range r.Rider.Modifiers {
			m.Amount += r.Rider.PerStep * steps
			out = append(out, m)
		}
	}
	if !doubled {
		return out
	}
	for i := range out {
		out[i].Amount *= 2
		if out[i].Dice != "" {
			out[i].Dice = engine.MultiplyDice(out[i].Dice, 2)
		}
	}
	return out
}

// ActivationRidden é o gesto como os acréscimos da ficha o deixam no que NÃO é número: o que cada degrau
// compra, e os avisos da linha dele, com o nome de quem avisa. É o par do `ActivationHeld`, que faz o mesmo
// pelos itens empunhados — o alcance não é número que o motor guarde, e mora no texto.
//
//	cura, avisos := book.ActivationRidden(cura, []Activation{compaixao})
//	cura.Scaling.StepLabel // "+2d6+1 PV curados"
//	avisos                 // ["alcance curto e 2d6+1 PV por PM (Virtude Paladinesca: Compaixão)"]
func ActivationRidden(spec Activation, riders []Activation) (Activation, []string) {
	notes := []string{}
	for _, r := range riders {
		if r.Rider == nil || r.Rider.On != spec.ID {
			continue
		}
		// O degrau é COPIADO antes de trocar o rótulo: o `Scaling` é ponteiro para o registro que toda ficha
		// lê, e escrever nele daria a Compaixão a todo paladino da casa.
		if r.Rider.StepLabel != "" && spec.Scaling != nil {
			scale := *spec.Scaling
			scale.StepLabel = r.Rider.StepLabel
			spec.Scaling = &scale
		}
		if r.Rider.Note != "" {
			notes = append(notes, r.Rider.Note+" ("+r.Name+")")
		}
	}
	return spec, notes
}

// RidersDouble diz se algum dos acréscimos dobra a concessão (o Inimigo).
func RidersDouble(spec Activation, riders []Activation) bool {
	for _, r := range riders {
		if r.Rider != nil && r.Rider.On == spec.ID && r.Rider.Doubles {
			return true
		}
	}
	return false
}

// O BÔNUS CUMULATIVO DE CENA (p42).
//
// A regra é de uma linha e mora aqui por isso: ela é do LIVRO — quanto cada
// gatilho dá e onde ele para —, e quem a chama é o caso de uso, que sabe o
// valor de agora e o nível. Nada aqui lê banco nem escreve nada.

// cumulativeTriggers é o vocabulário FECHADO dos gatilhos.
//
// Fechado pela mesma razão das cinco manobras: o casamento é por TEXTO, e um
// `on` digitado errado viraria um poder que nunca acumula — calado, como todo
// defeito de casamento por texto. Quem o cobra é o
// `TestEveryCumulativeBonusDeclaresATriggerTheEngineKnows`.
var cumulativeTriggers = map[string]bool{
	// "quando faz um acerto crítico ou reduz um inimigo a 0 PV" (p42).
	"criticalOrDrop": true,
	// "sempre que você acertar um ataque corpo a corpo em um inimigo" (p45).
	"meleeHit": true,
	// "sempre que um inimigo atacá-lo" (p54) — e o livro NÃO pede acerto: o
	// cavaleiro ganha o ponto pelo golpe que erra também.
	"attacked": true,
	// "quando um inimigo erra um ataque corpo a corpo contra você" (p76): é do lado de
	// quem APANHA, e quem diz que errou é o mestre.
	"meleeMissed": true,
}

// CumulativeTriggerIsKnown diz se o motor sabe disparar este gatilho.
func CumulativeTriggerIsKnown(on string) bool { return cumulativeTriggers[on] }

// CumulativeCap é o TETO do bônus — *"limitado pelo seu nível"* (p42).
//
// Devolve zero quando o teto é de uma espécie que o motor não conhece, e zero
// é o lado seguro: um teto desconhecido vira "não acumula", em vez de virar
// "acumula para sempre". É o oposto do `default` do `evalModifierScale`, que
// engole o `per` desconhecido e some com o bônus — aqui o silêncio também
// custa, mas custa um poder inerte e não um número inventado.
func CumulativeCap(spec CumulativeBonus, level int) int {
	if spec.CapPer == "level" {
		return level
	}
	return 0
}

// CumulativeNext é quanto o bônus vale DEPOIS de um gatilho, com o teto já
// aplicado.
//
// @example CumulativeNext(spec, 5, 6) // 6, e o próximo gatilho devolve 6 também
func CumulativeNext(spec CumulativeBonus, current, level int) int {
	return min(current+spec.Amount, CumulativeCap(spec, level))
}

// FlagCumulatives são as ativações que ACUMULAM enquanto esta flag está acesa.
//
// Irmã da `FlagGrants`, e separada dela porque as duas respondem perguntas
// diferentes: aquela diz o que a postura concede ao ENTRAR, esta diz o que ela
// faz crescer DEPOIS. Quem precisa das duas juntas é só o encerramento, que
// apaga os efeitos das duas famílias.
func FlagCumulatives(flag string) []Activation {
	outside := []Activation{}
	for _, spec := range Activations() {
		if spec.RequiresFlag == flag && spec.Cumulative != nil {
			outside = append(outside, spec)
		}
	}
	return outside
}

// writtenDegrees concorda o número com a palavra: a recusa é frase que uma
// pessoa lê no meio de uma sessão, e "até 1 degraus" é tropeço na leitura.
func writtenDegrees(howMany int) string {
	if howMany == 1 {
		return "1 degrau"
	}
	return strconv.Itoa(howMany) + " degraus"
}

// AmountFor é quanto a reserva dá a uma ficha de `level` com o atributo da concessão em
// `attribute`: nível + atributo (a Armadura da Honra, p53), ou só o atributo, mínimo 1,
// quando o verbete diz (o Estandarte, p54).
//
//	grant.AmountFor(14, 3) // 3 no Estandarte, 17 na Armadura da Honra
func (g PowerGrant) AmountFor(level, attribute int) int {
	if g.AttributeOnly {
		return max(1, attribute)
	}
	return level + attribute
}

// BannerPowerID é o Estandarte (p54): o cavaleiro recebe no início da cena, e os aliados
// que veem o estandarte recebem pelo mestre, na faixa da mesa.
const BannerPowerID = "class.cavaleiro.estandarte"

// SceneStartGrants são as ativações que concedem uma reserva NO INÍCIO DA CENA: de PV
// (a Armadura da Honra, p53) ou de PM (o Estandarte, p54).
//
// Elas são PASSIVAS com concessão: não há gesto que as dispare, e o que as
// separa das outras concessões é justamente isso — a da postura espera a flag
// acender, a do poder instantâneo espera o uso, e esta espera a cena começar.
func SceneStartGrants() []Activation {
	outside := []Activation{}
	for _, spec := range Activations() {
		if spec.Kind != "passive" || spec.Grant == nil || (spec.Grant.Kind != "temp-hp" && spec.Grant.Kind != "temp-mp") {
			continue
		}
		outside = append(outside, spec)
	}
	return outside
}

// activationClass é a classe da habilidade, quando ela é poder de classe, e vazia quando não é: é por ela que o
// fator de uma classe a alcança (a metade do Artista Completo nas de bardo, p45).
//
// Pelo id do poder ou, quando o id da ativação é outro, pelo NOME — a ligação que a ficha faz entre a ativação e
// o dono. Quinze ativações de classe só se ligam pelo nome (a Coragem, a Libertação…).
func activationClass(spec Activation) string {
	if class := ClassOfPower(spec.ID); class != "" {
		return class
	}
	return classByPowerName()[spec.Name]
}

// classByPowerName é a classe de cada nome de poder de classe. Um nome que duas classes usam fica VAZIO: escolher
// uma delas dependeria da ordem do mapa, e a resposta mudaria de uma execução para a outra.
var classByPowerName = sync.OnceValue(func() map[string]string {
	byName := map[string]string{}
	ambiguous := map[string]bool{}
	for _, power := range ClassPowers() {
		if class, seen := byName[power.Name]; seen && class != power.ClassName {
			ambiguous[power.Name] = true
		}
		byName[power.Name] = power.ClassName
	}
	for name := range ambiguous {
		byName[name] = ""
	}
	return byName
})

// ClassOfPower é a classe do poder de classe com esse id, e vazia quando não é um.
//
//	book.ClassOfPower(engine.WildTierPowerID(engine.WildTierNormal)) // "Druida"
func ClassOfPower(id string) string {
	if power, found := ClassPowers()[id]; found {
		return power.ClassName
	}
	return ""
}
