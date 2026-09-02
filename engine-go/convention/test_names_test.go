package convention

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

// portugueseMarkers são as palavras que NÃO cabem num nome de teste em inglês.
//
// A lista foi CALIBRADA, e não palpitada. A amostra é a árvore anterior à
// varredura, que é a única honesta que existe — 775 nomes sabidamente em
// português de um lado, e os 1.051 de hoje sabidamente em inglês do outro:
//
//	git archive 98067ad2 | tar -x -C /tmp/antes/engine-go
//	cp -r convention /tmp/antes/engine-go/ && cd /tmp/antes/engine-go
//	go test ./convention/          # 775 reprovados: a SENSIBILIDADE
//	                               # verde aqui na árvore de hoje: a ESPECIFICIDADE
//
// A especificidade tem de ser ZERO falso positivo: um guarda que cobra renome de
// um nome já certo é um guarda que alguém desliga.
//
// > A primeira calibração NÃO foi essa, e ela mentiu. Um script em Python
// > simulava o guarda para medir a lista — com outro algoritmo de quebrar
// > CamelCase, que separava `OBotao` em "O" e "Botao" enquanto o Go devolvia
// > "OBotao" inteiro. A sensibilidade medida era de palavras que o guarda nunca
// > produziria. Medir o instrumento com um segundo instrumento é medir o segundo.
//
// # O que ela deliberadamente NÃO pega
//
// Nome PRÓPRIO do livro. `TestBolaDeFogoWorkedExample`,
// `TestEspecializacaoEmArmadura`, `TestCaidoDefenseIsDirectionalAndCumulative`:
// "Bola de Fogo" e "Caído" são o nome da magia e da condição, não prosa em
// português, e o CLAUDE.md manda deixar termo sem tradução assentada como está.
// Então "armadura" e "em" ficam de fora, e um nome montado só com termo do livro
// passa. É uma fresta conhecida, e é mais barata que a alternativa: uma lista que
// pegasse esses nomes cobraria a tradução de um nome próprio.
//
// # O que acontece quando ela erra para o outro lado
//
// Se alguém escrever um nome novo em português com palavras que não estão aqui,
// ele passa. A lista cresce quando isso acontecer — e cresce com a medição do
// script, nunca por palpite.
var portugueseMarkers = map[string]bool{
	// funcionais
	"nao": true, "que": true, "com": true, "sem": true, "para": true,
	"pelo": true, "pela": true, "pelos": true, "pelas": true, "uma": true,
	"duas": true, "dois": true, "tres": true, "quem": true, "toda": true,
	"todo": true, "todas": true, "todos": true, "nenhum": true, "nenhuma": true,
	"cada": true, "mais": true, "menos": true, "ainda": true, "depois": true,
	"antes": true, "quando": true, "onde": true, "como": true, "porque": true,
	"mesmo": true, "mesma": true, "outra": true, "outro": true, "propria": true,
	"proprio": true, "junto": true, "fora": true, "dentro": true, "vazio": true,
	"vazia": true, "muito": true, "apenas": true, "tambem": true, "entre": true,
	// verbos
	"sao": true, "tem": true, "vem": true, "sai": true, "diz": true,
	"entra": true, "nasce": true, "volta": true, "devolve": true, "oferece": true,
	"desenha": true, "recusa": true, "recusado": true, "recusada": true,
	"leva": true, "guarda": true, "conta": true, "cabe": true, "acende": true,
	"mostra": true, "chega": true, "aparece": true, "existe": true, "segue": true,
	"pinta": true, "abre": true, "fecha": true, "manda": true, "escolhe": true,
	"limpa": true, "apaga": true, "muda": true, "anda": true, "mexe": true,
	// substantivos da mesa que JÁ têm grafia inglesa assentada no glossário
	"cena": true, "mesa": true, "mestre": true, "jogador": true, "peca": true,
	"ficha": true, "fila": true, "caminho": true, "aba": true, "pagina": true,
	"catalogo": true, "porta": true, "classe": true, "verbete": true,
	"acervo": true, "casa": true, "lista": true, "sessao": true, "condicao": true,
	"campanha": true, "livro": true, "busca": true, "linha": true, "nome": true,
	"servidor": true, "grupo": true, "vez": true, "tabuleiro": true,
	"pericia": true, "pericias": true, "personagem": true, "elenco": true,
	"marcador": true, "terreno": true, "especie": true, "rotulo": true,
	"trilho": true, "gabarito": true, "regua": true, "cortina": true,
	"lente": true, "palco": true, "pincel": true, "borracha": true,
	"parada": true, "paradas": true, "perna": true, "pernas": true,
	"traco": true, "ferramenta": true, "formulario": true, "folha": true,
	"dossie": true, "quadrinho": true, "rodape": true, "retangulo": true,
	"encontro": true, "fileira": true, "postura": true, "proficiencia": true,
	"atributo": true, "endereco": true, "rota": true, "carga": true,
	"defesa": true, "ataque": true, "origem": true, "oficio": true,
	"oficios": true, "dinheiro": true, "tabela": true, "papel": true,
	"turnos": true, "combatente": true, "mesas": true, "ilustracao": true, "esferas": true,
	"cubos": true, "faixas": true, "dificuldade": true,
	// segunda rodada de marcadores: cada um veio de um nome que ESCAPOU da
	// primeira, e a lista foi remedida rodando o guarda contra a árvore anterior.
	"ausencia": true, "copia": true, "ganha": true, "proximo": true,
	"numero": true, "livre": true, "grafia": true, "comeco": true,
	"arquivo": true, "ordem": true, "poe": true, "frente": true, "fim": true,
	"ponte": true, "pessoa": true, "dono": true, "saude": true, "raiz": true,
	"tela": true, "liga": true, "desliga": true, "validacao": true,
	"alcance": true, "losango": true, "pesada": true, "concede": true,
	"leve": true, "varre": true, "catalogos": true, "conjurar": true,
	"cobra": true, "editar": true, "remover": true, "entrar": true,
	"degraus": true, "pagamento": true, "equipar": true, "respeita": true,
	"eixo": true, "especies": true, "diferentes": true, "expira": true,
	"filtro": true, "separa": true, "guardar": true, "tira": true,
	"grade": true, "limpar": true, "zera": true, "polilinha": true,
	"atravessa": true, "dobra": true, "atalho": true, "fixo": true,
	"troca": true, "aceita": true, "seis": true, "buraco": true,
	"recebe": true, "gasta": true, "corrige": true, "limite": true,
	"palavra": true, "inteira": true, "caixa": true, "mantem": true,
	"fria": true, "fluxo": true, "redefinicao": true, "vale": true,
	"vinte": true, "quatro": true, "curinga": true, "antigo": true,
	"movimento": true, "pousa": true, "confirmar": true, "nivel": true,
	"tamanho": true, "absurdos": true, "caem": true, "padrao": true,
	"treino": true, "poder": true, "unico": true, "escolhivel": true,
	"tarefa": true, "reescreve": true, "enche": true, "nomeia": true,
	"acoes": true, "gastas": true, "titulo": true, "salva": true,
	"branco": true, "travessao": true, "sobrevive": true, "prende": true,
	"maximo": true, "voltar": true, "recarregar": true, "divide": true,
	"depende": true, "desfecho": true, "aprimoramentos": true, "abrem": true,
	"condicionais": true, "ligados": true, "entram": true, "reabrir": true,
	"teto": true, "abertos": true, "restaurar": true, "descarta": true,
	"ajuste": true, "rolagem": true, "descoberta": true, "minuscula": true,
	"situacionais": true, "substituem": true, "conjunto": true, "usar": true,
	"uso": true, "valida": true, "bordas": true, "linhas": true, "somam": true,
	"resistencias": true, "linearizacao": true, "sinal": true, "elo": true,
	"empate": true, "registra": true, "recorta": true, "destinatario": true,
	"ficar": true, "reaproveitado": true,
}

