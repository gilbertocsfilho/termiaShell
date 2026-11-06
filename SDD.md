# termiaShell — SDD

Software Design Document · versão 2.1 · 6 de novembro de 2026

Documento de referência do produto e da implementação atual. Resultados de testes e limites conhecidos estão em [VALIDATION.md](VALIDATION.md).

## 1. Objetivo e escopo

Aplicativo desktop em Go inspirado no fluxo básico do MobaXterm: terminal em abas e navegador gráfico de arquivos com menu de contexto para abrir/baixar, excluir, renomear, copiar caminhos, propriedades e permissões ao lado. A primeira entrega é Windows, com acesso ao WSL e SSH/SFTP. A estrutura também atende Linux, onde um shell local com PTY substitui o WSL.

Somente SSH é protocolo de conexão remota. Perfis SSH podem ser importados em massa por CSV. SFTP usa o transporte SSH para operações de arquivos. Terminal local e WSL são integrações do sistema operacional.

Ficam fora do escopo outros protocolos de acesso remoto, jogos, servidor X, encaminhamento gráfico, túneis, agent forwarding, plugins, macros, monitoramento, editor integrado, sincronização contínua e cópia recursiva de diretórios.

### Restrições

- Usar o Go instalado; não instalar ou atualizar ferramentas, runtimes, serviços ou pacotes globalmente.
- Dependências, caches e temporários de desenvolvimento ficam em `.local`, dentro do projeto. Não alterar PATH, Registro, configuração global do Go ou arquivos de inicialização do shell.
- Usar WebView2, WSL e runtimes de container somente quando já disponíveis. Pré-requisito ausente gera diagnóstico, sem instalador automático.
- Não exigir administrador para execução normal. Sem serviço em segundo plano, telemetria, CDN ou atualização automática.
- Containers podem validar SSH no runtime existente do WSL; pacotes de teste pertencem à imagem, nunca ao host.

## 2. Interface

Interface com temas **Claro, Escuro e Sistema**, fontes locais e ícones simples. A janela Windows inicia oculta até a interface carregar, com dimensões padrão de 1200 × 760, e então é centralizada antes de aparecer. Barra lateral compartilhada entre Windows e Linux, com navegação Fluent inspirada no WinUI: o WSL padrão (ou shell local no Linux) é um item independente, enquanto vários perfis SSH salvos ficam em uma seção própria; busca filtra os itens. As conexões abertas permanecem somente nas abas da sessão atual. Duas opções na tela inicial — terminal local e SSH/SFTP —, abas de terminal, painel de arquivos redimensionável e área de transferências. Terminal ocupa a maior parte da janela.

O seletor fica em Configurações/Preferências. **Sistema** é o padrão e acompanha mudanças do tema do SO sem reiniciar. Claro/Escuro são escolhas fixas. A troca atualiza interface, diálogos, terminais já abertos, novos terminais e solicita o tema correspondente à janela nativa pelo runtime Wails. A preferência persiste no armazenamento local do WebView. Valores inválidos são descartados em favor de Sistema; falha de armazenamento mantém o tema funcional na sessão atual.

O fechamento com trabalho ativo usa diálogo HTML do aplicativo, seguindo o tema, com **Encerrar** e **Cancelar**. Escape cancela e o foco inicial fica em Cancelar. Confirmar libera o segundo pedido de saída do runtime sem repetir o aviso; pedidos duplicados não empilham diálogos. Sem trabalho ativo, fechar é imediato.

Windows apresenta distribuições WSL; Linux apresenta terminal local. O frontend consulta capacidades do backend, sem identificar plataforma por user-agent. Perfis de terminal da outra plataforma são preservados e sinalizados como indisponíveis.

Navegador de arquivos tem caminho próprio: não acompanha `cd` nem interpreta o prompt. Pastas abrem por duplo clique; arquivos iniciam um diálogo de download. Ações permitem atualizar, subir um nível, mostrar ocultos, criar pasta, renomear, excluir item vazio e enviar arquivo.

Atalhos: `Ctrl+Shift+T` nova conexão, `Ctrl+Tab` próxima aba, `Ctrl+Shift+W` fechar aba e `Ctrl+Shift+C/V` copiar/colar. `Ctrl+C` continua sendo interrupção do terminal. Colagem multilinha pede confirmação. Confirmação também precede exclusão e encerramento de sessão ativa.

