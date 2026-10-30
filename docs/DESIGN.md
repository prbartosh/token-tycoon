# Token Tycoon: dokument projektowy

Wersja dokumentu: 0.1 (30.10.2026)
Stack: Go + Bubble Tea + Lip Gloss + Bubbles + Harmonica
Powiązane pliki: `ARCHITECTURE.md`, `balance.toml`, `save.example.json`

Wszystkie liczby w tym dokumencie to hipotezy startowe. Źródłem prawdy jest `balance.toml`, a weryfikacją symulator (`--simulate`).

---

## 1. Wizja

Terminalowa gra idle tycoon o branży AI. Startujesz jako solo founder z wrapperem na cudze API, kończysz budując AGI. Horyzont pełnej rozgrywki: ok. 30 dni realnego czasu przy 20 do 60 minutach aktywnej gry dziennie.

Filary:

1. **Dopamina bez frustracji**: częste, zmienne nagrody, krótkie i naprawialne kary.
2. **Decyzje, nie klikanie**: wąskie gardła, rynek i trening modeli wymuszają wybory.
3. **Stopniowa złożoność**: każdy system odblokowuje się dopiero, gdy poprzedni zaczyna nudzić.
4. **Lekkość**: poniżej 1% CPU i 20 MB RAM, jedna binarka.
5. **Czytelność**: kolor i symbol zawsze niosą to samo znaczenie.

---

## 2. Pętla gry

Biznesy produkują **tokeny** w cyklach (pasek postępu). Tokeny sprzedajesz na rynku za **$**. Za $ kupujesz kolejne biznesy, infrastrukturę, ulepszenia i trenujesz modele. Gdy wzrost hamuje, robisz prestiż.

Trzy horyzonty decyzji:

| Horyzont | Przykład decyzji |
|---|---|
| Sekundy | co kupić teraz, ile sztuk (x1, x10, x100, MAX) |
| Minuty i godziny | który milestone, kiedy rozbudować GPU, jaki model trenować |
| Dni | kiedy prestiż, którą gałąź badań rozwijać, kiedy runda finansowania |

---

## 3. Zasoby

Łańcuch produkcji: **Energia (MW) → Compute (GPU) → Tokeny → $**

| Zasób | Źródło | Zastosowanie |
|---|---|---|
| $ | sprzedaż tokenów, kontrakty, inwestorzy | wszystkie zakupy |
| Tokeny | biznesy | sprzedaż (nie są przechowywane, sprzedają się od razu) |
| Compute | racki GPU, chmura | limit produkcji, trening |
| Energia | sieć, PPA, SMR, orbita | zasilanie racków |
| Dane | scraping, licencje, dane syntetyczne | trening modeli |
| Hype | premiery, viralowe zdarzenia, ranking | popyt, wycena (zanika wykładniczo) |
| Talent | rekrutacja | szybkość badań i treningu |
| Parametry | prestiż warstwy 1 | trwały mnożnik przychodu |
| Kapitał VC | prestiż warstwy 2 | drzewko badań |

### 3.1 Wąskie gardła

```
dławienie = min(1, compute_dostępny / compute_potrzebny,
                   energia_dostępna / energia_potrzebna)
tokeny/s  = Σ(produkcja_biznesu) × dławienie
```

Do momentu odblokowania infrastruktury (ok. 1 h gry) dławienie = 1. W chwili odblokowania gracz dostaje „klaster startowy” z zapasem 20% ponad bieżące zapotrzebowanie, żeby odblokowanie nie było karą.

### 3.2 Zasada skalowania systemów pobocznych

Systemy poboczne (energia, pensje, licencje danych, kontrakty) wyceniane są **w minutach przychodu** lub **w procencie przychodu**, a nie w stałych kwotach. Dzięki temu pozostają istotne przy dowolnej skali liczb (od 1e3 do 1e150).

---

## 4. Biznesy (generatory)

Koszt n-tej sztuki: `koszt = bazowy × r^posiadane`

