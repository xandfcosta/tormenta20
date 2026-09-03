package master

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/web/bookui"
	"t20engine/web/ui"
)

// A PORTA da Mesa do Mestre (ALE-278).
//
// Ela é a MENOR porta com dependências até aqui — DUAS —, contra treze da
// administração, nove da porta de entrar, seis da forja e zero do buscador. E o
// tamanho não é mérito de desenho: é o que a cena É. As treze unidades do
// `/mestre/*` desenham o LIVRO, que é igual para todo mundo e chega por
// `go:embed`. Não há banco, não há campanha, não há personagem — o
// `requirePage` do grupo já respondeu a única pergunta de autorização que
// existe aqui, que é "entrou?".
//
// Medido antes de escrever: nos doze arquivos de produção que se mudaram, o
// hospedeiro era alcançado em exatamente dois lugares — o `WritePage` de cada
// handler e o `s.livro.endereco` que os componentes do livro recebem. Todo o
// resto já falava com `book`, `creature`, `search`, `web/ui`, `web/bookui` e
// `web/routes`, seis pacotes que já eram folha.
type Deps interface {
	// WritePage é a montagem da casca: ela injeta os estáticos e as
	// sobreposições que a cena não pode conhecer (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// BookAddress é onde o PDF do livro está, para o selo "p289" saber para
	// onde apontar. Ele vem por aqui e não por `book` porque não é dado do
	// livro: é CONFIGURAÇÃO do dono da mesa (`LIVRO_PDF`), e sem ela o selo
	// simplesmente não é desenhado.
	//
	// A porta pede o endereço PRONTO e não a `Config`, pela mesma razão que a
	// administração recusou o `s.cfg`: um tipo do hospedeiro com trinta campos
	// atravessando a fronteira para a cena ler dois.
	BookAddress() bookui.BookAddress
}

// Scene é a cena montada com as dependências dela.
//
// Ela existe porque esta cena TEM porta, ao contrário do buscador, que declara
// zero e por isso recebe só o roteador (ALE-278). A regra que a fatia do
// buscador deixou escrita continua valendo dos dois lados: interface vazia por
// simetria seria cerimônia, e porta com dois métodos é porta.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
