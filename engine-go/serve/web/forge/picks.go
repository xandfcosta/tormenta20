package forge

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/a-h/templ"
)

// AS LINHAS DE ESCOLHA da folha (ALE-535): raça, origem e deus abrem uma folha
// com busca, e a linha diz o que já foi escolhido. A classe tem lista própria,
// porque cada uma leva um nível.

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

// godPick é a linha do deus, que nunca fica vazia: o Nenhum é uma resposta.
func (v forgeView) godPick() pickSummary {
	if v.GodChosen == "Nenhum" {
		return pickSummary{Title: v.GodChosen, Meta: "opcional"}
	}
	if !v.GodGrantsPower {
		return pickSummary{Title: v.GodChosen, Meta: "não recebe poder concedido"}
	}
	return pickSummary{Title: v.GodChosen, Meta: "o poder concedido se escolhe na ficha"}
}

// classSheetTitle é o título da folha de classes: a primeira é a classe, e as seguintes são outra.
func (v forgeView) classSheetTitle() string {
	if len(v.ClassLines) == 0 {
		return "Classe"
	}
	return "Outra classe"
}

// classStepCommand é o `−` ou o `+` de uma classe. Manda o formulário inteiro, como todo redesenho.
func classStepCommand(class string, step int) string {
	return fmt.Sprintf("@post('/personagens/nova/esboco/nivel/%s/%d', {contentType: 'form', selector: '#forge-form'})",
		url.PathEscape(class), step)
}

// classStepDownLabel diz o que o `−` faz: no 1º nível ele tira a classe.
func classStepDownLabel(line classLine) string {
	if line.Level == 1 {
		return "Tirar " + line.Name
	}
	return "Diminuir " + line.Name
}

// classStepUpAttrs é o `+`, que no 20º nível do herói fica apagado.
func classStepUpAttrs(v forgeView, class string) templ.Attributes {
	attrs := templ.Attributes{"type": "button", "data-on:click": classStepCommand(class, 1)}
	if v.LevelCapped {
		attrs["disabled"] = true
	}
	return attrs
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

// itemPick é o que a linha de um item do kit mostra.
func itemPick(options []itemOption) pickSummary {
	for _, o := range options {
		if o.Chosen {
			return pickSummary{Title: o.Label, Meta: o.Meta}
		}
	}
	return pickSummary{}
}
