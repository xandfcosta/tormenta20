package tabuleiro

import (
	"fmt"
	"t20engine/aovivo"
	"testing"

	"t20engine/engine"
)

// combatenteDeFicha monta uma entrada de PC. Helper de teste local: o do
// `aovivo` mudou de pacote na ALE-254 e teste não exporta para o vizinho.
func combatenteDeFicha(label string, init int, charID int64) aovivo.InitiativeEntry {
	c := charID
	return aovivo.InitiativeEntry{Label: label, Initiative: init, Type: "character", CharacterID: &c}
}

// ContadorDeIds gera ids previsíveis para o teste. Era um helper compartilhado
// no `session_state_test.go`; quando aquele arquivo mudou de pacote (ALE-254) o
// helper foi junto, e teste não exporta para o vizinho — cada pacote tem o seu.
func ContadorDeIds() func() string {
	n := 0
	return func() string { n++; return "e" + itoaLocal(n) }
}

func itoaLocal(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte(48 + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// Mover peça no tabuleiro (ALE-124, fatia 3). O que se prova aqui é o que a mesa
// notaria quebrar: o jogador andando na vez de outro, a peça furando o
// deslocamento do livro, e o mestre impedido de fazer o que o mestre faz.
//
// A conta em si (diagonal custa o dobro, T20 p238) é do motor e está provada em
// `engine/board_movement_rules_test.go` — repeti-la aqui seria a terceira cópia
// da mesma regra.

// mesaEmCombate monta um tabuleiro com a peça do jogador na vez e um NPC fora.
func mesaEmCombate(t *testing.T) (*BoardState, *aovivo.SessionRuntimeState) {
	t.Helper()
	st := aovivo.EmptyRuntimeState()
	id := ContadorDeIds()
	_ = aovivo.AddEntry(st, combatenteDeFicha("Sílfide", 18, 7), id) // e1
	_ = aovivo.AddEntry(st, npc("Ogro", 12), id)                     // e2
	st.TurnIndex = 0

	b := newBoard("Taverna do Javali", "pedra")
	tokens := boardCounter()
	heroi := int64(7)
	_ = AddToken(b, BoardToken{Label: "Sílfide", X: 0, Y: 0, EntryID: strPtr("e1"), CharacterID: &heroi, SpeedSquares: 6}, tokens)
	_ = AddToken(b, BoardToken{Label: "Ogro", X: 9, Y: 9, EntryID: strPtr("e2")}, tokens)
	return b, st
}

func caminho(squares ...[2]int) []engine.Square {
	path := make([]engine.Square, len(squares))
	for i, s := range squares {
		path[i] = engine.Square{X: s[0], Y: s[1]}
	}
	return path
}

var (
	jogadorDono = Mover{UserID: 42, Role: "player", OwnsCharacter: true}
	mestre      = Mover{UserID: 1, Role: "gm"}
)

// O jogador DESENHA com a própria peça na própria vez, e o caminho é MEDIDO:
// seis quadrados de deslocamento (T20 p106: 9m) compram seis passos ortogonais.
//
// O que ele pode fazer é o desenho — quem o transforma em pouso é o mestre, e o
// guarda disso é o `TestOJogadorDesenhaMasNaoPoeAPecaNoLugar`. Aqui o que se
// prende é a outra metade: que ele PODE desenhar, e que a medida sai certa.
func TestJogadorDesenhaComAPropriaPecaNaPropriaVez(t *testing.T) {
	b, st := mesaEmCombate(t)

	err := ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}, [2]int{2, 0}), jogadorDono)
	if err != nil {
		t.Fatalf("desenho legítimo recusado: %v", err)
	}
	if b.Pending == nil || b.Pending.Cost != 2 {
		t.Fatalf("provisório saiu como %+v, esperava custo 2", b.Pending)
	}
	// A peça NÃO se mexeu: o desenho é intenção, e quem pousa é o mestre.
	if b.Tokens[0].X != 0 {
		t.Errorf("a peça andou com o desenho do jogador: x=%d", b.Tokens[0].X)
	}

	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("o mestre confirmar o desenho do jogador: %v", err)
	}
	if b.Tokens[0].X != 2 || b.Tokens[0].Y != 0 {
		t.Errorf("a peça pousou em (%d,%d), esperava (2,0)", b.Tokens[0].X, b.Tokens[0].Y)
	}
	if b.Pending != nil {
		t.Error("o provisório sobreviveu à confirmação")
	}
}

