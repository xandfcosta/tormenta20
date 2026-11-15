package live

// FollowUp é a manobra em seguida que um poder dá (o Lutador de Chão, p77; a Mão na Boca, p74): a manobra
// livre, o poder que a dá e o PM dela. Ver `engine.FollowUpManeuver`.
type FollowUp struct {
	PowerID  string `json:"powerId"`
	Name     string `json:"name"`
	Maneuver string `json:"maneuver"`
	PM       int    `json:"pm,omitempty"`
}

// OpenFollowUp é a manobra em seguida que o ✓ do mestre abriu na vez: de quem, e contra quem — "contra ela",
// a criatura agarrada ou atingida.
type OpenFollowUp struct {
	FollowUp
	EntryID  string `json:"entryId"`
	TargetID string `json:"targetId"`
}
