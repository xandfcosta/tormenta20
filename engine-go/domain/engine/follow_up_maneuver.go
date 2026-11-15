package engine

import "strings"

// A MANOBRA EM SEGUIDA (ALE-651, ALE-659).
//
// O Lutador de Chão (lutador, p77) e a Mão na Boca (ladino, p74) dão uma manobra LIVRE depois de outro gesto:
// a derrubada depois de agarrar, por 1 PM, e o agarrar depois de acertar o ataque furtivo. Ela abre pela
// DECISÃO do mestre sobre o gesto que vem antes — o Sucesso no agarrar, o acerto do golpe —, e o jogador decide
// depois se a faz, como o livro escreve ("quando agarra, pode gastar 1 PM", decisão do dono).
//
// Por que pela decisão e não por uma declaração antes de rolar: nada se rola antes de valer a condição de que
// depende, e quem diz se o agarrar venceu é o mestre — a manobra tem Sucesso e Falha para isso.
//
// O que abre, o que se faz, de que decisão depende e o preço moram no VERBETE (`ClassPower.FollowUp`), como o
// `ExtraStrike`.

// Os desfechos de que uma manobra em seguida depende: a decisão do mestre sobre o gesto anterior. Os valores são
// os da decisão gravada na mesa (`live.RulingOpens`).
const (
	OpensOnHit     = "hit"     // o acerto do golpe, crítico incluído
	OpensOnSuccess = "success" // o Sucesso da manobra
)

// ClassPowerFollowUp é a manobra em seguida que o poder dá — ver o arquivo. `After` é a manobra que vem antes,
// e vazio é o golpe; `On` é a decisão do mestre sobre ela que abre a manobra; `NeedsFlag` é o interruptor que
// o gesto precisa ter ligado quando é rolado (o Ataque Furtivo).
type ClassPowerFollowUp struct {
	Maneuver  string `json:"maneuver"`
	After     string `json:"after,omitempty"`
	On        string `json:"on"`
	PM        int    `json:"pm,omitempty"`
	NeedsFlag string `json:"needsFlag,omitempty"`
}

// FollowUpManeuver é a manobra em seguida que um poder da ficha dá. `Ready` diz que o interruptor que ela pede
// está ligado agora — sempre, quando ela não pede nenhum: é o que arma o gesto rolado agora.
type FollowUpManeuver struct {
	PowerID   string
	Name      string
	Maneuver  string
	After     string
	On        string
	PM        int
	NeedsFlag string
	Ready     bool
}

// FollowUpManeuvers são as manobras em seguida que os poderes da ficha dão, na ordem das classes, com o
// interruptor de cada uma conferido contra os ligados.
//
//	world.FollowUpManeuvers(lutador, active) // [{Lutador de Chão: derrubar depois do Sucesso no agarrar, 1 PM}]
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
				PowerID: power.ID, Name: power.Name, Maneuver: grant.Maneuver, After: grant.After, On: grant.On, PM: grant.PM,
				NeedsFlag: grant.NeedsFlag, Ready: grant.NeedsFlag == "" || active[FlagGroupID(grant.NeedsFlag)],
			})
		}
	}
	return out
}

// Condition é de que decisão a manobra depende, em frase: "depois do Sucesso no agarrar", "depois de acertar com
// o ataque furtivo". A linha das Ações a diz, e a recusa também.
func (f FollowUpManeuver) Condition() string {
	if f.On == OpensOnSuccess {
		return "depois do Sucesso no " + f.After
	}
	condition := "depois de acertar"
	if f.NeedsFlag != "" {
		condition += " com o " + strings.ReplaceAll(f.NeedsFlag, "-", " ")
	}
	return condition
}
