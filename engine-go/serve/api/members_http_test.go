package api

import (
	"context"
	"errors"
	"sync"
	"t20engine/app/campaign"
	"t20engine/infra/db/dbvalue"
	"testing"

	"t20engine/infra/db/sqlcgen"
)

// memberFixture: uma mesa do dono, um jogador membro, e uma segunda mesa com o membro
// dela — o vizinho que os testes usam para tentar atravessar a fronteira.
type memberFixture struct {
	s          *Server
	owner      int64
	player     int64
	campaignID int64
	memberID   int64
	otherOwner int64
	otherCamp  int64
	otherMemID int64
}

func newMemberFixture(t *testing.T) memberFixture {
	t.Helper()
	s := newTestServer(t)
	ctx := context.Background()
	owner := seedUser(t, s, "dono@t.com")
	player := seedUser(t, s, "jogador@t.com")
	otherOwner := seedUser(t, s, "vizinho@t.com")

	campaignID := seedCampaign(t, s, owner)
	otherCamp := seedCampaign(t, s, otherOwner)
	pcID := seedCharacter(t, s, player, "Herói", 20, 30, 5, 10)
	otherPc := seedCharacter(t, s, otherOwner, "Vizinho", 20, 30, 5, 10)

	member, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: pcID, Addedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed member: %v", err)
	}
	otherMember, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: otherCamp, Characterid: otherPc, Addedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed other member: %v", err)
	}

	return memberFixture{
		s: s, owner: owner, player: player, campaignID: campaignID, memberID: member.ID,
		otherOwner: otherOwner, otherCamp: otherCamp, otherMemID: otherMember.ID,
	}
}

// NÃO existe caso sobre promover um jogador a mestre porque NÃO existe o gesto:
// a rota que o fazia era inalcançável de qualquer cena, e foi apagada. Se a
// capacidade voltar a ser desejada, ela volta como gesto na cena das campanhas,
// com a regra no `joinTable`, que já sabe o que é um papel válido.

func TestADatabaseErrorClosesTheUniquenessGate(t *testing.T) {
	f := newMemberFixture(t)
	outroHeroi := seedCharacter(t, f.s, f.owner, "Segundo Herói", 10, 10, 0, 0)
	antes := membersOf(t, f.s, f.campaignID)

	// A checagem de unicidade não consegue responder. Antes disto, o erro virava
	// "pode entrar" e o jogador ganhava um segundo PC na mesma mesa.
	if _, err := f.s.db.Exec("ALTER TABLE campaign_members RENAME TO campaign_members_fora"); err != nil {
		t.Fatalf("esconder a tabela: %v", err)
	}
	err := f.addMember(t, f.owner, f.campaignID, outroHeroi)
	if _, err := f.s.db.Exec("ALTER TABLE campaign_members_fora RENAME TO campaign_members"); err != nil {
		t.Fatalf("devolver a tabela: %v", err)
	}

	if err == nil {
		t.Error("erro de banco passou — a trava tem de FECHAR, não abrir")
	}
	if depois := membersOf(t, f.s, f.campaignID); depois != antes {
		t.Errorf("entrou membro apesar do erro: %d → %d. O status importa menos que a escrita", antes, depois)
	}
}

// A entrada na mesa é UMA transação: se o membro não é criado, a cópia não
// fica. Cópia órfã é pior que nada — a deduplicação passa a dizer "já está na
// mesa" e o herói fica impedido de entrar para sempre, sem membro nenhum para
// remover.
func TestAFailedJoinLeavesNoOrphanSnapshot(t *testing.T) {
	f := newMemberFixture(t)
	heroi := seedCharacter(t, f.s, f.owner, "Terceiro Herói", 10, 10, 0, 0)
	copiasAntes := copiesOf(t, f.s, heroi)

	// A criação do membro falha DEPOIS de o clone já ter acontecido.
	if _, err := f.s.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	err := f.addMember(t, f.owner, f.campaignID, heroi)

	if err == nil {
		t.Error("a escrita falhou e o `joinTable` disse que deu certo")
	}
	if copias := copiesOf(t, f.s, heroi); copias != copiasAntes {
		t.Errorf("sobrou cópia órfã: %d → %d. O herói fica impedido de entrar para sempre", copiasAntes, copias)
	}
}

// A entrada normal continua funcionando — o guarda não pode fechar a porta de
// quem tem direito de entrar.
func TestJoiningStillWorks(t *testing.T) {
	f := newMemberFixture(t)
	heroi := seedCharacter(t, f.s, f.owner, "Quarto Herói", 10, 10, 0, 0)
	outraMesa := seedCampaign(t, f.s, f.owner)

	if err := f.addMember(t, f.owner, outraMesa, heroi); err != nil {
		t.Fatalf("entrada legítima foi recusada: %v", err)
	}
	// E a cópia de mesa nasceu junto: é ela que entra, não o original.
	if copias := copiesOf(t, f.s, heroi); copias != 1 {
		t.Errorf("a mesa ficou com %d cópias do herói, esperava 1", copias)
	}
}

