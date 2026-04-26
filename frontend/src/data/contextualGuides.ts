export interface GuideEntry {
  /** Titolo breve della guida */
  title: string
  /** Descrizione testuale della vista e del suo scopo */
  description: string
  /** Suggerimenti rapidi per l'uso */
  tips: string[]
  /** Collegamenti a viste correlate */
  relatedLinks: { label: string; view: string }[]
}

/**
 * Guide testuali contestuali per ogni vista principale del sistema.
 * Visualizzate quando showGuide=true (al termine del SetupWizard o su richiesta).
 * Ogni guida spiega lo scopo, le azioni principali e i collegamenti ad altre viste.
 */
export const contextualGuides: Record<string, GuideEntry> = {
  'agents-view': {
    title: 'Gestione Agenti',
    description:
      'Configura e orchestra i tuoi agenti AI. Ogni agente è un’entità autonoma dotata di un modello LLM, chiavi API e un system prompt specifico. Gli agenti trasformano i tuoi dati in insight attraverso analisi, ricerche web e generazioni di report.',
    tips: [
      'Definisci un system prompt rigoroso: più dettagli fornisci sul ruolo e l\'obiettivo, più l\'agente sarà preciso.',
      'Scegli il modello in base alla complessità: GPT-4o per ragionamenti complessi, Claude per analisi di testi voluminosi.',
      'Sfrutta gli agenti condivisi per mantenere la coerenza operativa tra diversi progetti di lavoro.',
      'Controlla regolarmente la sezione "Health" per assicurarti che le API key siano valide e i modelli raggiungibili.',
    ],
    relatedLinks: [
      { label: 'Skills', view: 'skills-view' },
      { label: 'Tools', view: 'tools-view' },
    ],
  },

  'skills-view': {
    title: 'Competenze (Skills)',
    description:
      'Le skills sono capacità specializzate che gli agenti possono acquisire ed eseguire. Esse incapsulano tool specifici per svolgere compiti definiti, come l\'analisi del sentiment o la previsione finanziaria, permettendo di costruire flussi di lavoro modulari.',
    tips: [
      'Mantieni le skills atomiche: una singola skill deve risolvere un unico problema specifico per essere riutilizzabile.',
      'Combina più tool all\'interno di una skill per creare capacità composite più potenti e flessibili.',
      'Testa le skills contrassegnate come "beta" in ambiente di staging prima di assegnarle ad agenti in produzione.',
      'Utilizza i filtri di ricerca per navigare rapidamente tra le centinaia di skill disponibili per dominio.',
    ],
    relatedLinks: [
      { label: 'Agenti', view: 'agents-view' },
      { label: 'Tools', view: 'tools-view' },
    ],
  },

  'tools-view': {
    title: 'Strumenti Operativi (Tools)',
    description:
      'I tools sono le primitive tecniche del sistema: funzioni eseguibili che permettono l\'interazione con database SQL, API esterne o sistemi di file. Sono raggruppati per domini (Finance, OSINT, Human Ecosystems) e validati in sandbox.',
    tips: [
      'Esegui sempre un test in sandbox prima di esporre un nuovo tool agli agenti per evitare errori runtime.',
      'I tool di categoria "OSINT" processano dati esterni: verifica sempre l\'affidabilità delle sorgenti estratte.',
      'Gestisci l\'attivazione dei singoli tool dal pannello di dettaglio per limitare l\'esposizione di funzioni critiche.',
      'Sfrutta i tool package predefiniti per implementare rapidamente funzionalità standard di dominio.',
    ],
    relatedLinks: [
      { label: 'Skills', view: 'skills-view' },
      { label: 'Agenti', view: 'agents-view' },
    ],
  },

  'datasources-view': {
    title: 'Fonti Dati',
    description:
      'Connetti e gestisci l\'integrità dei tuoi dati. Il sistema supporta l\'importazione di file CSV/JSON, database SQLite e URL remoti. Ogni fonte viene indicizzata per permettere query istantanee tramite il motore DuckDB.',
    tips: [
      'L\'indicizzazione automatica di DuckDB rende le tabelle importate subito disponibili per analisi complesse.',
      'L\'importazione via URL è dinamica: carica il contenuto e lo processa automaticamente secondo lo schema rilevato.',
      'Usa la funzione "Preview" per validare la struttura dei dati prima di confermare l\'importazione definitiva.',
      'Il sistema garantisce l\'isolamento dei dati per progetto tramite l\'uso di schemi database dedicati.',
    ],
    relatedLinks: [
      { label: 'Explorer', view: 'explore' },
      { label: 'Libreria', view: 'library-view' },
    ],
  },

  'library-view': {
    title: 'Libreria',
    description:
      'La libreria raccoglie tutti i dati, i file e le risorse importate. Qui puoi cercare, filtrare e organizzare il tuo knowledge base. Ogni entry mostra metadati come dimensione, tipo e data di importazione.',
    tips: [
      'Usa la ricerca globale (Cmd+K) per trovare rapidamente asset in tutta la libreria.',
      'Gli asset possono essere taggati e raggruppati per facilitarne il ritrovamento.',
      'Clicca su un asset per aprirlo nel pannello di dettaglio e vedere il contenuto.',
      'La libreria supporta file di testo, CSV, JSON, e dati strutturati.',
    ],
    relatedLinks: [
      { label: 'Fonti Dati', view: 'datasources-view' },
      { label: 'Explorer', view: 'explore' },
    ],
  },

  'components-view': {
    title: 'Componenti',
    description:
      'I componenti sono blocchi UI riutilizzabili che puoi configurare e combinare. Ogni componente incapsula logica di visualizzazione e interazione: tabelle, mappe, timeline, grafi. Usali per costruire dashboard personalizzate.',
    tips: [
      'Trascina i componenti dalla sidebar per riorganizzare la tua dashboard.',
      'Ogni componente puó essere configurato con parametri specifici (fonte dati, metriche, soglie).',
      'I componenti grafici (mappe, grafi) richiedono dati geospaziali o relazionali ben formati.',
      'Usa il componente Tabella per esplorare dati tabellari con sorting e filtri.',
    ],
    relatedLinks: [
      { label: 'Explorer', view: 'explore' },
      { label: 'Libreria', view: 'library-view' },
    ],
  },

  copilot: {
    title: 'Copilot',
    description:
      'Il Copilot é il terminale di comando principale. Qui puoi chattare con gli agenti, eseguire comandi rapidi, e orchestrare flussi di lavoro. Usa il linguaggio naturale per interagire con tutti i tools del sistema.',
    tips: [
      'Usa / per vedere la lista dei comandi disponibili (slash commands).',
      'Menziona il nome di un agente per attivarlo in una conversazione: "@analista".',
      'I comandi come /tool, /skill, /help danno accesso rapido a funzionalitá specifiche.',
      'Puoi arrestare una risposta lunga con il pulsante STOP.',
      'Usa Tab per autocompletare comandi e nomi di agenti.',
    ],
    relatedLinks: [
      { label: 'Agenti', view: 'agents-view' },
      { label: 'Tools', view: 'tools-view' },
    ],
  },

  explore: {
    title: 'Explorer',
    description:
      'L\'Explorer ti permette di esplorare visivamente i dati importati. Scegli un oggetto (tabella), applica filtri e visualizza i risultati in diverse modalitá: tabella, mappa, timeline, o grafo relazionale.',
    tips: [
      'Seleziona prima un oggetto dati, poi scegli la visualizzazione piú adatta.',
      'La vista Tabella supporta sorting cliccando sulle intestazioni delle colonne.',
      'La vista Mappa funziona solo con dati che hanno coordinate geografiche (lat/lng).',
      'La vista Grafo mostra le relazioni tra entitá basate su colonne con valori comuni.',
      'Usa la barra di ricerca per filtrare i risultati in tempo reale.',
    ],
    relatedLinks: [
      { label: 'Fonti Dati', view: 'datasources-view' },
      { label: 'Componenti', view: 'components-view' },
    ],
  },

  ontology: {
    title: 'Ontologia',
    description:
      'L\'ontologia definisce la struttura concettuale dei tuoi dati: entitá, relazioni e gerarchie. Qui puoi visualizzare e modificare lo schema semantico che il sistema usa per comprendere e correlare le informazioni.',
    tips: [
      'Definisci le entitá principali del tuo dominio prima di importare dati.',
      'Le relazioni tra entitá permettono al sistema di fare inferenze cross-dominio.',
      'Usa la vista grafo per visualizzare le connessioni tra entitá.',
      'L\'ontologia é condivisa tra agenti: tutti gli agenti la usano per contestualizzare le risposte.',
    ],
    relatedLinks: [
      { label: 'Explorer', view: 'explore' },
      { label: 'Libreria', view: 'library-view' },
    ],
  },

  settings: {
    title: 'Impostazioni',
    description:
      'Pannello di configurazione globale del sistema. Qui puoi gestire preferenze di visualizzazione, effetti terminale (scanlines, glow, flicker), lingua (IT/EN), e connettersi a provider LLM esterni (Ollama, Anthropic, OpenAI).',
    tips: [
      'Gli effetti terminale sono puramente estetici e non influiscono sulle performance.',
      'Puoi cambiare lingua in qualsiasi momento: le etichette si aggiornano immediatamente.',
      'Configura almeno un provider LLM prima di usare gli agenti.',
      'Le preferenze vengono salvate localmente e persistono tra sessioni.',
    ],
    relatedLinks: [
      { label: 'Agenti', view: 'agents-view' },
      { label: 'Copilot', view: 'copilot' },
    ],
  },

  health: {
    title: 'Salute del Sistema',
    description:
      'Pannello di diagnostica e monitoraggio. Mostra lo stato di salute di tutti i componenti del sistema: agenti, tools, fonti dati e connessioni. Ogni entry ha metriche, storico dei controlli e dettagli sugli eventuali errori.',
    tips: [
      'I controlli di salute vengono eseguiti automaticamente ogni 5 minuti.',
      'Un indicatore rosso significa che il componente non risponde — verifica la connettivitá.',
      'Clicca su un evento per vedere il dettaglio dell\'errore e suggerimenti di remediation.',
      'Usa la vista "History" per vedere l\'andamento della salute nel tempo.',
    ],
    relatedLinks: [
      { label: 'Tools', view: 'tools-view' },
      { label: 'Impostazioni', view: 'settings' },
    ],
  },
}