Tema escuro: fundo `#0B101A`, superfícies `#111925`, texto `#E6EDF7`. Tema claro: fundo `#F5F7FB`, superfícies brancas, texto `#243247`. Paletas próprias para ANSI, seleção e cursor do terminal; testes conferem contraste de texto e das cores ANSI principais. Fonte da interface `Segoe UI, system-ui, sans-serif`; terminal `Cascadia Mono, Consolas, DejaVu Sans Mono, monospace`. Nenhuma fonte é instalada.

## 3. Requisitos e aceite

| ID | Requisito | Aceite |
|---|---|---|
| RF01 | Atalho direto para WSL/local e perfis SSH | Criar, editar, buscar, excluir e recuperar após reiniciar, sem segredos persistidos |
| RF02 | Descoberta de alvos locais | Listar distribuições existentes no Windows ou shell local no Linux; erro claro se indisponível |
| RF03 | Terminal local interativo | PTY real, entrada, UTF-8, ANSI, cores, resize e interrupção |
| RF04 | SSH | Senha em memória ou chave privada com passphrase opcional; PTY remoto e resize |
| RF05 | Confiança SSH | Confirmar fingerprint nova; bloquear chave alterada; cancelar antes de confiar |
| RF06 | Arquivos WSL/POSIX | Listar, navegar, mostrar ocultos, criar pasta, renomear e excluir item vazio |
| RF07 | SFTP | Mesmas operações básicas; SFTP indisponível não impede terminal quando a abertura do subsystem retorna erro |
| RF08 | Transferência individual | Upload/download com progresso, fila, cancelamento e temporário no destino; não sobrescrever arquivo existente |
| RF09 | Abas independentes | Fechar uma sessão não encerra as outras; reconexão apenas por ação do usuário |
| RF10 | Encerramento | Confirmar trabalho ativo, cancelar operações e liberar recursos |
| RF11 | Preferências locais | Tema Claro/Escuro/Sistema, tamanho da fonte e exibição de ocultos preservados no armazenamento do WebView |

Links simbólicos não são seguidos. Diretórios só podem ser excluídos vazios. Arquivos especiais não são transferidos. Para substituir um arquivo existente, o usuário precisa gerenciar o destino explicitamente; não há sobrescrita automática.

Operações de rede e disco não bloqueiam a UI. Saída do terminal é limitada por lote e confirmação de consumo, com scrollback de 5.000 linhas. Até duas transferências executam simultaneamente; demais aguardam. Metas ainda a medir: abertura em até 3 s e resposta visual local à digitação abaixo de 50 ms no percentil 95. Não tratar metas como resultados.

## 4. Arquitetura e estrutura real

Wails v2 hospeda HTML/CSS/JavaScript local e xterm.js. Frontend não depende de Node/npm; assets distribuídos ficam em `frontend/vendor` e são embutidos no executável. Backend e operações de terminal/arquivos usam Go.

```text
frontend/                  interface e xterm.js, compartilhados
    │ bindings e eventos Wails
internal/app/              sessões, confiança SSH, transferências e casos de uso
    ├─ core/               perfis, eventos e contratos Terminal/Files
    ├─ remote/             SSH e SFTP, compartilhados
    ├─ localfiles/         operações de arquivos locais
    ├─ store/              persistência JSON
    └─ platform/
         ├─ *_windows.go   WSL, ConPTY, caminhos UNC e lock Windows
         └─ *_linux.go     PTY, caminhos POSIX e lock Linux
main.go                    composição e integração desktop Wails
scripts/                   build, testes, licenças, pacote e limpeza
tests/integration/ssh/     container opcional de validação
third_party/               licenças e manifestos de dependências
.local/                    módulos e artefatos de desenvolvimento
dist/                      executável, documentação e ZIP portátil
```

Núcleo não importa APIs Windows nem Wails. `app` é o ponto de integração com os adaptadores; contratos Go não expõem handles ou objetos WebView. Arquivos por plataforma e build constraints impedem carregar implementação Windows no Linux. Não criar framework de plugins nem camadas vazias para funcionalidades fora do escopo.

### Windows

Descobrir distribuições por `wsl.exe --list --quiet`, tratando UTF-16 e Unicode. Validar a distribuição escolhida contra o inventário. Iniciar WSL via ConPTY, com argumentos escapados individualmente, shell padrão e diretório inicial. Não editar arquivos do shell nem executar `wsl --shutdown`.

