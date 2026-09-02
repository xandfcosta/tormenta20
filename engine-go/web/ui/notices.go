package ui

import "fmt"

// A FRASE QUE TODA CENA PRECISA quando o servidor não conseguiu (ALE-278).
//
// Ela morava no `web/door/view.go`, entre as frases da porta — e não era da
// porta: a crônica e a administração já a usavam de lá. Quando a porta virou
// pacote, o compilador apontou, que é o ciclo funcionando como a ALE-254
// descreve: ele diz que a fronteira está no lugar errado.
//
// Fica no kit porque o kit é o que TODA cena alcança, e porque a frase é de
// APRESENTAÇÃO e não de domínio: ela não sabe o que falhou, e é justamente esse
// o ponto. Quando a cena sabe o que falhou, ela escreve a própria frase — esta é
// a que resta quando não há nada útil a dizer.
const NoticeInternal = "Não consegui completar agora. Tente de novo."

// Plural é a concordância que TODA tela precisa e que nenhuma quer reescrever.
//
// Ela morava no antigo `piloto_admin_view`, e a crônica já a usava de lá — o
// compilador apontou quando a administração virou pacote (ALE-278). É a segunda
// vez nesta épica que uma função de apresentação é encontrada hospedada numa
// cena por acidente de história; a primeira foi o `NoticeInternal` acima.
//
//	ui.Plural(1, "campanha", "campanhas")  // "1 campanha"
//	ui.Plural(3, "ficha", "fichas")        // "3 fichas"
//
// Ela leva as DUAS formas por parâmetro em vez de somar "s": o português tem
// plural irregular, e uma função que adivinha erra em "sessão" e em "papel"
// exatamente onde ninguém está olhando.
func Plural(n int64, um, muitos string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, um)
	}
	return fmt.Sprintf("%d %s", n, muitos)
}
