# Decisões de arquitetura (ADR)

Formato: Contexto, Decisão, Consequências. As ADRs de 0001 a 0005 registram decisões que já estavam no código antes deste documento. Por isso o status é "Aceita (registrada a posteriori)".

| ADR | Título | Status |
|---|---|---|
| [0001](0001-rclone-como-binario-externo.md) | rclone como binário externo, chamado pela linha de comando | Aceita (registrada a posteriori) |
| [0002](0002-gui-fyne-e-systray.md) | Interface em Fyne v2 e bandeja com getlantern/systray | Aceita (registrada a posteriori) |
| [0003](0003-montagem-winfsp-fuse.md) | Montagem por `rclone mount` sobre WinFsp, FUSE ou macFUSE | Aceita (registrada a posteriori) |
| [0004](0004-onde-fica-a-configuracao.md) | Onde fica a configuração | Aceita (registrada a posteriori) |
| [0005](0005-senha-em-memoria-e-obscure.md) | Senha da sessão em memória e senha do crypt ofuscada no `rclone.conf` | Aceita (registrada a posteriori) |
| [0006](0006-onde-ficam-os-segredos.md) | Onde ficam os segredos dos cofres | **Proposta** — decisão de Douglas pendente |
| [0007](0007-interface-wails.md) | Interface em Wails no lugar de Fyne | **Proposta** — escolha da versão (v3 beta, v2 sem bandeja ou v2 com bandeja à parte) pendente |

Novo ADR: copiar o formato, usar o próximo número e ligar à demanda que o pede.