// addMember chama o CASO DE USO direto, e não a rota: o que estes casos prendem
// é a TRAVA DE UNICIDADE do `Seat`, que nunca foi do transporte — a cena das
// campanhas senta pela mesma porta.
//
// Devolve ERRO em vez de status porque é a cena que traduz sentinela em FRASE.
func (f memberFixture) addMember(t *testing.T, caller, campaignID, characterID int64) error {
	t.Helper()
	return f.s.campaignSeating().Seat(context.Background(), caller, campaignID, characterID, "")
}

func membersOf(t *testing.T, s *Server, campaignID int64) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM campaign_members WHERE campaignId = ?`, campaignID).Scan(&n); err != nil {
		t.Fatalf("contar membros: %v", err)
	}
	return n
}

func copiesOf(t *testing.T, s *Server, sourceID int64) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM characters WHERE sourceCharacterId = ?`, sourceID).Scan(&n); err != nil {
		t.Fatalf("contar cópias: %v", err)
	}
	return n
}

// Dois cliques ao mesmo tempo não fazem dois personagens.
//
// A trava de unicidade é decidida no CÓDIGO, não no schema: uma pergunta ao
// banco seguida de uma escrita. Sem cuidado, dois pedidos simultâneos fazem as
// duas perguntas ANTES de qualquer escrita, recebem "não" os dois, e a mesa
// termina com dois personagens do mesmo jogador.
//
// Duas coisas resolvem, e MEDIDAS elas fazem trabalhos diferentes:
//
//   - a checagem REFEITA dentro da transação é o que torna o resultado CORRETO.
//     Sem ela, oito pedidos criaram QUATRO membros. A checagem de fora roda sem
//     transação nenhuma, então todos passam por ela.
//   - `_txlock=immediate` é o que torna o resultado HONESTO. Sem ele, um dos
//     perdedores recebe 500: com transação deferida, o SQLite recusa a escrita
//     sobre um snapshot que mudou (o correto), mas isso chega ao jogador como
//     erro do servidor. Com a trava no BEGIN, o perdedor espera, relê e recebe
//     o 409 que descreve o que houve — alguém chegou antes.
//
// Medido: sem `_txlock`, [409 409 500 409 409 201 409 409]; com ele, sete 409 e
// um 201.
//
// Oito pedidos porque um par é pouco para expor uma corrida: se ela existir, é
// quase certo que apareça, e o teste continua determinístico quando não existe.
func TestSimultaneousJoinsCreateOneMember(t *testing.T) {
	f := newMemberFixture(t)
	table := seedCampaign(t, f.s, f.owner)
	heroi := seedCharacter(t, f.s, f.owner, "Herói Disputado", 10, 10, 0, 0)

	const pedidos = 8
	var wg sync.WaitGroup
	erros := make([]error, pedidos)
	for i := 0; i < pedidos; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			erros[n] = f.addMember(t, f.owner, table, heroi)
		}(i)
	}
	wg.Wait()

	criados := 0
	for _, err := range erros {
		if err == nil {
			criados++
		}
	}
	if criados != 1 {
		t.Errorf("%d pedidos simultâneos criaram %d membros (erros %v), esperava 1", pedidos, criados, erros)
	}
	if n := membersOf(t, f.s, table); n != 1 {
		t.Errorf("a mesa ficou com %d membros", n)
	}
	// E nenhuma cópia sobrando: o pedido que perde a corrida desfaz o clone.
	if copias := copiesOf(t, f.s, heroi); copias != 1 {
		t.Errorf("sobraram %d cópias do herói, esperava 1", copias)
	}
	// Quem perde a corrida merece uma RECUSA, e não um erro de banco.
	//
	// São DOIS sentinelas, e não um, porque a corrida se perde em dois lugares:
	// quem chega atrasado na checagem de fora leva `ErrAlreadyHasHero`, e quem
	// passa por ela e perde a releitura DENTRO da transação leva
	// `ErrHeroAlreadyThere` — a trava dupla funcionando, e não um descuido.
	// Prender só o primeiro fazia este teste reprovar em três de dez corridas.
	for _, err := range erros {
		if err != nil && !errors.Is(err, campaign.ErrAlreadyHasHero) && !errors.Is(err, campaign.ErrHeroAlreadyThere) {
			t.Errorf("um perdedor recebeu um erro que não é recusa nenhuma: %v", err)
			break
		}
	}
}

// NÃO há caso aqui sobre o teto de 1 MB do corpo: a garantia não é da rota e sim
// do `httpio.DecodeJSON`, chamado por todo comando de cena. É lá que ela deve ser
// presa se alguém quiser um guarda dela.
