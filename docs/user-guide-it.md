# Guida Utente — Aleph-v2

> **Versione:** 2.0.0 · **Ultimo aggiornamento:** Aprile 2026 · **Lingua:** Italiano

---

## Cos'è Aleph-v2?

Immagina un magazzino in cui ogni scatola di dati ha un'etichetta, una mappa e un bibliotecario che parla la tua lingua. Tu entri e dici: "Mostrami le vendite di marzo". Il bibliotecario trova lo scaffale giusto, apre le scatole e ti consegna una risposta chiara. Questo è Aleph-v2.

In termini più concreti, è un sistema che organizza, analizza e interroga i tuoi dati usando AI. È come avere un data analyst personale 24 su 7. Carichi un foglio di calcolo, poni domande in italiano o in inglese e ottieni risposte estratte direttamente dai tuoi dati reali. Non serve scrivere codice né imparare a memoria comandi SQL.

---

## A chi è destinata questa guida

Questa guida è pensata per analisti, data scientist, product manager e founder. Non devi sapere programmare. Ti basta essere curioso dei tuoi dati e avere voglia di fare domande.

---

## Per iniziare in tre passaggi

### 1. Crea un progetto

Un progetto è il tuo spazio di lavoro privato. Tutti i tuoi file, le conversazioni e i risultati stanno lì dentro. Nulla traspira da un progetto all'altro.

Per crearne uno:

1. Apri Aleph nel browser (per esempio, `http://localhost:5173` se lo stai eseguendo in locale).
2. Guarda in alto sullo schermo il nome del progetto.
3. Cliccalo e scegli **Nuovo progetto**.
4. Dài un nome, come "Revisione vendite Q2" o "Feedback clienti 2026".

Il sistema costruisce automaticamente la struttura delle cartelle. Puoi sempre passare da un progetto all'altro senza disconnetterti.

### 2. Ottieni la tua API key

La prima volta che usi Aleph, ti chiede una API key. È come un badge di sicurezza che dice al sistema chi sei.

- Se qualcuno del tuo team ha installato Aleph, chiedigli di crearti una chiave dal pannello di amministrazione.
- Se sei tu l'amministratore, generane una dall'area impostazioni.
- Copiala subito. Viene mostrata una sola volta. Conservala in un password manager.

Incolla la chiave nel prompt all'avvio. Ora sei dentro.

### 3. Connetti il tuo primo file

Aleph legge CSV, Excel, JSON e può persino connettersi a Google Sheet o a API. Per la prima prova, un semplice CSV è l'ideale.

1. Nel terminale, digita `/` e cerca l'opzione per caricare un file o aggiungere una fonte dati.
2. Scegli **CSV** e seleziona il tuo file (per esempio, un report delle vendite esportato dal tuo CRM).
3. Il sistema legge le colonne, indovina i tipi di dato e costruisce una tabella.
4. In pochi secondi la tabella è pronta per le domande.

---

## Cosa puoi fare

### Chatta con i tuoi dati

Una volta connesso un file, puoi fare domande in linguaggio naturale. Aleph traduce la tua domanda in una query per il database, la esegue e ti restituisce il risultato.

Esempi di domande che funzionano bene:

- "Qual è stato il prodotto più venduto a marzo?"
- "Mostrami le entrate per regione, dalla più alta alla più bassa."
- "Quali clienti hanno acquistato più di una volta nell'ultimo trimestre?"
- "Confronta le vendite di questo mese con quelle dello stesso mese dell'anno scorso."

Dietro le quinte, Aleph costruisce la query esatta necessaria. Tu vedi il risultato in una tabella, un grafico o in formato testo. Se la risposta sembra sbagliata, puoi dirlo e l'agente riproverà.

### Agenti AI personalizzabili

Un agente è la personalità con cui chatti. Pensalo come un collega con una specialità. Un agente potrebbe essere bravo in analisi finanziaria. Un altro potrebbe concentrarsi sui ticket di assistenza clienti.

Puoi:

- Passare da un agente all'altro usando `/agent` o la palette dei comandi.
- Creare nuovi agenti con istruzioni specifiche (per esempio, "Arrotonda sempre la valuta a due decimali" oppure "Ignora le righe in cui lo stato è 'bozza'").
- Assegnare competenze agli agenti. Una competenza è un pacchetto di abilità, come "recupera dati di mercato" o "esegui analisi del sentiment su un testo".

Ogni agente ricorda il contesto della conversazione. Se chiedi le vendite di marzo e poi dici "E ad aprile?", capisce che stai parlando dello stesso dataset.

### Sandbox sicura per eseguire codice

A volte un agente ha bisogno di eseguire codice per rispondere alla tua domanda. Forse calcola una media mobile, pulisce un testo disordinato o unisce due tabelle. Questo accade dentro una stanza blindata chiamata sandbox.

La sandbox è progettata come un laboratorio con pareti di vetro spesso. Il codice può fare esperimenti, ma non può:

- Cancellare i tuoi file.
- Accedere a Internet (se necessario per le tue interrogazioni).
- Leggere dati di altri progetti.
- Eseguire comandi pericolosi.

Se un tool tenta qualcosa di sospetto, la sandbox lo blocca. Tu rimani al sicuro e i tuoi dati restano al loro posto.

### Decision engine automatico (PAORA)

Ogni volta che fai una domanda, l'agente segue un processo di ragionamento in cinque fasi:

