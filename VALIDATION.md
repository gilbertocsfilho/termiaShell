# Validação

Escopo atual: WSL no Windows, terminal local no Linux, SSH/SFTP, navegador gráfico de arquivos e temas Claro/Escuro/Sistema. Esta revisão adiciona temas persistentes e testes de segurança; integrações de outros protocolos continuam fora do produto.

## Evidências funcionais da implementação

- Windows: testes Go e vet; perfis, persistência sem sobrescrever JSON inválido, caminhos, operações de arquivos e transferências com conferência SHA-256.
- WSL Ubuntu real: shell interativo, UTF-8, resize, diretório inicial, fechamento de ConPTY sem leitor, com encerramento da árvore WSL vinculada à aba e criação/listagem/renomeação/leitura de arquivos Unicode.
- SSH/SFTP por loopback: senha, chave criptografada, passphrase e senha incorretas, rejeição de host incorreto, PTY, resize e transferência com hash.
- Confiança SSH: confirmação exige fingerprint exata de uma chave pendente; confiança persistida e repetição da solicitação rejeitada.
- OpenSSH em container Alpine no Docker existente do WSL: terminal Linux real, `stty size` em 37 × 115 e transferência de 1.500.000 bytes com SHA-256 conferido. Publicação somente em `127.0.0.1:22289`.
- Linux: pacotes internos compilados com Go e testes executados de fato no Ubuntu WSL, incluindo shell PTY, resize em 36 × 113 e filesystem POSIX.
- Frontend: sintaxe JavaScript conferida; um build anterior inicializou WebView2, frontend e bridge Go. Diagnóstico temporário usado naquela conferência foi removido do código.

O teste de OpenSSH em container foi repetido após atualizar a biblioteca SSH nesta revisão e passou, incluindo resize e SFTP de 1.500.000 bytes com SHA-256. O container e a tag da imagem de teste foram removidos, sem prune global. Cache de build pode permanecer no Docker existente.

## Verificação desta revisão

Fechamento: confirmação nativa substituída por diálogo no tema do aplicativo. Testes direcionados cobrem confirmar, cancelar, Escape, pedidos repetidos, erro de comunicação e a reentrada de Quit sem novo bloqueio. Isso corrige a comparação entre a resposta nativa `Yes` e o rótulo personalizado `Encerrar`.

Passaram os testes Go/vet, testes do JavaScript real pelo Goja, testes WSL reais e testes dos pacotes internos e frontend executados no Ubuntu. Claro/Escuro/Sistema foram testados quanto a persistência, mudança do SO, terminais abertos, janela nativa e contraste de texto/ANSI. Também há testes para preferências adulteradas e armazenamento indisponível.

As regressões de segurança e a consulta OSV estão descritas em [SECURITY.md](SECURITY.md). A consulta final incluiu 30 pacotes/versões e não retornou alertas, após atualização das dependências afetadas. Fuzzing de caminhos passou. O build Windows, pacote e hash são regenerados; caches e binários intermediários podem ser removidos com `scripts/clean.ps1`, preservando módulos e dados.

## Limites conhecidos

- O Windows bloqueou a abertura de um build anterior com “Uma política de Controle de Aplicativo bloqueou este arquivo”. Não foi alterada ou contornada a política. Compilação bem-sucedida não comprova que o dispositivo autorizará o novo binário.
- A ferramenta de captura/interação visual retornou ausência do native pipe. Layout, foco, atalhos, diálogos e escalas não tiveram revisão visual automatizada nesta sessão.
- Desktop Linux GTK/WebKitGTK não foi compilado ou inspecionado. Testes dos adaptadores e núcleo não substituem essa etapa.
- Metas de latência/memória, cinco sessões simultâneas e cenários extensos de cancelamento sob falha de rede ainda não foram medidos.
- Transferências individuais não sobrescrevem destinos existentes e não copiam diretórios recursivamente. Não há acompanhamento de `cd` ou navegação por links simbólicos.
- Preferências de tema, fonte e arquivos ocultos são locais ao perfil WebView; largura dos painéis não persiste.

Nenhum pacote foi instalado globalmente no Windows ou na distribuição WSL. A pasta portátil inclui executável, documentação, licenças e hash, sem dados de usuário ou caches de desenvolvimento.


## Correção de encerramento WSL

A rotina de fechamento agora termina a árvore de processos do PID WSL pertencente à sessão antes de liberar os handles ConPTY. O encerramento é limitado a esse PID e possui timeout; não usa wsl --shutdown. O teste sem leitor usa sleep 60 e exige retorno de Close em até cinco segundos, sem gerar saída ilimitada. 


## Perfis e sessões

A conexão não persiste automaticamente ao conectar. Apenas **Salvar perfil** grava ou atualiza a lista lateral; sessões em andamento ficam em memória e não são reabertas no próximo início.


## Janela inicial

No Windows, a janela é exibida somente depois do carregamento inicial da interface; o callback centraliza a janela antes de mostrá-la. Tamanho padrão: 1200 × 760, mínimo: 850 × 560.


## Menu de contexto do explorador

Clique com o botão direito em um item para abrir/baixar, excluir, renomear, copiar o caminho, enviar o caminho ao terminal, consultar propriedades e ajustar permissões POSIX octais (000–777). Clique do meio envia o caminho citado ao terminal.


WSL é um atalho único de conexão direta para a distribuição padrão identificada pelo próprio ambiente WSL. Não aparece como perfil salvo; o backend também rejeita salvar WSL/terminal local como perfil.


## Importação CSV

Importador SSH limita CSV a 2 MB e 5.000 linhas, exige `name`, `host` e `user`, usa porta 22 quando omitida, valida perfis, ignora campos de credenciais e pula duplicatas por host/porta/usuário. Mostra erros por linha (até 20 no resumo).


## Revisão de segurança adicional

Passaram `go test -tags wv2runtime.error ./internal/... ./tests/frontend` e `go vet -tags wv2runtime.error ./...`, fuzzing de caminhos (5 s, 17.126 execuções) e OSV (30 pacotes/versões, zero alertas em 2026-11-06 20:11 UTC). O Windows bloqueou apenas `termiashell.test.exe` do pacote raiz pela política de Controle de Aplicativo; não houve contorno. Detalhes em `SECURITY.md`.