Zakup wielu sztuk:

```
koszt(k) = bazowy × r^posiadane × (r^k − 1) / (r − 1)
max_k    = floor( log( kasa × (r−1) / (bazowy × r^posiadane) + 1 ) / log(r) )
```

| Tier | Biznes | Koszt | r | Cykl | Tokeny/cykl | Compute/szt. |
|---|---|---|---|---|---|---|
| 1 | Wrapper na API | 10 | 1.07 | 1 s | 1 | 0.01 |
| 2 | Chatbot z RAG | 120 | 1.15 | 3 s | 18 | 0.05 |
| 3 | Fine-tune LoRA | 1.4K | 1.14 | 6 s | 216 | 0.25 |
| 4 | Vibe coding studio | 17K | 1.13 | 12 s | 2.6K | 1.25 |
| 5 | Rój agentów (MCP) | 200K | 1.12 | 24 s | 31K | 6.25 |
| 6 | Model reasoningowy | 2.4M | 1.11 | 48 s | 373K | 31 |
| 7 | Klaster GPU na wynajem | 29M | 1.10 | 96 s | 4.5M | 156 |
| 8 | Data center z reaktorem SMR | 350M | 1.09 | 192 s | 54M | 780 |
| 9 | Orbitalne data center | 4.2B | 1.08 | 384 s | 648M | 3.9K |
| 10 | Frontier lab | 50B | 1.07 | 768 s | 7.8B | 19.5K |

Zasady konstrukcji: koszt ×12 na tier, ROI pojedynczej sztuki ×2 na tier, compute ×5 na tier (wyższe tiery są wydajniejsze w compute na token).

### 4.1 Milestone'y

| Próg sztuk | Efekt |
|---|---|
| 25, 50, 100 | czas cyklu ×0.5 |
| 200, 300, 400, 500… (co 100) | przychód ×2 |
| wszystkie biznesy ≥ 25, 50, 100, 200… | globalny przychód ×2 |

Gdy cykl spada poniżej 0.2 s, pasek przechodzi w tryb „ciągły” (patrz 13.2).

### 4.2 Menedżerowie (agenci autonomiczni)

Koszt = 100 × koszt bazowy biznesu. Automatyzują cykl. Każdy ma nazwę i one-liner przy zakupie. Pierwszy osiągalny w 30 do 60 s gry.

### 4.3 Ulepszenia

Jednorazowe, wyceniane tak, by zwracały się w 2 do 5 minut od momentu, gdy gracza na nie stać. Pula startowa:

| Ulepszenie | Efekt |
|---|---|
| Prompt caching | tier 1 i 2 ×3 |
| Kwantyzacja 4-bit | koszt biznesów −10% |
| Speculative decoding | cykl wszystkich ×0.75 |
| KV cache | compute/szt. −20% |
| Mixture of Experts | tier 5 do 7 ×3, compute −15% |
| Destylacja | tier 1 do 4 ×5 |
| Context window 10M | tier 6 ×4 |
| Model Context Protocol | tier 5 ×5 |
| Vibe coding 2.0 | tier 4 ×5 |

Pełna lista w `balance.toml`.

---

## 5. Infrastruktura

### 5.1 Compute

Racki GPU w generacjach. Każda nowa generacja daje ×10 compute na rack przy ×6 kosztu i ×3 energii. Odblokowanie generacji przez badania. Stare racki można wymienić (zwrot 30% ceny).

**Chmura**: natychmiastowy compute bez zakupu, koszt jako % przychodu (15%). Wyjście awaryjne z wąskiego gardła.

### 5.2 Energia

Indeks ceny sieci (0 do 100) zmienia się w cyklu dobowym z szumem (seedowany RNG). Koszt energii liczony jako % przychodu:

```
koszt_energii/s = przychód/s × udział_sieci × 0.08 × (indeks / 50)
```

