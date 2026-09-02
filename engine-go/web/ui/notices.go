package ui

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
