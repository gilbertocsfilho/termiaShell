# Servidor SSH descartável

Usar somente Docker já disponível no WSL. O Containerfile instala OpenSSH **na imagem**, sem modificar o Windows ou os pacotes da distribuição WSL. Conta `tester`, senha descartável `termiashell-test-only`. Nunca usar essa senha fora do container de teste.

Da pasta deste projeto no WSL:

```sh
docker build -f tests/integration/ssh/Containerfile -t termiashell-ssh-test:local tests/integration/ssh
docker run -d --name termiashell-ssh-validation --label app=termiashell-validation -p 127.0.0.1:22289:22 termiashell-ssh-test:local
docker exec termiashell-ssh-validation ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub -E sha256
```

Confirme que o container publica somente loopback. No PowerShell, com `scripts/env.ps1` carregado, defina `TERMIASHELL_SSH_TEST_PORT=22289` e `TERMIASHELL_SSH_TEST_FINGERPRINT` com o valor SHA256 obtido acima. Execute `go test ./internal/remote -run TestContainerSSH -v`.

Ao concluir, remova somente o container nomeado `termiashell-ssh-validation` e a imagem `termiashell-ssh-test:local`. Não faça prune global. A imagem base Alpine puxada para esse teste só deve ser removida se não estiver em uso por outros containers.