| Źródło | Charakter | Koszt bieżący |
|---|---|---|
| Sieć | tania pojemność, w szczycie dostępność 70% | zmienny (indeks) |
| PPA | kontrakt na 7 dni, stały indeks | zamrożony indeks z dnia zakupu |
| SMR | drogi, trwały | 0 |
| Orbita | bardzo drogi, trwały, +10% compute | 0 |

---

## 6. Rynek i konkurencja

### 6.1 Cena

```
cena = bazowa × premia_jakości × czynnik_popytu × deflacja
premia_jakości = clamp(1 + (benchmark − mediana_rywali) / 50, 0.5, 2.0)
czynnik_popytu = clamp(sqrt( (udział / 0.25) × (1 + hype) ), 0.25, 2.5)
deflacja       = 0.95 ^ wiek_modelu_w_dniach
```

Deflacja liczona od premiery aktualnego modelu, więc trening nowego modelu ją resetuje. To główny powód, żeby regularnie trenować.

### 6.2 Udział w rynku

```
udział_i = benchmark_i^4 / Σ benchmark_j^4
```

Gracz plus 4 fikcyjne laby:

| Lab | Osobowość | Tempo premier |
|---|---|---|
| Northwind | korporacyjny gigant, równe skoki | co 3 dni |
| OpenWeights | open-source, często, małe skoki, obniża ceny rynku | co 2 dni |
| Dragonfly | tani pretendent, nagłe duże skoki | nieregularnie |
| Haven | startup od safety, wolno, ale stabilnie | co 4 dni |

Harmonogram benchmarków rywali od ok. 10 pkt (dzień 1) do ok. 95 pkt (dzień 28).

### 6.3 Segmenty

Suwak alokacji tokenów: **Free** (generuje hype, 0 $), **Pro** (cena ×1), **Enterprise** (cena ×3, wymaga benchmarku ≥ 70 i compliance w danym regionie).

---

## 7. Trening modeli

Gracz wybiera liczbę parametrów N i tokenów treningowych D. Jakość wg scaling laws (Chinchilla):

```
nadmiar_loss = (406 / N^0.34 + 411 / D^0.28) × Π(bonusy_architektury)
benchmark    = 100 × exp(−0.55 × nadmiar_loss)
FLOPs        = 6 × N × D
czas         = FLOPs / (flops_klastra × udział_klastra_na_trening × mnożnik_talentu)
flops_klastra = flops_na_rack[gen] × racki^0.5
```

Przykładowe wyniki: N=1e7, D=2e8 → ok. 13 pkt. N=1e9, D=2e10 → ok. 60 pkt. N=1e12, D=2e13 → ok. 94 pkt.

Zasady:

1. Maksymalne N odblokowują badania (gałąź Skala).
2. D wymaga zapasu danych.
3. Compute przydzielony do treningu jest odejmowany od produkcji (decyzja: przychód teraz czy lepszy model później).
4. Bonusy architektury (MoE, RLHF, reasoning, lepszy tokenizer) mnożą `nadmiar_loss` przez < 1.
5. Optimum ok. 20 tokenów na parametr gracz odkrywa sam albo z karty „Chinchilla Paper”.
6. Docelowy czas treningu: minuty na początku, 8 do 14 h pod koniec (trening „na noc”).

**Premiera** to ceremonia (patrz 12). Można utrzymywać do 3 modeli naraz w różnych segmentach.

### 7.1 Dane

| Źródło | Koszt | Uwagi |
|---|---|---|
| Scraping | budynki ze skalowaniem kosztu | ryzyko zdarzenia regulacyjnego |
| Licencje | X minut przychodu | pewne, bez ryzyka |
| Dane syntetyczne | compute | wymaga modelu ≥ 60 pkt, przy nadmiarze > 50% kara do jakości |

---

## 8. Systemy średnio- i późnogrowe

### 8.1 Talent

| Klasa | Pensja | Efekt |
|---|---|---|
| Junior | 0.5% przychodu | +5% szybkość badań i treningu |
| Senior | 1.5% przychodu | +15% |
| Gwiazda | 4% przychodu | +40% i losowa cecha |

