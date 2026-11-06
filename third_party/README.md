# Dependências

Assets distribuídos localmente, obtidos do registry oficial npm sem instalar npm:

- `@xterm/xterm` 5.5.0: https://registry.npmjs.org/@xterm/xterm/-/xterm-5.5.0.tgz
- `@xterm/addon-fit` 0.10.0: https://registry.npmjs.org/@xterm/addon-fit/-/addon-fit-0.10.0.tgz

Licenças MIT em `xterm-LICENSE` e `addon-fit-LICENSE`. Hashes SHA-256 dos arquivos efetivamente embutidos em `assets-sha256.json`.

Módulos Go fixados em `go.mod` e `go.sum`, listados em `go-modules.json`, com cópias das licenças em `go/`. `scripts/licenses.ps1` regenera esse inventário a partir do cache local. Os diretórios `.local` e `data` não precisam acompanhar a distribuição do executável.
