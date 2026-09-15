package ui

import (
	"fmt"
	"strings"
)

// A IDENTIDADE VISUAL derivada do NOME.
//
// O app não tem campo de imagem para campanha nem para personagem: o que faz
// cada um ser reconhecível é um monogram sobre um gradiente cuja cor vem do
// NOME. Por isso a fórmula tem de ser a mesma em toda tela até o último dígito —
// um matiz diferente faz a mesma campanha ter duas capas, e a pessoa que abre a
// tela nova acha que abriu outra mesa.
//
// # Por que no KIT, e não numa cena
//
// CINCO famílias leem daqui — campanhas, a crônica, a entrada na campanha,
// personagens e a mesa. Enquanto isto morasse numa cena, cada cena que se
// mudasse levaria o hospedeiro junto.
//
// O kit é a casa certa porque nada aqui sabe do DOMÍNIO: são três funções de
// `string` para `string`. O `roleLabel` ficou de fora por isso — "Mestrando",
// "Jogando", "Mesa de X" é regra de quem é o quê numa campanha, e o kit não pode
// saber disso.

// NameHue é um hash de 31, e cada detalhe dele é para casar com o que o
// JavaScript calcula.
//
// A versão em JS itera por CODE POINT (`for...of`) mas lê `charCodeAt(0)`, que é
// a primeira unidade UTF-16. Para tudo no BMP — que inclui todo o pt-BR
// acentuado — os dois são o mesmo número, e o `range` do Go, que dá runas,
// concorda. Um nome com emoji divergiria; nenhum nome de campanha tem, e quando
// tiver, isto aqui é o lugar de olhar.
//
// O `>>> 0` do JS é truncamento para 32 bits sem sinal. Em Go o `uint32` já
// envolve sozinho, e o resultado é o mesmo porque multiplicação e soma
// distribuem sobre o módulo.
func NameHue(nome string) int {
	var hash uint32
	for _, r := range nome {
		hash = hash*31 + uint32(r)
	}
	return int(hash % 360)
}

// NameGradient é a capa: o gradiente de 155°, com a claridade e o croma de
// EMBLEMA (o retrato de personagem é um fio mais claro, e essa diferença é
// deliberada). Ele deriva de um nome QUALQUER — o cartão do herói o chama com o
// nome do personagem.
func NameGradient(nome string) string {
	m := NameHue(nome)
	return fmt.Sprintf(
		"linear-gradient(155deg, oklch(0.5 0.14 %d) 0%%, oklch(0.30 0.09 %d) 70%%, oklch(0.22 0.06 %d) 100%%)",
		m, m, m,
	)
}

// Monogram é o monogram de até duas letras.
func Monogram(nome string) string {
	partes := strings.Fields(nome)
	if len(partes) == 0 {
		return "?"
	}
	if len(partes) > 2 {
		partes = partes[:2]
	}
	var b strings.Builder
	for _, parte := range partes {
		for _, r := range parte {
			b.WriteString(strings.ToUpper(string(r)))
			break
		}
	}
	return b.String()
}