var declaracaoDeTeste = regexp.MustCompile(`(?m)^func (Test\w+)`)

// CADA NOME DE TESTE É EM INGLÊS (ALE-282).
//
// A varredura que fez esta regra valer trocou 773 nomes de uma vez. Ela existe
// porque a regra de idioma do CLAUDE.md — identificador em inglês, texto de
// gente em português — nomeia "nome de teste" com todas as letras desde sempre, e
// mesmo assim o repositório tinha 1.051 casos com 773 em português: uma
// convenção escrita e não varrida é aplicada exatamente aos arquivos que alguém
// apontou.
//
// O guarda é o que impede o 774º. Sem ele a lista volta a crescer em silêncio, um
// nome por vez, e cada um parece pequeno demais para justificar uma varredura.
func TestEveryTestNameIsEnglish(t *testing.T) {
	raiz := ".."
	medidos := 0
	for _, caminho := range arquivosDeTeste(t, raiz) {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for _, achado := range declaracaoDeTeste.FindAllStringSubmatch(string(conteudo), -1) {
			nome := achado[1]
			medidos++
			for _, palavra := range palavrasDe(nome) {
				if !portugueseMarkers[strings.ToLower(palavra)] {
					continue
				}
				t.Errorf("%s: %s tem %q, que é português.\n"+
					"Nome de teste é IDENTIFICADOR, e identificador é em inglês (CLAUDE.md § Idioma).\n"+
					"A grafia do termo sai do GLOSSARIO — traduzir na hora, duas vezes, é como um\n"+
					"conceito vira dois.",
					caminho, nome, palavra)
				break
			}
		}
	}

	// O denominador: uma lista de reprovados vazia e um caminho que não casa com
	// nada se parecem no terminal. Em setembro de 2026 eram 1.051 casos.
	if medidos < 900 {
		t.Fatalf("só %d nomes medidos — o guarda ficou cego", medidos)
	}
}

