package transcript

// GrantedPowers são os poderes concedidos que o motor sabe somar. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var GrantedPowers = concat(
	grantedPowersPart01,
	grantedPowersPart02,
)

type GrantedPower struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Gods      []string   `json:"deuses"`
	Effect    string     `json:"effect"`
	Kind      string     `json:"kind"`
	BookPage  int        `json:"bookPage"`
	UID       string     `json:"uid"`
	Modifiers []Modifier `json:"modifiers,omitempty"`
}
