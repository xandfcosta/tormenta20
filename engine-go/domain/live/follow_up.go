package live

// FollowUp é a manobra em seguida que um poder dá (o Lutador de Chão, p77; a Mão na Boca, p74): a manobra
// livre, o poder que a dá, a decisão de que ela depende (`On`, e `When` em frase) e o PM dela. Ver
// `engine.FollowUpManeuver`.
type FollowUp struct {
	PowerID  string `json:"powerId"`
	Name     string `json:"name"`
	Maneuver string `json:"maneuver"`
	On       string `json:"on"`
	When     string `json:"when"`
	PM       int    `json:"pm,omitempty"`
}

// OpenFollowUp é a manobra em seguida que a decisão do mestre abriu na vez: de quem, e contra quem — "contra
// ela", a criatura agarrada ou atingida.
type OpenFollowUp struct {
	FollowUp
	EntryID  string `json:"entryId"`
	TargetID string `json:"targetId"`
}

// RulingOpens diz se a decisão do mestre abre o que depende de `on`: "hit" abre no acerto e no crítico,
// "success" no Sucesso da manobra. Os valores de `on` são os do verbete (`engine.OpensOnHit`,
// `engine.OpensOnSuccess`), e o regime não importa o motor — o `TestEveryFollowUpTriggerIsOpenedBySomeRuling`
// confere que os dois lados falam a mesma língua.
//
//	RulingOpens(RulingCritical, "hit") // true
func RulingOpens(ruling, on string) bool {
	switch on {
	case RulingHit:
		return ruling == RulingHit || ruling == RulingCritical
	case RulingSuccess:
		return ruling == RulingSuccess
	}
	return false
}
