package transcript

type GMTableBook struct {
	Ruin             []RuinRoll         `json:"ruina"`
	ChaseEvents      []ChaseEvent       `json:"chaseEvents"`
	RewardPunishment []RewardRoll       `json:"rewardCastigo"`
	RewardLabels     OrderedMap[string] `json:"rewardLabels"`
	PunishmentLabels OrderedMap[string] `json:"castigoLabels"`
	ChaosDeck        []ChaosCard        `json:"chaosDeck"`
}

type RuinRoll struct {
	RollMin int    `json:"rollMin"`
	RollMax int    `json:"rollMax"`
	Outcome string `json:"outcome"`
	Label   string `json:"label"`
}

type ChaseEvent struct {
	RollMin int     `json:"rollMin"`
	RollMax int     `json:"rollMax"`
	Kind    string  `json:"kind"`
	Test    *string `json:"test"`
	DC      *int    `json:"cd"`
	Example string  `json:"example"`
}

type RewardRoll struct {
	Roll       int    `json:"roll"`
	Reward     string `json:"reward"`
	Punishment string `json:"castigo"`
}

type ChaosCard struct {
	RollMin int    `json:"rollMin"`
	RollMax int    `json:"rollMax"`
	Card    string `json:"card"`
	Effect  string `json:"effect"`
}
