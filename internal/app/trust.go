package app

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"termiashell/internal/core"
	"termiashell/internal/remote"
)

const trustLifetime = 2 * time.Minute
const maxPendingTrust = 64

type pendingTrust struct {
	key     ssh.PublicKey
	created time.Time
	remote  net.Addr
}

func (a *App) hostKeyCallback(p core.Profile) ssh.HostKeyCallback {
	return func(host string, addr net.Addr, key ssh.PublicKey) error {
		a.trustMu.Lock()
		defer a.trustMu.Unlock()
		cb, e := knownhosts.New(filepath.Join(a.data, "known_hosts"))
		if e != nil {
			return e
		}
		e = cb(host, addr, key)
		if e == nil {
			return nil
		}
		var ke *knownhosts.KeyError
		if !errors.As(e, &ke) || len(ke.Want) > 0 {
			return errors.New("Chave do servidor SSH mudou. Conexão bloqueada; confira o servidor e revise data/known_hosts manualmente")
		}
		now := time.Now()
		for host, pending := range a.pending {
			if now.Sub(pending.created) > trustLifetime {
				delete(a.pending, host)
			}
		}
		address := core.Address(p)
		if _, exists := a.pending[address]; !exists && len(a.pending) >= maxPendingTrust {
			return errors.New("Muitas confirmações SSH pendentes; aguarde a expiração")
		}
		a.pending[address] = pendingTrust{key: key, created: now, remote: addr}
		return remote.FingerprintError(key)
	}
}

func (a *App) TrustHost(host string, port int, fingerprint string) error {
	p := core.Profile{Name: "Verificação de host", Kind: "ssh", Host: host, Port: port, User: "trust-check"}
	if e := p.Validate(); e != nil {
		return e
	}
	a.trustMu.Lock()
	defer a.trustMu.Unlock()
	address := core.Address(p)
	pending, ok := a.pending[address]
	if !ok || pending.key == nil || pending.remote == nil || time.Since(pending.created) > trustLifetime {
		delete(a.pending, address)
		return errors.New("Solicitação de confiança expirou; conecte novamente")
	}
	if ssh.FingerprintSHA256(pending.key) != fingerprint {
		return errors.New("Fingerprint não corresponde à chave apresentada")
	}
	// O arquivo pode ter mudado enquanto o diálogo estava aberto. Uma solicitação
	// pendente não autoriza substituir uma chave que passou a ser conhecida.
	cb, e := knownhosts.New(filepath.Join(a.data, "known_hosts"))
	if e != nil {
		return e
	}
	e = cb(address, pending.remote, pending.key)
	if e == nil {
		delete(a.pending, address)
		return nil
	}
	var ke *knownhosts.KeyError
	if !errors.As(e, &ke) || len(ke.Want) > 0 {
		return errors.New("Confiança SSH mudou durante a confirmação; operação bloqueada")
	}
	f, e := os.OpenFile(filepath.Join(a.data, "known_hosts"), os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if _, e = fmt.Fprintln(f, knownhosts.Line([]string{knownhosts.Normalize(address)}, pending.key)); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	delete(a.pending, address)
	return nil
}
