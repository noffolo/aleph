# Development Bias Checklist

**Progetto:** aleph-v2  
**Scopo:** Prevenire errori sistematici nello sviluppo attraverso consapevolezza e check operativi  
**Quando usare:** Planning, code review, stime, decisioni architetturali

---

## Come Usare Questa Checklist

Prima di:
- **Stimare un task** → Vedi _Planning & Stime_
- **Iniziare implementazione** → Vedi _Decisioni Tecniche_
- **Fare code review** → Vedi _Code Review Template_
- **Valutare completamento** → Vedi _Verifica Completamento_

---

## Bias Catalog

### 1. Confirmation Bias

**Descrizione:** Cercare attivamente prove che confermano la soluzione già scelta, ignorando segnali di alternative migliori o problemi.

**Esempio aleph:**
- "Ho già deciso di usare Redis per la cache" → cerchi solo documentazione sui benefici di Redis, ignori che Memcached sarebbe più semplice per il caso d'uso
- Durante la review: cerchi solo conferme che il codice funziona, ignori edge cases non gestiti

**Mitigazione:**
- [ ] Prima di decidere, scrivi 3 alternative e 1 pro/contro per ciascuna
- [ ] Assegna a qualcuno il ruolo di "devil's advocate" nella discussione
- [ ] Chiediti: "Cosa dovrei vedere se la mia soluzione fosse sbagliata?"

**Remediation Status:** ✅ `ComputeDiversityScore()` in `internal/ethics/bias.go` — measures recommendation diversity via pairwise Jaccard similarity; detects when the system surface area narrows over time

---

### 2. Availability Bias

**Descrizione:** Scegliere la prima soluzione che viene in mente perché più "disponibile" mentalmente, non perché migliore.

**Esempio aleph:**
- "L'ultima volta ho usato PostgreSQL per tutto" → usi Postgres anche per dati temporanei dove SQLite basterebbe
- "Ho appena letto di Kubernetes" → proponi K8s per un deploy che sta bene su un singolo server

**Mitigazione:**
- [ ] Aspetta 24 ore prima di decidere architetture importanti (se possibile)
- [ ] Cerca attivamente soluzioni "noiose" e consolidate prima di quelle nuove
- [ ] Chiediti: "Sto scegliendo questo perché l'ho appena visto/usato?"

**Remediation Status:** ✅ `DecayWeight()` in `internal/ethics/bias.go` + `TimeDecayHalfLife` in `internal/tools/synthesis/synthesis.go` — exponential time decay weights recent data appropriately in recommendation scoring; configurable half-life controls how quickly older patterns are discounted

---

### 3. Data Bias

**Descrizione:** Squilibrio nei dati di training che porta il modello a performance diseguali su diversi sottogruppi (es. nodi tool vs nodi utente, categorie sovrarappresentate vs sottorappresentate).

**Esempio aleph:**
- "Il GNN ha AUC 0.92" → ma performa bene solo sui tool più usati, ignora quelli nuovi
- "Abbiamo 10k edge di training" → 9k sono dello stesso tipo funzionale, 1k per tutti gli altri

**Mitigazione:**
- [ ] Verifica la distribuzione per tipo/categoria prima del training
- [ ] Usa stratified sampling se lo squilibrio supera 70/30
- [ ] Monitora le performance per sottogruppo

**Remediation Status:** ✅ `CheckDataBalance()` in `internal/ethics/bias.go` — normalized entropy check on category distributions; `EdgeTypeBalance()` checks train/eval split balance; thresholds configurable per use case

---

### 4. Algorithmic Bias

**Descrizione:** Il modello apprende e amplifica pattern sistematici che favoriscono certi gruppi a discapito di altri, anche in assenza di feature esplicite sensibili.

**Esempio aleph:**
- "Il sistema raccomanda sempre tool popolari" → i tool nuovi non hanno chance di emergere
- "Le embedding dei tool A e B sono molto simili" → il modello non distingue contesti diversi

