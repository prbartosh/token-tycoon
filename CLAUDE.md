# Token Tycoon

- Decyzje projektowe: `docs/DECISIONS.md` (nadrzędne wobec DESIGN i ARCHITECTURE).
- Bez komentarzy w kodzie (DECISIONS 17).
- Uruchamiając binarkę w testach, ustaw absolutne `HOME` i `XDG_CONFIG_HOME` w katalogu tymczasowym; nigdy nie dotykaj `~/.config/token-tycoon`.
- Przed commitem: `gofmt -l .` pusty, `go vet ./...`, `go test ./...`.
- Wydanie i strona: README, sekcja „Wydanie i strona”. Wersja = tag `v*` na `main`; strona tylko w `site/`, synchronizacja do `prbartosh/token-tycoon-releases` jest automatyczna.
