package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"t20engine/engine"
)

// O CONSTRUTOR DE ENCONTROS (ALE-259), terceira ferramenta da Mesa do Mestre.
//
// Diferente do bestiário e dos catálogos, esta cena tem ESTADO: o encontro
// sendo montado. Decisão do dono: ele vive nos SINAIS do Datastar, como o sinal
// efêmero da SPA, e um botão monta o link sob demanda.
//
// A alternativa era o encontro SER o endereço, e ela foi recusada por um custo
// concreto: a quantidade muda a cada clique, então cada `[+]` viraria uma
// entrada no histórico e o botão Voltar passaria a desfazer cliques em vez de
// sair da tela. Compartilhar continua existindo — só deixa de ser automático.
//
// A conta NÃO está aqui: ela é regra do livro e mora no `engine`
// (`NDDeGrupo`, `DificuldadeDoEncontro`, `XPDoEncontro`). Esta camada resolve
// ids em verbetes e monta o que a tela lê.

// linhaDoEncontro é uma linha da composição: um verbete e quantos dele.
type linhaDoEncontro struct {
	ID  string `json:"id"`
	Qtd int    `json:"qtd"`
}

// grupoDoEncontro é a linha já resolvida, com o ND que ela vale.
type grupoDoEncontro struct {
	Verbete verbete
	Qtd     int
	ND      float64
}

const (
	nivelMinimo   = 1
	nivelMaximo   = 20
	grupoMinimo   = 1
	grupoMaximo   = 8
	quantidadeMax = 99
)

// Sem campanha por perto — a Mesa do Mestre monta encontro no vazio —, o grupo
// padrão é o do livro: quatro personagens de 1º (p282).
const (
	nivelPadrao = 1
	grupoPadrao = 4
)

type encontrosView struct {
	Nivel  int
	Grupo  int
	Linhas []grupoDoEncontro
	// Busca e Achados são do painel de adicionar criatura, que reusa o filtro
	// do bestiário — o mestre que aprendeu a buscar lá não reaprende aqui.
	Busca   string
	Achados []verbete
}

// ND do encontro é a SOMA dos grupos. O livro cala sobre composição mista,
// então somar é o padrão permissivo e o mestre confere no olho.
func (v encontrosView) ND() float64 {
	total := 0.0
	for _, l := range v.Linhas {
		total += l.ND
	}
	return total
}

func (v encontrosView) Dificuldade() engine.Dificuldade {
	return engine.DificuldadeDoEncontro(v.ND() - float64(v.Nivel))
}

// XPPorPersonagem assume VITÓRIA: o construtor planeja o combate, e planejar
// perder não é o caso de uso. O desfecho existe na regra para quando a sessão
// registrar o resultado.
func (v encontrosView) XPPorPersonagem() int {
	return engine.XPDoEncontro(v.ND(), v.Nivel, v.Grupo, engine.Vitoria)
}

func (v encontrosView) Vazio() bool { return len(v.Linhas) == 0 }

// carregaEncontros resolve as linhas em verbetes e faz as contas.
//
// Linha cujo verbete sumiu do bestiário é DESCARTADA, e não desenhada vazia:
// um id velho colado numa URL renderizaria uma linha sem nome com quantidade
// viva, que é pior que não existir.
func carregaEncontros(nivel, grupo int, linhas []linhaDoEncontro, busca string) encontrosView {
	v := encontrosView{
		Nivel: aperta(nivel, nivelMinimo, nivelMaximo, nivelPadrao),
		Grupo: aperta(grupo, grupoMinimo, grupoMaximo, grupoPadrao),
		Busca: busca,
	}
	porID := map[string]verbete{}
	for _, m := range criaturasDoLivro() {
		porID[m.ID] = m
	}
	for _, l := range linhas {
		m, ok := porID[l.ID]
		if !ok {
			continue
		}
		qtd := aperta(l.Qtd, 1, quantidadeMax, 1)
		v.Linhas = append(v.Linhas, grupoDoEncontro{
			Verbete: m,
			Qtd:     qtd,
			ND:      engine.NDDeGrupo(m.ND, qtd),
		})
	}
	// A busca do painel de adicionar só corre quando há termo: mostrar as 80
	// criaturas abaixo do encontro empurraria a conta para fora da tela, e a
	// conta é o assunto desta ferramenta (ALE-170).
	if strings.TrimSpace(busca) != "" {
		v.Achados = filtraCriaturas(criaturasDoLivro(), filtroDeCriaturas{
			Busca: busca, NDMin: ndMinimo, NDMax: ndMaximo,
		})
	}
	return v
}

// aperta prende um número na faixa; fora dela ou ausente, cai no padrão.
func aperta(v, min, max, padrao int) int {
	if v < min || v > max {
		return padrao
	}
	return v
}

// ── a álgebra do rascunho ────────────────────────────────────────────────────

