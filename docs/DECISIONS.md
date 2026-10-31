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

## Strojenie (etap 4)

Diagnoza symulatorem (seed 42, przed zmianami):

- Wykładnik `a` w zależności przychód ~ pieniądze^a w obrębie runu: ok. 0.8 do 1.0 poniżej ok. 1e18 $ (milestone'y prędkości, ulepszenia), ok. 0.16 później (x2 co 100 sztuk wobec kosztu r^n). Prestiż `sqrt(run)` z bonusem liniowym jest stabilny tylko przy `a` ok. 0.5, więc na początku rozpędzał się (gra ciągła: dzień 1 kończył się na ok. 5e25 $/s po 25 prestiżach), a później dławił (ok. 1e36 łącznie po 30 dniach, runda 1e40 nieosiągalna).
- Runda finansowania zerowała parametry i nie dawała nic do przychodu, więc po rundzie gra cofała się o kilkadziesiąt rzędów i stawała.
- Bot: runda wykonywana przed prestiżem po nocy (przepadał dzienny skok), kontrakty przyjęte pod koniec sesji blokowały prestiż następnego dnia.

Zmiany wzorów i mechanik:

18. **Milestone'y**: przychód x2 od 100 sztuk co 25 (było od 200 co 100), globalne x2 po 100 co 50 (było co 100). Późna gra ma `a` ok. 0.5, wcześniej ok. 0.16.
19. **Limit skoku wersji (prestiż 1)**: `zysk = surowy × limit / (surowy + limit)`, `limit = (parametry + param_coef) × długość_runu / leap_s`, `surowy = param_coef × sqrt(run / 1e12)` jak dotąd. Jedna nowa wersja mnoży bonus najwyżej ok. `1 + run_s / leap_s` (`leap_s = 1800`). Długi run (noc offline) daje duży skok, seria krótkich runów mały. Tempo gry = stała liczba rzędów wielkości na dzień, niezależnie od wykładnika `a`.
20. **Mnożnik rund**: runda nadal zeruje parametry, ale ich bonus przechodzi do trwałego mnożnika przychodu: `mnożnik_rund *= bonus_parametrów ^ round_bank_share` (`1.0` = runda nic nie zabiera). Nowe pole stanu `prestige.round_mult`, zapis w wersji 8, migracja 7 -> 8 ustawia 1.
21. **Bezpiecznik liczb**: przychód i mnożniki obcięte do 1e280/s, pieniądze i przychód łączny do 1e300. Po finale wzrost trwa bez końca (gra ciągła dochodzi do limitu ok. 11. dnia), wcześniej groziło Inf i NaN.
22. **Generacja GPU 6** od łącznie 1e34 (dzień 6 nie miał nowego celu).
23. **Bot greedy = gracz z DESIGN 10**: jedna sesja 60 min dziennie, reszta offline (`[pacing]` w `balance.toml`). `greedy_session_min = 1440` to gra bez przerwy, test obciążeniowy, nie kryterium. Bot robi prestiż przed rundą, rundę tylko bez aktywnych kontraktów i nie przyjmuje kontraktu dłuższego niż reszta sesji.
24. **Raport symulatora** sprawdza wszystkie kryteria z `[pacing]` i z `--ci` zwraca kod 1. Dla bota greedy: płaski odcinek, menedżer, infra i prestiż w czasie gry, kolejność, sesje, nowy cel każdego dnia do dnia 20, pasmo przychodu z tabeli DESIGN 10 (±1 rząd), dzień progu rundy (5 do 7) i AGI (18 do 20), dzień finału (25 do 32). Dla pasywnego: płaski odcinek, menedżer, kolejność, sesje, dzień finału (40 do 50). Cel dnia = odblokowanie, generacja GPU, runda, ukończone badanie, etap AGI, finał, poziom SI. Pierwsza sesja (start gry) nie liczy się do kryterium „coś nowego w 5 min”.

Wyniki (seed 42; dzień:log10 najlepszego przychodu $/s):

| Przebieg | Przed | Po |
|---|---|---|
| greedy 30d, sesje 60 min | 1:8 2:11 3:12 5:17 10:23 20:25 30:26, finał nie | 1:9 2:13 3:20 5:29 7:37 10:53 13:69 18:91 20:100 30:144, finał dzień 29, próg rundy dzień 7, AGI dzień 18 |
| greedy 30d bez przerwy | 1:26 10:30 30:29, łącznie 1e36 | 1:44 5:152, finał dzień 5 (nie jest celem) |
| pasywny 50d, 10 min dziennie | 1:4 10:12 30:22 50:23, finał nie | 1:6 10:31 30:102 50:176, finał dzień 41 |
| greedy 30d, czas symulacji | 4 s | 9 s |

Otwarte dla strojenia liczb: infra po 6 min 43 s (cel 20 do 30), płaski odcinek 14 min od 46. minuty (brak treści między 1e10 a 1e12 przy 6 tierach), sesje dni 3, 5, 6, 7 bez nowości w 5 min (progi przekraczane w 7. do 17. minucie sesji), pasma dni 3 (1e20) i 13 (1e69). Gra ciągła bez przerw jest kilka razy szybsza od tabeli (prestiż co ok. 30 min).

## Strojenie liczb (etap 4b, tylko `balance.toml`)

25. Strojenie symulatorem (seedy 42, 7, 1234) zmieniło wyłącznie liczby:
    - biznesy 1 do 6 (koszt, wzrost `r`, tokeny), koszt menedżera API 50, `manager_mult` 94, milestone zysku x1.6 co 25 sztuk od 100, koszty i wartości ulepszeń: dzień 1 kończy się na ok. 1e9 $/s, menedżer po 30 do 60 s, prestiż 1 w ok. 1 h.
    - infrastruktura odblokowuje się od 2.1e9 $ łącznie (było 1e7 z punktu 13): przy 1e7 gracz miał ją po 7 minutach zamiast po 20 do 30.
    - `leap_s = 940`, runda finansowania od 6.6e33 (dzień 7), `capital_log_offset = 30`, koszty etapów AGI 1e95, 1e104, 1e110, 1e118, 1e128.
    - progi odblokowań dobrane do okien sesji (komentarz w nagłówku `balance.toml`): w pierwszych 5 minutach każdej sesji greedy pojawia się nowy cel.
    - Znane niespełnione kryteria: płaski odcinek w pierwszej godzinie (bot czeka na zakup w kawałkach do milestone'u, a między 1e10 a 1e12 przy 6 biznesach i 9 ulepszeniach nie ma wystarczająco drobnej treści; wynik jest chaotyczny, 2 do 9 min zależnie od seeda) oraz nowy cel w każdej sesji bota pasywnego (jego dni nie pokrywają się z oknami greedy, brakuje dodatkowych odblokowań).


## Przegląd etapu 4

26. Poprawki po przeglądzie (bez zmian strojenia):
    - model czekający na decyzję gracza (13b) wchodzi na rynek z wiekiem 0 w chwili decyzji, a nie w chwili końca treningu (deflacja nie nalicza się, zanim model trafi na rynek);
    - biznesy 7 do 10 odblokowują się w kolejności tierów (te same progi 7.9e14, 9.1e14, 1.1e15, 1.2e15, tylko przestawione); walidacja balansu pilnuje kolejności;
    - zakładka Rynek pokazuje stałą produkcję (14a) w podsumowaniu i w krotności oferty kontraktu;
    - podgląd rundy pokazuje mnożnik rund przed i po, podgląd prestiżu pokazuje zysk bez limitu skoku (gdy limit działa) i komunikat o zbyt krótkim runie; Statystyki pokazują mnożnik rund, a rekord przychodu bierze też najlepszy przychód z zapisu;
    - raport symulatora podaje progi odblokowań z balansu zamiast liczb z DESIGN.

## Treść 1e8-1e12

27. Treść 1e8-1e12 (płaski odcinek pierwszej godziny):
    - 18 nowych ulepszeń z branży AI (Chain of thought, Function calling, Few-shot prompting, Agentowe IDE, Code interpreter, Drzewo myśli, Zespoły agentów, Pamięć agentów, FlashAttention, Wyszukiwanie w sieci, RLHF, Korzystanie z narzędzi, DPO, Multimodalność, Dane syntetyczne, Test-time compute, Constitutional AI, RL na długim kontekście), koszty od 1.4e7 do 5.5e13, mechanika bez zmian (profit/cycle na tierach 1 do 6, nigdy na 7 do 10, żeby nie przyspieszać późnej gry). Lista ulepszeń w `balance.toml` posortowana po koszcie. Zapis bez zmian (ulepszenia po id).
    - Liczby: tokeny Roju agentów 7640 i Modelu reasoningowego 309000 (było 19100 i 660000), MoE 4.2e7 x2.3, Context 10M 4e8 x2.7, Destylacja 9.6e9 x7.9, infra i GPU gen 1 od 2.7e8. Progi dni przesunięte do nowych okien sesji (seed 42): dzień 3 od 1.2e16 do 1.9e16, talent 1.5e20, kontrakty 2e20, GPU gen 6 4e30, runda 8.3e33, polityki 1e39, GPU gen 7 1.5e39.
    - Bot greedy: przy zakupie biznesu, który przekracza compute, dolicza brakujące racki i sieć (wcześniej taki zakup miał ujemny zysk i bot stał); zbierając na paczkę do milestone'u kupuje po drodze tyle sztuk, ile go stać; w pierwszym runie (przed pierwszym prestiżem, nie przy oszczędzaniu na menedżera) kupuje wypełniacz, jeśli jego zwrot jest najwyżej 1.25 x wynik najlepszego celu. Same ulepszenia bez tych zmian nie wystarczały: bot oszczędzał na każde mocne ulepszenie i przerwy rosły.
    - Wynik greedy 1h: płaski odcinek 1m55s (seed 42), 1m35s (7), 1m35s (1234); menedżer 42 s, 59 s, 53 s; infra 24, 28, 28 min; prestiż 60, 53, 49 min. Greedy 30d seed 42 spełnia wszystkie kryteria.
    - Znane: wynik jest bardzo czuły na liczby (zmiana tokenów o 0.5% potrafi wydłużyć przerwę o 50 s), inne seedy (1, 2, 3, 5, 99) mają 93 do 139 s, seed 99 ma prestiż po 44 min. Bot pasywny 50d nie kończy gry (dzień 54, wcześniej 50 na granicy).
    - Dostrojenie końcówki: koszty etapów AGI 4 i 5 obniżone do 1e117 i 1e126 (było 1e118 i 1e128). Bot pasywny 50d kończy grę na seedach 7 (dzień 42) i 1234 (dzień 49), seed 42 nadal nie (stage 4 w dniu 46, koniec po dniu 50). Bot greedy kończy w dniu 25 na wszystkich trzech seedach. Dalsze obniżenie kosztu etapu 5 (do 1e125) dałoby pasywnemu seedowi 42 dzień 50, ale greedy seeda 7 skończyłby w dniu 24 (poniżej kryterium), więc nie da się spełnić obu bez zmiany mechaniki.
