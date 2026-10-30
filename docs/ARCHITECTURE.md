# Token Tycoon: architektura

Stack: Go, Bubble Tea (pętla i input), Lip Gloss (style), Bubbles (komponenty), Harmonica (animacje sprężynowe), go-toml v2 (balans). Wersje przypięte w `go.mod`.

## 1. Zasady

1. **Symulacja jest czysta**: `sim` nie importuje niczego z UI, nie zna czasu systemowego, nie robi I/O. Wejście: stan, balans, dt, komendy. Wyjście: nowy stan i zdarzenia.
2. **Jeden silnik dla trzech trybów**: gra, nadrabianie offline, symulator balansu.
3. **Determinizm**: cała losowość z seedowanych strumieni RNG zapisanych w stanie.
4. **UI tylko czyta stan** i wysyła komendy. Animacje żyją wyłącznie w UI.
5. **Balans poza kodem**: wszystkie stałe w `balance.toml`.

## 2. Struktura repozytorium

```
token-tycoon/
├── cmd/token-tycoon/main.go     # flagi, wczytanie save'a, start UI lub symulatora
├── internal/
│   ├── balance/                 # wczytanie i walidacja balance.toml (go:embed + nadpisanie lokalne)
│   ├── sim/                     # stan gry, Step(dt), komendy, offline
│   │   ├── state.go
│   │   ├── step.go
│   │   ├── commands.go          # Buy, Manager, Prestige, Train, ...
│   │   ├── offline.go
│   │   └── events.go            # zdarzenia wychodzące do UI (krytyk, milestone, karta, ...)
│   ├── econ/                    # czyste wzory: koszty, max_k, mnożniki, prestiż
│   ├── market/                  # cena, udział, rywale, segmenty, regiony
│   ├── infra/                   # compute, energia, dławienie
│   ├── training/                # scaling laws, kolejka treningów, dane
│   ├── research/                # drzewko badań
│   ├── rules/                   # silnik polityk
│   ├── rng/                     # strumienie PCG (math/rand/v2)
│   ├── save/                    # JSON, zapis atomowy, kopie, migracje
│   ├── format/                  # formatowanie liczb i czasu
│   ├── simulate/                # boty i eksport CSV
│   └── ui/
│       ├── app.go               # główny tea.Model, routing zakładek
│       ├── theme/               # paleta, wykrywanie profilu kolorów
│       ├── anim/                # tween, sprężyny, cząsteczki, kolejka toastów
│       ├── widgets/             # pasek cyklu, sparkline, wykres Braille, karta
│       ├── tabs/                # businesses, models, market, infra, research, rules, stats
│       ├── palette/             # paleta komend i wyszukiwanie rozmyte
│       └── layout/              # responsywny układ wg rozmiaru terminala
├── balance.toml
├── docs/
└── testdata/                    # golden files, save'y do testów migracji
```

## 3. Pętla

Jeden tick Bubble Tea z adaptacyjnym interwałem. Symulacja na stałym kroku z akumulatorem:

```go
const simStep = 100 * time.Millisecond

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case frameMsg:
        now := time.Time(msg)
        m.acc += now.Sub(m.last)
        m.last = now
        for m.acc >= simStep {
            m.state, m.evs = sim.Step(m.state, m.bal, simStep, m.pending)
            m.pending = m.pending[:0]
            m.anim.Consume(m.evs)       // krytyki, milestone'y, karty -> animacje
            m.acc -= simStep
        }
        m.anim.Advance(now)
        return m, m.nextFrame()          // interwał wg trybu
    case tea.KeyMsg:
        return m.handleKey(msg)          // dodaje komendę do m.pending, reakcja w tej klatce
    case tea.FocusMsg:
        m.focused = true
    case tea.BlurMsg:
        m.focused = false
    case tea.WindowSizeMsg:
        m.layout = layout.For(msg.Width, msg.Height)
    }
    return m, nil
}
```

Interwał klatki:

| Tryb | FPS | Warunek |
|---|---|---|
| Animacja | 30 | aktywna animacja, toast, ceremonia |
| Normalny | 10 | fokus, brak animacji |
| Tło | 1 | brak fokusu (`tea.WithReportFocus()`) |

Opcje programu: `tea.WithAltScreen()`, `tea.WithMouseCellMotion()`, `tea.WithReportFocus()`.

Input jest obsługiwany natychmiast: komenda zakupu trafia do kolejki i jest stosowana w najbliższym kroku, ale UI od razu pokazuje efekt wizualny (błysk), więc odczuwalna reakcja < 16 ms.

## 4. Stan

```go
type State struct {
    Version     int
    Seed        uint64
    RNG         rng.Streams          // events, crits, cards, rivals, energy
    Clock       float64              // sekundy gry od startu
    Money       float64
    Lifetime    Lifetime             // przychód runu, całkowity, warstwy
    Businesses  [10]Business
    Upgrades    Bitset
    Infra       Infra
    Market      Market
    Training    Training
    Talent      []Researcher
    Research    Bitset
    Cards       map[string]int
    Rules       []rules.Rule
    Prestige    Prestige             // parametry, kapitał, wersja modelu, runda
    AGI         AGI
    Streak      Streak
    Stats       Stats
    Unlocks     Bitset
}
```

