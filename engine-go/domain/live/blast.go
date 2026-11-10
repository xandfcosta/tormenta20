package live

import "fmt"

// BlastRoll é o que um encanto dispara EM VEZ do golpe, rolado na ficha: a bola de fogo da Flamejante (p335).
//
// NÃO É UM PROVISÓRIO, como o `SkillTestRoll` não é: o mestre aplica o dano e a resistência do alvo com os
// gestos de PV que já existem, e a faixa só INFORMA — a rolagem, o tipo do dano e a CD de quem disparou.
// Fica no estado da sessão até a próxima, pela razão do teste: a mesa redesenha, e quem chega atrasado tem de
// ver a bola de fogo que acabou de cair.
type BlastRoll struct {
	// Who é o nome de quem disparou, pronto, e Weapon a arma do encanto.
	Who    string `json:"who"`
	Weapon string `json:"weapon"`
	Name   string `json:"name"`
	// AttackerEntryID e TargetEntryID são as linhas da fila: quem dispara está na vez, e o alvo é outra linha.
	AttackerEntryID string `json:"attackerEntryId"`
	TargetEntryID   string `json:"targetEntryId"`
	// Dice é a notação ("6d6"), Rolls os dados quando o app rolou, e Total a soma. ByHand: o total veio da mesa.
	Dice       string `json:"dice"`
	Rolls      []int  `json:"rolls,omitempty"`
	Total      int    `json:"total"`
	ByHand     bool   `json:"byHand,omitempty"`
	DamageType string `json:"damageType"`
	Range      string `json:"range"`
	// Save, DC e OnSave são a resistência do alvo: "Reflexos CD 17 reduz à metade".
	Save   string `json:"save"`
	DC     int    `json:"dc"`
	OnSave string `json:"onSave"`
	// BoostID é o gasto do encanto (`engine.EnchantBoostID`): a vez da rodada é dele. PM é o que ele custa.
	BoostID     string `json:"boostId"`
	PM          int    `json:"pm"`
	CharacterID int64  `json:"characterId"`
	ByUserID    int64  `json:"byUserId"`
}

// RecordBlast guarda o disparo, substituindo o anterior. O alvo tem de estar na fila: a bola de fogo é
// "contra um alvo", e a faixa o nomeia.
func RecordBlast(st *SessionRuntimeState, blast BlastRoll) error {
	if FindEntryIndex(st, blast.TargetEntryID) < 0 {
		return fmt.Errorf("o alvo %q não está na fila", blast.TargetEntryID)
	}
	if blast.TargetEntryID == blast.AttackerEntryID {
		return fmt.Errorf("a bola de fogo de %s é contra outro alvo da fila", blast.Who)
	}
	st.LastBlast = &blast
	return nil
}