Arquivos WSL usam `\\wsl.localhost\<distribuição>`, com alternativa `\\wsl$`. Caminhos lógicos usam semântica POSIX e são convertidos somente no adaptador. Não acessar VHDX diretamente. Drenar a saída ao fechar ConPTY para evitar bloqueio.

WebView2 usa o runtime existente e grava seu perfil dentro da pasta portátil. Build Windows exige `wv2runtime.error`; falta do runtime produz erro, sem bootstrapper.

### Linux

PTY Unix inicia o shell existente e propaga tamanho. Arquivos usam filesystem POSIX e permissões do usuário corrente, sem sudo. SSH/SFTP, domínio, UI e transferências são compartilhados com Windows.

Desktop Linux depende de GTK/WebKitGTK e compilador compatíveis já disponíveis. Não instalar esses pré-requisitos, não prometer binário estático universal e não confundir testes de pacotes internos com validação do desktop. O ambiente gráfico Linux continua pendente de compilação e revisão próprias.

### SSH/SFTP

`golang.org/x/crypto/ssh` executa o protocolo, sem montar comando em shell local. Solicitar PTY `xterm-256color`; encaminhar entrada, saída, erro e resize. Timeout de conexão e handshake, keepalive e fechamento da sessão tratam perda de transporte. Não repetir comandos nem reconectar automaticamente.

Host novo exige confirmação com endereço, algoritmo e SHA-256 da chave pública. A confirmação deve corresponder exatamente a uma chave pendente, expira em dois minutos e tem limite de 64 solicitações pendentes. Reler `known_hosts` antes de confirmar impede que uma solicitação antiga autorize outra chave após mudança de confiança. `known_hosts` pertence ao aplicativo; chave alterada bloqueia o acesso. Não usar `InsecureIgnoreHostKey` nem modificar a confiança SSH pessoal.

Senha/passphrase ficam somente na memória necessária, sem promessa de apagamento garantido pelo runtime Go. Chave privada é selecionada por caminho. Não há agent, keyboard-interactive/MFA ou importação automática de credenciais.

SFTP usa canal próprio no transporte SSH. Falha de subsystem retornada pelo servidor permite terminal sem arquivos; timeout de abertura encerra a tentativa de conexão. Temporários têm nome aleatório e só são promovidos após concluir a cópia. Cancelamento tenta fechar streams e remover o temporário da própria operação.

## 5. API e persistência

API: `GetCapabilities`, `ListLocalTargets`, `ListProfiles`, `SaveProfile`, `DeleteProfile`, `OpenSession`, `StartReading`, `SendInput`, `Ack`, `ResizeSession`, `CloseSession`, `TrustHost`, `ListFiles`, `Mkdir`, `Rename`, `Remove`, `StartTransfer` e `CancelTransfer`.

Eventos: `terminal:data`, `session:closed`, `session:error` e `transfer:update`, associados ao ID da operação/sessão. Saída binária usa base64 por lote de até 32 KiB, aguardando confirmação da UI; não descartar silenciosamente saída.

`data/profiles.json` contém somente perfis SSH: schema, ID, nome, host, porta, usuário, caminho de chave e diretório inicial. WSL/terminal local conectam diretamente e não geram perfil salvo. Não há campos de senha, passphrase ou opções de protocolos excluídos. Conectar não grava nem atualiza perfis: somente a ação explícita **Salvar perfil** persiste a conexão. Sessões abertas são estado temporário em memória e não são restauradas ao iniciar o aplicativo. Gravação usa temporário e substituição atômica por plataforma; JSON inválido não é sobrescrito. `data/known_hosts` guarda apenas confiança pública SSH.

Dados ficam ao lado do executável, ou em caminho explícito por `--data-dir`. Sem fallback silencioso para AppData. Lock impede duas instâncias com os mesmos dados. Preferências visuais ficam em `data/webview`; largura dos painéis não persiste nesta versão. Conteúdo do terminal não é salvo em disco.

## 6. Segurança

Validar IDs, caminhos absolutos, caracteres, portas, dimensões e tamanho de entrada no backend. Rejeitar segmentos `.`/`..`, barras invertidas, UTF-8 inválido e controles em caminhos lógicos; nomes de entradas remotas não podem conter separadores ou segmentos especiais. Frontend não escolhe executáveis arbitrários ou fornece scripts ao backend. Nunca interpretar nome de arquivo como comando.

