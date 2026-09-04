package book

import (
	"encoding/json"

	"t20engine/catalog"

	"t20engine/engine"
)

// DOIS RECORTES do `class-powers` que o catálogo tipado não expunha (ALE-278).
//
// O `ClassPower` guarda id, nome, classe, nível e descrição — o que a lista de
// poderes desenha. Estes dois leem campos que ela não carrega: os MODIFICADORES
// (para saber que flag um poder liga) e o bloco de ESCOLHA (para saber que magia
// um poder ensina).
//
// Eles moravam na cena da ficha, lendo `catalog.Resource("class-powers")`
// direto — duas vezes, em dois arquivos, com o mesmo `Unmarshal` anônimo. É a
// mesma forma do `items.go` da forja e do improviso do trilho do mestre, e vêm
// para cá pela mesma regra: **o destino de uma função é a DEPENDÊNCIA dela.**

// ClassPowerFlags mapeia id do poder → flag que os modificadores dele ligam.
func ClassPowerFlags() map[string]string {
	bruto, ok := catalog.Resource("class-powers")
	if !ok {
		return map[string]string{}
	}
	var poderes []struct {
		ID        string            `json:"id"`
		Modifiers []engine.Modifier `json:"modifiers"`
	}
	_ = json.Unmarshal(bruto, &poderes)
	achados := map[string]string{}
	for _, p := range poderes {
		for _, m := range p.Modifiers {
			if m.Condition != nil && m.Condition.C == "flagOn" && m.Condition.Flag != "" {
				achados[p.ID] = m.Condition.Flag
				break
			}
		}
	}
	return achados
}

// PowerTeachingSpells é um poder que ENSINA magia, com o que ele oferece.
type PowerTeachingSpells struct {
	ID   string
	Name string
	// Options mapeia o id da escolha para o NOME da magia, que é o que a `note`
	// do catálogo guarda.
	Options map[string]string
}

// PowersThatTeachSpells são os poderes cujo bloco de escolha concede magia.
func PowersThatTeachSpells() []PowerTeachingSpells {
	bruto, ok := catalog.Resource("class-powers")
	if !ok {
		return nil
	}
	var poderes []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Choice *struct {
			GrantsSpellAttribute string `json:"grantsSpellAttribute"`
			Options              []struct {
				ID   string `json:"id"`
				Note string `json:"note"`
			} `json:"options"`
		} `json:"choice"`
	}
	if err := json.Unmarshal(bruto, &poderes); err != nil {
		return nil
	}
	fora := []PowerTeachingSpells{}
	for _, p := range poderes {
		if p.Choice == nil || p.Choice.GrantsSpellAttribute == "" {
			continue
		}
		opcoes := map[string]string{}
		for _, o := range p.Choice.Options {
			opcoes[o.ID] = o.Note
		}
		fora = append(fora, PowerTeachingSpells{ID: p.ID, Name: p.Name, Options: opcoes})
	}
	return fora
}
