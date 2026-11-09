package book

import "slices"

// PowerChange é o que um item EMPUNHADO muda em poderes de quem o empunha: a ação que eles custam, e o
// aviso da linha deles quando a mudança não é número que o motor guarde. É a forma do Alaúde Élfico e do
// Tambor das Profundezas (p158).
type PowerChange struct {
	Powers []string `json:"powers"`
	Action string   `json:"action"`
	Note   string   `json:"note"`
}

// ActivationHeld é o poder como ele fica para quem empunha os itens `wielded` (ids do catálogo), e os
// avisos que os itens deixam na linha dele.
//
// Quem pergunta a ação de um poder pergunta AQUI, e não ao catálogo cru: a cobrança da vez, o grupo do
// chip na superfície Ações e a linha da aba Poderes têm de concordar, e um que lesse o catálogo direto
// cobraria a ação padrão de quem tem o alaúde na mão enquanto a tela diz movimento.
//
//	held, _ := book.ActivationHeld(*book.ActivationOf("class.bardo.inspiracao", ""), []string{"alaude-elfico"})
//	held.Action // "movimento"
func ActivationHeld(spec Activation, wielded []string) (Activation, []string) {
	notes := []string{}
	for _, id := range wielded {
		item := ItemByID(id)
		if item == nil {
			continue
		}
		for _, change := range item.PowerChanges {
			if !slices.Contains(change.Powers, spec.ID) {
				continue
			}
			if change.Action != "" {
				spec.Action = change.Action
			}
			if change.Note != "" {
				notes = append(notes, change.Note+" ("+item.Name+")")
			}
		}
	}
	return spec, notes
}
