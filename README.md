# Token Tycoon

Terminalowa gra idle tycoon o branży AI (Go, Bubble Tea v2). Etap **M0**: silnik symulacji, balans, zapis i symulator balansu. Interfejs gry pojawi się w M1.

Dokumentacja: `docs/DESIGN.md`, `docs/ARCHITECTURE.md`, `docs/DECISIONS.md` (ma pierwszeństwo).

## Budowanie

```
go build -o token-tycoon ./cmd/token-tycoon
go test ./...
go vet ./...
```

Wymaga Go 1.27.3. Zależności są przypięte w `go.mod`.

## Uruchomienie

```
./token-tycoon            # w M0 wypisuje informację o M1 i kończy z kodem 0
./token-tycoon --lang en  # język: pl | en (domyślnie wg LANG)
```

## Symulator balansu

```
./token-tycoon --simulate 1h
./token-tycoon --simulate 1d --strategy greedy --seed 42 --out run.csv
./token-tycoon --simulate 7d --strategy passive --sessions 10m/day
./token-tycoon --simulate 1h --strategy random
```

Flagi:

| Flaga | Znaczenie |
|---|---|
| `--simulate` | czas gry: `90s`, `10m`, `1h`, `1d`, `30d` |
| `--strategy` | `greedy` (stała gra, najlepszy stosunek koszt / przyrost $/s), `passive` (sesja dziennie, reszta offline), `random` |
| `--seed` | ziarno RNG (ten sam seed = ten sam wynik) |
| `--out` | zapis próbek do CSV |
| `--sessions` | dla `passive`: czas aktywnej gry dziennie, np. `10m/day` |
| `--combo-rate` | naciśnięcia Spacji na sekundę w trakcie gry (domyślnie 0, wyłączone) |
| `--click-latency` | po ilu sekundach bot ponownie startuje cykl biznesu bez menedżera (domyślnie 0.5) |
| `--balance` | plik nadpisujący balans (domyślnie `~/.config/token-tycoon/balance.toml`) |
| `--ci` | kod wyjścia 1, gdy kryteria akceptacji nie są spełnione |

Raport sprawdza kryteria z DESIGN sekcja 10 mierzalne w M0: płaskie odcinki w 1. godzinie, czas do pierwszego menedżera, próg infrastruktury (łącznie 1e7), próg prestiżu (run 1e12), kolejność, odblokowania w sesjach.

## Balans

Domyślny balans jest wbudowany w binarkę (`internal/balance/balance.toml`). Lokalny plik `~/.config/token-tycoon/balance.toml` nadpisuje tylko podane klucze; tablice `[[business]]`, `[[upgrade]]` itd. są łączone po `id`. Przykład:

```toml
[crit]
chance = 0.05

[[business]]
id = "gpu_rental"
unlock_lifetime = 0
```

## Zapis gry

`~/.config/token-tycoon/save.json` (Windows: `%APPDATA%\token-tycoon\save.json`), zapis atomowy, 3 kopie zapasowe (`save.json.1` do `.3`), migracje wersji.

## Pakiety

| Pakiet | Rola |
|---|---|
| `internal/balance` | wczytanie, scalanie i walidacja balansu, typy wszystkich sekcji |
| `internal/econ` | czyste wzory: koszty, MAX, milestone'y, prestiż, krytyki |
| `internal/sim` | stan gry, komendy, `Step`, zdarzenia, offline |
| `internal/save` | zapis JSON, kopie, migracje |
| `internal/rng` | deterministyczne strumienie PCG |
| `internal/i18n` | katalogi PL i EN |
| `internal/format` | liczby (K, M, B, T, aa...) i czasy |
| `internal/simulate` | boty, raport, CSV |
