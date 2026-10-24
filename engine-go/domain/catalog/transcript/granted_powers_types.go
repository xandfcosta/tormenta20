package transcript

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
