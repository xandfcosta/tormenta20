package transcript

// DivinePowers são os poderes concedidos, um por par (deus, poder). A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var DivinePowers = concat(
	divinePowersPart01,
	divinePowersPart02,
)

type DivinePower struct {
	GodID       string  `json:"deusId"`
	Name        string  `json:"name"`
	Action      string  `json:"action"`
	PMCost      PMCost  `json:"pmCost"`
	Uses        *string `json:"uses"`
	BookPage    int     `json:"bookPage"`
	Description string  `json:"description"`
	UID         string  `json:"uid"`
}
