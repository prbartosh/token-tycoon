# Token Tycoon

Terminalowa gra idle tycoon o branży AI (Go, Bubble Tea v2). Zaczynasz od jednego darmowego Wrappera na API i budujesz imperium: biznesy i menedżerowie, własna infrastruktura (racki, energia, scrapery), trening modeli, rynek z rywalami i segmentami, talent, kontrakty enterprise, prestiż (nowa wersja modelu), rundy finansowania, drzewko badań, polityki automatyzacji, regiony, incydenty safety, inwestorzy i wyścig do AGI z finałem oraz trybem Superinteligencji. Gra liczy też czas offline, ma 71 osiągnięć i satyryczny pasek newsów. Interfejs w pełni po polsku i angielsku.

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
./token-tycoon --lang en  # język: pl | en (domyślnie wg LANG, pl* = PL, inaczej EN)
./token-tycoon --no-color # tryb ASCII bez kolorów
./token-tycoon --debug    # zapis z wcięciami, FPS w nagłówku
./token-tycoon --balance moj.toml  # własny plik balansu zamiast ~/.config/token-tycoon/balance.toml
```

Język można też zmienić w grze (paleta komend, `ustawienia`). Zapis następuje przy wyjściu, automatycznie co 60 s (`autosave_s` w balansie) i komendą `zapisz`.

## Sterowanie

| Klawisz | Akcja |
|---|---|
| ← → , 1 do 7 | zakładki (Biznesy, Modele, Rynek, Infra, Badania, Polityki, Statystyki) |
| ↑ ↓ , k j | wybór wiersza (w Statystykach przewijanie) |
| Home, End, PgUp, PgDn | skok na początek, koniec listy, strona w górę, w dół |
| Enter | akcja główna: kup, przełącz, trenuj, zmień segment modelu, wejdź do regionu, przywróć model z archiwum, okno inwestycji w etap AGI |
| Tab, Shift+Tab | ilość x1, x10, x100, MAX (biznesy, racki, scrapery), Shift+Tab w drugą stronę |
| + - | zmiana wartości: N, D i udział klastra treningu, udział zadania, dane syntetyczne, suwak segmentów, priorytet polityki |
| x | sprzedaj rack, anuluj trening, wycofaj model lub usuń go z archiwum (z potwierdzeniem), zwolnij osobę, odrzuć ofertę kontraktu, usuń politykę |
| s | podpowiedź treningu, optymalny podział N/D (Modele) |
| m | zatrudnij menedżera (Biznesy) |
| u | cofnięcie ostatniego zakupu (5 s) |
| Spacja | boost: +0.1 combo (max x5, gaśnie po 2 s), combo mnoży cały przychód; startuje też cykl zaznaczonego biznesu bez menedżera |
| p | podgląd i potwierdzenie prestiżu (nowa wersja modelu) |
| P (Shift+p) | podgląd i potwierdzenie rundy finansowania (prestiż warstwy 2, kapitał VC) |
| H, G | reakcja na zdarzenie (Halucynacja, Inwestor anioł) |
| o | kontroferta dla podkupywanego badacza (60 s) |
| r | reakcja na incydent safety (10 s): zamiast utraty hype premia reputacji |
| d | dzienny grant |
| n, a, c | notacja liczb, doradca, symbole dla daltonistów |
| : , Ctrl+P | paleta komend z wyszukiwaniem rozmytym (PL i EN): `kup rag 50`, `kup max`, `trenuj`, `idź badania`, `prestiż podgląd`, `runda finansowania podgląd`, `polityki`, `zapisz`, `ustawienia` (język na żywo, notacja, daltonizm, doradca, dzwonek), `pomoc`, `wyjdź` |
| Esc | zamyka okno, pomija ceremonię (prestiż, premiera, runda; ceremonie nie blokują klawiszy) |
| ? | pomoc kontekstowa (najpierw klawisze bieżącej zakładki) |
| q, Ctrl+C | zapis i wyjście |
| mysz | klik zaznacza wiersz i zakładkę, podwójny klik wykonuje akcję główną, kółko przewija |

W oknach potwierdzeń (prestiż, runda, wycofanie modelu, usunięcie z archiwum) Enter lub y zatwierdza, Esc lub n anuluje.

Limit 3 modeli (DECISIONS 13b): gdy ręcznie rozpoczęty trening kończy się przy komplecie, gra pyta, który model wycofać (można odrzucić nowy, Esc odkłada decyzję, wiersz „czeka na decyzję” w zakładce Modele). Offline i przy politykach zostaje lepszy model, drugi trafia do archiwum. Żaden model nie przepada bez decyzji gracza.

## Systemy M4

- **Talent** (zakładka Modele, od łącznie 4e20): junior, senior, gwiazda; pensja jako procent przychodu, premia do szybkości badań i treningu, gwiazda z losową cechą (szybki, perfekcjonista, ewangelista, oszczędny). Opłata rekrutacyjna w minutach najlepszego przychodu w historii (jak SMR), więc zatrudnianie tuż po prestiżu nie jest darmowe. Podkupienie: 60 s na kontrofertę (`o`), podwójna pensja przez 3 dni zegara gry. Offline nikt nie odchodzi.
- **Kontrakty enterprise** (zakładka Rynek, od 8.4e20): do 3 ofert na tablicy, do 3 aktywnych (więcej z badań). Wolumen = 1.5 do 4 × stała produkcja × czas (dostawa liczy stałą produkcję: bez combo, krytyków i boostów zdarzeń, DECISIONS 14a). Nagroda w minutach przychodu plus reputacja, porażka tylko obniża reputację. Termin stoi w miejscu, gdy gra jest offline.
- **Runda finansowania** (`P`, od łącznie 6.6e33): kapitał = floor(5 × (log10(przychód od ostatniej rundy) − 30)^1.5), resetuje to co prestiż 1 oraz parametry, numer wersji i aktywne kontrakty. Bonus parametrów przechodzi do trwałego mnożnika rund (DECISIONS 20); ceremonia z licznikiem wyceny.
- **Badania** (zakładka Badania): 34 węzły w 6 gałęziach (`[[research_node]]` w `balance.toml`), kupowane kapitałem VC, czas badania skracany przez talent.
- **Polityki** (zakładka Polityki, od 2.7e37): kreator z listami wyboru (Enter na „+ nowa polityka”), do 2 warunków (I) i jedna akcja, sloty 3 do 12 z badań, kolejność = priorytet, ocena co 1 s, log wykonania; offline tylko z badaniem „Polityki offline”.

## Systemy M5

- **Regiony** (zakładka Rynek, od łącznie 4.2e48): US, EU, Azja z udziałami rynku 40/30/30. Odblokowanie niczego nie zmienia, każdy region po wejściu mnoży popyt (udział regionów z wejściem / udział US, do x2.5). Wejście do EU to compliance AI Act (120 min przychodu), do Azji lokalne data center (180 min). Segment Enterprise sprzedaje tylko w regionach z compliance enterprise (US od razu, EU i Azja po 60 min przychodu), gdzie indziej jego tokeny idą po cenie Pro. Koszty liczone od najlepszego przychodu (jak SMR), węzeł badań „Biuro AI Act” obniża je o połowę.
- **Incydenty safety** (od 3.1e12, DECISIONS 13a): jailbreak w mediach, halucynacja u klienta, wyciek danych. Szansa 1 na 4 h gry, z badaniami safety do 1 na 48 h. Okno 10 s na `r`: reakcja daje reputację, brak reakcji zabiera część hype. Tylko online; offline aktywny incydent znika bez kary. Etap AGI „Alignment” wyłącza incydenty.
- **Inwestorzy** (zakładka Rynek, od 1.7e26): wycena = przychód roczny x (1 + hype) x mnożnik wzrostu 7-dniowego x `valuation_mult`. Kwartał co 24 h zegara gry z celem +50 do +200% względem poprzedniego; pobity kwartał dodaje +25% do kapitału następnej rundy finansowania.
- **Wyścig do AGI** (zakładka Badania, od 1e94): 5 etapów z `[[agi_stage]]`, Enter otwiera okno z porcjami 10%, 50%, 100% gotówki albo brakującą kwotą i potwierdzeniem (wkład przetrwa prestiż i rundę). Efekty: incydenty wyłączone, wszystkie biznesy x2, dane syntetyczne bez kary, jeden trening 24 h z modelem 100 pkt, finał z ekranem napisów i statystyk. Potem tryb Superinteligencji: poziom co 10 rzędów wielkości przychodu, każdy dodaje x1 do przychodu, czasy poziomów zapisywane jako rekordy.
- **Osiągnięcia**: 71 definicji w `internal/sim/achievements.go`, sprawdzane raz na sekundę gry, toast wjeżdżający z prawej, lista w Statystykach.
- **Pasek newsów**: 45 satyrycznych nagłówków z szablonami reagującymi na stan gry (rywale, hype, energia, regiony, AGI...) na przemian z 12 ogólnymi.

## Symulator balansu

Ten sam binarny plik uruchamia symulator zamiast gry, gdy podasz `--simulate`:

```
./token-tycoon --simulate 1h
./token-tycoon --simulate 1d --strategy greedy --seed 42 --out run.csv
./token-tycoon --simulate 7d --strategy passive --sessions 10m/day
./token-tycoon --simulate 1h --strategy random
./token-tycoon --simulate 30d --strategy greedy --ci
./token-tycoon --simulate 1d --balance moj.toml
```

Flagi:

| Flaga | Znaczenie |
|---|---|
| `--simulate` | czas gry: `90s`, `10m`, `1h`, `1d`, `30d` |
| `--strategy` | `greedy` (gracz z DESIGN 10: jedna sesja 60 min dziennie, reszta offline, najlepszy stosunek koszt / przyrost $/s; `[pacing]` w `balance.toml`), `passive` (krótka sesja dziennie, reszta offline), `random` |
| `--seed` | ziarno RNG (domyślnie 42; ten sam seed = ten sam wynik) |
| `--out` | zapis próbek do CSV |
| `--sessions` | dla `passive`: czas aktywnej gry dziennie, np. `10m/day` (domyślnie) |
| `--combo-rate` | naciśnięcia Spacji na sekundę w trakcie gry (domyślnie 0, wyłączone) |
| `--click-latency` | po ilu sekundach bot ponownie startuje cykl biznesu bez menedżera (domyślnie 0.5) |
| `--balance` | plik nadpisujący balans (domyślnie `~/.config/token-tycoon/balance.toml`) |
| `--ci` | kod wyjścia 1, gdy kryteria akceptacji nie są spełnione (do CI); bez niego raport tylko informuje |
| `--lang` | język raportu: pl lub en |

Kody wyjścia: 0 sukces, 1 niespełnione kryteria przy `--ci`, 2 błąd argumentów, balansu lub zapisu CSV.

Raport pokazuje prestiże, karty, zdarzenia, infrastrukturę, rynek, rywali, modele i najlepszy przychód każdego dnia oraz sprawdza kryteria z `[pacing]` (DECISIONS 24): płaski odcinek w 1. godzinie, pierwszy menedżer, infrastruktura i prestiż w czasie gry oraz ich kolejność, coś nowego w pierwszych 5 min każdej sesji, nowy cel każdego dnia do dnia 20, pasmo przychodu z tabeli DESIGN 10, dzień rundy (5 do 7), AGI (18 do 20) i finału (greedy 25 do 32, pasywny 40 do 50).

## Balans

Domyślny balans jest wbudowany w binarkę (`internal/balance/balance.toml`). Plik `~/.config/token-tycoon/balance.toml` (albo ścieżka z `--balance`) nadpisuje tylko podane klucze; tablice `[[business]]`, `[[upgrade]]` itd. są łączone po `id`. Nieprawidłowy plik kończy start kodem 2 z listą błędów. Dotyczy gry i symulatora. Przykład:

```toml
[crit]
chance = 0.05

[[business]]
id = "gpu_rental"
unlock_lifetime = 0
```

## Zapis gry

`~/.config/token-tycoon/save.json` (Windows: `%APPDATA%\token-tycoon\save.json`, przy ustawionym `XDG_CONFIG_HOME` na Linuksie i macOS: `$XDG_CONFIG_HOME/token-tycoon/save.json`). Zapis atomowy, 3 kopie zapasowe (`save.json.1` do `.3`), migracje wersji (obecna wersja zapisu: 8). Przy uszkodzonym `save.json` gra wczytuje najnowszą dobrą kopię zapasową i pokazuje komunikat; gdy nie da się odczytać żadnego pliku, zapis trafia do `save.json.corrupt`, a gra startuje od nowa. Zapis z nowszej wersji gry nie jest wczytywany. Przykładowy zapis z symulatora (1 dzień, bot greedy, seed 42): `docs/save.example.json`.

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
