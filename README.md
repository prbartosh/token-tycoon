# Token Tycoon

Terminalowa gra idle tycoon o branży AI (Go, Bubble Tea v2). Stan: **M0 do M5** (silnik, balans, zapis, symulator, interfejs TUI, infrastruktura, prestiż 1, karty, zdarzenia, dzienny grant, rynek z rywalami i segmentami, trening modeli i dane, talent, kontrakty enterprise, runda finansowania, drzewko badań, polityki, regiony, incydenty safety, inwestorzy, wyścig do AGI z finałem i trybem Superinteligencji, osiągnięcia, satyryczny pasek newsów).

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
| Enter | akcja główna: kup, przełącz, trenuj, zmień segment modelu, wejdź do regionu, zainwestuj w etap AGI |
| Tab | ilość x1, x10, x100, MAX (biznesy, racki, scrapery) |
| + - | zmiana wartości: N, D i udział klastra treningu, udział zadania, dane syntetyczne, suwak segmentów, priorytet polityki |
| x | sprzedaj rack, anuluj trening, wycofaj model (z potwierdzeniem), zwolnij osobę, odrzuć ofertę kontraktu, usuń politykę |
| s | podpowiedź treningu (Modele) |
| m, u | menedżer, cofnięcie zakupu (5 s) |
| Spacja | boost i combo |
| p | podgląd i potwierdzenie prestiżu (nowa wersja modelu) |
| P (Shift+p) | podgląd i potwierdzenie rundy finansowania (prestiż warstwy 2, kapitał VC) |
| H, G | reakcja na zdarzenie (Halucynacja, Inwestor anioł) |
| o | kontroferta dla podkupywanego badacza (60 s) |
| r | reakcja na incydent safety (10 s): zamiast utraty hype premia reputacji |
| d | dzienny grant |
| n, a, c | notacja, doradca, symbole dla daltonistów |
| ? , q | pomoc, zapis i wyjście |

## Systemy M4

- **Talent** (zakładka Modele, od łącznie 1e22): junior, senior, gwiazda; pensja jako procent przychodu, premia do szybkości badań i treningu, gwiazda z losową cechą (szybki, perfekcjonista, ewangelista, oszczędny). Opłata rekrutacyjna w minutach najlepszego przychodu w historii (jak SMR), więc zatrudnianie tuż po prestiżu nie jest darmowe. Podkupienie: 60 s na kontrofertę (`o`), podwójna pensja przez 3 dni zegara gry. Offline nikt nie odchodzi.
- **Kontrakty enterprise** (zakładka Rynek, od 1e22): do 3 ofert na tablicy, do 3 aktywnych (więcej z badań). Wolumen = 1.5 do 4 × bieżąca produkcja × czas. Nagroda w minutach przychodu plus reputacja, porażka tylko obniża reputację. Termin stoi w miejscu, gdy gra jest offline.
- **Runda finansowania** (`P`, od łącznie 1e40): kapitał = floor(5 × (log10(przychód od ostatniej rundy) − 39)^1.5), resetuje to co prestiż 1 oraz parametry; ceremonia z licznikiem wyceny.
- **Badania** (zakładka Badania): 34 węzły w 6 gałęziach (`[[research_node]]` w `balance.toml`), kupowane kapitałem VC, czas badania skracany przez talent.
- **Polityki** (zakładka Polityki, od 1e44): kreator z listami wyboru (Enter na „+ nowa polityka”), do 2 warunków (I) i jedna akcja, sloty 3 do 12 z badań, kolejność = priorytet, ocena co 1 s, log wykonania; offline tylko z badaniem „Polityki offline”.

## Systemy M5

- **Regiony** (zakładka Rynek, od łącznie 1e54): US, EU, Azja z udziałami rynku 40/30/30. Odblokowanie niczego nie zmienia, każdy region po wejściu mnoży popyt (udział regionów z wejściem / udział US, do x2.5). Wejście do EU to compliance AI Act (120 min przychodu), do Azji lokalne data center (180 min). Segment Enterprise sprzedaje tylko w regionach z compliance enterprise (US od razu, EU i Azja po 60 min przychodu), gdzie indziej jego tokeny idą po cenie Pro. Koszty liczone od najlepszego przychodu (jak SMR), węzeł badań „Biuro AI Act” obniża je o połowę.
- **Incydenty safety** (od 1e13, DECISIONS 13a): jailbreak w mediach, halucynacja u klienta, wyciek danych. Szansa 1 na 4 h gry, z badaniami safety do 1 na 48 h. Okno 10 s na `r`: reakcja daje reputację, brak reakcji zabiera część hype. Tylko online; offline aktywny incydent znika bez kary. Etap AGI „Alignment” wyłącza incydenty.
- **Inwestorzy** (zakładka Rynek, od 1e29): wycena = przychód roczny x (1 + hype) x mnożnik wzrostu 7-dniowego x `valuation_mult`. Kwartał co 24 h zegara gry z celem +50 do +200% względem poprzedniego; pobity kwartał dodaje +25% do kapitału następnej rundy finansowania.
- **Wyścig do AGI** (zakładka Badania, od 1e94): 5 etapów z `[[agi_stage]]`, Enter inwestuje całą gotówkę w bieżący etap (wkład przetrwa prestiż). Efekty: incydenty wyłączone, wszystkie biznesy x2, dane syntetyczne bez kary, jeden trening 24 h z modelem 100 pkt, finał z ekranem napisów i statystyk. Potem tryb Superinteligencji: poziom co 10 rzędów wielkości przychodu, każdy dodaje x1 do przychodu, czasy poziomów zapisywane jako rekordy.
- **Osiągnięcia**: 71 definicji w `internal/sim/achievements.go`, sprawdzane raz na sekundę gry, toast wjeżdżający z prawej, lista w Statystykach.
- **Pasek newsów**: 45 satyrycznych nagłówków z szablonami reagującymi na stan gry (rywale, hype, energia, regiony, AGI...) na przemian z 12 ogólnymi.

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
