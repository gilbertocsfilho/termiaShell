# termiaShell

Um terminal simples para trabalhar com Linux local, WSL e servidores SSH, com navegador de arquivos integrado.

![Tela inicial do termiaShell](screenshots/termiaShell.png)

## Recursos

- Terminal em abas: Linux local, WSL no Windows e SSH.
- Navegador gráfico de arquivos para WSL e SFTP.
- Sessões SSH salvas e importação de conexões por CSV.
- Temas claro, escuro e sistema.

## Requisitos

- **Windows:** WebView2. WSL é necessário apenas para usar distribuições WSL.
- **Linux:** Go e dependências gráficas GTK/WebKitGTK já disponíveis no sistema.

## Compilar

Os scripts mantêm caches e módulos dentro de `.local` e não instalam pacotes globalmente.

**Windows (PowerShell):**

```powershell
.\scripts\build.ps1
.\dist\termiaShell.exe
```

**Linux:**

```sh
sh scripts/build.sh
./dist/termiaShell
```

Senhas SSH não são salvas. Antes de confiar em um servidor, confira sua chave de host.

Mais detalhes: [arquitetura](SDD.md) · [segurança](SECURITY.md) · [validação](VALIDATION.md) · [licenças de terceiros](third_party/README.md).
