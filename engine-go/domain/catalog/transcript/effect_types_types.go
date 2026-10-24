package transcript

type EffectType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	BookPage    int    `json:"bookPage"`
	UID         string `json:"uid"`
}