// acrescenta põe uma criatura no encontro, ou SOBE a contagem se ela já está.
//
// Duas linhas do mesmo verbete calculariam cada uma o próprio ND de grupo, e a
// regra da dobra (p282) só significa alguma coisa sobre UM grupo — dois grupos
// de dois ogros valeriam menos que um grupo de quatro, que é o oposto da regra.
func acrescenta(linhas []linhaDoEncontro, id string) []linhaDoEncontro {
	for i := range linhas {
		if linhas[i].ID == id {
			if linhas[i].Qtd < quantidadeMax {
				linhas[i].Qtd++
			}
			return linhas
		}
	}
	return append(linhas, linhaDoEncontro{ID: id, Qtd: 1})
}

// diminui tira um da contagem, e a última unidade TIRA A LINHA: um grupo de
// zero criaturas não é um grupo, e deixar a linha com 0 mostraria "ND 0" numa
// linha que ainda parece parte do encontro.
func diminui(linhas []linhaDoEncontro, id string) []linhaDoEncontro {
	var fora []linhaDoEncontro
	for _, l := range linhas {
		if l.ID != id {
			fora = append(fora, l)
			continue
		}
		if l.Qtd > 1 {
			l.Qtd--
			fora = append(fora, l)
		}
	}
	return fora
}

func removeLinha(linhas []linhaDoEncontro, id string) []linhaDoEncontro {
	var fora []linhaDoEncontro
	for _, l := range linhas {
		if l.ID != id {
			fora = append(fora, l)
		}
	}
	return fora
}

// ── o encontro como ENDEREÇO, sob demanda ────────────────────────────────────

// enderecoDoEncontro monta o link que o botão de copiar entrega.
//
// O formato é curto de propósito — `?c=goblin:4,ogro:1` —, porque ele acaba
// colado no chat da mesa e uma URL de trezentos caracteres não sobrevive a
// quebra de linha. A ordem é a da composição, para o link reabrir a lista como
// o mestre a montou.
func enderecoDoEncontro(v encontrosView) string {
	q := url.Values{}
	q.Set("nivel", strconv.Itoa(v.Nivel))
	q.Set("grupo", strconv.Itoa(v.Grupo))
	if len(v.Linhas) > 0 {
		var partes []string
		for _, l := range v.Linhas {
			partes = append(partes, fmt.Sprintf("%s:%d", l.Verbete.ID, l.Qtd))
		}
		q.Set("c", strings.Join(partes, ","))
	}
	return "/mestre/encontros?" + q.Encode()
}

// linhasDaURL lê `?c=goblin:4,ogro:1`.
//
// Entrada mal formada é DESCARTADA por linha, não recusada em bloco: o link
// chega por chat e um caractere a mais não pode custar o encontro inteiro.
func linhasDaURL(bruto string) []linhaDoEncontro {
	if bruto == "" {
		return nil
	}
	var fora []linhaDoEncontro
	for _, parte := range strings.Split(bruto, ",") {
		id, qtd, achou := strings.Cut(strings.TrimSpace(parte), ":")
		if !achou || id == "" {
			continue
		}
		n, err := strconv.Atoi(qtd)
		if err != nil || n < 1 {
			continue
		}
		fora = append(fora, linhaDoEncontro{ID: id, Qtd: min(n, quantidadeMax)})
	}
	return fora
}

// ── o que a cena escreve ─────────────────────────────────────────────────────

// sinaisDosEncontros: o rascunho inteiro vive aqui, e é isso que faz o clique
// não mexer no histórico.
func sinaisDosEncontros(v encontrosView) string {
	linhas := make([]linhaDoEncontro, 0, len(v.Linhas))
	for _, l := range v.Linhas {
		linhas = append(linhas, linhaDoEncontro{ID: l.Verbete.ID, Qtd: l.Qtd})
	}
	encontro, _ := json.Marshal(linhas)
	busca, _ := json.Marshal(v.Busca)
	return fmt.Sprintf(`{nivel: %d, grupo: %d, encontro: %s, buscaCriatura: %s, copiado: false}`,
		v.Nivel, v.Grupo, encontro, busca)
}

// ndArredondado é o que a tela mostra: duas casas, porque o log2 da regra da
// dobra produz dízima e "ND 4.999999999999999" não é um número que se lê.
func ndArredondado(nd float64) string {
	return ndEscrito(math.Round(nd*100) / 100)
}

// A cor da dificuldade sai do TOM, nunca do rótulo: o texto é o que o leitor de
// tela anuncia, e a cor é enfeite por cima dele.
var tintaDoTom = map[string]string{
	"calmo":   "var(--hp-full)",
	"parelho": "var(--grimorio-gold)",
	"duro":    "var(--hp-hurt)",
	"mortal":  "var(--grimorio-crimson-bright)",
}

func corDaDificuldade(tom string) string {
	if c, ok := tintaDoTom[tom]; ok {
		return "color: " + c
	}
	return "color: var(--grimorio-gold)"
}

// xpEscrito com separador de milhar pt-BR: 1000 vira "1.000".
func xpEscrito(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var partes []string
	for len(s) > 3 {
		partes = append([]string{s[len(s)-3:]}, partes...)
		s = s[:len(s)-3]
	}
	return strings.Join(append([]string{s}, partes...), ".")
}

// rotuloDoPasso escreve o nome acessível do botão de quantidade.
func rotuloDoPasso(passo, nome string) string {
	if passo == "mais" {
		return "Mais um " + nome
	}
	return "Menos um " + nome
}