// O DESENHO DO JOGADOR É SÓ VISUAL: ele não põe a peça no lugar.
//
// As palavras do dono: *"o desenho do movimento do jogador é sempre só visual
// para o mestre e outros jogadores entenderem o que ele quer fazer"*. Quem muda
// o estado do tabuleiro é o mestre, e só ele.
//
// Isto SUBSTITUI a trava de deslocamento que morava aqui. Ela existia porque o
// jogador chegava direto ao estado da cena e precisava de um guarda no servidor;
// com o confirmar sendo do mestre, o guarda perdeu o objeto — e o deslocamento
// virou desenho (as três faixas da seta), não recusa.
func TestOJogadorDesenhaMasNaoPoeAPecaNoLugar(t *testing.T) {
	b, st := mesaEmCombate(t)

	if err := ProposeMove(b, st, "t1",
		caminho([2]int{0, 0}, [2]int{1, 1}, [2]int{2, 2}, [2]int{3, 3}), jogadorDono); err != nil {
		t.Fatalf("o jogador não conseguiu DESENHAR o movimento dele: %v", err)
	}
	if b.Pending == nil {
		t.Fatal("o desenho do jogador não ficou na cena: a mesa não vê o que ele quer fazer")
	}

	if err := CommitMove(b, st, b.Version, jogadorDono); err == nil {
		t.Fatal("o jogador confirmou o próprio movimento: quem põe a peça no lugar é o mestre")
	}
	if b.Tokens[0].X != 0 || b.Tokens[0].Y != 0 {
		t.Errorf("a peça andou para (%d,%d) por conta do jogador", b.Tokens[0].X, b.Tokens[0].Y)
	}
	// E o DESENHO SOBREVIVE à recusa: ele é o produto do gesto do jogador, e
	// apagá-lo aqui tiraria da mesa exatamente o que ela precisa ver para decidir.
	if b.Pending == nil {
		t.Error("a recusa jogou fora o desenho do jogador")
	}
}

// O MESTRE NÃO TEM LIMITE: ele põe a peça onde quiser, inclusive além do que ela
// anda num turno.
//
// As palavras do dono: *"o mestre não tem limite, ele faz o que quiser no
// tabuleiro, mas a parte visual serve para todos"*. Quatro diagonais custam 8
// (T20 p238) sobre um deslocamento de 6 — antes esta era a recusa, e agora é um
// pouso com a seta contando a história em azul.
func TestOMestrePousaAPecaAlemDoDeslocamento(t *testing.T) {
	b, st := mesaEmCombate(t)

	if err := ProposeMove(b, st, "t1",
		caminho([2]int{0, 0}, [2]int{1, 1}, [2]int{2, 2}, [2]int{3, 3}, [2]int{4, 4}), mestre); err != nil {
		t.Fatalf("propor um caminho caro: %v", err)
	}
	if b.Pending.Cost != 8 {
		t.Errorf("o caminho custou %d, esperado 8 (quatro diagonais, p238)", b.Pending.Cost)
	}
	// O ORÇAMENTO DE DESENHO é o deslocamento da PEÇA, mesmo para o mestre: é ele
	// que parte a seta nas três faixas, e -1 esconderia da mesa o que ela quer ver.
	if b.Pending.Budget != 6 {
		t.Errorf("o mestre desenhou com orçamento %d, esperado o deslocamento da peça (6)", b.Pending.Budget)
	}

	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("o mestre foi barrado pelo deslocamento, e ele não tem limite: %v", err)
	}
	if b.Tokens[0].X != 4 || b.Tokens[0].Y != 4 {
		t.Errorf("a peça pousou em (%d,%d), esperava (4,4)", b.Tokens[0].X, b.Tokens[0].Y)
	}
}

