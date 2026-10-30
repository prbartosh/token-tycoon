# Token Tycoon

Terminalowa gra idle tycoon o branży AI (Go, Bubble Tea v2). Stan: **M0 do M3** (silnik, balans, zapis, symulator, interfejs TUI, infrastruktura, prestiż 1, karty, zdarzenia, dzienny grant, rynek z rywalami i segmentami, trening modeli i dane). Zakładki Badania i Polityki czekają na kolejne etapy.

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
./token-tycoon            # gra w terminalu (min. 60x14, pełny układ od 120 kolumn)
./token-tycoon --lang en  # język: pl | en (domyślnie wg LANG)
./token-tycoon --no-color # tryb ASCII bez kolorów
./token-tycoon --debug    # zapis z wcięciami, FPS w nagłówku
```

Sterowanie:

| Klawisz | Akcja |
|---|---|
| ← → , 1 do 7 | zakładki (Biznesy, Modele, Rynek, Infra, Badania, Polityki, Statystyki) |
| ↑ ↓ , k j | wybór wiersza (w Statystykach przewijanie) |
| Enter | akcja główna: kup, przełącz, trenuj, zmień segment modelu |
| Tab | ilość x1, x10, x100, MAX (biznesy, racki, scrapery) |
| + - | zmiana wartości: N, D i udział klastra treningu, udział zadania, dane syntetyczne, suwak segmentów |
| x | sprzedaj rack, anuluj trening, wycofaj model (z potwierdzeniem) |
| s | podpowiedź treningu (Modele) |
| m, u | menedżer, cofnięcie zakupu (5 s) |
| Spacja | boost i combo |
| p | podgląd i potwierdzenie prestiżu |
| H, G | reakcja na zdarzenie (Halucynacja, Inwestor anioł) |
| d | dzienny grant |
| n, a, c | notacja, doradca, symbole dla daltonistów |
| ? , q | pomoc, zapis i wyjście |

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

Raport pokazuje prestiże, karty, zdarzenia, infrastrukturę, rynek, rywali i modele oraz sprawdza kryteria z DESIGN sekcja 10 mierzalne od M0: płaskie odcinki w 1. godzinie, czas do pierwszego menedżera, próg infrastruktury (łącznie 1e7), próg prestiżu (run 1e12), kolejność, odblokowania w sesjach.

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
| `internal/sim` | stan gry, komendy, `Step`, zdarzenia, offline, infra, prestiż, karty, rynek, trening |
| `internal/infra` | indeks energii, dyspozycja źródeł, dławienie, koszt energii |
| `internal/cards` | efekty kolekcji kart i losowanie |
| `internal/market` | cena, udział, popyt, deflacja, hype, harmonogram rywali |
| `internal/training` | scaling laws, FLOPs, optymalny podział N/D |
| `internal/ui` | Bubble Tea: pętla klatek, zakładki, nakładki, animacje (`anim`, `theme`, `widgets`) |
| `internal/save` | zapis JSON, kopie, migracje |
| `internal/rng` | deterministyczne strumienie PCG |
| `internal/i18n` | katalogi PL i EN |
| `internal/format` | liczby (K, M, B, T, aa...) i czasy |
| `internal/simulate` | boty, raport, CSV |