Stałe tablice zamiast map tam, gdzie to możliwe (mniej alokacji, szybsza serializacja).

## 5. Offline

1. Przy starcie: `elapsed = now − saved_at`, przycięte do limitu offline.
2. Symulacja w krokach 60 s z wyłączonymi elementami interaktywnymi (combo, zdarzenia z oknem reakcji, krytyki zastąpione wartością oczekiwaną).
3. Przychód × wydajność offline.
4. Polityki działają offline, jeśli gracz ma badanie „Polityki offline”.
5. Wynik: ekran „Gdy Cię nie było” z animowanym licznikiem.

Cel: 24 h offline (1440 kroków) w < 200 ms.

## 6. Animacje

Warstwa `ui/anim` nie zna logiki gry, dostaje tylko zdarzenia z `sim`.

| Typ | Implementacja |
|---|---|
| Tween wartości | wykładnicze doganianie celu, stała 0.2 na klatkę przy 30 FPS (korekta przy innym FPS) |
| Sprężyna | `harmonica.NewSpring(harmonica.FPS(fps), freq, damping)` dla toastów i paneli |
| Cząsteczki | pula stała (64), bez alokacji, pozycja + wiek + kolor |
| Błysk | licznik klatek per wiersz |
| Gradient | prekomputowane tablice kolorów per tier przy starcie |

Przy braku aktywnych animacji `anim.Active()` zwraca false i pętla schodzi do 10 FPS.

## 7. Rendering

1. Lip Gloss style tworzone raz przy starcie (theme), nie w `View()`.
2. Bufory `strings.Builder` z prealokacją, reużywane między klatkami.
3. Formatowanie liczb z cache dla wartości, które się nie zmieniły.
4. Pasek cyklu: indeks znaku z tablicy `[]rune("▏▎▍▌▋▊▉█")`.
5. Wykresy Braille: bitmapa 2×4 na komórkę, offset `0x2800`.
6. Profil kolorów wykrywany przez Lip Gloss (termenv), fallback 256 i 16, flaga `--no-color`.

## 8. Zapis

```go
func Save(path string, s *sim.State) error {
    tmp := path + ".tmp"
    // 1. json.Marshal (bez wcięć w produkcji, z wcięciami w trybie --debug)
    // 2. zapis do tmp, f.Sync()
    // 3. rotacja: .2 -> .3, .1 -> .2, path -> .1
    // 4. os.Rename(tmp, path)
}
```

Migracje: `map[int]func(raw map[string]any) error`, stosowane kolejno od wersji pliku do bieżącej. Każda migracja ma test na pliku z `testdata/`.

Wczytanie: plik główny, przy błędzie `.1`, `.2`, `.3`, przy braku nowa gra.

## 9. Symulator balansu

```
token-tycoon --simulate 30d --strategy greedy --seed 42 --out run.csv
token-tycoon --simulate 30d --strategy passive --sessions 10m/day
```

Strategie:

| Strategia | Zachowanie |
|---|---|
| greedy | co krok kupuje opcję o najlepszym stosunku koszt / przyrost $/s, prestiż wg podglądu |
| passive | sesje o zadanej długości dziennie, reszta offline |
| random | losowe sensowne zakupy (test odporności) |

Wyjście CSV: czas, $/s, $, parametry, kapitał, benchmark, udział, odblokowania. Raport kontroluje kryteria akceptacji z `DESIGN.md` (sekcja 10) i zwraca kod błędu, jeśli są złamane, więc da się go podpiąć pod CI.

Symulacja 30 dni w trybie przyspieszonym: cel < 30 s.

## 10. Testy

1. `econ`, `market`, `training`: testy tabelaryczne wzorów.
2. `format`: golden files dla liczb od 1 do 1e300.
3. `sim`: testy determinizmu (ten sam seed i komendy = identyczny stan).
4. `save`: migracje, uszkodzone pliki, zapis atomowy.
5. `simulate`: regresja krzywej postępu (tolerancja ±15% względem zatwierdzonego CSV).
6. Benchmarki: `Step`, `View`, offline 24 h.

## 11. Kamienie milowe

| Etap | Zakres | Gotowe, gdy |
|---|---|---|
| M0 | `balance`, `econ`, `sim` (biznesy, menedżerowie, milestone'y), symulator | krzywa dnia 1 spełnia kryteria |
| M1 | UI: zakładka Biznesy, paski, licznik, kolory, zapis | grywalna pierwsza godzina |
| M2 | infra, prestiż 1, karty, zdarzenia, krytyki, combo | grywalny dzień 1 |
| M3 | rynek, rywale, trening, dane | grywalne dni 2 do 4 |
| M4 | talent, kontrakty, runda finansowania, badania, polityki | grywalne dni 5 do 13 |
| M5 | regiony, safety, inwestorzy, AGI, finał | pełna gra |
| M6 | szlif: animacje ceremonii, statystyki, pomoc, paleta komend, wydajność | wszystkie cele z sekcji 17 DESIGN |
