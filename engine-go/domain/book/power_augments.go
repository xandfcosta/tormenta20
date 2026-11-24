package book

import (
	"slices"
	"strings"
)

// OS PODERES QUE SÃO APRIMORAMENTO (p131): "Estes poderes acrescentam melhorias
// às magias conhecidas pelo conjurador. Eles seguem todas as regras para
// aprimoramentos". Por isso eles se escolhem no diálogo de conjurar e não têm
// gesto próprio: pagos à parte, o custo escaparia do limite de PM da magia.
//
// Seis poderes de CLASSE se pagam do mesmo jeito — "quando lança uma magia, você
// pode pagar" (p38, p58, p62) —, e entram aqui pelo mesmo campo, cada um uma vez
// por conjuração (decisão do dono: o livro não os chama de aprimoramento).
const (
	QuickenedSpellPower = "magia-acelerada"
	AmplifiedSpellPower = "magia-ampliada"
	DiscreetSpellPower  = "magia-discreta"
	// PiercingSpellPower é a Magia Pungente do arcanista (p38).
	PiercingSpellPower = "class.arcanista.magia-pungente"
	// CombatPrayerPower é a Prece de Combate do clérigo (p58).
	CombatPrayerPower = "class.clerigo.prece-de-combate"
	// PrairieFreedomPower é a Liberdade da Pradaria do druida (p62).
	PrairieFreedomPower = "class.druida.liberdade-da-pradaria"
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
	for _, power := range ClassPowers() {
		if power.SpellAugment != nil {
			augments = append(augments, PowerAugment{PowerID: power.ID, Name: power.Name, Augment: *power.SpellAugment})
		}
	}
	slices.SortFunc(augments, func(a, b PowerAugment) int { return strings.Compare(a.Name, b.Name) })
	return augments
}

// PowerAugmentOf acha o poder que é aprimoramento pelo id.
func PowerAugmentOf(powerID string) (PowerAugment, bool) {
	if power, found := GeneralPowers()[powerID]; found && power.SpellAugment != nil {
		return PowerAugment{PowerID: power.ID, Name: power.Name, Augment: *power.SpellAugment}, true
	}
	if power, found := ClassPowers()[powerID]; found && power.SpellAugment != nil {
		return PowerAugment{PowerID: power.ID, Name: power.Name, Augment: *power.SpellAugment}, true
	}
	return PowerAugment{}, false
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
	case PiercingSpellPower:
		// "aumentar em +2 a CD para resistir a ela" (p38): a magia sem teste de resistência não tem CD.
		if spell.SaveType == "" || spell.SaveType == "none" {
			return "só em magia com teste de resistência"
		}
	case CombatPrayerPower:
		// "com tempo de conjuração de uma ação padrão" (p58). A magia divina é a marca "apenas divinos".
		if spell.Execution != "padrao" || spell.ExecutionNote != "" {
			return "só em magia de ação padrão"
		}
	case PrairieFreedomPower:
		// "de toque para curto, de curto para médio ou de médio para longo" (p62).
		if !slices.Contains([]string{"toque", "curto", "medio"}, spell.Range) {
			return "só com alcance de toque, curto ou médio"
		}
	}
	return ""
}
