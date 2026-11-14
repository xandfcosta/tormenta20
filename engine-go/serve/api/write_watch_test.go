package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"t20engine/infra/events"
)

// TODA GRAVAÇÃO QUE A MESA MOSTRA PUBLICA O EVENTO DELA (ALE-639).
//
// O fluxo da Mesa só se redesenha quando um evento avisa (CQRS, decisão do dono): uma gravação sem evento
// deixa a tela de todo mundo parada, sem erro em lugar nenhum. Este detector é o guarda disso, e ele vê o
// que o código FAZ, e não o que ele diz: o gancho de escrita do SQLite (`db.OpenObserved`) anota cada
// linha gravada durante um pedido da bancada (`everyRequest`, em todo pedido aos roteadores), um espião anota cada evento
// publicado, e no fim do pedido toda gravação numa tabela que a Mesa mostra precisa ter o evento do ALVO
// dela — a ficha, a sessão ou a campanha. As violações juntam-se e reprovam a suíte no fim, no `TestMain`,
// com a rota e a tabela.
//
// O TERRENO é a suíte inteira: os pedidos dos testes de cena são os gestos de verdade. Rota que nenhum
// teste exercita não é medida — e é por isso que gesto novo pede teste.
//
// O gancho existe SÓ aqui. O vocabulário da produção continua por ATO (o `infra.md`): o gancho é o
// detector, e não o despertador.

// tableWrite é uma linha gravada, com o alvo que ela toca.
type tableWrite struct {
	table                        string
	character, session, campaign int64
	// anyCharacter: a tabela é da ficha mas não carrega o id dela (os encantos, as doses e as escolhas
	// do item moram pelo `itemId`); qualquer evento de ficha satisfaz.
	anyCharacter bool
}

// writeWatch é o detector de um servidor de teste.
type writeWatch struct {
	mu      sync.Mutex
	columns map[string][]string
	writes  []tableWrite
	unknown map[string]bool
	events  events.Subscription
	active  bool
}

var (
	watches          sync.Map // *Server → *writeWatch
	watchViolationsM sync.Mutex
	watchViolations  = map[string]bool{}
)

// unseenTables são as tabelas que a Mesa NÃO mostra, cada uma com o porquê. Tabela nova cai fora desta
// lista e das famílias de `targetOf`, e reprova como "sem classificação" até alguém decidir.
var unseenTables = map[string]string{
	"users":                "a conta: a Mesa mostra o nome do personagem, e não o da conta",
	"password_resets":      "a troca de senha",
	"account_invite_links": "o convite de cadastro",
	"account_invite_uses":  "o convite de cadastro",
	"goose_db_version":     "as migrações",
	"sqlite_sequence":      "o contador interno do SQLite",
	"campaign_items":       "a emenda do catálogo: muda o livro da campanha, e a ficha recalcula ao ser lida",
}

// newWriteWatch lê as colunas de cada tabela no banco de `path` antes de o servidor abrir: dentro do
// gancho não se pode consultar o banco.
func newWriteWatch(t *testing.T, path string) *writeWatch {
	t.Helper()
	plain, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("abrir %s para ler as colunas: %v", path, err)
	}
	defer func() { _ = plain.Close() }()
	rows, err := plain.Query(`SELECT m.name, p.name FROM sqlite_master m, pragma_table_info(m.name) p
		WHERE m.type = 'table' ORDER BY m.name, p.cid`)
	if err != nil {
		t.Fatalf("ler as colunas de %s: %v", path, err)
	}
	defer func() { _ = rows.Close() }()
	w := &writeWatch{columns: map[string][]string{}, unknown: map[string]bool{}}
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			t.Fatalf("ler uma coluna: %v", err)
		}
		w.columns[table] = append(w.columns[table], column)
	}
	return w
}

