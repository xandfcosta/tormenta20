package door

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/app/accounts"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA da porta. `door` e `port` são a mesma palavra em português e vão
// confundir alguém: `door` é a CENA de entrar, `port` é a interface que uma cena
// declara. Ver GLOSSARY.md.
//
// Ela era de NOVE métodos, e sete deles eram conta e senha: autenticar,
// cadastrar, classificar a recusa do cadastro, conferir o link de redefinição,
// redefinir. Esses sete viraram DOIS casos de uso — `accounts.Gate` e
// `accounts.Resets` —, e eles chegam por PARÂMETRO do `New`, não pela porta: o
// `app/` está abaixo da cena, então lê-lo direto não fecha ciclo nenhum
// (ALE-349).
//
// O que sobrou é o que de fato pertence ao hospedeiro: a casca da página e o
// BISCOITO. Pôr cookie é transporte — quem sabe o que é `Secure` e `SameSite` é
// quem responde HTTP —, e por isso a assinatura da sessão mora no caso de uso e
// a escrita dela mora aqui.
//
// O USUÁRIO INTEIRO continua sem atravessar: `AuthUser` é tipo do `api`, e
// pedi-lo faria a cena importar o hospedeiro, que a importa de volta para montar
// rota. A cena só precisa saber SE há sessão.
type Deps interface {
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
	HasSession(r *http.Request) bool
	// IssueSession escreve o cookie da sessão já assinada pelo caso de uso.
	// `false` é não ter conseguido assinar, e aí a cena mostra a recusa em vez de
	// mandar para dentro.
	IssueSession(w http.ResponseWriter, user sqlcgen.User) bool
}

// Scene é a porta montada com as dependências dela.
//
// Uma struct e não a interface direta porque o Go não aceita interface como
// RECEPTOR, e os handlers precisam ser métodos para as rotas ficarem legíveis.
type Scene struct {
	deps   Deps
	gate   accounts.Gate
	resets accounts.Resets
}

// New recebe os casos de uso por PARÂMETRO e o hospedeiro pela porta. A
// diferença não é estilo: o que chega por parâmetro é regra que outro transporte
// também chama, o que chega pela porta é o que só este hospedeiro sabe fazer.
func New(d Deps, gate accounts.Gate, resets accounts.Resets) Scene {
	return Scene{deps: d, gate: gate, resets: resets}
}