Cechy: szybki, perfekcjonista (wolniej, +jakość), ewangelista (+hype), oszczędny (−compute). Zdarzenie „podkupienie”: 60 s na kontrofertę (2× pensja przez 3 dni).

### 8.2 Kontrakty enterprise

Szablon: „dostarcz X tokenów w T minut, model ≥ Y pkt”. X liczone jako 1.5 do 4 × bieżąca produkcja × T. Nagroda: 20 do 60 minut przychodu + reputacja. Porażka: tylko utrata reputacji. Do 3 aktywnych naraz.

### 8.3 Regiony i regulacje

| Region | Rynek | Wymóg |
|---|---|---|
| US | 40% | brak |
| EU | 30% | compliance AI Act (jednorazowo 2 h przychodu) |
| Azja | 30% | lokalne data center (budynek) |

Bez wejścia do regionu jego część rynku jest niedostępna (mnożnik popytu).

### 8.4 Bezpieczeństwo

Incydenty (jailbreak w mediach, halucynacja u klienta, wyciek danych) obniżają hype. Szansa bazowa 1 na 4 h, redukowana badaniami safety do 1 na 48 h. Każdy incydent ma okno reakcji (klawisz), które zamienia karę w bonus reputacji.

### 8.5 Inwestorzy

```
wycena = przychód_roczny × mnożnik(hype, wzrost_7d)
```

Co 24 h realnego czasu „kwartał” z celem przychodu (wzrost 50 do 200% względem poprzedniego). Pobicie celu: bonus do następnej rundy finansowania.

---

## 9. Prestiż

### 9.1 Warstwa 1: Nowa wersja modelu

Odblokowanie: ok. 1 h gry. Reset: $, biznesy, ulepszenia, infrastruktura.

```
parametry = floor(100 × sqrt(przychód_runu / 1e12))
bonus     = +2% przychodu za każdy parametr
```

Numer wersji rośnie (v1.0, v2.0, v2.5 Turbo, v3.0 Pro, v4.0 Ultra…). Podgląd przed resetem: zysk i szacowany czas powrotu do obecnego poziomu.

### 9.2 Warstwa 2: Runda finansowania

Odblokowanie: przychód całkowity ≥ 1e40 (dzień 5 do 7). Reset: wszystko z warstwy 1 plus parametry.

```
kapitał = floor(5 × (log10(przychód_całkowity) − 39)^1.5)
```

Nazwy rund: Pre-seed, Seed, Seria A do F, IPO, potem „Mega-runda N”. Kapitał wydawany w drzewku badań.

### 9.3 Drzewko badań

| Gałąź | Przykłady węzłów |
|---|---|
| Efektywność | compute/szt. −10% (5 poziomów), kwantyzacja 2-bit, sparsity |
| Skala | max N ×10 (6 poziomów), nowe generacje GPU |
| Produkt | segment Enterprise, kontrakty, limit modeli |
| Automatyzacja | auto-prestiż warstwy 1, sloty polityk, offline 100% i 24 h |
| Bezpieczeństwo | redukcja incydentów, Enterprise w EU taniej |
| Szczęście | szansa na krytyk, rzadsze karty częściej |

### 9.4 Warstwa 3: Wyścig do AGI

Odblokowanie: dzień 18 do 20. Megaprojekt w 5 etapach, każdy zmienia zasady:

| Etap | Koszt | Efekt po ukończeniu |
|---|---|---|
| Alignment | 1e95 | incydenty wyłączone |
| Compute | 1e105 | wszystkie biznesy ×2 na stałe w tym runie |
| Dane | 1e115 | dane syntetyczne bez kary |
| Trening | 1e125 | jeden trening 24 h, benchmark do 100 |
| Ewaluacja | 1e140 | finał, napisy, statystyki |

Po finale: tryb nieskończony „Superinteligencja” z rekordami czasów.

---

## 10. Tempo na 30 dni

Założenia: 20 do 60 min aktywnie dziennie, offline 50% (do 100%), limit 8 h (do 24 h).