// observe pendura os ganchos numa conexão do pool. A linha gravada fica PENDENTE na conexão até a
// transação dela fechar: o commit a conta, e o rollback a esquece — o ritual que falha e desfaz a cobrança
// gravou e voltou atrás, e não mudou nada que a Mesa mostre.
func (w *writeWatch) observe(conn sqlite.HookRegisterer) {
	var pending []tableWrite
	unknown := map[string]bool{}
	conn.RegisterPreUpdateHook(func(data sqlite.SQLitePreUpdateData) {
		columns := w.columns[data.TableName]
		values := make([]any, len(columns))
		if data.Op == sqlite3.SQLITE_DELETE {
			_ = data.Old(values...)
		} else {
			_ = data.New(values...)
		}
		// A REGRAVAÇÃO IGUAL não é mudança: o `UPDATE` que põe na linha o que ela já tinha não muda nada que
		// a Mesa mostre, e não deve evento a ninguém.
		if data.Op == sqlite3.SQLITE_UPDATE {
			before := make([]any, len(columns))
			_ = data.Old(before...)
			if reflect.DeepEqual(before, values) {
				return
			}
		}
		row := map[string]int64{}
		for i, column := range columns {
			if id, ok := values[i].(int64); ok {
				row[column] = id
			}
		}
		write, seen := targetOf(data.TableName, row, columns)
		switch {
		case !seen:
		case write.table == "":
			unknown[data.TableName] = true
		default:
			pending = append(pending, write)
		}
	})
	conn.RegisterCommitHook(func() int32 {
		w.mu.Lock()
		if w.active {
			w.writes = append(w.writes, pending...)
			for table := range unknown {
				w.unknown[table] = true
			}
		}
		w.mu.Unlock()
		pending, unknown = nil, map[string]bool{}
		return 0
	})
	conn.RegisterRollbackHook(func() {
		pending, unknown = nil, map[string]bool{}
	})
}

// targetOf diz que alvo a linha gravada toca. `seen` falso é a tabela que a Mesa não mostra; `seen`
// verdadeiro com tabela vazia é a tabela que ninguém classificou.
func targetOf(table string, row map[string]int64, columns []string) (tableWrite, bool) {
	if _, unseen := unseenTables[table]; unseen {
		return tableWrite{}, false
	}
	has := func(c string) bool {
		for _, column := range columns {
			if column == c {
				return true
			}
		}
		return false
	}
	switch {
	case table == "characters":
		return tableWrite{table: table, character: row["id"]}, true
	case strings.HasPrefix(table, "character_") || table == "active_effects":
		if has("characterId") {
			return tableWrite{table: table, character: row["characterId"]}, true
		}
		return tableWrite{table: table, anyCharacter: true}, true
	case table == "sessions":
		return tableWrite{table: table, session: row["id"]}, true
	case table == "open_boards":
		return tableWrite{table: table, session: row["sessionId"]}, true
	case table == "campaigns":
		return tableWrite{table: table, campaign: row["id"]}, true
	case strings.HasPrefix(table, "campaign_"):
		return tableWrite{table: table, campaign: row["campaignId"]}, true
	}
	return tableWrite{}, true
}

// everyRequest é o middleware que os roteadores da bancada penduram (`Server.watchRequests`): cada
// pedido abre a janela e a fecha, venha ele do `f.requests`, do `f.posts` ou de um teste que chama o
// roteador direto. Ficar no ROTEADOR, e não num helper da bancada, é o que o impede de ficar cego para o
// próximo teste que chamar o roteador por fora — foi assim que o `Seat` gravou sem evento (ALE-645).
//
// O FLUXO LONGO fica de fora: ele fica aberto enquanto a tela estiver, e a janela dele engoliria o que os
// pedidos de outros gestos gravam ao mesmo tempo. Ele só lê.
//
// A janela é UMA por servidor. Dois pedidos ao mesmo tempo (o `TestSimultaneousJoinsCreateOneMember`) se atropelam: o
// segundo `begin` esquece o que o primeiro gravou, e o detector deixa de ver, sem acusar à toa.
func (w *writeWatch) everyRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if liveStreamPath(r.URL.Path) {
			next.ServeHTTP(rw, r)
			return
		}
		w.begin()
		next.ServeHTTP(rw, r)
		for _, violation := range w.check(r.Method, r.URL.Path) {
			recordViolation(violation)
		}
	})
}