// TestOCONTROLE: o mestre pousando um caminho que CABE.
//
// Sem ele, "o mestre pousou o caro" não se distingue de "o mestre pousa
// qualquer coisa porque o confirmar deixou de conferir a vez e a posse".
func TestOMestrePousaOCaminhoQueCabe(t *testing.T) {
	b, st := mesaEmCombate(t)

	if err := ProposeMove(b, st, "t1",
		caminho([2]int{0, 0}, [2]int{1, 1}, [2]int{2, 2}, [2]int{3, 3}), mestre); err != nil {
		t.Fatalf("propor seis quadrados sobre um deslocamento de seis: %v", err)
	}
	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("confirmar um caminho que cabe: %v", err)
	}
	if b.Tokens[0].X != 3 || b.Tokens[0].Y != 3 {
		t.Errorf("a peça pousou em (%d,%d), esperava (3,3)", b.Tokens[0].X, b.Tokens[0].Y)
	}
}

// A vez é do rastreador, não do tabuleiro: fora dela o jogador não anda.
func TestJogadorNaoAndaForaDaPropriaVez(t *testing.T) {
	b, st := mesaEmCombate(t)
	st.TurnIndex = 1 // a vez é do Ogro

	if err := ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}), jogadorDono); err == nil {
		t.Fatal("o jogador andou na vez do Ogro")
	}
}

// A peça do vizinho não é da pessoa, mesmo estando na vez dela.
func TestJogadorNaoMovePecaDeOutro(t *testing.T) {
	b, st := mesaEmCombate(t)
	naoDono := Mover{UserID: 42, Role: "player", OwnsCharacter: false}

	if err := ProposeMove(b, st, "t2", caminho([2]int{9, 9}, [2]int{8, 9}), naoDono); err == nil {
		t.Fatal("o jogador moveu a peça do Ogro")
	}
}

// O mestre move qualquer peça, a qualquer hora, SEM ser barrado — é a saída para
// voo, empurrão, teleporte e "pode ir" — e mesmo assim VÊ o deslocamento dela.
//
// As duas metades são a frase do dono inteira: *"o mestre não tem limite, ele faz
// o que quiser no tabuleiro, mas a parte visual serve para todos"*. O orçamento
// deixou de ser permissão e virou desenho, então mandá-lo como -1 aqui apagaria
// da tela do mestre as três faixas que a mesa está lendo.
func TestOMestreMoveSemLimiteMasVeODeslocamentoDaPeca(t *testing.T) {
	b, st := mesaEmCombate(t)

	longe := caminho([2]int{9, 9}, [2]int{10, 10}, [2]int{11, 11}, [2]int{12, 12},
		[2]int{13, 13}, [2]int{14, 14}, [2]int{15, 15}, [2]int{16, 16})
	if err := ProposeMove(b, st, "t2", longe, mestre); err != nil {
		t.Fatalf("o mestre foi barrado: %v", err)
	}
	if b.Pending.Budget != speedOf(b.Tokens[1]) {
		t.Errorf("o mestre desenhou com orçamento %d, esperava o deslocamento da peça (%d)",
			b.Pending.Budget, speedOf(b.Tokens[1]))
	}
	// E ele POUSA: catorze quadrados de caminho sobre um deslocamento de seis.
	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("o mestre foi barrado no confirmar, e ele não tem limite: %v", err)
	}
}

// Fora de combate não existe vez nem deslocamento de turno: a cena da taverna
// também tem posição, e ali cada um anda com a própria peça.
func TestForaDeCombateCadaUmAndaComASua(t *testing.T) {
	b, st := mesaEmCombate(t)
	st.TurnIndex = -1

	longe := caminho([2]int{0, 0}, [2]int{1, 1}, [2]int{2, 2}, [2]int{3, 3}, [2]int{4, 4})
	if err := ProposeMove(b, st, "t1", longe, jogadorDono); err != nil {
		t.Fatalf("fora de combate o jogador foi barrado: %v", err)
	}
	if b.Pending.Budget != -1 {
		t.Errorf("orçamento %d fora de combate, esperava nenhum (-1)", b.Pending.Budget)
	}
}