**Mitigazione:**
- [ ] Calcola demographic parity sulle embedding per gruppo
- [ ] Verifica che i punteggi medi siano comparabili tra categorie
- [ ] Aggiungi regolarizzazione per favorire embeddings bilanciate

**Remediation Status:** ✅ `ComputeDemographicParity()` in `internal/ethics/bias.go` — compares mean embedding magnitude across demographic groups; detects when the model systematically favors certain categories

---

### 5. Sunk Cost Fallacy

**Descrizione:** Continuare su una strada sbagliata perché già investito tempo/energia, invece di tagliare le perdite.

**Esempio aleph:**
- "Ho già passato 3 ore su questo approccio di caching" → continui anche dopo aver scoperto che non scala
- "Il modulo di auth è quasi finito" → lo tieni nonostante sia insicuro, invece di riscriverlo

**Mitigazione:**
- [ ] Definisci criteri di stop PRIMA di iniziare (es: "se dopo 2h non ho X, cambio approccio")
- [ ] A ogni checkpoint: "Se iniziassi oggi da zero, rifarei questa scelta?"
- [ ] Normalizza il "pivot" come successo, non fallimento

**Remediation Status:** ⏳ Processo — nessuna remediation code; mitigazione affidata ai criteri di stop pre-definiti

---

### 6. Overconfidence Bias

**Descrizione:** Sottostimare sistematicamente la complessità di feature stimate come S/M.

**Esempio aleph:**
- "È solo un endpoint CRUD, 2 ore" → 8 ore dopo: validazione, edge cases, test, fix
- "L'integrazione con l'API esterna è banale" → non consideri rate limiting, retry, fallback

**Mitigazione:**
- [ ] Moltiplica le stime S/M per 2.5x (baseline storica del team)
- [ ] Per ogni task: scrivi 3 cose che potrebbero andare storto
- [ ] Tieni traccia delle stime vs effettivo, reviewa mensilmente

**Remediation Status:** ⏳ Processo — mitigazione affidata al fattore 2.5x e al velocity tracking

---

### 7. Planning Fallacy

**Descrizione:** Ignorare la storia passata di task simili che hanno sforato, trattando ogni task come "questa volta è diverso".

**Esempio aleph:**
- Task "migrazione dati" stimato 4 ore → l'ultima volta ha richiesto 16 ore
- "Questa feature è simile a quella di marzo" → ignori che quella slittò di 2 settimane

**Mitigazione:**
- [ ] Prima di stimare, cerca task simili nel backlog/history
- [ ] Usa la media degli effettivi passati, non le stime originali
- [ ] Mantieni un "velocity log" delle stime vs realtà

**Remediation Status:** ⏳ Processo — mitigazione affidata alla ricerca di task simili prima della stima

---

### 8. Survivorship Bias

**Descrizione:** Guardare solo i task completati con successo, ignorando quelli falliti o abbandonati.

**Esempio aleph:**
- "Tutte le feature rilasciate funzionano bene" → ignori le 3 feature abbandonate a metà
- "Il nostro processo di stima è accurato" → guardi solo i task completati, non quelli ancora in corso da settimane

**Mitigazione:**
- [ ] Reviewa anche i task cancellati/abbandonati ogni sprint
- [ ] Chiediti: "Cosa hanno in comune i task falliti?"
- [ ] Tieni un "cimitero delle feature" con lezioni apprese

**Remediation Status:** ⏳ Processo — mitigazione affidata alla retrospective e al feature graveyard

---

### 9. Anthropomorphism Bias

**Descrizione:** Attribuire qualità umane al sistema (intenzionalità, coscienza, comprensione) fraintendendone i limiti e i comportamenti.

**Esempio aleph:**
- "Il sistema ha rifiutato la mia richiesta perché non gli piaccio" → nessuna preferenza, solo scoring probabilistico
- "Aleph ha 'capito' cosa intendo" → corrispondenza statistica, non comprensione

