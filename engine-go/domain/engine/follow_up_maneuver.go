package engine

// A MANOBRA EM SEGUIDA (ALE-651).
//
// O Lutador de Chão (lutador, p77) e a Mão na Boca (ladino, p74) dão uma manobra LIVRE depois de outro gesto:
// a derrubada depois de agarrar, por 1 PM, e o agarrar depois de acertar o ataque furtivo. O jogador DECLARA a
// ação completa antes de rolar (decisão do dono): o interruptor vai no gesto que a abre, e o ✓ do mestre a esse
// gesto abre a manobra na mesma vez.
//
// Por que não abrir pela VITÓRIA do agarrar: a manobra não tem decisão de vencedor — o mestre dá ✓ também na
// que falhou, e marca a condição na mão. A declaração deixa o mestre ler o pedido inteiro antes de responder.
//
// O que abre, o que se faz e o preço moram no VERBETE (`ClassPower.FollowUp`), como o `ExtraStrike`.

// ClassPowerFollowUp é a manobra em seguida que o poder dá — ver o arquivo. `After` é a manobra que a declara,
// e vazio é o golpe; `NeedsFlag` é o interruptor que o golpe precisa ter ligado (o Ataque Furtivo).
type ClassPowerFollowUp struct {
	Maneuver  string `json:"maneuver"`
	After     string `json:"after,omitempty"`
	PM        int    `json:"pm,omitempty"`
	NeedsFlag string `json:"needsFlag,omitempty"`
}

// FollowUpManeuver é a manobra em seguida que a ficha pode declarar. `Ready` diz que o interruptor que ela pede
// está ligado agora — sempre, quando ela não pede nenhum.
type FollowUpManeuver struct {
	PowerID   string
	Name      string
	Maneuver  string
	After     string
	PM        int
	NeedsFlag string
	Ready     bool
}

// FollowUpManeuvers são as manobras em seguida que os poderes da ficha dão, na ordem das classes, com o
// interruptor de cada uma conferido contra os ligados.
//
//	world.FollowUpManeuvers(lutador, active) // [{Lutador de Chão: derrubar depois de agarrar, 1 PM}]
func (r *Ruleset) FollowUpManeuvers(ch Character, active map[string]bool) []FollowUpManeuver {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	var out []FollowUpManeuver
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			grant := power.FollowUp
			if grant == nil {
				continue
			}
			out = append(out, FollowUpManeuver{
				PowerID: power.ID, Name: power.Name, Maneuver: grant.Maneuver, After: grant.After, PM: grant.PM,
				NeedsFlag: grant.NeedsFlag, Ready: grant.NeedsFlag == "" || active[FlagGroupID(grant.NeedsFlag)],
			})
		}
	}
	return out
}
