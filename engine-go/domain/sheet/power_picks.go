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
		power := book.ClassPowers()[id]
		if power.Choice != nil && power.Choice.Repeatable && len(picks[id]) > 1 {
			used += len(picks[id])
			continue
		}
		used++
	}
	return used
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

// WithoutPowerPicksOf apaga as escolhas de um poder — o que acontece quando o
// poder sai da ficha, para não sobrar escolha de poder que ninguém tem.
func WithoutPowerPicksOf(blob, power string) string {
	picks := PowerPicks(blob)
	delete(picks, power)
	return marshalPowerPicks(picks)
}

// powerPicksFit confere as escolhas: só de poder que a ficha tem — escolhido ou
// concedido pelo nível —, só opção que o livro oferece, e mais de uma só no
// poder repetível. A escolha contada por atributo tem conferência própria, que
// depende da ficha (`ChosenExpertisesFit`).
func powerPicksFit(dto CharacterDTO) error {
	owned := map[string]bool{}
	for _, p := range OwnedPowers(dto) {
		owned[p.ID] = true
	}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		power, known := book.ClassPowers()[id]
		if !known || power.Choice == nil {
			return fmt.Errorf("o poder %q não pede escolha", id)
		}
		if !owned[id] {
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