**Mitigazione:**
- [ ] Non usare termini come "autocoscienza", "capisce", "vuole" nella documentazione
- [ ] Usa sempre "non disponibile" invece di "non trovato" per errori di sistema
- [ ] Specifica sempre il modello e la versione nelle risposte

**Remediation Status:** ✅ Già affrontato in P2-05 — nessun claim di "autocoscienza"; tutte le risposte sono qualificate come probabilistiche

---

### 10. Anchoring Bias

**Descrizione:** Fermarsi alla prima stima o numero sentito, senza rivederlo con nuove informazioni.

**Esempio aleph:**
- "Qualcuno ha detto 4 ore" → quella diventa la stima, anche dopo aver scoperto requisiti aggiuntivi
- "Il cliente si aspetta 2 settimane" → ancori il planning a quello, non alla realtà tecnica

**Mitigazione:**
- [ ] Fai stime indipendenti PRIMA di discutere in gruppo
- [ ] Rivedi le stime dopo ogni scoperta significativa
- [ ] Chiediti: "Se non avessi sentito quel numero, cosa stimerei?"

**Remediation Status:** ⏳ Processo — mitigazione affidata a stime indipendenti pre-discussione

---

## Code Review Template

Includi questo snippet nelle PR description:

```markdown
## Bias Check (pre-review)

- [ ] **Confirmation**: Ho considerato almeno 2 alternative a questo approccio
- [ ] **Availability**: Questa non è la prima soluzione che mi è venuta in mente
- [ ] **Sunk Cost**: Sono disposto a riscrivere se la review mostra problemi fondamentali
- [ ] **Overconfidence**: La stima include buffer per edge cases e test
- [ ] **Anchoring**: La stima è stata rivista dopo aver implementato

## Reviewer Focus

- [ ] Cerca attivamente cosa potrebbe andare storto (non solo conferme)
- [ ] Verifica che gli edge cases siano gestiti
- [ ] Chiedi "perché questa scelta?" per decisioni architetturali
```

---

## Planning & Stime Checklist

Prima di finalizzare una stima:

- [ ] Ho cercato task simili nel history del progetto
- [ ] Ho moltiplicato per 2.5x se la stima è < 1 giorno
- [ ] Ho scritto 3 rischi specifici per questo task
- [ ] Ho definito criteri di stop chiari
- [ ] La stima è indipendente da aspettative esterne

---

## Decisioni Tecniche Checklist

Prima di decidere un approccio:

- [ ] Ho scritto 3 alternative con pro/contro
- [ ] Ho aspettato almeno 1 ora prima di decidere (se possibile)
- [ ] Ho chiesto a qualcuno di fare devil's advocate
- [ ] Ho cercato attivamente prove che la mia scelta è sbagliata
- [ ] La decisione è reversibile? Se no, ho coinvolto il team

---

## Verifica Completamento

Prima di segnare un task come "Done":

- [ ] Ho testato gli edge cases, non solo il happy path
- [ ] Ho considerato cosa potrebbe rompersi in produzione
- [ ] La documentazione è aggiornata
- [ ] Qualcuno altro ha reviewato il codice
- [ ] Sono disposto a dire "non è finito" se mancano pezzi

---

## Metriche da Monitorare

Traccia mensilmente:

| Metrica | Target | Attuale |
|---------|--------|---------|
| Stime S/M vs Effettivo | < 2.5x | _da compilare_ |
| Task con criteri di stop definiti | > 80% | _da compilare_ |
| PR con bias check compilato | > 90% | _da compilare_ |
| Task abbandonati con lezioni documentate | 100% | _da compilare_ |

---

## Revisioni

Questo documento va rivisto:
- Ogni 3 mesi
- Dopo ogni post-mortem significativo
- Quando un bias causa un problema grave

**Ultima revisione:** _da compilare_  
**Prossima revisione:** _da compilare_