| Dzień | Przychód $/s | Odblokowania |
|---|---|---|
| 1 | 1e3 do 1e9 | biznesy 1 do 6, menedżerowie, infrastruktura, prestiż 1, karty |
| 2 | 1e9 do 1e13 | rynek, rywale, segmenty |
| 3 | 1e13 do 1e18 | trening, dane, biznesy 7 do 10 |
| 4 | 1e18 do 1e25 | kontrakty, talent |
| 5 do 7 | 1e25 do 1e40 | runda finansowania, badania, inwestorzy |
| 8 do 9 | 1e40 do 1e50 | polityki (automatyzacja) |
| 10 do 13 | 1e50 do 1e65 | regiony, wyzwania |
| 14 do 17 | 1e65 do 1e90 | orbita, pełna gałąź safety |
| 18 do 30 | 1e90 do 1e150 | AGI, finał |

Kryteria akceptacji symulatora:

1. W pierwszej godzinie żaden płaski odcinek (brak sensownego zakupu) nie trwa dłużej niż 2 min.
2. W każdej sesji coś nowego odblokowuje się w pierwszych 5 min.
3. Żaden dzień bez nowej mechaniki lub nowego celu do dnia 20.
4. Bot „greedy” kończy grę w 25 do 32 dni, bot „pasywny” (tylko offline + 10 min dziennie) w 40 do 50.

Wszystkie wartości mieszczą się w float64 (max ok. 1e308).

---

## 11. Zdarzenia i karty

### 11.1 Zdarzenia losowe (co 2 do 5 min)

| Zdarzenie | Efekt | Czas |
|---|---|---|
| Viralowe demo | przychód ×7 | 30 s |
| Leak benchmarku | szansa na kartę ×3 | 60 s |
| Niedobór GPU | racki droższe o 20% | 60 s |
| Halucynacja! | wciśnij H w 5 s: bonus zamiast straty | 5 s |
| Inwestor anioł | wciśnij G w 8 s: 15 min przychodu | 8 s |
| Godzina szczytu | tier 1 do 3 ×5 | 45 s |
| Konferencja | hype +50% | 120 s |

### 11.2 Karty badań (papers)

Wypadają z cykli (0.05% na cykl) i zdarzeń. Trwałe bonusy, przetrwają prestiż warstwy 1. Duplikaty podnoszą poziom karty.

| Rzadkość | Kolor | Szansa | Przykład |
|---|---|---|---|
| Common | szary | 70% | „Dropout”, „Adam Optimizer” |
| Rare | niebieski | 22% | „LoRA”, „RAG” |
| Epic | fioletowy | 7% | „Chinchilla Paper”, „Mixture of Experts” |
| Legendary | złoty | 1% | „Attention Is All You Need” |

Kolekcja: ok. 40 kart, licznik ukończenia, bonus za komplet w rzadkości.

---

## 12. Mechaniki dopaminowe

1. **Krytyki**: 3% szansy na ×10 na cykl („TOKEN BURST”), złoty napis, rozbłysk wiersza.
2. **Combo**: szybkie ręczne odpalanie cykli nabija mnożnik do ×5, wygasa po 2 s bezczynności.
3. **Karty z animacją odsłaniania**: migotanie kolorów, potem rzadkość.
4. **Licznik do następnego celu zawsze widoczny**.
5. **Rozbicie mnożników** obok przychodu.
6. **Ceremonie**: prestiż (animacja treningu ze spadającym loss), premiera modelu (benchmarki, skok w rankingu), runda finansowania (wycena rośnie na liczniku).
7. **Dzienny grant**: streak do 7 dni, przy przerwie spada o 1, nie do zera.
8. **Osiągnięcia**: ok. 60, toast wjeżdżający z prawej.
9. **Pasek newsów**: satyryczne nagłówki reagujące na stan gry.
10. **Cofnięcie zakupu** w ciągu 5 s.

