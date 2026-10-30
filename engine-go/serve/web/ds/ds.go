// Package ds é o KIT do sistema Tempestade (ALE-497): as peças com que toda
// tela migrada se monta.
//
// A peça escreve a própria classe e a variante num `data-*`, e a tela não
// escolhe cor nenhuma. Uma tela que precisa de algo que não está aqui
// acrescenta a peça a este pacote e à folha viva `/sistema` antes de usá-la —
// é o que impede cada tela de resolver o botão do seu jeito, que é um dos
// quatro incômodos que a épica veio consertar.
//
// O estilo mora em `serve/web/assets/src/ds.css`, e só vale dentro de `.scene-ds`,
// que é a classe que a casca põe na cena migrada.
package ds

import (
	"strconv"

	"github.com/a-h/templ"
)

// ButtonVariant é o PESO do botão. Um primário por região; o resto é
// secundário ou fantasma. O destrutivo é o único vermelho preenchido do app.
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = ""
	ButtonSecondary ButtonVariant = "secondary"
	ButtonGhost     ButtonVariant = "ghost"
	ButtonDanger    ButtonVariant = "danger"
)

// Field é um campo de texto com rótulo. `Error` vazio é o campo válido; cheio,
// ele troca a ajuda pela frase do erro, que diz o valor e o formato esperado.
type Field struct {
	ID          string
	Name        string
	Label       string
	Type        string
	Value       string
	Placeholder string
	Help        string
	Error       string
}

// SumTerm é uma PARCELA da conta: a origem, uma nota opcional (de onde ela
// vem) e o valor já escrito com sinal, como a ficha o mostra.
//
// O sinal vem pronto e a penalidade é explícita porque quem sabe se a parcela
// tira é a regra, e não a primeira letra de uma string.
type SumTerm struct {
	Label   string
	Note    string
	Value   string
	Penalty bool
}

// SumHead é a linha fechada da conta. Muted é o total que existe e não vale —
// a perícia só para treinados, sem treino —, e ele sai riscado; Alert é o
// porquê, em destaque depois da nota. Um sem o outro não se explica.
type SumHead struct {
	Title string
	Hint  string
	Alert string
	Total string
	Open  bool
	Muted bool
}

// MeterKind escolhe a cor da barra: PV e PM têm cor própria, e o resto é tinta.
type MeterKind string

const (
	MeterPlain MeterKind = ""
	MeterPV    MeterKind = "pv"
	MeterPM    MeterKind = "pm"
)

// Meter é um número protagonista com a barra embaixo, quando há máximo.
//
// Sem `Max`, ele é só o número (Defesa, deslocamento). `Temp` é a reserva que
// se gasta antes (PV temporário), desenhada listrada depois da barra cheia.
type Meter struct {
	Label string
	Value int
	Max   int
	Temp  int
	Unit  string
	Kind  MeterKind
}

// share devolve a fração da barra, em porcentagem, que `n` ocupa.
//
// O denominador é o máximo MAIS a reserva, e não só o máximo: com 50 de 52 PV e
// 6 temporários, medir sobre 52 daria uma barra de 107%, que transborda. A
// reserva é parte da barra, e a soma das duas partes nunca passa de 100.
func (m Meter) share(n int) string {
	total := m.Max + m.Temp
	if total <= 0 || n <= 0 {
		return "0%"
	}
	if n > total {
		n = total
	}
	return formatPercent(n * 100 / total)
}

// RollTerm é uma parcela da rolagem, depois do dado.
type RollTerm struct {
	Label string
	Value int
}

// Roll é um resultado de rolagem: o que foi rolado, o total grande, e a conta.
//
// Sem veredito, de propósito — dizer se passou é do mestre (CLAUDE.md, "O
// sistema INFORMA; o mestre DECIDE").
type Roll struct {
	What    string
	Total   int
	Die     string
	Natural int
	Terms   []RollTerm
	// Lines são as linhas EXTRAS já escritas: o dano, o crítico. Elas não são
	// parcelas do total, e por isso não entram em `Terms`.
	Lines []string
}

func formatPercent(n int) string { return strconv.Itoa(n) + "%" }

// rollTermText escreve a parcela como a conta se lê em voz alta: o sinal, a
// origem e o tamanho. O sinal é o de MENOS tipográfico (−), e não o hífen, como
// no `stepSignal` da ficha: o hífen é mais curto que o "+" e desalinha a conta.
func rollTermText(t RollTerm) string {
	if t.Value < 0 {
		return " − " + t.Label + " " + strconv.Itoa(-t.Value)
	}
	return " + " + t.Label + " " + strconv.Itoa(t.Value)
}

// dialogBackdropCloses fecha o diálogo no toque no fundo. O fundo entrega o
// clique ao próprio `<dialog>`, e por isso o alvo tem de ser ele E o ponto tem
// de cair fora da caixa: o clique de teclado (Enter num botão) chega com as
// coordenadas zeradas, e só o alvo o separa. O `closedby="any"` faria isto
// nativo, mas ainda não chegou a todos os navegadores.
const dialogBackdropCloses = "evt.target === el && ((r) => evt.clientX < r.left || evt.clientX > r.right || " +
	"evt.clientY < r.top || evt.clientY > r.bottom)(el.getBoundingClientRect()) && el.close()"

// SoftNav é a TROCA MACIA de aba (ver o GLOSSARY): o link guarda o endereço no
// histórico e pede a cena pelo `@get`, sem recarregar a página. Com Ctrl, Cmd
// ou Shift ele volta a ser link comum, para abrir noutra guia. A entrada de
// onde se saiu também é marcada, senão o voltar até ela não recarregaria (ver
// o `popstate` do `scene.ts`).
//
//	@ds.Tab(href, current, ds.SoftNav())
func SoftNav() templ.Attributes {
	return templ.Attributes{"data-on:click": softNavigation}
}

const softNavigation = "!(evt.ctrlKey || evt.metaKey || evt.shiftKey) && (evt.preventDefault(), " +
	"history.state?.softNav || history.replaceState({softNav: true}, ''), " +
	"history.pushState({softNav: true}, '', el.href), @get(el.href))"
