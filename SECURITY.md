# Segurança — validação de 6 de novembro de 2026

Testes automatizados pertencem ao projeto e são executados pelo `go test`; não dependem de Node ou de instalação global. Este relatório descreve o escopo testado, sem afirmar ausência de vulnerabilidades desconhecidas.

## Correções e regressões

| Área | Proteção testada |
|---|---|
| Caminhos | Recusa de `.`/`..`, caminhos relativos, controles, UTF-8 inválido e barras invertidas antes de acessar filesystem/SFTP |
| Entradas remotas | Nomes recebidos do servidor não podem injetar separadores, subir diretórios ou inserir controles |
| Arquivos | Links estáticos no caminho são recusados; criação exclusiva; renomeação concorrente não sobrescreve destinos |
| Transferências | Cancelamento fecha streams, remove só seu temporário e preserva arquivos preexistentes; upload/download conferidos por SHA-256 |
| Sessões | IDs desconhecidos/encerrados, entrada acima de 256 KiB e dimensões excessivas não chegam ao adaptador |
| SSH | Host incorreto, senha/passphrase erradas, confiança expirada, fingerprint divergente e mudança de chave durante a confirmação são recusados |
| Recursos SSH | Confirmações de host expiram em dois minutos; limite de 64 pendências; confirmação não é reutilizável |
| Credenciais | Senha/passphrase, inclusive campos extras injetados no JSON de perfil, não aparecem nos arquivos de dados |
| Interface | Nomes maliciosos passam por `textContent`; saída do terminal segue como bytes, sem interpretação HTML pela aplicação |
| Assets | CSP restringe scripts/conexões a recursos locais; sem handlers inline; hashes SHA-256 dos assets vendorizados conferidos |
| Preferências | Temas usam enumeração fechada; preferências adulteradas não injetam CSS/HTML; tamanho da fonte fica limitado |
| Plataforma | Windows rejeita perfil de terminal Linux; Linux rejeita WSL; a interface exibe o tipo local correspondente |

Foram executados testes no Windows e nos pacotes internos/JavaScript no Ubuntu WSL. Fuzzing de caminhos foi executado por cinco segundos, sem falha encontrada. Testes de tema verificam persistência, acompanhamento do SO, atualização dos terminais abertos, chamadas à janela nativa e contraste das paletas principais.

## Dependências

A consulta à [API OSV](https://google.github.io/osv.dev/post-v1-querybatch/) verificou **30 pacotes/versões**: módulos importados no build/testes Windows, núcleo/adaptadores/testes Linux, biblioteca padrão do Go e dois assets npm. O resultado final teve **zero alertas retornados**. Evidência e horário UTC: [dependencies-osv.json](security/dependencies-osv.json).

A consulta inicial identificou estes alertas, tratados com atualizações apenas em `go.mod`, `go.sum` e no cache local:

| Registro | Dependência | Tratamento |
|---|---|---|
| [GO-2026-5932](https://vuln.go.dev/ID/GO-2026-5932.json) | `golang.org/x/crypto` | Versão fixada em `v0.58.1-0.20261009181718-2f1f834330b2`, que remove o pacote OpenPGP descontinuado. OpenPGP já não era importado pelo aplicativo; o alerta inicial era do módulo, não evidência de exploração do SSH |
| [GO-2026-6860](https://vuln.go.dev/ID/GO-2026-6860.json) | `golang.org/x/net/html` | Atualizado para `v0.62.0`, com correção do comportamento quadrático de parsing |
| [GO-2026-6866](https://vuln.go.dev/ID/GO-2026-6866.json) | `golang.org/x/net/html` | Atualizado para `v0.62.0`, com correção da renderização em transições de namespace |

`x/sys` e `x/text` foram alinhados às versões exigidas pelas dependências. O Go instalado e o sistema operacional não foram atualizados.

O `govulncheck` oficial foi compilado **somente em `.local/bin`**, mas o Windows bloqueou sua execução por política de Controle de Aplicativo. Não houve contorno ou alteração da política. A consulta OSV identifica alertas por pacote/versão; não substitui a análise de alcance por funções que o [govulncheck](https://go.dev/doc/security/vuln/) faria.

## Reproduzir

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\test.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\security.ps1 -AuditDependencies
```

A primeira rotina inclui regressões Go e JavaScript e vet, sem rede após disponibilizar as dependências. A segunda adiciona fuzzing e consulta OSV. Alertas ou falhas de consulta não são tratados como auditoria aprovada. Não é necessário baixar ou executar `govulncheck` para essas rotinas.

## Limites

- Não foi executado pentest externo, análise dinâmica de todo o WebView ou análise de alcance por `govulncheck`.
- Verificação visual automatizada permanece indisponível; os testes JavaScript usam DOM simulado e não comprovam layout/foco na janela real.
- O terminal tem as permissões do usuário e não é sandbox. Verificação de symlinks não promete isolamento contra outro processo com as mesmas permissões trocando caminhos simultaneamente; um servidor SFTP malicioso controla seu próprio filesystem.
- Em filesystems Linux sem `RENAME_NOREPLACE`, o fallback seguro por hard link funciona para arquivos regulares; renomeação de diretórios retorna erro. Se remover a origem falhar após criar o link, ambos os nomes são preservados.
- Resultado sem alertas vale para a base consultada e versões registradas; não é certificação de segurança.


## Revisão adicional — 6 de novembro de 2026

A importação CSV foi revisada e passou a limitar o arquivo a 2 MB e 5.000 linhas, rejeitar UTF-8 inválido, validar os perfis por linha, ignorar campos de senha e deduplicar por host/porta/usuário. Testes de regressão confirmam esses limites e que segredos de colunas não são persistidos. A importação é serializada para evitar duplicatas em cliques concorrentes.

Nesta revisão passaram `go test -tags wv2runtime.error ./internal/... ./tests/frontend`, `go vet -tags wv2runtime.error ./...` e fuzzing de caminhos por 5 segundos (17.126 execuções; 27 entradas novas no corpus). A consulta OSV de 30 pacotes/versões em 2026-11-06 20:11 UTC retornou zero alertas conhecidos.

O teste completo `go test ./...` não foi concluído: o Windows bloqueou a execução de `termiashell.test.exe` pela política de Controle de Aplicativo. Não tentei contornar a política. Os pacotes internos e o teste de frontend foram executados separadamente e passaram; testes da raiz permanecem sem execução nesta revisão.
