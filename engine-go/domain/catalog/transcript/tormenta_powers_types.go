package transcript

type TormentaPower struct {
	NaturalWeapon       *NaturalWeapon `json:"naturalWeapon,omitempty"`
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	RequiresOtherPowers int            `json:"requiresOtherPowers"`
	BookPage            int            `json:"bookPage"`
	UID                 string         `json:"uid"`
	RequiresPower       string         `json:"requiresPower,omitempty"`
}
