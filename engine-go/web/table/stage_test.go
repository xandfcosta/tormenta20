package table

// A FORMA DO MESTRE (ALE-269) — os guardas do palco.
//
// O que se prende aqui NÃO é o desenho: é o contrato que o desenho pode quebrar
// em SILÊNCIO. A Mesa é remendada por REGIÃO, e o stream acha a região pelo id.
// Um id que some, ou que apareça duas vezes porque alguém pendurou a mesma
// região nas duas formas, faz o remendo escrever no vazio — e o sintoma disso
// não é uma exceção, é uma tela que PARA DE ATUALIZAR sozinha, que é a família
// de defeito mais cara deste repositório.
//
// Estes três casos foram vistos VERMELHOS antes de valerem:
//   - tirar `@tableRailTracker` do palco → o primeiro falha
//   - pendurar `@tableTracker` no palco E na gaveta → o segundo falha, dizendo 2
//   - listar `mesa-trilho-fila` sem a guarda de papel → o terceiro falha

// tableRegionNames são os ids que o stream remenda. A lista é a mesma do
// `TableRegions`, escrita aqui de novo DE PROPÓSITO: derivá-la da produção
// faria o teste concordar com o defeito: uma região removida sumiria dos dois
// lados e o guarda ficaria verde sobre nada.
var tableRegionNames = []string{
	"mesa-cabecalho",
	"mesa-registrar",
	"mesa-grupo",
	"mesa-tabuleiro",
	"mesa-por-no-mapa",
	"mesa-acervo",
	"mesa-config-da-sessao",
	"mesa-fila",
	"mesa-comandos",
}
