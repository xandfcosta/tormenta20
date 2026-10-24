package transcript

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
