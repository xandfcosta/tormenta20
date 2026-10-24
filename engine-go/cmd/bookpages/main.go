// Command bookpages deriva a PÁGINA DO LIVRO de cada entrada dos catálogos, do
// próprio livro.
//
// O botão "abrir no livro" (ALE-264) precisa de `bookPage` em toda entrada, e
// transcrever mil páginas à mão produz erro silencioso — o botão abre uma página
// plausível e ninguém confere. A fonte é o ÍNDICE REMISSIVO do livro, que já é
// uma tabela nome→página; o que este comando acrescenta é a CONFERÊNCIA: nenhuma
// página é aceita sem que o nome da entrada apareça no texto dela. Entrada que
// não confere fica SEM página — o botão não nasce, que é honesto.
//
//	go run ./cmd/bookpages                     # relatório, não grava
//	go run ./cmd/bookpages --gravar            # aplica nos catálogos
//	go run ./cmd/bookpages /caminho/livro.pdf  # outro PDF
//
// Sem caminho, o livro se acha como em `bookpdf.Open` (`T20_BOOK_PDF`,
// `LIVRO_PDF`, subindo do diretório atual, ou o do checkout principal).
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		var f fatal
		if errors.As(err, &f) {
			fmt.Fprintln(os.Stderr, f.msg)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	path := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		path = args[0]
		if _, err := os.Stat(path); err != nil {
			return fatal{fmt.Sprintf("não achei o livro em %s — passe o caminho ou LIVRO_PDF", path)}
		}
	} else {
		found, err := bookpdf.Open()
		if err != nil {
			return err
		}
		path = found.Path
	}
	write := contains(args, "--gravar")

	b, err := openBook(path)
	if err != nil {
		return err
	}
	fmt.Printf("livro: %d páginas · abertura %d (apoiada por %d rodapés) · índice com %d nomes\n",
		b.pages, b.opening, b.support, len(b.index))

	changed, missing := 0, 0
	for _, c := range catalogs {
		m, s := b.process(c)
		changed += m
		missing += s
	}
	if write {
		if err := rebuildAndSave(b); err != nil {
			return err
		}
	}
	fmt.Printf("\n%d páginas mudadas · %d entradas sem página\n", changed, missing)
	if write {
		fmt.Println("GRAVADO")
	} else {
		fmt.Println("nada gravado (use --gravar)")
	}
	return nil
}

// rebuildAndSave grava as páginas acertadas e depois cada catálogo recriado, um
// de cada vez: um que pare o script deixa gravados os anteriores, como o Python
// deixava.
func rebuildAndSave(b *book) error {
	dir, err := transcriptdir.Find()
	if err != nil {
		return err
	}
	if err := transcript.WriteSource(dir); err != nil {
		return err
	}
	b.rebuildClasses()
	for _, rebuild := range []func() error{b.rebuildEffectTypes, b.rebuildSpellSchools, b.rebuildExpertises} {
		if err := transcript.WriteSource(dir); err != nil {
			return err
		}
		if err := rebuild(); err != nil {
			return err
		}
	}
	return transcript.WriteSource(dir)
}
