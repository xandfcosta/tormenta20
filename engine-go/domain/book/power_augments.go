package book

import (
	"regexp"
	"slices"
	"strings"
)

// OS PODERES QUE SÃO APRIMORAMENTO (p131): "Estes poderes acrescentam melhorias
// às magias conhecidas pelo conjurador. Eles seguem todas as regras para
// aprimoramentos". Por isso eles se escolhem no diálogo de conjurar e não têm
// gesto próprio: pagos à parte, o custo escaparia do limite de PM da magia.
const (
	QuickenedSpellPower = "magia-acelerada"
	AmplifiedSpellPower = "magia-ampliada"
	DiscreetSpellPower  = "magia-discreta"
)

// PowerAugment é um desses poderes, com o que ele custa ao conjurar.
type PowerAugment struct {
	PowerID string
	Name    string
	Augment SpellAugment
}

// PowerAugments são os poderes que são aprimoramento, em ordem de nome.
//
//	PowerAugments()[0] // {PowerID: "magia-acelerada", Augment: {PmCost: 4, Kind: "muda", …}}
func PowerAugments() []PowerAugment {
	augments := []PowerAugment{}
	for _, power := range GeneralPowers() {
		if power.SpellAugment != nil {
			augments = append(augments, PowerAugment{PowerID: power.ID, Name: power.Name, Augment: *power.SpellAugment})
		}
	}
	slices.SortFunc(augments, func(a, b PowerAugment) int { return strings.Compare(a.Name, b.Name) })
	return augments
}

// PowerAugmentOf acha o poder que é aprimoramento pelo id.
func PowerAugmentOf(powerID string) (PowerAugment, bool) {
	power, found := GeneralPowers()[powerID]
	if !found || power.SpellAugment == nil {
		return PowerAugment{}, false
	}
	return PowerAugment{PowerID: power.ID, Name: power.Name, Augment: *power.SpellAugment}, true
}

// PowerAugmentBar diz por que o poder não se aplica a esta magia, ou "" quando se
// aplica. O texto é o mesmo da linha trancada e da recusa do servidor.
//
//	PowerAugmentBar(QuickenedSpellPower, reaction) // "só em execução de movimento, padrão ou completa"
func PowerAugmentBar(powerID string, spell Spell) string {
	switch powerID {
	case QuickenedSpellPower:
		// "Você só pode aplicar este aprimoramento em magias com execução de
		// movimento, padrão ou completa" (p131).
		// A execução longa é "completa" no campo, que é o que ela cobra, e a nota diz o resto.
		if spell.ExecutionNote != "" || !slices.Contains([]string{"movimento", "padrao", "completa"}, spell.Execution) {
			return "só em execução de movimento, padrão ou completa"
		}
	case AmplifiedSpellPower:
		// "de curto para médio, de médio para longo", "ou dobra a área" (p131).
		// A palavra é a do alvo no livro: "Área: esfera", "Alvo ou Área: …".
		hasStep := spell.Range == "curto" || spell.Range == "medio"
		if !hasStep && !strings.Contains(strings.ToLower(spell.Targeting), "área") {
			return "só com alcance curto ou médio, ou com área"
		}
	}
	return ""
}

// changesExecutionText casa o início da frase, e não a palavra solta: "Como parte
// da execução: ataque corpo a corpo" não muda a execução de nada.
var changesExecutionText = regexp.MustCompile(`(?i)^(muda (a )?)?execução `)

// AugmentChangesExecution diz se o texto do aprimoramento muda a execução da magia,
// que é a característica que a Magia Acelerada também muda (p171).
func AugmentChangesExecution(description string) bool {
	return changesExecutionText.MatchString(description)
}