// A versão é o que impede o commit de ser escrito sobre outra cena: dois
// clientes na mesma peça, o mestre arrastando enquanto o jogador confirma, e o
// broadcast atrasado que chega depois da re-hidratação.
func TestCommitSobreTabuleiroMudadoERecusado(t *testing.T) {
	b, st := mesaEmCombate(t)
	_ = ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}), jogadorDono)
	vista := b.Version

	// O mestre mexe em outra peça: a cena que o jogador tinha na mão não existe mais.
	_ = UpdateToken(b, "t2", tokenPatch{X: intPtr(5), Y: intPtr(5)})

	if err := CommitMove(b, st, vista, mestre); err == nil {
		t.Fatal("o commit passou por cima de um tabuleiro que já tinha mudado")
	}
	if b.Pending == nil {
		t.Error("o provisório foi perdido junto com a recusa: o jogador refaz o movimento do zero")
	}
	// Com a versão em dia, o mesmo commit entra.
	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("commit com a versão em dia recusado: %v", err)
	}
}

// O mestre confirma pelo jogador — é ele quem toca a mesa quando o jogador
// travou ou caiu da rede. O contrário não vale: um jogador não decide o
// provisório de outro.
func TestMestreConfirmaPeloJogadorEOContrarioNao(t *testing.T) {
	b, st := mesaEmCombate(t)
	_ = ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}), jogadorDono)

	outroJogador := Mover{UserID: 99, Role: "player", OwnsCharacter: true}
	if err := CommitMove(b, st, b.Version, outroJogador); err == nil {
		t.Fatal("um jogador confirmou o movimento proposto por outro")
	}
	if err := CommitMove(b, st, b.Version, mestre); err != nil {
		t.Fatalf("o mestre não pôde confirmar pelo jogador: %v", err)
	}
}

// Peça escondida some inteira da cópia do jogador — e o provisório dela também,
// senão um caminho desenhado saindo do nada entregaria a emboscada.
func TestProvisorioDePecaEscondidaNaoVazaParaOJogador(t *testing.T) {
	b, st := mesaEmCombate(t)
	_ = UpdateToken(b, "t2", tokenPatch{Hidden: boolPtr(true)})
	_ = ProposeMove(b, st, "t2", caminho([2]int{9, 9}, [2]int{8, 9}), mestre)

	visto := BoardForRole("player", b)

	if visto.Pending != nil {
		t.Errorf("o provisório da peça escondida saiu para o jogador: %+v", visto.Pending)
	}
	if len(visto.Tokens) != 1 {
		t.Errorf("o jogador viu %d peças, esperava só a dele", len(visto.Tokens))
	}
	// E o mestre continua vendo tudo.
	if BoardForRole("gm", b).Pending == nil {
		t.Error("o mestre perdeu o próprio provisório")
	}
}

// A peça removida leva o provisório junto: um movimento para quem não está mais
// no tabuleiro nunca poderia ser confirmado e ficaria pendurado no estado.
func TestRemoverAPecaLevaOProvisorioJunto(t *testing.T) {
	b, st := mesaEmCombate(t)
	_ = ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}), jogadorDono)

	RemoveToken(b, "t1")

	if b.Pending != nil {
		t.Errorf("o provisório sobreviveu à peça: %+v", b.Pending)
	}
}

// O caminho tem de começar onde a peça está: um caminho que nasce em outro
// lugar é um cliente desatualizado, e aceitá-lo teleportaria a peça.
func TestCaminhoTemDeComecarNaPeca(t *testing.T) {
	b, st := mesaEmCombate(t)

	if err := ProposeMove(b, st, "t1", caminho([2]int{5, 5}, [2]int{6, 5}), jogadorDono); err == nil {
		t.Fatal("caminho que começa longe da peça foi aceito")
	}
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }

// "Trazer a iniciativa" entrega uma CENA, não uma fila (ALE-166).
//
// Antes todos caíam numa fileira única no meio do mapa, e esse é o estado em
// que o mestre encontra o tabuleiro no segundo em que o combate começa: ele
// tinha de arrastar nove peças antes de a cena servir para alguma coisa.
func TestPopulateStartsTheSidesApart(t *testing.T) {
	st := aovivo.EmptyRuntimeState()
	id := ContadorDeIds()
	_ = aovivo.AddEntry(st, combatenteDeFicha("Sílfide", 18, 7), id)
	_ = aovivo.AddEntry(st, combatenteDeFicha("Paladino", 15, 8), id)
	_ = aovivo.AddEntry(st, npc("Ogro", 12), id)
	_ = aovivo.AddEntry(st, npc("Goblin", 9), id)
	b := newBoard("Cripta", "pedra")

	populateBoard(b, st, boardCounter(), nil)

	pcs, npcs := []BoardToken{}, []BoardToken{}
	for _, token := range b.Tokens {
		if token.Kind == "character" {
			pcs = append(pcs, token)
			continue
		}
		npcs = append(npcs, token)
	}
	if len(pcs) != 2 || len(npcs) != 2 {
		t.Fatalf("separou %d personagens e %d do outro lado", len(pcs), len(npcs))
	}

	// Ninguém nasce dentro do bloco do outro lado.
	for _, pc := range pcs {
		for _, inimigo := range npcs {
			if pc.X == inimigo.X && pc.Y == inimigo.Y {
				t.Fatalf("%s e %s nasceram no mesmo quadrado", pc.Label, inimigo.Label)
			}
		}
	}

	// A distância entre as bordas é de 6 quadrados — 9m, o alcance CURTO do
	// livro (T20 p224): perto o bastante para a briga começar sem ninguém
	// atravessar meia tela, longe o bastante para o primeiro turno ter escolha.
	maisADireitaDoGrupo, maisAEsquerdaDoInimigo := pcs[0].X, npcs[0].X
	for _, pc := range pcs {
		if pc.X > maisADireitaDoGrupo {
			maisADireitaDoGrupo = pc.X
		}
	}
	for _, inimigo := range npcs {
		if inimigo.X < maisAEsquerdaDoInimigo {
			maisAEsquerdaDoInimigo = inimigo.X
		}
	}
	if vao := maisAEsquerdaDoInimigo - maisADireitaDoGrupo; vao < 4 {
		t.Errorf("os dois lados nasceram a %d quadrados um do outro — perto demais para ser começo de combate", vao)
	}
}

// Continua idempotente: clicar duas vezes não duplica ninguém, e quem já está
// posicionado NÃO é movido — o mestre pode ter colocado o vilão onde queria
// antes de trazer o resto.
func TestPopulateLeavesWhoIsAlreadyThere(t *testing.T) {
	st := aovivo.EmptyRuntimeState()
	id := ContadorDeIds()
	_ = aovivo.AddEntry(st, npc("Ogro", 12), id)
	b := newBoard("Cripta", "pedra")
	tokens := boardCounter()
	_ = AddToken(b, BoardToken{Label: "Ogro", X: 40, Y: 40, EntryID: strPtr("e1")}, tokens)

	populateBoard(b, st, tokens, nil)
	populateBoard(b, st, tokens, nil)

	if len(b.Tokens) != 1 {
		t.Fatalf("o tabuleiro ficou com %d peças, esperava 1", len(b.Tokens))
	}
	if b.Tokens[0].X != 40 || b.Tokens[0].Y != 40 {
		t.Errorf("a peça já posicionada foi movida para (%d,%d)", b.Tokens[0].X, b.Tokens[0].Y)
	}
}

// Duas peças avulsas criadas seguidas não nascem uma em cima da outra — o
// defeito que o "+ Peça" da ALE-178 traria com a posição fixa em (0,0).
func TestLoosePiecesDoNotStack(t *testing.T) {
	b := newBoard("Cripta", "pedra")
	tokens := boardCounter()

	for _, nome := range []string{"Porta", "Baú", "Barril"} {
		spot := nextFreeSpot(b)
		if err := AddToken(b, BoardToken{Label: nome, Kind: "object", X: spot.x, Y: spot.y}, tokens); err != nil {
			t.Fatalf("criar %s: %v", nome, err)
		}
	}

	vistos := map[string]bool{}
	for _, token := range b.Tokens {
		chave := fmt.Sprintf("%d,%d", token.X, token.Y)
		if vistos[chave] {
			t.Errorf("duas peças no mesmo quadrado %s", chave)
		}
		vistos[chave] = true
	}
}

