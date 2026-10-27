package sheet

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/book"
)

// AS ESCOLHAS DE UM PODER (ALE-491).
//
// O poder que pede uma escolha — o atributo do Aumento de Atributo (p38), o
// animal do Totem Espiritual (p42) — guarda o que foi escolhido em
// `powerChoices`, pelo id do poder. O motor lê dali o alvo do número
// (`engine.expandChoiceModifiers`).
//
// O REPETÍVEL ("você pode escolher este poder várias vezes") entra UMA vez em
// `classPowers` e uma vez por escolha aqui, e cada escolha ocupa uma vaga. A
// alternativa — o id repetido em `classPowers` — quebraria o conjunto que todo
// leitor dele supõe, e a escolha teria de se casar com a repetição por ordem.

// PowerPicks é o `powerChoices` da ficha, pelo id do poder. Coluna ruim degrada
// para "nenhuma escolha", como os irmãos.
func PowerPicks(blob string) map[string][]string {
	picks := map[string][]string{}
	if json.Unmarshal([]byte(blob), &picks) != nil {
		return map[string][]string{}
	}
	return picks
}

func marshalPowerPicks(picks map[string][]string) string {
	b, err := json.Marshal(picks)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// UsedPowerSlots são as vagas de poder que a ficha ocupa: uma por poder
// escolhido, e uma por ESCOLHA no poder repetível.
//
//	// Aumento de Atributo em Força e em Sabedoria, e o Golpe Poderoso
//	UsedPowerSlots(dto) // 3
func UsedPowerSlots(dto CharacterDTO) int {
	picks := PowerPicks(dto.PowerChoices)
	used := 0
	for _, id := range UnmarshalStrings(dto.ClassPowers) {
		power, chooses := book.PowerWithChoice(id)
		if chooses && power.Choice.Repeatable && len(picks[id]) > 1 {
			used += max(1, len(picks[id])-FreePicks(dto, power))
			continue
		}
		used++
	}
	return used
}

// FreePicks são as escolhas do poder que NÃO custam vaga, porque o poder veio
// de fora da lista de escolhidos: o Foco em Perícia do Membro de Guilda (p92),
// o poder geral da Memória Póstuma (ALE-510). Uma por fonte.
func FreePicks(dto CharacterDTO, power book.ClassPower) int {
	free := 0
	for _, owned := range OwnedPowers(dto) {
		if owned.Source != PowerFromClass && owned.Source != PowerFromGeneral &&
			(owned.ID == power.ID || owned.Name == power.Name) {
			free++
		}
	}
	return free
}

// WithPowerPick acrescenta uma escolha ao poder. No poder que NÃO se repete, a
// escolha nova substitui a antiga: o Totem Espiritual tem um animal só. A
// escolha contada por atributo acumula opções DISTINTAS — as perícias do
// Especialista —, e quem a limita é o `ChosenExpertisesFit`.
func WithPowerPick(blob, power, option string, choice book.PowerChoice) string {
	picks := PowerPicks(blob)
	switch {
	case choice.Repeatable || len(choice.CountLevels) > 0:
		picks[power] = append(picks[power], option)
	case choice.CountAttribute != "":
		if !slices.Contains(picks[power], option) {
			picks[power] = append(picks[power], option)
		}
	default:
		picks[power] = []string{option}
	}
	return marshalPowerPicks(picks)
}

// WithoutPowerPick tira UMA ocorrência da escolha, e devolve quantas sobraram.
// A cópia é de uma só porque Força em dois patamares são dois +1, e tirar um
// não tira o outro.
func WithoutPowerPick(blob, power, option string) (string, int) {
	picks := PowerPicks(blob)
	at := slices.Index(picks[power], option)
	if at < 0 {
		return marshalPowerPicks(picks), len(picks[power])
	}
	picks[power] = slices.Delete(slices.Clone(picks[power]), at, at+1)
	left := len(picks[power])
	if left == 0 {
		delete(picks, power)
	}
	return marshalPowerPicks(picks), left
}

// spellPicksFit confere as escolhas do Foco em Magia (p131): "uma magia que
// possa lançar" — uma do grimório — e "magias diferentes".
func spellPicksFit(dto CharacterDTO, power book.ClassPower, picks []string) error {
	for i, pick := range picks {
		if slices.Contains(picks[:i], pick) {
			return fmt.Errorf("%s já escolheu %s; cada vez é uma magia diferente", power.Name, book.SpellName(pick))
		}
		if !slices.ContainsFunc(dto.Spells, func(s SpellDTO) bool { return s.CatalogSpellID == pick }) {
			return fmt.Errorf("%s escolhe uma magia que a ficha possa lançar, e %s não está no grimório", power.Name, book.SpellName(pick))
		}
		if !CanCastCircleOf(dto, pick) {
			return fmt.Errorf("%s escolhe uma magia que a ficha possa lançar, e %s é de um círculo que ela ainda não alcança",
				power.Name, book.SpellName(pick))
		}
	}
	return nil
}

// WithoutPowerPicksOf apaga as escolhas de um poder — o que acontece quando o
// poder sai da ficha, para não sobrar escolha de poder que ninguém tem.
func WithoutPowerPicksOf(blob, power string) string {
	picks := PowerPicks(blob)
	delete(picks, power)
	return marshalPowerPicks(picks)
}

// powerPicksFit confere as escolhas: só de poder que a ficha tem — escolhido ou
// concedido pelo nível —, só opção que o livro oferece, e mais de uma só no
// poder repetível. A escolha que depende do treino ou do atributo da ficha tem
// conferência própria (`ChosenExpertisesFit`).
func powerPicksFit(dto CharacterDTO) error {
	// A POSSE é pelo id ou pelo NOME: o Foco em Perícia que a origem dá tem o
	// id do benefício, e a escolha mora no id do poder (ALE-510).
	owned := map[string]bool{}
	for _, p := range OwnedPowers(dto) {
		owned[p.ID], owned["name:"+p.Name] = true, true
	}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		power, chooses := book.PowerWithChoice(id)
		if !chooses {
			return fmt.Errorf("o poder %q não pede escolha", id)
		}
		if !owned[id] && !owned["name:"+power.Name] {
			return fmt.Errorf("há escolha de %q, e a ficha não tem o poder", power.Name)
		}
		if power.Choice.CountAttribute != "" {
			continue
		}
		if len(picks) > 1 && !power.Choice.Repeatable && len(power.Choice.CountLevels) == 0 {
			return fmt.Errorf("%q se escolhe uma vez só, e há %d escolhas", power.Name, len(picks))
		}
		if err := picksAreOptions(power, picks); err != nil {
			return err
		}
		if power.Choice.Kind == "proficiency" {
			if err := proficiencyPicksFit(dto, power, picks); err != nil {
				return err
			}
		}
		if power.Choice.RequiresProficiency {
			if err := weaponPicksFit(dto, power, picks); err != nil {
				return err
			}
		}
		if power.Choice.Kind == "spell" {
			if err := spellPicksFit(dto, power, picks); err != nil {
				return err
			}
		}
	}
	return nil
}

// picksAreOptions recusa a escolha que o livro não oferece, dizendo quais são.
// A lista é a do `Offered`: a do verbete, ou as armas do catálogo.
func picksAreOptions(power book.ClassPower, picks []string) error {
	offered := power.Choice.Offered()
	if len(offered) == 0 {
		return fmt.Errorf("%s pede escolha e não oferece opção nenhuma", power.Name)
	}
	names := make([]string, 0, len(offered))
	for _, o := range offered {
		names = append(names, o.Name)
	}
	for _, pick := range picks {
		if !slices.ContainsFunc(offered, func(o book.PowerChoiceOption) bool { return o.ID == pick }) {
			return fmt.Errorf("%q não é opção de %s: as opções são %s",
				pick, power.Name, strings.Join(names, ", "))
		}
	}
	return nil
}
