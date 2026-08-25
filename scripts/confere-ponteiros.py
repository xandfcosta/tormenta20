#!/usr/bin/env python3
"""Confere se os arquivos CITADOS nos comentários dos testes ainda existem.

Um teste que sai deixa escrito para onde a garantia foi, e um destino que não
existe é pior que nenhuma nota: manda a próxima pessoa procurar um arquivo que
não está lá. A ALE-187 escreveu dezenas desses ponteiros de uma vez.

A ideia veio da sessão da migração Datastar, que achou três ponteiros quebrados
na base dela VARRENDO em vez de lembrar — ela conhecia um dos três.

    python3 scripts/confere-ponteiros.py

LIMITAÇÃO, e ela importa: a varredura não distingue "o destino sumiu" de "o
comentário cita um arquivo que FOI SUBSTITUÍDO e diz isso". O
`live_gating_test.go` abre com "este arquivo é o `realtime_gating_test.go`
traduzido para HTTP (ALE-253)" — o nome antigo é história, não endereço, e
aparece aqui como falso positivo. Trate a saída como candidatos, não veredito.
"""
import io
import os
import re
import subprocess
import sys

RAIZ = subprocess.run(
    ['git', 'rev-parse', '--show-toplevel'], capture_output=True, text=True,
).stdout.strip()

CITADO = re.compile(r'`([\w./-]+\.(?:ts|tsx|go))`')


def arquivos_de_teste() -> list[str]:
    saida = subprocess.run(
        ['git', 'ls-files', '*.test.ts', '*.test.tsx', '*_test.go', '*.spec.ts'],
        cwd=RAIZ, capture_output=True, text=True,
    ).stdout
    return saida.split()


def existe(nome: str) -> bool:
    base = os.path.basename(nome)
    achou = subprocess.run(['git', 'ls-files', '*' + base],
                           cwd=RAIZ, capture_output=True, text=True).stdout
    return bool(achou.strip())


def main() -> int:
    quebrados: list[tuple[str, str]] = []
    vivos = 0
    for rel in arquivos_de_teste():
        try:
            texto = io.open(os.path.join(RAIZ, rel), encoding='utf-8').read()
        except OSError:
            continue
        for linha in texto.split('\n'):
            # Só COMENTÁRIO: um import ou uma string de código não é ponteiro.
            if '//' not in linha and '*' not in linha:
                continue
            for nome in CITADO.findall(linha):
                if existe(nome):
                    vivos += 1
                else:
                    quebrados.append((rel, nome))

    total = vivos + len(quebrados)
    print(f'ponteiros conferidos: {total} · vivos: {vivos} · candidatos: {len(quebrados)}')
    for rel, nome in quebrados:
        print(f'  {rel}  →  {nome}')
    if quebrados:
        print('\nConfira cada um: citar um arquivo que foi SUBSTITUÍDO, dizendo isso,')
        print('é história e não endereço quebrado.')
    return 0


if __name__ == '__main__':
    sys.exit(main())
