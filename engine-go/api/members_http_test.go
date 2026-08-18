package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"t20engine/db/sqlcgen"
)

// Gestão de membros pelo ROUTER real. `PATCH /campaigns/{cid}/members/{id}` promove
// alguém a mestre da mesa, e só tinha o 401 anônimo da tabela de rotas — nada provava
// que um membro comum não se promove sozinho, nem que um id de OUTRA mesa é recusado.

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
		Campaignid: campaignID, Characterid: pcID, Role: "player", Addedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed member: %v", err)
	}
	otherMember, err := s.queries.CreateMember(ctx, sqlcgen.CreateMemberParams{
		Campaignid: otherCamp, Characterid: otherPc, Role: "player", Addedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed other member: %v", err)
	}

	return memberFixture{
		s: s, owner: owner, player: player, campaignID: campaignID, memberID: member.ID,
		otherOwner: otherOwner, otherCamp: otherCamp, otherMemID: otherMember.ID,
	}
}

func (f memberFixture) roleOf(t *testing.T, memberID int64) string {
	t.Helper()
	m, err := f.s.queries.GetMember(context.Background(), memberID)
	if err != nil {
		t.Fatalf("ler membro %d: %v", memberID, err)
	}
	return m.Role
}

func (f memberFixture) patchRole(t *testing.T, caller, campaignID, memberID int64, role string) int {
	t.Helper()
	path := "/campaigns/" + strconv.FormatInt(campaignID, 10) + "/members/" + strconv.FormatInt(memberID, 10)
	return authed(t, f.s, caller, http.MethodPatch, path, `{"role":"`+role+`"}`).Code
}

func TestUpdateMemberRole(t *testing.T) {
	t.Run("o dono promove um membro a mestre", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.memberID, "gm"); code != http.StatusOK {
			t.Fatalf("code=%d, queria 200", code)
		}
		if got := f.roleOf(t, f.memberID); got != "gm" {
			t.Errorf("papel=%q, queria gm", got)
		}
	})

	// A escalação de privilégio: o próprio jogador se promovendo na mesa de outro.
	t.Run("o jogador não se promove", func(t *testing.T) {
		f := newMemberFixture(t)
		code := f.patchRole(t, f.player, f.campaignID, f.memberID, "gm")
		if code == http.StatusOK {
			t.Fatal("o jogador conseguiu se promover")
		}
		if got := f.roleOf(t, f.memberID); got != "player" {
			t.Errorf("papel=%q — a recusa devolveu %d mas GRAVOU mesmo assim", got, code)
		}
	})

	// O membro existe, mas é de outra mesa: o id sozinho não pode bastar.
	t.Run("membro de outra mesa é 404 e não muda", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.otherMemID, "gm"); code != http.StatusNotFound {
			t.Errorf("code=%d, queria 404", code)
		}
		if got := f.roleOf(t, f.otherMemID); got != "player" {
			t.Errorf("papel do vizinho=%q — mudou o membro de outra mesa", got)
		}
	})

	t.Run("papel fora da lista é recusado", func(t *testing.T) {
		f := newMemberFixture(t)
		if code := f.patchRole(t, f.owner, f.campaignID, f.memberID, "admin"); code == http.StatusOK {
			t.Fatal("papel inventado foi aceito")
		}
		if got := f.roleOf(t, f.memberID); got != "player" {
			t.Errorf("papel=%q, queria player intacto", got)
		}
	})
}

