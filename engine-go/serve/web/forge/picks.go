package forge

import (
	"fmt"
	"strconv"
)

// AS LINHAS DE ESCOLHA da folha (ALE-535): raça, classe e origem abrem uma
// folha com busca, e a linha diz o que já foi escolhido.

// pickSummary é o que a linha mostra: o nome escolhido e uma linha do que ele
// dá. Vazio quando nada foi escolhido — aí a linha convida a escolher.
type pickSummary struct {
	Title string
	Meta  string
}

func (v forgeView) racePick() pickSummary {
	for _, c := range v.Races {
		if c.Chosen {
			meta := c.Size + " · " + strconv.Itoa(c.Displacement) + "m"
			if c.Attributes != "" {
				meta = c.Attributes + " · " + meta
			}
			return pickSummary{Title: c.Name, Meta: meta}
		}
	}
	return pickSummary{}
}

func (v forgeView) classPick() pickSummary {
	for _, c := range v.Classes {
		if c.Chosen {
			return pickSummary{Title: c.Name, Meta: fmt.Sprintf("PV %d · PM %d · %s", c.PV, c.PM, c.Expertises)}
		}
	}
	return pickSummary{}
}

func (v forgeView) originPick() pickSummary {
	for _, o := range v.Origins {
		if o.Chosen {
			return pickSummary{Title: o.Name, Meta: "dois benefícios se escolhem na ficha"}
		}
	}
	return pickSummary{}
}

// pickFilter é a expressão que mostra a opção quando a busca da folha casa com
// ela. A busca é do CLIENTE: as listas já vêm inteiras, e cada opção traz o
// próprio texto dobrado em `data-find`; aqui só se dobra o que foi digitado
// (minúsculas, sem acento), do mesmo jeito que o `search.Fold`.
func pickFilter(signal string) string {
	return fmt.Sprintf(`$%s.trim() === '' || el.dataset.find.includes($%s.trim().toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, ''))`, signal, signal)
}

// pickChosen fecha a folha e redesenha a forja com a escolha nova. O fechar vem
// ANTES do pedido: o remendo troca a folha inteira, e uma folha modal fechada
// por remendo pode ficar presa na camada de cima.
func pickChosen() string {
	return "el.closest('dialog').close(); " + forgeRedraw()
}
