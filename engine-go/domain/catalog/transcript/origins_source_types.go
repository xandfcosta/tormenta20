package transcript

type OriginSource struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Expertises          []string `json:"pericias"`
	Powers              []string `json:"poderes"`
	UniquePower         string   `json:"poderUnico"`
	StartingItems       []string `json:"itensIniciais"`
	BookPage            int      `json:"bookPage"`
	UID                 string   `json:"uid"`
	RequiredPower       string   `json:"poderObrigatorio,omitempty"`
	GMDriven            bool     `json:"gmDriven,omitempty"`
	PowerChoiceCategory string   `json:"poderChoiceCategory,omitempty"`
}