// liveStreamPath diz se o endereço é de um fluxo que fica aberto: o da Mesa, o da tela da mesa e o da
// ficha solta.
func liveStreamPath(path string) bool {
	return strings.HasSuffix(path, "/fluxo") || strings.HasSuffix(path, "/ao-vivo")
}

// begin abre a janela de um pedido: esquece o que veio antes, de outro pedido ou de trabalho de fundo.
func (w *writeWatch) begin() {
	w.mu.Lock()
	w.writes, w.active = nil, true
	w.mu.Unlock()
	w.drain()
}

func (w *writeWatch) drain() []events.Target {
	var targets []events.Target
	for {
		select {
		case ev := <-w.events.C:
			targets = append(targets, ev.Target())
		default:
			return targets
		}
	}
}

// numberInPath troca os ids do endereço por `{n}`, para a mesma rota contar uma vez.
var numberInPath = regexp.MustCompile(`/(\d+|[0-9a-f]{8}-[0-9a-f-]{27}|[a-z-]+-dmv[0-9a-z]+)(/|$)`)

// check fecha a janela e devolve o que gravou sem avisar.
func (w *writeWatch) check(method, path string) []string {
	published := w.drain()
	w.mu.Lock()
	writes, unknown := w.writes, w.unknown
	w.writes, w.active, w.unknown = nil, false, map[string]bool{}
	w.mu.Unlock()
	var violations []string
	route := method + " " + numberInPath.ReplaceAllString(numberInPath.ReplaceAllString(strings.SplitN(path, "?", 2)[0], "/{id}$2"), "/{id}$2")
	for table := range unknown {
		violations = append(violations, fmt.Sprintf("%s grava em %s, que ninguém classificou: ponha-a em `unseenTables` "+
			"(com o porquê) ou numa família de `targetOf`", route, table))
	}
	for _, write := range writes {
		if announced(write, published) {
			continue
		}
		violations = append(violations, fmt.Sprintf("%s grava em %s sem publicar o evento do alvo dela (%s; publicou %s)",
			route, write.table, write.describe(), describeTargets(published)))
	}
	return violations
}

func (w tableWrite) describe() string {
	switch {
	case w.character != 0:
		return fmt.Sprintf("ficha %d", w.character)
	case w.anyCharacter:
		return "alguma ficha"
	case w.session != 0:
		return fmt.Sprintf("sessão %d", w.session)
	default:
		return fmt.Sprintf("campanha %d", w.campaign)
	}
}

func announced(write tableWrite, published []events.Target) bool {
	for _, target := range published {
		switch {
		case write.anyCharacter && target.CharacterID != 0:
			return true
		case write.character != 0 && target.CharacterID == write.character:
			return true
		case write.session != 0 && target.SessionID == write.session:
			return true
		case write.campaign != 0 && target.CampaignID == write.campaign:
			return true
		}
	}
	return false
}

func recordViolation(v string) {
	watchViolationsM.Lock()
	watchViolations[v] = true
	watchViolationsM.Unlock()
}

// writeWatchBaseline é a DÍVIDA: os pares "rota grava em tabela" que ainda gravam sem evento. A linha de
// base só encolhe — par novo reprova, e par da lista que deixou de violar também, para sair dela.
const writeWatchBaseline = "testdata/write_watch_baseline.txt"

// violationKey é a violação sem o detalhe entre parênteses: a rota e a tabela.
func violationKey(v string) string {
	key := strings.SplitN(v, " sem publicar", 2)[0]
	return strings.SplitN(key, ", que ninguém classificou", 2)[0]
}

