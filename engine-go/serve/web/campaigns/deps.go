package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA das campanhas, e a mais larga da série: a cena cobre quatro telas com
// um endereço cada — a lista, a campanha aberta, a folha em branco e a carta de
// entrar — e três delas ESCREVEM.
//
// Duas coisas NÃO atravessam esta porta, e as duas por um motivo só: tipo com
// tag `json:` é a forma de um protocolo, e uma tela que o lesse passaria a
// depender do formato de um fio que ela não fala. Por isso a cena declara o
// `ListRow` e o `PlaceRow` dela, e por isso ela pede PERGUNTAS (`SaveText`) em
// vez de montar SQL.
type Deps interface {
	// Queries é o banco. O `Queries` continua na porta porque três telas leem e
	// escrevem as próprias tabelas; o sinal de que ele está no lugar é nenhum
	// handler daqui tocar banco fora dele.
	Queries() *sqlcgen.Queries
	CurrentUserID(r *http.Request) int64
	// RequesterIsAdmin olha a REQUISIÇÃO. O hospedeiro já tem um `IsAdmin`, que
	// olha a configuração e recebe um e-mail — dois nomes porque são duas
	// perguntas, e o compilador recusaria um só.
	RequesterIsAdmin(r *http.Request) bool
	// List são as campanhas que esta pessoa vê, com o papel dela em cada uma.
	List(ctx context.Context, userID int64, admin bool) ([]ListRow, error)
	RoleIn(ctx context.Context, userID int64, c sqlcgen.Campaign) (papel string, membros int, err error)
	// OwnerNames traduz o dono de cada campanha em nome, para a lista do admin.
	OwnerNames(ctx context.Context, campanhas []sqlcgen.Campaign, quemPede int64) map[int64]string
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)
	// IgnoredRules é o que o mestre DESLIGOU das regras opcionais.
	IgnoredRules(ctx context.Context, campanhaID int64) []string
	SaveIgnoredRules(ctx context.Context, campanhaID int64, regras []string) error
	// OpenTable abre a mesa já com link de convite. Cunhar é do hospedeiro
	// porque é `crypto/rand` e é a política de quem entra — e uma mesa que nasce
	// sem link recusa todo mundo menos o dono.
	OpenTable(ctx context.Context, donoID int64, nome, descricao string) (id int64, err error)
	// InviteLink é o link da mesa, ou "" quando ela não tem um — estado NORMAL
	// em toda campanha aberta antes de o link existir.
	InviteLink(ctx context.Context, campanhaID int64) string
	// RotateInvite cunha um novo e derruba o anterior: um gesto só, porque
	// "nunca teve link" e "quero cortar quem tem" pedem a mesma coisa.
	RotateInvite(ctx context.Context, campanhaID int64) (string, error)
	// SaveText grava nome e descrição. Descrição vazia vira NULL no hospedeiro,
	// para a regra não carregar `database/sql`.
	SaveText(ctx context.Context, campanhaID int64, nome, descricao string) error
	// Join devolve o MOTIVO da recusa, não o erro: quem classifica é o
	// hospedeiro, quem escolhe a frase é a cena. Ler os sentinelas de erro daqui
	// alcançaria o `api`.
	Join(ctx context.Context, campanhaID, heroiID, quemPede int64, convite string) JoinRefusal

	// O ACERVO DE LUGARES são três perguntas e não o `BoardStore` inteiro: o
	// store é o vocabulário do domínio AO VIVO, e esta cena não é ao vivo.
	Places(ctx context.Context, campanhaID int64) []PlaceRow
	// NewPlace cria o lugar vazio quando ele ainda não existe. Nome repetido
	// leva ÀQUELE lugar: o nome é a identidade dele dentro da campanha.
	NewPlace(ctx context.Context, campanhaID int64, nome, chao string) (int64, error)
	RemovePlace(ctx context.Context, campanhaID, lugarID int64) error
	// Grounds são as aparências que um lugar pode ter. Vêm pela porta porque são
	// do tabuleiro: uma cópia aqui ofereceria um chão que o servidor não conhece
	// no dia em que a sexta nascer.
	Grounds() []GroundOption

	// CampaignDeleted é chamada ANTES do `DeleteCampaign`: apagar leva as
	// sessões por cascata, e depois não há como perguntar quais eram. Sem ela o
	// tabuleiro de cada sessão fica no mapa em memória batendo na chave
	// estrangeira, e a mesa se declara suja para sempre.
	CampaignDeleted(ctx context.Context, campanhaID int64)

	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// PlaceRow é um lugar do acervo, na forma que esta tela desenha.
type PlaceRow struct {
	ID   int64
	Nome string
	// Pecas é a CONTAGEM, e ela separa a cena montada da cena abandonada:
	// "Cripta · 9 peças" é uma noite de trabalho, "cena vazia" é lixo.
	Pecas int
	// Quando é a última mudança, já legível.
	Quando string
	// NaMesaID é a sessão que mostra este lugar agora, ou zero. Ela decide os
	// gestos da linha: o lugar que está numa mesa não se monta nem se apaga. As
	// travas de verdade são do servidor; isto é a cortesia de não oferecer o
	// gesto que ele vai recusar.
	NaMesaID int64
}

// GroundOption é uma aparência de lugar, para o formulário do lugar novo.
type GroundOption struct {
	ID     string
	Rotulo string
}

// ListRow é uma campanha na LISTA, na forma que esta cena precisa.
type ListRow struct {
	ID          int64
	Name        string
	Description string
	// Role é o papel de quem pede: `gm` ou `player`.
	Role string
	// OwnerName vem preenchido SÓ numa campanha que quem pede não possui — hoje,
	// um admin vendo as de todo mundo. A tela marca a exceção, não toda linha.
	OwnerName string
	// Character é o herói de quem pede NESTA campanha, quando há um.
	Character *RowCharacter
}

// RowCharacter é o herói de quem pede numa campanha da lista.
type RowCharacter struct {
	ID      int64
	Name    string
	Level   int64
	Classes []sheet.ClassDTO
}

// JoinRefusal é o MOTIVO de a pessoa não conseguir sentar à mesa.
//
// São SEIS valores para as sete travas do hospedeiro, e a diferença é
// deliberada: "personagem não existe" e "personagem é de outra pessoa" viram a
// mesma frase, e distinguir diria a um estranho se um id existe.
type JoinRefusal int

const (
	// JoinOK é a pessoa sentada.
	JoinOK JoinRefusal = iota
	// JoinNoSuchCampaign: o número digitado não é de campanha nenhuma.
	JoinNoSuchCampaign
	// JoinNeedsInvite: a mesa é fechada e o convite não serve.
	JoinNeedsInvite
	// JoinNotYourHero cobre as DUAS travas de personagem do hospedeiro.
	JoinNotYourHero
	// JoinAlreadyHasHero: esta pessoa já tem um herói nesta mesa.
	JoinAlreadyHasHero
	// JoinHeroAlreadyThere: este herói já está nesta mesa.
	JoinHeroAlreadyThere
	// JoinFailed é qualquer outra coisa, e vira o aviso interno.
	JoinFailed
)

// As montagens (`LoadList`, `LoadOne`, `LoadJoin`, `JoinBody`) são EXPORTADAS
// porque quem prova o caminho banco → tela é a bancada do `api`: este pacote
// não tem banco, e importar o `db/testdb` com um `*api.Server` seria o ciclo
// que a divisão existe para evitar.
//
// A cena diz COMO montar a si mesma; o hospedeiro prova que o que está no banco
// chega até lá.

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