// palavrasDe quebra o CamelCase em palavras.
//
// Duas quebras, e a SEGUNDA é a que uma primeira versão errou: além de cortar
// antes de uma maiúscula que segue minúscula (`...ContaLetras` -> "Conta",
// "Letras"), é preciso cortar a última maiúscula de uma CORRIDA quando vem
// minúscula depois — senão `OBotaoDiz` sai como uma palavra só, "OBotao", e
// nenhum marcador casa com ela.
//
// O erro não aparecia porque o script que calibrava a lista quebrava as palavras
// com OUTRO algoritmo, e media sensibilidade sobre palavras que o guarda nunca
// produziria. É a família do medidor que responde outra pergunta: a calibração
// desta lista agora é feita RODANDO o guarda contra a árvore anterior à
// varredura, e não simulando-o.
//
// A sigla gritada continua sendo uma palavra só:
// `…LimitesContamCARACTERESENaoBytes` tem "CARACTERES".
func palavrasDe(nome string) []string {
	corpo := []rune(strings.TrimPrefix(nome, "Test"))
	var palavras []string
	inicio := 0
	for i := 1; i < len(corpo); i++ {
		anterior, atual := corpo[i-1], corpo[i]
		seguinteEhMinuscula := i+1 < len(corpo) && unicode.IsLower(corpo[i+1])
		corta := unicode.IsUpper(atual) && !unicode.IsUpper(anterior)
		corta = corta || (unicode.IsUpper(anterior) && unicode.IsUpper(atual) && seguinteEhMinuscula)
		if corta {
			palavras = append(palavras, string(corpo[inicio:i]))
			inicio = i
		}
	}
	if inicio < len(corpo) {
		palavras = append(palavras, string(corpo[inicio:]))
	}
	return palavras
}

func arquivosDeTeste(t *testing.T, raiz string) []string {
	t.Helper()
	var achados []string
	err := filepath.WalkDir(raiz, func(caminho string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() && (entrada.Name() == "node_modules" || entrada.Name() == ".git") {
			return filepath.SkipDir
		}
		if !entrada.IsDir() && strings.HasSuffix(entrada.Name(), "_test.go") {
			achados = append(achados, caminho)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer %s: %v", raiz, err)
	}
	return achados
}