Zasady: kary zawsze krótkie (≤ 60 s) i naprawialne; żadnej mechaniki, która karze za nieobecność.

---

## 13. Interfejs

### 13.1 Układ

```
 TOKEN TYCOON v4.5 Turbo    $ 1.28B  +12.4M/s ▁▂▃▅▆▇█    PWR 82%  GPU 94%  HYPE 1.8x
 [1 Biznesy]  2 Modele  3 Rynek  4 Infra  5 Badania •NEW  6 Polityki  7 Statystyki
┌───────────────────────────────────────────────┬──────────────────────────────────┐
│ > Wrapper na API   x142 ███████████████▌ 38M/s │ CHATBOT Z RAG              x87   │
│   Chatbot z RAG    x87  ███████▋         3.1M  │ Przychód   3.1M / cykl (3.0 s)   │
│   Fine-tune LoRA   x31  ███▎            18.9M  │ Compute    0.4 GPU / szt.        │
│   Vibe coding      x12  ▊                74M   │ Milestone  87/100  cykl x0.5     │
│   Rój agentów      --   [zablok.] 200M 2m14s   │ ████████████████░░░              │
│                                               │ Kup x10    412M   * najlepsze ROI │
├───────────────────────────────────────────────┴──────────────────────────────────┤
│ NEWS  Northwind wypuszcza model 92 pkt. Twój udział: 31% (-4)                       │
└ up/dn wybór  Enter kup  Tab ilość: x10  m menedżer  u cofnij  : komendy  ? pomoc ──┘
```

Zakładki: Biznesy, Modele (trening, premiery, ranking), Rynek (ceny, udział, segmenty, regiony, kontrakty), Infra (GPU, energia, dane), Badania, Polityki, Statystyki. Zakładki pojawiają się z plakietką NEW w momencie odblokowania.

### 13.2 Animacje

| Element | Technika | Czas |
|---|---|---|
| Pasek cyklu | znaki `▏▎▍▌▋▊▉█`, 8 kroków na komórkę | ciągły |
| Pasek przy cyklu < 0.2 s | pełny, przesuwający się gradient + wartość/s | ciągły |
| Licznik $ | tween: `wyświetlana += (cel − wyświetlana) × 0.2` na klatkę | ok. 300 ms |
| Pływające liczby | `+1.2K` w górę o 2 do 3 wiersze, blednięcie w 6 do 8 klatek | 300 ms |
| Zakup | błysk wiersza, pulsujący licznik sztuk | 120 ms |
| Milestone | puls całego wiersza + komunikat | 300 ms |
| Krytyk | złoty rozbłysk + napis | 300 ms |
| Toast | wjazd sprężynowy (Harmonica) z prawej | 250 ms |
| Sparkline | `▁▂▃▄▅▆▇█`, ostatnie 60 s | ciągły |
| Wykresy | Braille (2×4 punkty na komórkę) | statyczne |

Zasada: żadna animacja nie blokuje inputu.

### 13.3 Sterowanie

| Klawisz | Akcja |
|---|---|
| ↑ ↓ / k j | wybór |
| ← → / h l, 1 do 7 | zakładki |
| Enter | kup / akcja główna |
| Spacja | ręczny cykl (combo) |
| Tab | ilość: x1, x10, x100, MAX |
| m | menedżer |
| u | cofnij ostatni zakup (5 s) |
| : lub Ctrl+P | paleta komend z wyszukiwaniem rozmytym |
| ? | pomoc kontekstowa |
| p | podgląd prestiżu |
| q | zapis i wyjście |
| mysz | klik zaznacza, podwójny klik kupuje, scroll przewija |

### 13.4 Zasady UX

