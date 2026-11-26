package table

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"t20engine/domain/book"
	"t20engine/domain/live"
)

// O PERIGO NA MESA (ver GLOSSARY, **perigo**): a página Perigos do mestre, com o Disparar de cada armadilha e o Expor
// de cada forma de ficar no ácido e na lava, e a faixa do perigo disparado, que só ele vê — na mesa e no topo da
// página, para ele não trocar de aba para ler o que rolou. A regra do que se rola é do `session.FireHazard`.

// hazardRoutes pendura os dois gestos da página. O perigo vai no CAMINHO, como a linha da fila: nesta superfície o
// verbo é o caminho.
func (s Scene) hazardRoutes(r chi.Router) {
	r.Post(sessionPattern+"/"+string(pageHazards)+"/{perigo}/disparar", s.gmCommand(firesTheHazard))
	r.Post(sessionPattern+"/"+string(pageHazards)+"/{perigo}/expor/{exposicao}", s.gmCommand(firesTheHazard))
}

// firesTheHazard dispara a armadilha, ou expõe pela forma do caminho; sem a forma, é a armadilha.
func firesTheHazard(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.deps.Sessions().FireHazard(c.R.Context(), c.SessionID,
		chi.URLParam(c.R, "perigo"), chi.URLParam(c.R, "exposicao"), st.deps.RollDie)
}

// hazardRow é uma linha da página Perigos: o que se dispara ou a que se expõe, o que a ficha diz, e o gesto.
type hazardRow struct {
	Name string
	Meta string
	// Verb é o rótulo do botão ("Disparar", "Expor"), Aria o nome acessível dele, e Post o caminho do gesto
	// depois da sessão.
	Verb string
	Aria string
	Post string
}

// hazardRows são as linhas da página: as formas de ficar no ambiente, uma por linha, e as armadilhas, na ordem do
// livro. A doença e o resto do ambiente não rolam na mesa e não entram.
func hazardRows() (ambient, traps []hazardRow) {
	for _, h := range book.Hazards() {
		for _, e := range h.Exposures {
			name := h.Name + " · " + e.Name
			meta := e.Dice + " por rodada"
			if e.Kind != "" {
				meta = e.Dice + " de " + e.Kind + " por rodada"
			}
			if e.Lingers {
				meta += ", e mais uma depois de sair"
			}
			ambient = append(ambient, hazardRow{
				Name: name, Meta: meta, Verb: "Expor", Aria: "Expor a " + name,
				Post: fmt.Sprintf("%s/%s/expor/%s", pageHazards, h.ID, e.ID),
			})
		}
		if h.Trap == nil {
			continue
		}
		traps = append(traps, hazardRow{
			Name: h.Name, Meta: strings.Join(trapSheet(*h.Trap), " · "), Verb: "Disparar", Aria: "Disparar " + h.Name,
			Post: fmt.Sprintf("%s/%s/disparar", pageHazards, h.ID),
		})
	}
	return ambient, traps
}

// trapSheet é a ficha da armadilha numa linha: o ND, o dano, a resistência com a CD e se é mágica.
func trapSheet(t book.HazardTrap) []string {
	sheet := []string{"ND " + book.CRWritten(t.ND)}
	if damage := t.DamageWritten(); damage != "" {
		sheet = append(sheet, damage)
	}
	if save := t.SaveWritten(); save != "" {
		sheet = append(sheet, save)
	}
	if t.Magical {
		sheet = append(sheet, "mágica")
	}
	return sheet
}

// hazardBandOf traduz o perigo disparado na faixa, ou nil quando não há — e não há no estado do jogador, que chega
// sem ele (`live.HazardRoll`).
func hazardBandOf(st *live.SessionRuntimeState) *rollBand {
	if st == nil || st.LastHazard == nil {
		return nil
	}
	h := st.LastHazard
	band := &rollBand{
		Badge: book.HazardKindName(h.Kind) + " · p" + strconv.Itoa(h.BookPage),
		What:  h.Hazard,
	}
	if h.Exposure != "" {
		band.What += " · " + h.Exposure
	}
	for _, part := range h.Parts {
		band.Lines = append(band.Lines, rollLine{Total: part.Total, Tally: writtenRoll(part.Dice, part.Rolls, part.Kind, false)})
	}
	if h.Save != "" {
		band.Help = append(band.Help, fmt.Sprintf("%s · CD %d", h.Save, h.DC))
	}
	switch {
	case h.Rounds > 0:
		band.Help = append(band.Help, fmt.Sprintf(
			"Dura %d rodadas (%s): esta, e mais %d na vez de quem estiver nele. Marque quem está nele: “+ Marcar: %s” na folha da linha.",
			h.Rounds, writtenRoll(h.RoundsDice, h.RoundsRolls, "", false), h.Rounds-1, h.Hazard))
	case h.Lingering != nil:
		said := partsWritten(h.Lingering.Parts) + " por rodada, no início do turno de quem está exposto."
		if h.Lingers {
			said += " O dano persiste uma rodada depois de sair: tire a marca na rodada seguinte."
		}
		band.Help = append(band.Help, said,
			fmt.Sprintf("Marque quem está nele: “+ Marcar: %s” na folha da linha. A marca fica até você tirar.", h.Lingering.Source))
	case h.Description != "":
		band.Help = append(band.Help, h.Description)
	}
	return band
}
