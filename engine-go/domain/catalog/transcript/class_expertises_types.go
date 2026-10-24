package transcript

type ClassExpertiseRule struct {
	Fixed       []string           `json:"fixed"`
	ChooseCount int                `json:"chooseCount"`
	ChoosePool  []string           `json:"choosePool"`
	EitherOr    *ExpertiseEitherOr `json:"eitherOr,omitempty"`
}

type ExpertiseEitherOr struct {
	Options []string `json:"options"`
}
