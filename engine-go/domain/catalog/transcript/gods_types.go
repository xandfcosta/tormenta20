package transcript

type God struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Major           bool     `json:"major"`
	PaladinEligible bool     `json:"paladinoEligible"`
	DruidEligible   bool     `json:"druidaEligible"`
	Portfolio       string   `json:"portfolio"`
	Energy          string   `json:"energia"`
	Symbol          string   `json:"simbolo"`
	FavoredWeapon   *string  `json:"armaPreferida"`
	GrantedPowers   []string `json:"poderesConcedidos"`
	Devotees        []string `json:"devotos"`
	BookPage        int      `json:"bookPage"`
	UID             string   `json:"uid"`
}
