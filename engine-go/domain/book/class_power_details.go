package book

import (
	"encoding/json"

	"t20engine/domain/engine"
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
	findings := map[string]string{}
	for _, p := range ClassPowersWithModifiers() {
		for _, m := range p.Modifiers {
			if m.Condition != nil && m.Condition.C == "flagOn" && m.Condition.Flag != "" {
				findings[p.ID] = m.Condition.Flag
				break
			}
		}
	}
	return findings
}

// PowerModifiers é um poder de classe com os modificadores dele, que é tudo o
// que as duas perguntas deste arquivo precisam.
type PowerModifiers struct {
	ID        string            `json:"id"`
	Modifiers []engine.Modifier `json:"modifiers"`
}

// ClassPowersWithModifiers lê o catálogo resolvido UMA vez por chamada e
// devolve só o par que interessa.
//
// Ela nasceu quando o segundo leitor apareceu — a `StanceTargets` —, e o que
// ela evita é a segunda `json.Unmarshal` do mesmo blob com a mesma struct
// anônima: duas cópias da mesma leitura divergem no dia em que uma delas
// ganhar um campo.
func ClassPowersWithModifiers() []PowerModifiers {
	raw := classPowersResolved()
	if raw == nil {
		return nil
	}
	var powers []PowerModifiers
	_ = json.Unmarshal(raw, &powers)
	return powers
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
	raw := classPowersResolved()
	if raw == nil {
		return nil
	}
	var powers []struct {
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
	if err := json.Unmarshal(raw, &powers); err != nil {
		return nil
	}
	outside := []PowerTeachingSpells{}
	for _, p := range powers {
		if p.Choice == nil || p.Choice.GrantsSpellAttribute == "" {
			continue
		}
		options := map[string]string{}
		for _, o := range p.Choice.Options {
			options[o.ID] = o.Note
		}
		outside = append(outside, PowerTeachingSpells{ID: p.ID, Name: p.Name, Options: options})
	}
	return outside
}
