package sheetui

import (
	"fmt"
	"net/http"

	"t20engine/infra/db/sqlcgen"
)

// sheetMutation é um gesto da ficha, como o `sheetCommand` o recebe.
type sheetMutation = func(Scene, *http.Request, sqlcgen.Character, Signals) error

// onlyInPlay marca o gesto de JOGAR: o rascunho só monta o personagem, e quem joga é a cópia da
// campanha (ALE-564). A trava é na rota porque a tela só deixa de oferecer: a fronteira é o
// servidor.
func onlyInPlay(gesture sheetMutation) sheetMutation {
	return func(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
		if !row.CampaignId.Valid {
			return fmt.Errorf("%s é um rascunho: para jogar, entre com ele numa campanha", row.Name)
		}
		return gesture(s, r, row, signals)
	}
}