1. **Pianifica.** Capisce cosa deve sapere e come ottenerlo.
2. **Agisce.** Esegue la query o chiama il tool giusto.
3. **Osserva.** Guarda cosa è tornato indietro.
4. **Riflette.** Verifica se la risposta ha senso. Se no, si regola e riprova.
5. **Ammette.** Ti presenta il risultato finale, insieme a una breve spiegazione di come ci è arrivato.

Questo ciclo avviene automaticamente. Non devi gestirlo tu. Il vantaggio è che l'agente si accorge dei propri errori prima di mostrarti qualcosa. Se i dati sembrano strani, si ferma e chiede chiarimenti invece di servirti nonsensi.

---

## Esempi concreti

**La responsabile vendite**

Ogni lunedì Elena scarica i dati delle vendite della settimana in un CSV e li carica su Aleph. Chiede: "Quali linee di prodotto sono cresciute di più del dieci per cento?" L'agente evidenzia i vincitori. Elena poi chiede: "Mostrami lo stesso dato per il mese scorso", e confronta le tendenze senza toccare una singola formula di Excel.

**Il product manager**

Marcus importa un dump di ticket di feedback degli utenti. Chiede: "Quali sono le tre lamentele principali di questo mese?" L'agente conta le parole chiave e le classifica. Marcus segue con: "Quali lamentele sono legate al flusso di checkout?" e ottiene una lista filtrata in pochi secondi.

**La founder**

Priya connette l'esportazione di Stripe della sua startup e un Google Sheet con le spese marketing. Chiede: "Qual è stato il nostro costo per acquisizione per canale nel primo trimestre?" L'agente unisce i due dataset e restituisce una ripartizione pulita. Lei esporta la tabella e la incolla nella sua presentazione per gli investitori.

**L'analista**

Jamal riceve un file Excel disordinato da un cliente, con celle unite e date incoerenti. Lo carica e dice: "Puliscilo e mostrami il valore medio della transazione per città." L'agente aggiusta la formattazione, esegue il calcolo e restituisce un risultato ordinato.

---

## Comandi rapidi

Una volta entrato, queste scorciatoie ti aiutano a muoverti più velocemente:

| Scorciatoia | Cosa fa |
|-------------|---------|
| `Cmd+K` (Mac) oppure `Ctrl+K` (Windows) | Apri la palette dei comandi |
| `↑` e `↓` | Scorri i comandi recenti |
| `Tab` | Autocompleta ciò che stai scrivendo |
| `Esc` | Chiudi qualsiasi pannello aperto |
| `/` | Vedi l'elenco dei comandi slash integrati |

---

## Risoluzione dei problemi

### L'agente non risponde o dice "Non posso accedere ai dati"

Questo di solito significa che i dati non sono stati caricati correttamente, oppure l'agente ha perso il filo di quale tabella usare.

- Verifica che il caricamento del file sia completato. Cerca un messaggio di conferma.
- Prova a essere specifico nella domanda: "Dalla tabella vendite, mostrami le entrate totali."
- Se il file è grande, attendi qualche secondo in più dopo il caricamento prima di fare domande.

### La query non restituisce risultati

Un risultato vuoto non è sempre un errore. Potrebbe semplicemente voler dire che nulla corrisponde.

- Controlla la tua domanda per eventuali errori di battitura, specialmente nei nomi o nelle date.
- Prova prima una versione più ampia. Invece di "Mostrami le vendite per il 15 marzo 2026", chiedi "Mostrami le vendite di marzo."
- Assicurati che la colonna su cui stai filtrando esista davvero nel file caricato.

### Grafici o tabelle sembrano strani

A volte i numeri appaiono come testo, o le date si presentano come timestamp grezzi.

- Chiedi esplicitamente all'agente di correggere il formato: "Converti la colonna data in date leggibili."
- Verifica se il tuo file sorgente aveva formati misti (per esempio, alcune righe con date nel formato americano e altre in quello europeo).
- Se necessario, ricarica il file dopo averlo pulito in Excel o Google Sheet.

### La mia API key non funziona

- Verifica di aver copiato la chiave per intera, senza spazi extra.
- Conferma che la chiave non sia stata revocata da un amministratore.
- Se stai gestendo tu l'installazione di Aleph, controlla che il server sia in esecuzione visitando l'URL base nel browser.

---

## Glossario

| Termine | Significato in parole povere |
|---------|------------------------------|
| **Progetto** | Uno spazio di lavoro privato che contiene i tuoi file, tabelle e conversazioni |
| **Agente** | Una personalità AI con cui chatti, configurata per un compito specifico |
| **Competenza** | Un pacchetto di abilità che un agente può usare, come "analizzare testo" o "eseguire query" |
| **Sandbox** | Una stanza blindata in cui il codice gira in sicurezza, senza poter toccare i tuoi dati reali |
| **PAORA** | Il ciclo di ragionamento in cinque passi che l'agente usa: Pianifica, Agisci, Osserva, Rifletti, Ammetti |
| **Query** | Una richiesta inviata al database per recuperare o calcolare qualcosa |
| **Ingestione** | Il processo di importare un file e trasformarlo in una tabella strutturata |
| **Ontologia** | Una mappa dei tuoi dati, che mostra quali colonne esistono e come si relazionano |

---

## Altre guide

- [`docs/user-guide-en.md`](./user-guide-en.md) — Guida utente in inglese
- [`docs/api-reference.md`](./api-reference.md) — Riferimento API completo per chi integra il sistema
- [`docs/deployment-guide.md`](./deployment-guide.md) — Come installare e gestire Aleph sui tuoi server
