# proxy

`ncp/` contains reusable reverse proxy code extracted from Traefik and kept close to the upstream implementation.

- HTTP/HTTPS share the same reverse proxy core, matching Traefik's `pkg/proxy/httputil` behavior.
- TCP mirrors Traefik's bidirectional stream proxy and dialer handling from `pkg/tcp`.
- UDP mirrors Traefik's session-style UDP listener and proxy from `pkg/udp`.

Key entry points:

- `NewHTTPTransportManager`
- `NewHTTPProxyBuilder`
- `NewSingleHostHTTPProxy`
- `NewSingleHostHTTPSProxy`
- `NewTCPDialerManager`
- `NewTCPProxy`
- `ListenUDP`
- `NewUDPProxy`