// O terreno que o mestre PINTA chega à régua (ALE-124, fatia 4).
//
// A conta do dobro é do motor e está provada contra a p238 em
// `engine/board_movement_rules_test.go`. O que se prova AQUI é a ligação, que é
// onde ela pode se perder: até esta fatia o estado nem tinha onde guardar o
// chão, e o `ProposeMove` chamava a régua com um mapa VAZIO — o mestre pintava
// o brejo e a peça o atravessava como se fosse pedra lisa.
func TestOTerrenoPintadoEncareceOCaminho(t *testing.T) {
	b, st := mesaEmCombate(t)
	// Quatro passos ortogonais custam 4 de 6 — cabe com folga.
	reto := caminho([2]int{0, 0}, [2]int{1, 0}, [2]int{2, 0}, [2]int{3, 0}, [2]int{4, 0})
	if err := ProposeMove(b, st, "t1", reto, jogadorDono); err != nil {
		t.Fatalf("caminho de quatro quadrados em chão limpo foi recusado: %v", err)
	}
	if b.Pending.Cost != 4 {
		t.Fatalf("chão limpo custou %d, esperado 4", b.Pending.Cost)
	}
	b.Pending = nil

	// O mestre pinta DUAS casas do caminho: cada uma passa a custar 2.
	PaintTerrain(b, engine.Square{X: 2, Y: 0}, TerrenoDificil, true)
	PaintTerrain(b, engine.Square{X: 3, Y: 0}, TerrenoDificil, true)

	if err := ProposeMove(b, st, "t1", reto, jogadorDono); err != nil {
		t.Fatalf("seis quadrados de custo num deslocamento de seis foram recusados: %v", err)
	}
	if b.Pending.Cost != 6 {
		t.Errorf("com duas casas difíceis o caminho custou %d, esperado 6 (T20 p238)", b.Pending.Cost)
	}
}

// Quem apaga é a BORRACHA, mandando `false` — e pintar duas vezes a mesma casa
// tem de ser inofensivo, porque o pincel pinta arrastando e o arraste passa
// pela mesma casa várias vezes. Alternar faria a casa piscar debaixo do dedo.
func TestPintarEApagarSaoExplicitosEIdempotentes(t *testing.T) {
	b, _ := mesaEmCombate(t)
	casa := engine.Square{X: 2, Y: 0}

	PaintTerrain(b, casa, TerrenoDificil, true)
	if len(b.Difficult) != 1 {
		t.Fatalf("pintar não marcou a casa: %+v", b.Difficult)
	}
	versaoDepoisDePintar := b.Version

	// O arraste passando de novo: nada muda, e a versão NÃO sobe à toa — cada
	// subida é um broadcast para a mesa inteira.
	PaintTerrain(b, casa, TerrenoDificil, true)
	if len(b.Difficult) != 1 || b.Version != versaoDepoisDePintar {
		t.Errorf("pintar de novo mexeu no estado: %d casas, versão %d", len(b.Difficult), b.Version)
	}

	PaintTerrain(b, casa, TerrenoDificil, false)

	if len(b.Difficult) != 0 {
		t.Errorf("a borracha não apagou: %+v", b.Difficult)
	}
	if b.Version <= versaoDepoisDePintar {
		t.Errorf("apagar não subiu a versão: %d", b.Version)
	}
}

// O provisório carrega a CONTA, não só o total (ALE-190): quantos passos
// dobraram por diagonal e quantos por terreno difícil. É esse estado que deixa
// a tela NOMEAR a regra do livro em vez de refazer a aritmética em JavaScript —
// uma segunda implementação livre para divergir do motor é a classe de defeito
// que a ALE-104 apagou.
func TestProvisorioCarregaAContaQueProduziuOCusto(t *testing.T) {
	b, st := mesaEmCombate(t)
	b.Difficult = []engine.Square{{X: 1, Y: 0}}

	// Um reto no brejo (2) e uma diagonal limpa (2): custo 4, uma causa de cada.
	err := ProposeMove(b, st, "t1", caminho([2]int{0, 0}, [2]int{1, 0}, [2]int{2, 1}), jogadorDono)
	if err != nil {
		t.Fatalf("movimento legítimo recusado: %v", err)
	}
	if b.Pending == nil {
		t.Fatal("nenhum provisório")
	}
	if b.Pending.Cost != 4 {
		t.Fatalf("custo = %d, esperado 4 (2 do brejo + 2 da diagonal)", b.Pending.Cost)
	}
	if b.Pending.Diagonals != 1 || b.Pending.Difficult != 1 {
		t.Errorf(
			"a conta não viajou: diagonais=%d difícil=%d, esperado 1 e 1",
			b.Pending.Diagonals, b.Pending.Difficult,
		)
	}
}