// Com o banco com problema, entrar na mesa não escreve nada e responde 500.
//
// HONESTIDADE SOBRE O QUE ESTE TESTE NÃO PROVA: ele não isola o `hasPc, _ :=`
// que a ALE-156 corrigiu. A trava consulta `campaign_members` e `characters`,
// exatamente as duas tabelas que a escrita também usa, então qualquer falha que
// derrube a checagem derruba a inserção junto — e o desfecho (500, nada
// escrito) fica igual com e sem a correção. Isolar exigiria um seam de injeção
// de falha por query, e trocar `*sqlcgen.Queries` por interface no servidor
// inteiro é caro demais para provar um ramo de quatro linhas.
//
// O que ele protege, e que vale: sob falha de banco, ninguém entra na mesa. Foi
// o SILÊNCIO que a auditoria achou lendo — `false` significando "pode entrar" —
// e a regra que fica escrita é: checagem de autorização ou de unicidade nunca
// descarta erro; na dúvida, nega.
func TestADatabaseErrorClosesTheUniquenessGate(t *testing.T) {
	f := newMemberFixture(t)
	outroHeroi := seedCharacter(t, f.s, f.owner, "Segundo Herói", 10, 10, 0, 0)
	antes := membersOf(t, f.s, f.campaignID)

	// A checagem de unicidade não consegue responder. Antes disto, o erro virava
	// "pode entrar" e o jogador ganhava um segundo PC na mesma mesa.
	if _, err := f.s.db.Exec("ALTER TABLE campaign_members RENAME TO campaign_members_fora"); err != nil {
		t.Fatalf("esconder a tabela: %v", err)
	}
	code := f.addMember(t, f.owner, f.campaignID, outroHeroi)
	if _, err := f.s.db.Exec("ALTER TABLE campaign_members_fora RENAME TO campaign_members"); err != nil {
		t.Fatalf("devolver a tabela: %v", err)
	}

	if code != http.StatusInternalServerError {
		t.Errorf("erro de banco respondeu %d — a trava tem de FECHAR, não abrir", code)
	}
	if depois := membersOf(t, f.s, f.campaignID); depois != antes {
		t.Errorf("entrou membro apesar do erro: %d → %d. O status importa menos que a escrita", antes, depois)
	}
}

// A entrada na mesa é UMA transação: se o membro não é criado, a cópia não
// fica. Cópia órfã é pior que nada — o `campaignHasCopyOf` passa a dizer "já
// está na mesa" e o herói fica impedido de entrar para sempre, sem membro
// nenhum para remover.
func TestAFailedJoinLeavesNoOrphanSnapshot(t *testing.T) {
	f := newMemberFixture(t)
	heroi := seedCharacter(t, f.s, f.owner, "Terceiro Herói", 10, 10, 0, 0)
	copiasAntes := copiesOf(t, f.s, heroi)

	// A criação do membro falha DEPOIS de o clone já ter acontecido.
	if _, err := f.s.db.Exec("DROP TABLE campaign_members"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	code := f.addMember(t, f.owner, f.campaignID, heroi)

	if code != http.StatusInternalServerError {
		t.Errorf("respondeu %d numa falha de escrita", code)
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

	if code := f.addMember(t, f.owner, outraMesa, heroi); code != http.StatusCreated {
		t.Fatalf("entrada legítima respondeu %d", code)
	}
	// E a cópia de mesa nasceu junto: é ela que entra, não o original (ALE-33).
	if copias := copiesOf(t, f.s, heroi); copias != 1 {
		t.Errorf("a mesa ficou com %d cópias do herói, esperava 1", copias)
	}
}

func (f memberFixture) addMember(t *testing.T, caller, campaignID, characterID int64) int {
	t.Helper()
	path := "/campaigns/" + strconv.FormatInt(campaignID, 10) + "/members"
	body := `{"characterId":` + strconv.FormatInt(characterID, 10) + `,"role":"player"}`
	return authed(t, f.s, caller, http.MethodPost, path, body).Code
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

// Dois cliques ao mesmo tempo não fazem dois personagens (ALE-156).
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
	mesa := seedCampaign(t, f.s, f.owner)
	heroi := seedCharacter(t, f.s, f.owner, "Herói Disputado", 10, 10, 0, 0)

	const pedidos = 8
	var wg sync.WaitGroup
	codigos := make([]int, pedidos)
	for i := 0; i < pedidos; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			codigos[n] = f.addMember(t, f.owner, mesa, heroi)
		}(i)
	}
	wg.Wait()

	criados := 0
	for _, code := range codigos {
		if code == http.StatusCreated {
			criados++
		}
	}
	if criados != 1 {
		t.Errorf("%d pedidos simultâneos criaram %d membros (códigos %v), esperava 1", pedidos, criados, codigos)
	}
	if n := membersOf(t, f.s, mesa); n != 1 {
		t.Errorf("a mesa ficou com %d membros", n)
	}
	// E nenhuma cópia sobrando: o pedido que perde a corrida desfaz o clone.
	if copias := copiesOf(t, f.s, heroi); copias != 1 {
		t.Errorf("sobraram %d cópias do herói, esperava 1", copias)
	}
	// Quem perde a corrida merece a resposta CERTA: 409 diz "alguém chegou
	// antes"; 500 diria "o servidor quebrou", que é falso e manda o jogador
	// tentar de novo achando que o app está com defeito.
	for _, code := range codigos {
		if code == http.StatusInternalServerError {
			t.Errorf("um perdedor recebeu 500 em vez de 409 (códigos %v)", codigos)
			break
		}
	}
}
