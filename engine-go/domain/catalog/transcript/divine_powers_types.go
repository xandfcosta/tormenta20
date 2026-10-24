package transcript

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
