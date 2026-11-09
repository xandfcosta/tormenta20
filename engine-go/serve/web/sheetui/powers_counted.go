package sheetui

import (
	"strconv"

	"t20engine/domain/sheet"
)

// A ESCOLHA CONTADA do poder que a classe concedeu: as perícias do Especialista (por atributo), os terrenos
// do Explorador (por nível) e as escolas das Magias do bardo e do druida (pelo número do verbete).

// countedChoice é o bloco do Especialista: as perícias treinadas, quantas de
// quantas, e o atributo que dá o limite.
type countedChoice struct {
	ID      string
	Name    string
	Detail  string
	Options []powerOption
	Chosen  int
	Limit   int
	// Attribute é o que dá o limite ("Int", "nível") e Value o valor dele.
	Attribute string
	Value     int
	// Repeats diz que tocar uma opção escolhida SOMA outra (o terreno
	// repetido do Explorador), e o "−" tira — o padrão do Aumento de Atributo.
	Repeats bool
	// Fixed diz que o limite é o número do verbete (as três escolas, p44): não
	// há atributo nem nível a escrever ao lado dele.
	Fixed bool
}

// Hint é a nota de rodapé do bloco.
func (c countedChoice) Hint() string {
	if c.Repeats {
		return "Escolher de novo soma +2 no mesmo terreno. Área de Tormenta a partir do 11º nível."
	}
	if c.Fixed {
		return "O livro diz: uma vez feita, essa escolha não pode ser mudada."
	}
	return "Só as perícias treinadas."
}

// Full diz que não cabe mais nenhuma: as não escolhidas se apagam.
func (c countedChoice) Full() bool { return c.Chosen >= c.Limit }

// Written é "1 de 2 · Int 2", ou "1 de 3" quando o limite é o do verbete.
func (c countedChoice) Written() string {
	if c.Fixed {
		return strconv.Itoa(c.Chosen) + " de " + strconv.Itoa(c.Limit)
	}
	return strconv.Itoa(c.Chosen) + " de " + strconv.Itoa(c.Limit) + " · " + c.Attribute + " " + strconv.Itoa(c.Value)
}

// countedChoicesOf monta os blocos da classe a partir do que a ficha tem.
func countedChoicesOf(dto sheet.CharacterDTO, class string) []countedChoice {
	out := []countedChoice{}
	for _, counted := range sheet.CountedChoices(dto) {
		if counted.Power.ClassName != class {
			continue
		}
		block := countedChoice{
			ID: counted.Power.ID, Name: counted.Power.Name, Detail: counted.Power.Description,
			Chosen: len(counted.Picks), Limit: counted.Limit, Value: counted.Value,
			Attribute: counted.Basis, Repeats: counted.Repeats, Fixed: counted.Fixed,
		}
		for _, option := range counted.Offered {
			label := counted.Labels[option]
			if label == "" {
				label = option
			}
			count := 0
			for _, pick := range counted.Picks {
				if pick == option {
					count++
				}
			}
			block.Options = append(block.Options, powerOption{Value: option, Label: label, Count: count})
		}
		out = append(out, block)
	}
	return out
}
