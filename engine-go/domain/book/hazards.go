package book

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"t20engine/domain/catalog"
)

// Os três tipos de perigo (p317-319), como o recurso `hazards` os guarda (ver GLOSSARY, **perigo**).
const (
	HazardKindTrap    = "armadilha"
	HazardKindDisease = "doenca"
	HazardKindAmbient = "ambiente"
)

// HazardKinds são os tipos na ordem em que a tela os oferece, com o nome que ela escreve.
var HazardKinds = []struct{ ID, Name string }{
	{HazardKindTrap, "Armadilha"},
	{HazardKindDisease, "Doença"},
	{HazardKindAmbient, "Ambiente"},
}

// Hazard é um perigo do Capítulo 7: a ameaça que não é criatura. A armadilha e a doença trazem a ficha; o perigo
// do ambiente é só a descrição, porque cada um tem regra própria.
type Hazard struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	BookPage    int            `json:"bookPage"`
	Description string         `json:"description"`
	Trap        *HazardTrap    `json:"trap"`
	Disease     *HazardDisease `json:"disease"`
	// Exposures é o dano por rodada de quem fica no perigo do ambiente: a exposição e a imersão total do ácido e da
	// lava. Vazio no resto.
	Exposures []HazardExposure `json:"exposures"`
}

// HazardExposure é uma forma de ficar no perigo do ambiente ("exposicao", "imersao") e o dano dela por rodada.
// Lingers diz que o dano persiste uma rodada depois de sair (p317, p319).
type HazardExposure struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Dice    string `json:"dice"`
	Kind    string `json:"kind"`
	Lingers bool   `json:"lingers"`
}

// ExposureByID acha a forma de ficar no perigo pelo id ("imersao").
func (h Hazard) ExposureByID(id string) (HazardExposure, bool) {
	for _, e := range h.Exposures {
		if e.ID == id {
			return e, true
		}
	}
	return HazardExposure{}, false
}

// HazardTrap é a ficha da armadilha (p317). Investigation e Thievery são as CDs para achá-la e para desarmá-la.
type HazardTrap struct {
	ND      float64        `json:"nd"`
	Magical bool           `json:"magical"`
	Damage  []HazardDamage `json:"damage"`
	Area    string         `json:"area"`
	Save    *HazardSave    `json:"save"`
	// Rounds é o dado de quantas rodadas o dano se repete ("2d4" no Gás Venenoso), vazio no dano de uma vez.
	Rounds        string `json:"rounds"`
	Investigation int    `json:"investigation"`
	Thievery      int    `json:"thievery"`
}

// DamageWritten é o dano da armadilha como TODA tela o escreve: curto, o dado e o tipo de cada parcela — "1
// perfuração + 1d12 perda de vida por veneno" —, porque o mestre o lê de relance e a frase do livro fica no selo de
// página (decisão do dono). Vazio na armadilha sem dano.
func (t HazardTrap) DamageWritten() string {
	parts := make([]string, len(t.Damage))
	for i, d := range t.Damage {
		parts[i] = strings.TrimSpace(d.Dice + " " + d.Kind)
	}
	return strings.Join(parts, " + ")
}

// SaveWritten é a resistência da vítima escrita: "Reflexos CD 20 evita". Vazio na armadilha sem resistência.
func (t HazardTrap) SaveWritten() string {
	if t.Save == nil {
		return ""
	}
	return fmt.Sprintf("%s CD %d %s", t.Save.Test, t.Save.DC, t.Save.Effect)
}

// HazardDamage é uma parcela do dano da armadilha ("1d12" de "perda de vida por veneno").
type HazardDamage struct {
	Dice string `json:"dice"`
	Kind string `json:"kind"`
}

// HazardSave é a resistência da vítima e o que passar nela faz ("evita", "reduz à metade").
type HazardSave struct {
	Test   string `json:"test"`
	DC     int    `json:"dc"`
	Effect string `json:"effect"`
}

// HazardDisease é a ficha da doença (p318): o contágio, a CD de Fortitude e os efeitos, um por falha seguida.
type HazardDisease struct {
	Contagion string   `json:"contagion"`
	DC        int      `json:"dc"`
	Effects   []string `json:"effects"`
}

var (
	hazardsOnce sync.Once
	hazards     []Hazard
)

// Hazards são os perigos na ordem do livro.
//
//	for _, h := range book.Hazards() { if h.Trap != nil { … } }
func Hazards() []Hazard {
	hazardsOnce.Do(func() {
		raw, ok := catalog.Resource("hazards")
		if !ok {
			panic("o recurso `hazards` não está no catálogo embutido: confira a lista de recursos em domain/catalog/catalog.go")
		}
		if err := json.Unmarshal(raw, &hazards); err != nil {
			panic(fmt.Sprintf("o recurso `hazards` não tem a forma de []book.Hazard: %v", err))
		}
	})
	return hazards
}

// HazardByID acha o perigo pelo id ("fosso-camuflado").
func HazardByID(id string) (Hazard, bool) {
	for _, h := range Hazards() {
		if h.ID == id {
			return h, true
		}
	}
	return Hazard{}, false
}

// HazardKindName é o nome que a tela escreve para o tipo ("doenca" → "Doença").
func HazardKindName(kind string) string {
	for _, k := range HazardKinds {
		if k.ID == kind {
			return k.Name
		}
	}
	return kind
}

// HazardFields é o que a busca dos catálogos varre no perigo.
func HazardFields(h Hazard) []string {
	return []string{h.Name, h.Description, HazardKindName(h.Kind)}
}