Renomeação local recusa sobrescrita no filesystem, inclusive em corrida entre processos. Linux usa `RENAME_NOREPLACE`; filesystems sem suporte podem usar criação exclusiva por hard link para arquivos regulares. Diretórios nesses filesystems retornam erro, sem fallback inseguro para renomeação com sobrescrita. A aplicação não é sandbox contra outro processo com as mesmas permissões que altere simultaneamente todos os componentes de um caminho.

WebView carrega assets locais com CSP; não renderizar terminal ou nomes de arquivos como HTML. Links externos do terminal, OSC 52 e drop de conteúdo externo são desativados. O terminal continua executando os comandos que o usuário digita: não é sandbox de shell.

Não registrar senhas, passphrases, chaves privadas, terminal ou conteúdo de arquivos em logs. Perfil do WebView e erro de inicialização ficam locais. Não alterar políticas de segurança para abrir o executável.

## 7. Build, limpeza e distribuição

`scripts/env.ps1` limita `GOPATH`, `GOMODCACHE`, `GOCACHE`, `GOBIN`, temporários e toolchain ao processo e à pasta `.local`. `GOTOOLCHAIN=local` impede download automático de outro Go. `go.mod`, `go.sum` e hashes de assets fixam as dependências.

`build.ps1` compila Windows; `test.ps1` executa testes/vet; `licenses.ps1` gera o inventário; `package.ps1` cria ZIP e SHA-256; `clean.ps1` remove apenas caches de build, logs de teste, arquivos baixados já incorporados e binários intermediários conhecidos. Dependências em `.local/gomodcache`, dados e código são preservados.

No Linux, `build.sh` usa variáveis equivalentes e os pré-requisitos existentes. Nenhum script usa instaladores globais ou modifica configuração persistente do sistema. Goja é dependência exclusivamente dos testes para executar o JavaScript real de temas/interface via `go test`, sem instalar Node.

Distribuição Windows contém executável, ícone SVG, README, SDD, validação, SHA-256 e licenças. ZIP não inclui `.local`, dados do usuário ou perfil do WebView.

## 8. Ciclo de vida do terminal Windows

Cada aba WSL possui um PID WSL iniciado pelo ConPTY. Ao fechar, o aplicativo encerra somente a árvore desse PID (`taskkill /T /F`), com timeout de três segundos, e então libera os handles ConPTY. O encerramento é idempotente; não executa `wsl --shutdown` e preserva outras distribuições e sessões. Teste de fechamento sem leitor usa `sleep 60`, sem saída contínua, e tem limite de cinco segundos.

## 9. Validação e pronto

Testar perfis, JSON corrompido, validação de entradas, confiança SSH, autenticação positiva/negativa, arquivos, hash de transferências, temporários e fechamento de sessões, incluindo confirmação de que processos filhos da aba encerrada não persistem. Incluir regressões de segurança para caminhos ambíguos, nomes remotos hostis, segredos persistidos, sobrescrita concorrente, limites de recursos, cancelamento, CSP e integridade de assets. Executar testes JavaScript de temas e prevenção de interpretação HTML no próprio `go test`.

`scripts/security.ps1` inclui testes/vet e fuzzing de caminhos. A opção `-AuditDependencies` consulta a OSV com nomes e versões públicos, sem código ou dados pessoais, e registra o resultado. A consulta cobre módulos importados no Windows, núcleo/adaptadores Linux, bibliotecas de teste, Go e assets npm; não equivale a análise de alcance por grafo de chamadas. Resultados e limites em `SECURITY.md`.

Validar WSL real e, quando disponível, SSH/OpenSSH em container vinculado apenas a loopback. Remover somente recursos criados pelo teste, sem prune global.

Para Linux, compilar e executar testes dos pacotes internos e do PTY em Linux real/WSL. Validação desktop exige sessão gráfica e dependências nativas compatíveis já existentes. Revisar foco, atalhos, diálogos e escalas de interface antes de declarar revisão visual concluída.

A entrega está pronta como implementação inicial quando build e testes passam, o pacote portátil contém apenas os arquivos necessários e os limites efetivos estão documentados. Ausência de revisão visual ou bloqueio do executável pelo Windows devem ser informados; não apresentar funcionalidade bloqueada como validada.