// reportWriteWatch reprova a suíte com a lista das gravações sem evento. Chamada pelo `TestMain`, depois
// de todos os testes: um teste no meio da fila não veria os que vêm depois dele.
func reportWriteWatch(code int) int {
	watchViolationsM.Lock()
	defer watchViolationsM.Unlock()
	// A suíte FILTRADA (`-run`) não exercita tudo: ela não pode dizer que um par da base saiu. Só a
	// rodada inteira julga a linha de base.
	if filtered() {
		return code
	}
	baseline := map[string]bool{}
	if raw, err := os.ReadFile(writeWatchBaseline); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
				baseline[line] = true
			}
		}
	}
	var fresh, stale []string
	violated := map[string]bool{}
	for v := range watchViolations {
		key := violationKey(v)
		violated[key] = true
		if !baseline[key] {
			fresh = append(fresh, v)
		}
	}
	for key := range baseline {
		if !violated[key] {
			stale = append(stale, key)
		}
	}
	if len(fresh) == 0 && len(stale) == 0 {
		return code
	}
	sort.Strings(fresh)
	sort.Strings(stale)
	fmt.Fprintf(os.Stderr, "\n--- FAIL: TestEveryWriteTheTableShowsPublishesItsEvent\n")
	if len(fresh) > 0 {
		fmt.Fprintf(os.Stderr, "    %d gravações que a Mesa mostra não publicaram o evento delas, e a tela de quem está "+
			"na Mesa ficaria parada (ver o cabeçalho de write_watch_test.go):\n", len(fresh))
		for _, v := range fresh {
			fmt.Fprintf(os.Stderr, "    %s\n", v)
		}
	}
	if len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "    %d linhas de %s deixaram de violar: tire-as de lá (a base só encolhe):\n",
			len(stale), writeWatchBaseline)
		for _, v := range stale {
			fmt.Fprintf(os.Stderr, "    %s\n", v)
		}
	}
	return 1
}

// filtered diz se a suíte rodou com `-run`: aí ela não exercita o terreno inteiro.
func filtered() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.run") {
			return true
		}
	}
	return false
}

// watchOf é o detector do servidor de teste, ou nil para o servidor montado sem ele.
func watchOf(s *Server) *writeWatch {
	if w, ok := watches.Load(s); ok {
		return w.(*writeWatch)
	}
	return nil
}

func describeTargets(targets []events.Target) string {
	if len(targets) == 0 {
		return "nada"
	}
	parts := make([]string, 0, len(targets))
	for _, t := range targets {
		parts = append(parts, fmt.Sprintf("{sessão %d ficha %d campanha %d}", t.SessionID, t.CharacterID, t.CampaignID))
	}
	return strings.Join(parts, " ")
}

// O DETECTOR acusa o que veio medir: a gravação na ficha sem o evento dela, e só ela. Sem esta prova, o
// verde da suíte inteira podia ser o de um gancho que nunca disparou.
func TestEveryWriteTheTableShowsPublishesItsEvent(t *testing.T) {
	f := newSceneFixture(t)
	watch := watchOf(f.s)
	write := func() {
		t.Helper()
		if _, err := f.s.db.Exec("UPDATE characters SET name = name || '!' WHERE id = ?", f.charID); err != nil {
			t.Fatalf("gravar a ficha %d: %v", f.charID, err)
		}
	}

	watch.begin()
	write()
	if got := watch.check("POST", "/sonda"); len(got) != 1 || !strings.Contains(got[0], "grava em characters") {
		t.Errorf("a ficha gravada sem evento deu %q; o detector devia acusá-la", got)
	}

	watch.begin()
	write()
	f.s.bus.Publish(events.CharacterChanged{CharacterID: f.charID})
	if got := watch.check("POST", "/sonda"); len(got) != 0 {
		t.Errorf("a ficha gravada COM o evento dela foi acusada: %q", got)
	}
}