1. Jasny tekst = mogę teraz, szary = jeszcze nie (z czasem do odblokowania).
2. Pasek klawiszy pokazuje tylko akcje możliwe dla zaznaczonego elementu.
3. Potwierdzenia tylko dla akcji nieodwracalnych (prestiż, wycofanie modelu).
4. Responsywny układ: ≥ 120 kolumn trzy panele (dochodzi log), 80 do 119 dwa, < 80 jeden.
5. Bez emoji (niespójna szerokość). Tylko znaki blokowe, ramki, Braille.
6. Tryb dla daltonistów: obok koloru zawsze symbol (`+ - * !`).
7. Doradca: gwiazdka przy najlepszym zakupie, wyłączalny.
8. Liczby: K, M, B, T, potem aa, ab…; przełącznik na notację naukową.

### 13.5 Paleta komend

Przykłady: `kup rag 50`, `kup max`, `trenuj`, `idź badania`, `prestiż podgląd`, `polityki`, `zapisz`, `ustawienia`.

---

## 14. Polityki (automatyzacja)

Od dnia 8. Kreator z listami wyboru, bez pisania kodu. Limit slotów rośnie z badaniami (start 3, max 12).

```
GDY  $ > 10 × koszt(najlepsze_ROI)     TO kup najlepsze_ROI x10
GDY  wykorzystanie_GPU > 90%           TO kup rack_GPU MAX
GDY  cena_energii > 80                 TO kup PPA
GDY  udział < 25% i trening wolny      TO trenuj model(optymalny)
```

Warunki: $, przychód, wykorzystanie GPU i energii, indeks energii, udział, hype, stan treningu, czas do zdarzenia.
Akcje: kup (biznes, rack, energia, dane), trenuj, przełącz segment, prestiż warstwy 1.
Reguły wykonywane co 1 s, kolejność wg priorytetu, log wykonania w zakładce Polityki.

---

## 15. Kolory

| Rola | Truecolor | Fallback 256 |
|---|---|---|
| Tło | #0B0E14 | 233 |
| Tekst główny | #E6E6E6 | 254 |
| Tekst nieaktywny | #5C6370 | 241 |
| $ i przychód | #00E5FF → #39FF88 | 51 / 84 |
| Koszt niedostępny | #FF4D6D | 204 |
| Złoty (krytyk, Legendary, prestiż) | #FFC300 | 220 |
| Magenta (combo, zdarzenia) | #FF3CAC | 199 |
| Parametry | #A66CFF | 141 |
| Kapitał VC | #2EC4B6 | 37 |
| Rare / Epic | #3A86FF / #B15CFF | 33 / 135 |

Paski: gradient per tier od chłodnych (tier 1) do ciepłych (tier 10). Fallback do 16 kolorów i flaga `--no-color`.

---

## 16. Plik stanu

Ścieżka: `~/.config/token-tycoon/save.json` (Linux, macOS), `%APPDATA%\token-tycoon\save.json` (Windows).

1. Format JSON, przykład w `save.example.json`.
2. Autozapis co 60 s, przy wyjściu i po każdym prestiżu.
3. Zapis atomowy: plik tymczasowy, fsync, rename.
4. 3 rotowane kopie zapasowe (`save.json.1` do `.3`).
5. Pole `version` i łańcuch migracji przy wczytaniu.
6. Zdarzenia, rywale i ceny energii wyliczane z `seed`, nie są zapisywane.
7. Uszkodzony plik: próba z kolejnej kopii, komunikat dla gracza.

---

## 17. Budżet zasobów

| Metryka | Cel |
|---|---|
| CPU (normalny tryb) | < 1% |
| CPU (terminal bez fokusu) | < 0.1% |
| RAM | < 20 MB |
| Binarka | < 15 MB |
| Start z offline 24 h | < 200 ms |
| Reakcja na klawisz | < 16 ms |

Techniki: adaptacyjny FPS (30 animacje, 10 normalnie, 1 bez fokusu), symulacja 10 ticków/s, offline w krokach 1 min, zero alokacji w pętli klatki, rendering przez diff (Bubble Tea).

---

## 18. Poza zakresem v1

Multiplayer, rankingi online, dźwięk (poza opcjonalnym `bell`), mody poza nadpisaniem `balance.toml`, lokalizacja inna niż PL i EN.
