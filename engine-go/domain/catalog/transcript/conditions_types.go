package transcript

type Condition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	UpgradesTo  string   `json:"upgradesTo,omitempty"`
	BookPage    int      `json:"bookPage"`
	UID         string   `json:"uid"`
}
