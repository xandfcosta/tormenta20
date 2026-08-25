#!/usr/bin/env python3
"""Deriva a PÁGINA DO LIVRO de cada entrada dos catálogos, do próprio livro.

O botão "abrir no livro" (ALE-264) precisa de `bookPage` em toda entrada, e a
tentação é transcrever à mão. Transcrever 1.072 páginas à mão produz erro
silencioso — o botão abre uma página plausível e ninguém confere.

A fonte aqui é o ÍNDICE REMISSIVO do próprio livro (impressas 396-399), que já
é uma tabela nome→página feita por quem diagramou. O que o script acrescenta é
a CONFERÊNCIA: nenhuma página é aceita sem que o nome da entrada apareça no
texto daquela página. Entrada que não confere fica SEM página — o botão não
nasce, que é honesto; página errada seria o defeito.

    python3 scripts/paginas-do-livro.py               # relatório, não grava
    python3 scripts/paginas-do-livro.py --gravar      # aplica nos catálogos

O livro é ignorado pelo git e vive fora do repositório: passe o caminho em
LIVRO_PDF ou como primeiro argumento.
"""

import json
import os
import re
import subprocess
import sys
import unicodedata
from collections import Counter, defaultdict

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DADOS = os.path.join(RAIZ, "engine-go", "catalog", "data")
PADRAO_DO_LIVRO = "/mnt/HD/projects/tormenta20/t20-book.pdf"


def dobra(texto):
    """Minúsculas e sem acento: ninguém digita til, e o índice usa caixa baixa."""
    texto = unicodedata.normalize("NFD", texto)
    texto = "".join(c for c in texto if unicodedata.category(c) != "Mn").lower()
    return re.sub(r"[ \t]+", " ", texto).strip()


def achata(texto):
    """Uma linha só: hífen de quebra emendado e todo espaço virando um."""
    emendado = re.sub(r"-\n\s*", "", texto)
    return re.sub(r"\s+", " ", dobra(emendado))


def com_selo_no_meio(alvo, texto):
    """O nome com o SELO da diagramação enfiado no meio dele.

    Conserto de um falso negativo que o guarda do bestiário denunciou: a ficha
    imprime o ND à direita do nome, e o texto extraído sai "cavaleiro do leopardo
    nd 9 sangrento". A comparação por substring reprovava a página CERTA, e o
    script tirava a página de uma criatura que estava correta.

    A folga é apertada de propósito: até dois pedaços de no máximo três letras
    entre uma palavra e a seguinte. Cabe "nd 9" e "t$ 45"; não cabe uma frase, o
    que impede o nome de casar espalhado por uma página de prosa.
    """
    partes = [re.escape(p) for p in alvo.split(" ") if p]
    if len(partes) < 2:
        return False
    padrao = r"(?:\s+\S{1,3}){0,2}\s+".join(partes)
    return re.search(r"\b" + padrao + r"\b", texto) is not None


def paginas_do_pdf(caminho):
    saida = subprocess.run(
        ["pdftotext", caminho, "-"], capture_output=True, text=True, check=True
    ).stdout
    return saida.split("\f")


def mede_a_abertura(paginas):
    """Quantas páginas o ARQUIVO tem antes da impressa 1.

    MEDIDA e não suposta: o rodapé de cada página traz o número impresso, e a
    diferença entre ele e o índice do arquivo é a abertura. Tira-se a MODA
    porque nem toda página imprime o rodapé (as de abertura de capítulo não), e
    uma página solta com um número no fim do texto não pode decidir sozinha.
    """
    votos = Counter()
    for i, pagina in enumerate(paginas):
        for linha in reversed(pagina.split("\n")[-6:]):
            if re.fullmatch(r"\d{1,3}", linha.strip()):
                votos[i + 1 - int(linha.strip())] += 1
                break
    if not votos:
        raise SystemExit("não achei rodapé nenhum: o PDF não é o livro esperado")
    abertura, quantas = votos.most_common(1)[0]
    if quantas < 100:
        raise SystemExit(f"abertura {abertura} apoiada por só {quantas} páginas — não confio")
    return abertura, quantas


