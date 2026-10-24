package transcript

// Origins são as origens achatadas em benefícios, que a ficha aplica. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var Origins = concat(
	originsPart01,
	originsPart02,
	originsPart03,
	originsPart04,
	originsPart05,
)

type Origin struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Benefits    []OriginBenefit   `json:"benefits"`
	UniquePower OriginUniquePower `json:"poderUnico"`
	UID         string            `json:"uid"`
}

type OriginBenefit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description,omitempty"`
	Expertise   string `json:"expertise,omitempty"`
	UID         string `json:"uid"`
	PowerUID    string `json:"powerUid,omitempty"`
	PowerPick   string `json:"powerPick,omitempty"`
}

type OriginUniquePower struct {
	ID          string     `json:"id"`
	Unmodeled   string     `json:"unmodeled,omitempty"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	Description string     `json:"description"`
	Modifiers   []Modifier `json:"modifiers,omitempty"`
}
