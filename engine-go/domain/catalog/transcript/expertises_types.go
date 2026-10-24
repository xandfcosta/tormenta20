package transcript

type Expertise struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Attribute    string `json:"attribute"`
	TrainedOnly  bool   `json:"soTreinada"`
	ArmorPenalty bool   `json:"penalidadeDeArmadura"`
	BookPage     int    `json:"bookPage"`
	UID          string `json:"uid"`
}
