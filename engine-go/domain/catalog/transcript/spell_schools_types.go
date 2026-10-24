package transcript

type SpellSchool struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	BookPage     int    `json:"bookPage"`
	Abbreviation string `json:"abrev,omitempty"`
	UID          string `json:"uid"`
}
