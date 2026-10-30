# Token Tycoon: decyzje uzgodnione z właścicielem

Data: 30.10.2026. Te decyzje mają pierwszeństwo przed DESIGN.md i ARCHITECTURE.md, jeśli są sprzeczne.

## Środowisko

1. Go 1.27.3 (Homebrew). Moduł lokalny: `token-tycoon` (bez ścieżki GitHub).
2. Git lokalny, commit po każdym etapie.
3. Zależności (przypięte w go.mod):
   - `charm.land/bubbletea/v2` v2.1.1
   - `charm.land/lipgloss/v2` v2.0.6
   - `charm.land/bubbles/v2` v2.2.1
   - `github.com/charmbracelet/harmonica` v0.2.0
   - `github.com/pelletier/go-toml/v2` v2.4.3
4. Bubble Tea v2 ma inne API niż przykłady w ARCHITECTURE.md (v1). Przed pisaniem kodu UI czytaj źródła w `$(go env GOMODCACHE)/charm.land/bubbletea/v2@v2.1.1`. Znane różnice: `View()` zwraca `tea.View` (pola `AltScreen`, `ReportFocus`, `MouseMode` zamiast opcji programu), klawisze to `tea.KeyPressMsg`, mysz to `tea.MouseClickMsg` / `tea.MouseWheelMsg`.
5. `balance.toml` leży w `internal/balance/` (wymóg `go:embed`). Nadpisanie lokalne: `~/.config/token-tycoon/balance.toml`.

## Język

6. Pełna wersja PL i EN. Wszystkie teksty UI, nazwy biznesów, ulepszeń, kart, zdarzeń, newsy i pomoc w katalogu tłumaczeń (`internal/i18n`). Język w ustawieniach, domyślnie wg `LANG` (pl* = PL, inaczej EN), flaga `--lang pl|en`.
7. Identyfikatory i komentarze w kodzie po angielsku.

## Ekonomia i postęp

8. **Odblokowania zależą od przychodu, nie od dni.** Wszystkie `unlock_day` zastąpione progami przychodu łącznego (`lifetime.total`, $), wyprowadzonymi z tabeli tempa w DESIGN sekcja 10. Mechaniki czasowe (deflacja modelu, PPA 7 dni, kwartał inwestorów, hype half-life) liczone zegarem gry (`clock_s`).
9. Kapitał startowy: **0 $** i jeden darmowy Wrapper na API (bez tego gra nie ruszy).
10. Menedżer tier 1 kosztuje **60 $** (wyjątek od `cost × manager_mult`, klucz `manager_cost` w `[[business]]`).
11. Cena tokena przed odblokowaniem rynku: **1 $/token** (`price_base`).
12. Globalne milestone'y: 25, 50, 100, potem **co 100 bez końca**, każdy x2. Liczy się minimum po **odblokowanych** biznesach; zdobyte x2 nie przepadają po odblokowaniu nowych tierów.
13. Progi infra (`lifetime 1e7`) i prestiżu 1 (`run 1e12`) zostają jako liczby. Kolejność: najpierw infra, potem prestiż. Realne czasy mierzy symulator; cel strojenia: infra ok. 20 do 30 min, prestiż 1 ok. 1 h.
13a. Progi bez pokrycia w DESIGN, zatwierdzone: zdarzenia losowe i karty od lifetime 1e7 (razem z infra), incydenty safety od 1e13 (razem z rynkiem), drzewko badań od 1e40 (razem z pierwszą rundą finansowania).
13b. Limit 3 modeli: gdy trening kończy się przy komplecie, online gracz dostaje modal „który model wycofać” (lista 4 modeli, w tym nowy, więc można odrzucić nowy). Offline i przy politykach: nowy wchodzi tylko, jeśli jest lepszy od najsłabszego; inaczej trafia do archiwum. Z archiwum gracz ręcznie wymienia model. Żaden model nie przepada bez decyzji gracza.
14. Krytyki i karty w trybie ciągłym (cykl < 0.2 s): losowanie dla efektywnej liczby cykli **max 5/s na biznes** (nadwyżka liczona jako wartość oczekiwana bez losowania kart). Krytyki zbalansowane tak, by średni wkład był stały niezależnie od długości cyklu.

14a. Dostawa kontraktów liczy stałą produkcję: bez combo, krytyków i boostów zdarzeń.
14b. Inwestycja w etap AGI: Enter otwiera okno z porcjami (10%, 50%, 100% $, brakująca kwota) i potwierdzeniem.
14c. Zatwierdzone interpretacje: regiony jako ekspansja (odblokowanie neutralne, US = x1, wszystkie regiony = x2.5); inwestycja i efekty AGI przetrwają prestiż 1 i rundę finansowania; runda finansowania zeruje wersję modelu i aktywne kontrakty; compliance Enterprise w US automatyczny (koszt 0).

## Styl kodu

17. **Bez komentarzy w kodzie.** Dozwolone wyłącznie: krótki doc string na górze pliku (opis pakietu lub pliku) oraz krótki komentarz przy bardzo skomplikowanej logice biznesowej (np. wyprowadzenie wzoru). Komentarze nie mogą tłumaczyć, co robi kod. Każdy agent przy okazji usuwa istniejące komentarze łamiące tę zasadę w plikach, które dotyka. Komentarze w `balance.toml` opisujące znaczenie kluczy są dozwolone.

## Sterowanie

15. Zakładki: **← →** i cyfry **1 do 7**. Wybór: **↑ ↓** i **k / j**. Klawisze h i l nie zmieniają zakładek (wolne dla zdarzeń: Halucynacja = H, Inwestor anioł = G).
16. **Spacja = globalny boost**: każde wciśnięcie nabija combo (+0.1, max x5, gaśnie po 2 s ciszy), combo mnoży cały przychód. Gdy zaznaczony biznes nie ma menedżera i stoi, Spacja dodatkowo startuje jego cykl. Enter na biznesie bez menedżera, którego nie stać na zakup, też startuje cykl.