def primeira_do_indice(paginas):
    """O índice do PDF (0-based) onde o Índice Remissivo começa.

    A PRIMEIRA da METADE DE TRÁS. Duas armadilhas de uma vez, as duas medidas: o
    SUMÁRIO escreve "Índice Remissivo" na abertura do livro (com a página ao
    lado), e o próprio índice repete a frase como cabeçalho corrido em cada uma
    de suas páginas. Pegar a primeira ocorrência lia o sumário (130 nomes);
    pegar a última começava no meio do índice (306). São 1.217.
    """
    citam = [i for i, pagina in enumerate(paginas) if dobra("Índice Remissivo") in dobra(pagina)]
    tras = [i for i in citam if i >= len(paginas) // 2]
    if not tras:
        raise SystemExit("não achei o Índice Remissivo no PDF")
    return tras[0]


def le_o_indice(paginas, abertura):
    """Lê o Índice Remissivo: nome → [(qualificador, [páginas impressas])].

    O qualificador entre parênteses é do próprio livro e resolve as colisões que
    mais importam aqui: "grifo (criatura)" contra "grifo (parceiro)", "guarda
    (origem)" contra "guarda (criatura)".
    """
    # DE TRÁS PARA A FRENTE, e isto é conserto de uma medição que veio errada
    # sem falhar: o SUMÁRIO também escreve "Índice Remissivo" (com a página dele
    # ao lado), então a busca do começo parava na primeira página do livro e lia
    # quatro páginas de sumário como se fossem o índice — 130 nomes em vez de
    # 1.235, e nenhum erro.
    #
    # `dobra` tira o acento dos DOIS lados: procurar "índice" acentuado num
    # texto já dobrado não acha nada.
    comeco = primeira_do_indice(paginas)
    # Até o FIM do livro, e não um número fixo de páginas: o índice tem quatro
    # hoje e teria cinco numa edição revista. Página que não é índice não tem
    # linha pontilhada e não contribui com nada.
    linhas = []
    for pagina in paginas[comeco:]:
        linhas.extend(l.strip() for l in pagina.split("\n"))

    # A linha pode terminar no pontilhado com o número na seguinte: o texto sai
    # coluna a coluna e a quebra cai no meio da entrada.
    juntas, i = [], 0
    while i < len(linhas):
        linha = linhas[i]
        if linha and re.search(r"\.{2,}$", linha) and i + 1 < len(linhas):
            linha += " " + linhas[i + 1]
            i += 1
        juntas.append(linha)
        i += 1

    entrada = re.compile(r"^(.+?)[\.\s]*\.{3,}\s*([\d,\s]+)$")
    achado = defaultdict(list)
    for linha in juntas:
        casa = entrada.match(linha)
        if not casa:
            continue
        nome = casa.group(1).strip().rstrip(".").strip()
        paginas_citadas = [int(n) for n in re.findall(r"\d+", casa.group(2))]
        if not nome or not paginas_citadas:
            continue
        qualificador = re.search(r"\(([^)]*)\)\s*$", nome)
        base = dobra(re.sub(r"\s*\([^)]*\)\s*$", "", nome))
        achado[base].append((qualificador.group(1) if qualificador else "", paginas_citadas))
    return achado


class Livro:
    """O livro como duas visões: o texto de cada página impressa e o índice."""

    def __init__(self, caminho):
        paginas = paginas_do_pdf(caminho)
        self.abertura, self.apoio = mede_a_abertura(paginas)
        # O texto de conferência é ACHATADO: uma linha só, sem hífen de quebra.
        # O livro é diagramado em duas colunas estreitas, e um nome de duas
        # palavras quase sempre atravessa a quebra de linha ("Abençoar Ali-\nmentos").
        # Comparar contra o texto cru reprovava 35 magias cuja página estava certa
        # — e "não confere" seria lido como "a página está errada".
        self.texto = [achata(p) for p in paginas]
        self.linhas = [[dobra(l) for l in p.split("\n")] for p in paginas]
        self.indice = le_o_indice(paginas, self.abertura)
        self.paginas = len(paginas)
        # As páginas do PRÓPRIO índice são proibidas como resposta, e isto é
        # conserto de um vermelho do script: uma condição foi resolvida para a
        # impressa 396, que é a primeira página do índice remissivo — o nome
        # aparece lá porque aquilo é uma lista de nomes. Sem esta trava, o botão
        # "abrir no livro" abriria o índice, com a conferência de pé e tudo.
        self.indice_de = primeira_do_indice(paginas) - self.abertura + 1

    def _pos(self, impressa):
        return impressa + self.abertura - 1

    def pagina(self, impressa):
        """O texto achatado de uma página impressa, ou vazio fora do livro."""
        i = self._pos(impressa)
        return self.texto[i] if 0 <= i < len(self.texto) else ""

    def confere(self, nome, impressa):
        """Ver o comentário de `indice_de`: página do índice nunca é resposta."""
        """O nome aparece no texto daquela página impressa?

        É o CONTROLE de tudo o que sai daqui: sem ele, uma página do índice mal
        lida entraria no catálogo com cara de dado conferido.
        """
        if impressa >= self.indice_de:
            return False
        i = self._pos(impressa)
        if not (0 <= i < len(self.texto)):
            return False
        texto = self.texto[i]
        if achata(nome) in texto:
            return True
        return com_selo_no_meio(achata(nome), texto)

    def por_indice(self, nome, qualificadores=()):
        """Páginas que o índice cita para este nome, na ordem em que ele as cita."""
        for qualificador, paginas in self.indice.get(dobra(nome), []):
            if qualificadores and qualificador not in qualificadores:
                continue
            return paginas
        return []

    def por_titulo(self, nome, faixa):
        """Páginas de uma faixa em que alguma LINHA começa com o nome.

        Serve para o que o índice não lista. A faixa é o que a torna confiável:
        um nome de item começa linha em dezenas de páginas de prosa, e dentro do
        capítulo de equipamento ele é a linha da tabela.
        """
        alvo = dobra(nome)
        fora = []
        for impressa in range(faixa[0], faixa[1] + 1):
            i = self._pos(impressa)
            if not (0 <= i < len(self.linhas)):
                continue
            for linha in self.linhas[i]:
                if linha == alvo or linha.startswith(alvo + " ") or linha.startswith(alvo + "."):
                    fora.append(impressa)
                    break
        return fora


# ── os catálogos, e o que cada um diz ao índice ──────────────────────────────
#
# `qualificadores` são os do próprio livro, e existem onde o nome colide: "grifo
# (criatura)" contra "grifo (parceiro)". Vazio na lista aceita a entrada sem
# parênteses, que é o caso da maioria.

# `assinatura` é como o LIVRO imprime o começo daquele verbete, e ela é a prova
# mais forte que existe aqui — mais forte que o índice e que o `bookPage` que já
# estava. Ela nasceu de uma imprecisão medida: a p289 contém "lobos-das-cavernas"
# no texto corrido, então a conferência por substring aprovava a página errada
# enquanto o bloco do Lobo abre na 290.
#
# `{nome}` é substituído pelo nome dobrado da entrada.
#
#   criatura:  "lobo nd 1/2 ..."      → o ND vem colado no nome, sempre
#   condição:  "abalado. o personagem ..."
#
# Onde não há assinatura confiável (item, poder, magia), a escada continua sendo
# índice → título na faixa. Assinatura inventada seria pior que nenhuma: ela
# passaria a decidir com a autoridade de uma prova.
CATALOGOS = [
    {"arquivo": "conditions.json", "rotulo": "condições", "qualificadores": ("condição", ""),
     "assinatura": r"{nome}\.\s+(o personagem|voce|a criatura)"},
    {"arquivo": "items.json", "rotulo": "itens", "qualificadores": ("",)},
    {"arquivo": "spells.json", "rotulo": "magias", "qualificadores": ("magia", "")},
    {"arquivo": "bestiary.json", "rotulo": "criaturas", "qualificadores": ("criatura", ""),
     "assinatura": r"{nome}\s+nd\s+\d"},
    {"arquivo": "races.json", "rotulo": "raças", "qualificadores": ("raça", "")},
    {"arquivo": "deuses.json", "rotulo": "deuses", "qualificadores": ("",)},
    {"arquivo": "general-powers.json", "rotulo": "poderes gerais", "qualificadores": ("",)},
    {"arquivo": "class-powers.json", "rotulo": "poderes de classe", "qualificadores": ("",)},
    {"arquivo": "granted-powers.json", "rotulo": "poderes concedidos", "qualificadores": ("",)},
    {"arquivo": "tormenta-powers.json", "rotulo": "poderes da Tormenta", "qualificadores": ("",)},
    {"arquivo": "divine-powers.json", "rotulo": "poderes divinos", "qualificadores": ("",)},
    {"arquivo": "origens.json", "rotulo": "origens", "qualificadores": ("origem", "")},
]


def carrega(arquivo):
    with open(os.path.join(DADOS, arquivo), encoding="utf-8") as f:
        return json.load(f)


def grava(arquivo, dado):
    """Grava no MESMO formato em que estava: minificado e com acento literal.

    Formato preservado não é estética: um `indent=2` aqui reescreveria 1.072
    linhas e o diff da correção de página sumiria dentro dele.
    """
    caminho = os.path.join(DADOS, arquivo)
    with open(caminho, "w", encoding="utf-8") as f:
        json.dump(dado, f, ensure_ascii=False, separators=(",", ":"))


def entradas(dado):
    """Os catálogos são dict-por-id ou lista; as duas formas viram lista aqui."""
    return list(dado.values()) if isinstance(dado, dict) else dado


def nome_da(entrada):
    return entrada.get("name") or entrada.get("id") or ""


def por_assinatura(livro, nome, assinatura, faixa):
    """As páginas em que o livro ABRE este verbete, pela assinatura do catálogo.

    Só serve quando a resposta é ÚNICA: duas páginas casando a assinatura
    significam que ela não distingue, e escolher uma seria dar a uma dúvida a
    cara de uma prova.
    """
    if not assinatura:
        return None
    padrao = re.compile(assinatura.replace("{nome}", re.escape(dobra(nome))))
    achadas = [
        impressa
        for impressa in range(faixa[0], faixa[1] + 1)
        if impressa < livro.indice_de and padrao.search(livro.pagina(impressa))
    ]
    return achadas[0] if len(achadas) == 1 else None


def resolve(livro, entrada, qualificadores, faixa, assinatura=None):
    """A escada, da fonte mais forte para a mais fraca. Devolve (página, fonte).

    Página que não confere não é usada, venha de onde vier — inclusive do
    `bookPage` que já estava no catálogo, que é como as cinco erradas do
    bestiário foram achadas.
    """
    nome = nome_da(entrada)
    tinha = entrada.get("bookPage")

    # A ASSINATURA vem antes de tudo, inclusive do que já estava: ela é a única
    # fonte aqui que sabe a diferença entre a página que CITA o verbete e a que
    # o ABRE. Foi o que corrigiu o Lobo (289→290), o Troll (309→308) e o Trog
    # (292→291) — os três com o bloco uma página adiante do que o catálogo dizia.
    if faixa:
        pagina = por_assinatura(livro, nome, assinatura, faixa)
        if pagina:
            return pagina, "assinatura do livro"

    if tinha and livro.confere(nome, tinha):
        return tinha, "já estava"

    for pagina in livro.por_indice(nome, qualificadores):
        if livro.confere(nome, pagina):
            return pagina, "índice"
        # JANELA em volta do que o índice diz, e ela é medida e não folga
        # arbitrária: o índice às vezes aponta a ABERTURA da seção e não o
        # verbete — "Raio Arcano" está indexado em 38 e escrito em 40; "Combate
        # Defensivo", em 124 e escrito em 126. Aceita-se a página mais PRÓXIMA
        # em que o nome de fato aparece, o que mantém a conferência de pé.
        perto = proxima_que_confere(livro, nome, pagina)
        if perto:
            return perto, "índice (janela)"

    if faixa:
        candidatas = [p for p in livro.por_titulo(nome, faixa) if livro.confere(nome, p)]
        if len(candidatas) == 1:
            return candidatas[0], "título na faixa"
        if len(candidatas) > 1:
            # A ÚLTIMA do trecho, e a escolha é CALIBRADA contra o próprio livro
            # em vez de arbitrada: nos 52 itens que o índice resolve E que têm
            # mais de uma página candidata, a página do índice é a última em 48,
            # a primeira em 2 e uma do meio em 2. O padrão tem explicação — o
            # nome aparece antes na TABELA e depois na descrição, e é a
            # descrição que o índice aponta.
            #
            # Erra em 4 de 52 (8%) no conjunto ambíguo, e isso fica dito: a
            # página escolhida CONTÉM a entrada (a conferência continua valendo),
            # só pode não ser a que o índice escolheria.
            #
            # Trecho curto porque candidatas espalhadas pelo livro são
            # ocorrências diferentes do mesmo nome, e aí não há o que escolher.
            if max(candidatas) - min(candidatas) <= 8:
                return max(candidatas), "última do trecho (calibrado)"
            return None, f"ambígua ({len(candidatas)} páginas na faixa)"

    # O nome sem o DEGRAU: "Fúria +3", "Instinto Selvagem +2", "Redução de Dano
    # 4" e "Magias (2° círculo)" não são verbetes — o livro escreve o poder uma
    # vez e a tabela da classe mostra a progressão. Sem isto, 152 poderes de
    # classe ficavam sem página por causa do sufixo.
    base = sem_degrau(nome)
    if base != nome:
        entrada_base = dict(entrada, name=base)
        entrada_base.pop("bookPage", None)
        pagina, fonte = resolve(livro, entrada_base, qualificadores, faixa)
        if pagina:
            return pagina, fonte + " (sem o degrau)"

    if tinha:
        return None, f"a que estava (p{tinha}) não confere"
    return None, "sem fonte"


def sem_degrau(nome):
    """Tira o degrau de progressão e o prefixo de família do nome do poder.

        "Fúria +3"                  → "Fúria"
        "Marca da Presa +1d8"       → "Marca da Presa"
        "Magias (2° círculo)"       → "Magias"
        "Música: Melodia Curativa"  → "Melodia Curativa"

    O livro escreve o poder UMA vez; a tabela da classe é que mostra a
    progressão, e a família ("Música:", "Caminho:") é rótulo nosso para agrupar.
    """
    cru = re.sub(r"\s*(\+\s*\d+(d\d+)?|\(\s*\d+.*\)|\b\d+)\s*$", "", nome).strip()
    return cru.split(": ", 1)[1] if ": " in cru else cru


def proxima_que_confere(livro, nome, alvo, janela=4):
    """A página mais próxima de `alvo` em que o nome aparece, ou None."""
    for distancia in range(1, janela + 1):
        for pagina in (alvo + distancia, alvo - distancia):
            if pagina > 0 and livro.confere(nome, pagina):
                return pagina
    return None


def faixa_dos_confirmados(paginas):
    """O envelope do que já foi resolvido POR ÍNDICE vira a faixa de busca dos
    que sobraram. Data-driven em vez de faixa escrita à mão: o capítulo se
    declara pelo que o índice já apontou, e não por um número que envelhece."""
    if len(paginas) < 3:
        return None
    return (min(paginas), max(paginas))


def processa(livro, catalogo, gravar):
    dado = carrega(catalogo["arquivo"])
    lista = entradas(dado)
    quali = catalogo["qualificadores"]

    # Primeira volta: só as fontes fortes, para o envelope nascer de página
    # conferida e não de chute.
    assinatura = catalogo.get("assinatura")
    primeira = {}
    for entrada in lista:
        pagina, fonte = resolve(livro, entrada, quali, None, assinatura)
        primeira[id(entrada)] = (pagina, fonte)
    faixa = faixa_dos_confirmados([p for p, _ in primeira.values() if p])

    contagem = Counter()
    mudou = []
    sem = []
    for entrada in lista:
        pagina, fonte = primeira[id(entrada)]
        if faixa:
            # A segunda volta roda para TODOS e não só para quem ficou sem
            # página: é nela que a faixa existe, e é a faixa que habilita a
            # assinatura — sem ela o Lobo continuaria em 289, resolvido pelo
            # "já estava" da primeira volta.
            pagina, fonte = resolve(livro, entrada, quali, faixa, assinatura)
        contagem[fonte if pagina else "SEM PÁGINA"] += 1
        if pagina is None:
            sem.append((nome_da(entrada), entrada.get("bookPage"), fonte))
            entrada.pop("bookPage", None)
            continue
        if entrada.get("bookPage") != pagina:
            mudou.append((nome_da(entrada), entrada.get("bookPage"), pagina, fonte))
        entrada["bookPage"] = pagina

    print(f"\n── {catalogo['rotulo']} ({len(lista)}) · faixa {faixa}")
    for fonte, quantas in contagem.most_common():
        print(f"   {quantas:4d}  {fonte}")
    for nome, antes, agora, fonte in mudou[:12]:
        print(f"   ~ {nome}: {antes} → {agora} ({fonte})")
    if len(mudou) > 12:
        print(f"   ~ … e mais {len(mudou) - 12}")
    for nome, antes, motivo in sem[:8]:
        print(f"   ✗ {nome} (tinha {antes}): {motivo}")
    if len(sem) > 8:
        print(f"   ✗ … e mais {len(sem) - 8} sem página")

    if gravar:
        grava(catalogo["arquivo"], dado)
    return len(mudou), len(sem)


# O catálogo existe em DUAS cópias, e quem mexe numa tem de mexer na outra.
#
# O motor resolve regras a partir de `parity/_catalogs.json` (lido do disco, é o
# `CATALOG_PATH`); o navegador e os testes de schema leem os arquivos embutidos
# em `catalog/data`. O `TestDumpAgreesWithEmbeddedCatalog` existe justamente para
# as duas nunca divergirem — e foi ele que acusou este script na primeira
# gravação. A mensagem dele manda regenerar com `cmd/genoracle`, mas o genoracle
# LÊ o dump, não o escreve: o produtor era o `t20-data`, que foi apagado na
# ALE-104. Então o dump é artefato commitado, e sincronizá-lo é trabalho de quem
# edita o catálogo.
GEMEOS = {"items": "items.json", "classPowers": "class-powers.json",
          "generalPowers": "general-powers.json", "racas": "races.json"}


def sincroniza_o_dump():
    """Copia para o dump do motor o que mudou nos arquivos embutidos."""
    caminho = os.path.join(RAIZ, "engine-go", "parity", "_catalogs.json")
    with open(caminho, encoding="utf-8") as f:
        dump = json.load(f)
    for chave, arquivo in GEMEOS.items():
        if chave in dump:
            dump[chave] = carrega(arquivo)
    # `indent=2` porque é o formato em que ele está: reescrevê-lo minificado
    # trocaria um diff de páginas por um diff do arquivo inteiro.
    with open(caminho, "w", encoding="utf-8") as f:
        json.dump(dump, f, ensure_ascii=False, indent=2)
        f.write("\n")
    print(f"dump do motor sincronizado: {', '.join(GEMEOS)}")


# ── as CLASSES, que não tinham catálogo ──────────────────────────────────────

def cria_o_catalogo_de_classes(livro):
    """Escreve `catalog/data/classes.json`: id, nome e página das 14 classes.

    Elas existiam só como uma LISTA DE NOMES dentro de `options.json`, sem lugar
    onde guardar a página — e sem página não há botão para o livro. O catálogo
    novo é mínimo de propósito: o que a classe tem de PV, PM e perícias é
    transcrição de tabela, e transcrever à mão é o que este script existe para
    não fazer. O que a tela mostra além do nome ela deriva do que já existe
    (perícias treinadas, poderes da classe).

    O `id` segue a convenção que os poderes de classe já usam
    ("class.arcanista.…"): o nome dobrado, sem acento e sem espaço.
    """
    nomes = carrega("options.json")["classes"]
    classes = []
    for nome in nomes:
        paginas = [p for p in livro.por_indice(nome, ("classe", "")) if livro.confere(nome, p)]
        if not paginas:
            print(f"   ✗ classe {nome} sem página — fica de fora")
            continue
        classes.append({"id": dobra(nome).replace(" ", "-"), "name": nome, "bookPage": paginas[0]})
    grava("classes.json", classes)
    print(f"catálogo de classes: {len(classes)} de {len(nomes)}")


def main():
    caminho = sys.argv[1] if len(sys.argv) > 1 and not sys.argv[1].startswith("-") else None
    caminho = caminho or os.environ.get("LIVRO_PDF") or PADRAO_DO_LIVRO
    if not os.path.exists(caminho):
        raise SystemExit(f"não achei o livro em {caminho} — passe o caminho ou LIVRO_PDF")
    gravar = "--gravar" in sys.argv

    livro = Livro(caminho)
    print(f"livro: {livro.paginas} páginas · abertura {livro.abertura} "
          f"(apoiada por {livro.apoio} rodapés) · índice com {len(livro.indice)} nomes")

    mudadas = faltando = 0
    for catalogo in CATALOGOS:
        m, s = processa(livro, catalogo, gravar)
        mudadas += m
        faltando += s
    if gravar:
        cria_o_catalogo_de_classes(livro)
        sincroniza_o_dump()
    print(f"\n{mudadas} páginas mudadas · {faltando} entradas sem página")
    print("GRAVADO" if gravar else "nada gravado (use --gravar)")


if __name__ == "__main__":
    main()
