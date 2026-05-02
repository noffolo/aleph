# i18n Evaluation Document — Aleph-v2 Frontend

> **Scopo:** Valutare lo stato di internazionalizzazione del frontend Aleph-v2,
> proporre un approccio tecnico, stimare lo sforzo per le 3 lingue target (IT, EN, FR)
> e definire il percorso di migrazione per le stringhe italiane esistenti.

**Data:** 2 Maggio 2026  
**Autore:** C09 Frontend Test Coverage / i18n Assessment  
**Target Locales:** IT (italiano), EN (inglese), FR (francese)

---

## 1. Executive Summary

Il frontend Aleph-v2 dispone già di una **infrastruttura i18n custom** con ~172
chiavi di traduzione in italiano e ~165 chiamate a `t()` distribuite su 37 file
TSX. Tuttavia, esistono ancora **67+ stringhe hardcoded** (italiano e inglese)
in 20 componenti, un sistema di pluralizzazione assente, e nessun meccanismo di
language switching.

**Raccomandazione:** Migrare a `react-i18next` (con `i18next-http-backend` per
lazy-loading) — la soluzione più matura con supporto nativo per pluralizzazione,
interpolazione, namespaces e formattazione. Lo sforzo stimato è **~3gg per la
migrazione completa + ~1.5gg per ogni nuova lingua** (traduzione + QA).

---

## 2. Stato Attuale: i18n-Ready Assessment

### 2.1 Infrastruttura Esistente

| Componente | File | Stato |
|---|---|---|
| Funzione `t()` | `frontend/src/i18n/index.ts` | **Attivo** — interpola `{key}`, fallback al key se mancante |
| Dizionario IT | `frontend/src/i18n/locale.json` | **172 chiavi**, solo italiano |
| Type safety | `LocaleKey` type da `keyof typeof locale` | Definito ma non usato nella firma di `t()` |
| Pluralizzazione | — | **Assente** |
| Language Detection | — | **Assente** (sempre IT) |
| Namespace/Lazy Loading | — | **Assente** (singolo file JSON) |
| Lingua EN in SetupWizard | `SetupWizard.tsx:35-46` | Hardcoded inline, duplicato |

### 2.2 Copertura `t()` — Chiamate Esistenti

**165 chiamate** a `t()`, distribuite su **37 file TSX**. I componenti principali
sono quasi completamente tradotti:

| Componente | Chiamate `t()` | Stato |
|---|---|---|
| `AgentsView.tsx` | ~12 | ✅ Quasi completo (manca `Aggiorna Agente` button) |
| `SkillsView.tsx` | ~8 | ✅ Quasi completo (manca `Salvataggio...`/`Aggiorna Skill`) |
| `ToolsView.tsx` | ~8 | ✅ Quasi completo (stesso pattern) |
| `DataSourcesView.tsx` | ~12 | ✅ Buona copertura, status label miste IT/EN |
| `ComponentsView.tsx` | ~12 | ✅ Buona copertura, stringhe vuoto hardcoded |
| `SetupWizard.tsx` | ~6 | ⚠️ Solo parziale — doppio set inline IT/EN |
| `OracleView.tsx` | ~18 | ⚠️ Buona copertura `t()` ma confidenza label hardcoded |
| `LibraryView.tsx` | ~8 | ✅ Drag-and-drop testi via `t()` |
| `SettingsView.tsx` | ~12 | ✅ Tutti i label via `t()` |
| `CopilotView.tsx` | ~5 | ✅ Azioni principali via `t()` |
| `CommandPalette.tsx` | ~6 | ✅ Prompt e sezioni via `t()` |
| `GenericCommandPalette.tsx` | ~3 | ✅ Ricerca via `t()` |
| `Toast.tsx` | ~5 | ✅ Tipi toast via `t()` |
| `ConfirmDialog.tsx` | ~2 | ✅ Actions via `t()` |
| `DetailPanel.tsx` | ~3 | ✅ Labels via `t()` |
| `SlideOverPanel.tsx` | ~3 | ⚠️ Fullscreen labels hardcoded |
| `TerminalPrompt.tsx` | 0 | ❌ Nessuna chiamata — `placeholder` hardcoded |
| `ToolIntelligenceView.tsx` | 0 | ❌ Nessuna chiamata — tutto in EN hardcoded |
| `ScenarioComparisonView.tsx` | 0 | ❌ Nessuna chiamata — tutto in EN hardcoded |
| `DataHealthView.tsx` | 0 | ❌ `Profilo Dati`, `Unici`, `Record` hardcoded |
| `OntologyView.tsx` | 0 | ❌ `Modellazione Business`, `Emergenza Automatica`, etc. |
| `GuideTour.tsx` | 0 | ❌ `Suggerimenti`, `Correlati`, `Precedente`, etc. |

### 2.3 Inventario Completo — Stringhe Hardcoded

Elenco esaustivo delle stringhe che NON passano attraverso `t()`:

#### A. Form Components (validazione errori)

| File | Stringa | Linea |
|---|---|---|
| `AgentFormSlideOver.tsx` | `'Il nome è obbligatorio'` | 32 |
| `AgentForm.tsx` | `'Aggiorna Agente'` (edit mode) | 161 |
| `ComponentFormSlideOver.tsx` | `'Il nome è obbligatorio'` | 36 |
| `DataSourceFormSlideOver.tsx` | `'Il nome è obbligatorio'` | 58 |
| `DataSourceFormSlideOver.tsx` | `'Il JSON di configurazione non è valido'` | 63 |
| `DataSourceFormSlideOver.tsx` | `'La stringa di connessione è obbligatoria'` | 71 |
| `DataSourceFormSlideOver.tsx` | `'Indietro'` (step nav) | 274 |
| `SkillForm.tsx` | `'Salvataggio...'` (saving state) | 148 |
| `SkillForm.tsx` | `'Aggiorna Skill'` (edit mode) | 148 |
| `ToolForm.tsx` | `'Salvataggio...'` (saving state) | 140 |
| `ComponentForm.tsx` | `'Modifica Componente'` / `'Registra Componente'` | 59 |
| `DataSourceForm.tsx` | `'Nuova Sorgente Dati'` | 97 |

#### B. View Components (label UI)

| File | Stringa | Linea | Lingua |
|---|---|---|---|
| `OntologyView.tsx` | `'Modellazione Business'` | 24 | IT |
| `OntologyView.tsx` | `"L'Ontologia è il filtro..."` | 25 | IT |
| `OntologyView.tsx` | `'Emergenza Automatica'` | 30 | IT |
| `OntologyView.tsx` | `'Pubblica Modello'` | 34 | IT |
| `OntologyView.tsx` | `'Editor Codice DSL Aleph'` | 44 | IT |
| `OntologyView.tsx` | `'Glossario Visivo'` | 60 | IT |
| `OntologyView.tsx` | `'Struttura rilevata...'` | 61 | IT |
| `OntologyView.tsx` | `'Tips'` | 117 | EN |
| `OntologyView.tsx` | `'Usa <b>relation</b> per...'` | 119 | IT |
| `DataHealthView.tsx` | `'Profilo Dati'` | 40 | IT |
| `DataHealthView.tsx` | `'Unici'` | 45 | IT |
| `DataHealthView.tsx` | `'Record'` | 49 | IT |
| `DataHealthView.tsx` | `'Distribuzione Top 5'` | 57 | IT |
| `DataHealthView.tsx` | `'[Vuoto]'` | 64 | IT |
| `DataHealthView.tsx` | `'Seleziona un oggetto...'` | 87 | IT |
| `OracleView.tsx` | `'Alta confidenza'` | 252 | IT |
| `OracleView.tsx` | `'Confidenza media'` | 253 | IT |
| `OracleView.tsx` | `'Bassa confidenza'` | 254 | IT |
| `ToolManagementView.tsx` | `'Sconosciuto'` (health status) | 81 | IT |
| `ToolManagementView.tsx` | `'Ultimo check:'` | 111 | IT |
| `ToolManagementView.tsx` | `'Mai'` (never checked) | 111 | IT |
| `ComponentsView.tsx` | `'Nessun componente corrisponde al filtro'` | 191 | IT |
| `ComponentsView.tsx` | `'Nessun componente registrato...'` | 191 | IT |
| `DataSourcesView.tsx` | `'esecuzione'` (status match) | 80 | IT |
| `DataSourcesView.tsx` | `'completato'` (status match) | 80 | IT |
| `DataSourcesView.tsx` | `'fallito'` (status match) | 80 | IT |
| `ExplorerView.tsx` | `` `Cerca in ${selectedObject}...` `` | 50 | IT |
| `ExplorerView.tsx` | `'DuckDB Engine • No-ETL Query'` | 78 | EN |

#### C. Setup & Onboarding

| File | Stringa | Linea |
|---|---|---|
| `SetupWizard.tsx` | `'Errore nella creazione del progetto'` | 58 |
| `SetupWizard.tsx` | `'Errore nella generazione della chiave'` | 72 |
| `SetupWizard.tsx` | Tutto il set EN: `'Create your workspace'`... | 35–46 |

#### D. Guide & Tutorial

| File | Stringa | Linea |
|---|---|---|
| `GuideTour.tsx` | `'Suggerimenti'` | 83 |
| `GuideTour.tsx` | `'Correlati'` | 99 |
| `GuideTour.tsx` | `'Precedente'` | 129 |
| `GuideTour.tsx` | `'Chiudi'` / `'Prossimo'` | 151 |
| `contextualGuides.ts` | **Tutti i 195 contenuti** — titoli, descrizioni, tips | 1–195 |

#### E. UI / Shell Components

| File | Stringa | Linea | Lingua |
|---|---|---|---|
| `SlideOverPanel.tsx` | `'Esci da schermo intero'` / `'Schermo intero'` | 114–115 | IT |
| `TerminalPrompt.tsx` | `'inserisci un comando...'` (placeholder) | 75 | IT |
| `TerminalPrompt.tsx` | `'type freely...'` (modalità libera) | 75 | EN |
| `ChatExportMenu.tsx` | `'ESPORTA'` | 47 | IT |
| `ChatExportMenu.tsx` | CSV header: `'Role,Content,CreatedAt,ToolCall\n'` | 24 | EN |
| `CopilotView.tsx` | `'Torna in fondo'` (scroll button) | 137 | IT |
| `DataPanel.tsx` | `'DETAIL'` | 27 | EN |
| `DataPanel.tsx` | `'No data selected'` | 41 | EN |
| `InlineRenderer.tsx` | 11× `label="..."` (AgentsView, SkillsView, etc.) | 97–234 | EN |

#### F. Componenti Solo-Inglese (non toccati da i18n)

| File | Stringhe Hardcoded | Lingua |
|---|---|---|
| `ToolIntelligenceView.tsx` | `"CodeFlow Analysis"`, `"Usage Patterns"`, `"Security Intelligence"`, `"Cross-Context Recommendations"`, `"Top Users"`, `"Execs"`, `"Avg ms"`, `"Risk:"`, `"Loading intelligence data..."`, `"No tool data available"`, `"Related:"` — **~25 stringhe** | EN |
| `ScenarioComparisonView.tsx` | `"No Scenarios Found"`, `"Select 2 or 3 scenarios..."`, `"Compare:"`, `"Conf:"`, `"Confidence Score"`, `"Trend Direction"`, `"Key Signals"`, `"Core Assumptions"`, `"Scenario Detail"`, `"Probability"` — **~15 stringhe** | EN |

---

## 3. Raccomandazione: react-i18next

### 3.1 Confronto Approcci

| Criterio | **Custom attuale** | **react-i18next** | **Custom migliorato** |
|---|---|---|---|
| Tempo setup | ✅ Già fatto | 🟡 1gg | 🟡 2gg (reimplementare pluralizzazione, namespaces, detection) |
| Pluralizzazione | ❌ Assente | ✅ `t('key', {count})` via ICU | 🔴 Da implementare manualmente |
| Language Detection | ❌ Assente | ✅ `i18next-browser-languagedetector` | 🔴 Da implementare manualmente |
| Lazy Loading namespaces | ❌ Singolo JSON | ✅ `i18next-http-backend` nativo | 🔴 Da implementare manualmente |
| Interpolazione | ✅ Base (`{key}`) | ✅ ICU (`{{key}}`) + formattazione | 🟡 Da estendere |
| Tooling (editor, CLI) | ❌ Nessuno | ✅ `i18next-scanner`, `i18n-ally` (VSCode) | 🔴 Da costruire |
| Bundle size | ✅ ~2KB (locale.json) | 🟡 ~15KB gzippato | 🟡 ~5KB |
| Ecosystem | 🟡 Interno | ✅ 4M+ download/settimana, React Native compatibile | 🟡 Interno |
| Curva apprendimento | ✅ Il team lo conosce | 🟡 1–2gg per setup avanzato | 🟡 Il team lo conosce |

### 3.2 Perché react-i18next

1. **Pluralizzazione ICU** — indispensabile per IT (`1 agente`, `2 agenti`, `5 agenti`), EN (`1 agent`, `2 agents`), FR (`1 agent`, `2 agents`)
2. **Lazy loading namespaces** — `locale.json` è già 6.8KB; con 3 lingue diventerebbe 20KB. Con namespaces (`common`, `agents`, `tools`, ...) si carica solo il necessario
3. **Language detection automatica** — ` navigator.language` → fallback IT
4. **VSCode i18n-ally** — anteprime inline, editing tabellare
5. **Maturità** — 4M+ download/settimana, compatibile SSR/React Native, battle-tested
6. **Transifex/Lokalise ready** — se in futuro servirà esternalizzare le traduzioni

### 3.3 Alternative Considerate e Scartate

| Alternativa | Motivo scarto |
|---|---|
| **Custom migliorato** | Reimplementare pluralizzazione + detection + namespaces richiede 3–4gg di dev, risultato inferiore a react-i18next |
| **FormatJS (react-intl)** | Più pesante (50KB+), meno diffuso nell'ecosistema React |
| **LinguiJS** | Ottimo ma richiede macro babel/swc plugin — overhead build non giustificato per 3 lingue |
| **next-intl** | Solo per Next.js, non applicabile a Vite |
| **Continuare col custom attuale** | Bloccante: manca pluralizzazione, nessun EN/FR senza refactor pesante |

---

## 4. Percorso di Migrazione

### 4.1 Fase 1: Setup (0.5gg)

1. **Installare dipendenze:**
   ```bash
   cd frontend
   npm install react-i18next i18next i18next-http-backend i18next-browser-languagedetector
   ```

2. **Creare `frontend/src/i18n/config.ts`:**
   ```ts
   import i18n from 'i18next';
   import { initReactI18next } from 'react-i18next';
   import LanguageDetector from 'i18next-browser-languagedetector';
   import HttpBackend from 'i18next-http-backend';

   i18n
     .use(HttpBackend)
     .use(LanguageDetector)
     .use(initReactI18next)
     .init({
       fallbackLng: 'it',
       supportedLngs: ['it', 'en', 'fr'],
       ns: ['common', 'agents', 'skills', 'tools', 'datasources', 'components', 'oracle', 'settings', 'guides'],
       defaultNS: 'common',
       backend: { loadPath: '/locales/{{lng}}/{{ns}}.json' },
       detection: { order: ['localStorage', 'navigator'] },
       interpolation: { escapeValue: false },
     });

   export default i18n;
   ```

3. **Aggiungere `I18nextProvider` in `main.tsx`** (wrap esistente).

### 4.2 Fase 2: Migrare `locale.json` → namespace (1.5gg)

1. **Suddividere le 172 chiavi esistenti** nei namespace logici:
   - `common.json` — toast, errori, conferme, azioni generiche (40 chiavi)
   - `agents.json` — gestione agenti (12 chiavi)
   - `skills.json` — skill framework (10 chiavi)
   - `tools.json` — toolbox (10 chiavi)
   - `datasources.json` — sorgenti dati (15 chiavi)
   - `components.json` — registry componenti (10 chiavi)
   - `oracle.json` — oracle engine (20 chiavi)
   - `settings.json` — impostazioni (15 chiavi)
   - `guides.json` — guide contestuali (nuovo, ~195 entry)

2. **Copiare in `public/locales/it/`** — struttura cartelle:
   ```
   public/locales/
     it/
       common.json
       agents.json
       skills.json
       ...
     en/   (vuoto inizialmente)
     fr/   (vuoto inizialmente)
   ```

3. **Rimpiazzare `import { t } from '../i18n'`** con `import { useTranslation } from 'react-i18next'`:
   - Nei componenti funzione: `const { t } = useTranslation('agents')`
   - Per namespace multipli: `const { t } = useTranslation(['common', 'agents'])`
   - Per utility fuori React (raro): `import i18n from '../i18n/config'; i18n.t(...)`

### 4.3 Fase 3: Eliminare Stringhe Hardcoded (1gg)

**Priorità 1 (bloccanti UX):**
- `AgentFormSlideOver.tsx`, `ComponentFormSlideOver.tsx`, `DataSourceFormSlideOver.tsx`: errori validazione → `t('validation.nameRequired')`
- `GuideTour.tsx` + `contextualGuides.ts`: tutte le guide → namespace `guides`
- `SetupWizard.tsx`: set IT/EN inline → `t('setup.*')` con language switch

**Priorità 2 (viste principali):**
- `OntologyView.tsx`: 9 stringhe → `t('ontology.*')`
- `DataHealthView.tsx`: 7 stringhe → `t('dataHealth.*')`
- `OracleView.tsx`: 3 stringhe confidenza → `t('oracle.confidence.*')`
- `ExplorerView.tsx`: 2 stringhe → `t('explorer.*')`
- `TerminalPrompt.tsx`: 2 placeholder → `t('terminal.*')`

**Priorità 3 (viste EN-only):**
- `ToolIntelligenceView.tsx`: ~25 stringhe → namespace `toolIntel`
- `ScenarioComparisonView.tsx`: ~15 stringhe → namespace `scenarioComparison`
- `DataPanel.tsx`: 2 stringhe → `t('common.*')`

### 4.4 Fase 4: Aggiungere Pluralizzazione (0.5gg)

Sostituire costrutti condizionali con `t('agents.count', { count })`:

```json
// agents.json (IT)
{
  "count_one": "{{count}} agente",
  "count_other": "{{count}} agenti"
}
```

Chiamata: `t('agents.count', { count: n })` — react-i18next risolve automaticamente la forma plurale via suffissi `_one`, `_other` (e `_zero`, `_two`, `_few`, `_many` per arabo/russo in futuro).

### 4.5 Fase 5: Script di Validazione CI (0.5gg — opzionale)

Aggiungere al workflow CI un check che impedisca stringhe hardcoded in italiano
o inglese nei componenti:

```bash
# Rileva stringhe IT/EN non intercettate da t()
grep -rPn "(?<!t\()'[A-ZÀ-ÿ][a-zà-ÿ].*[a-zà-ÿ]'" frontend/src/components/
```

---

## 5. Stima Sforzo per Nuova Lingua

### 5.1 Per EN (prima lingua dopo IT)

| Attività | Ore |
|---|---|
| Traduzione 172 chiavi esistenti (assistita da LLM, revisione umana) | 4h |
| Traduzione ~90 nuove chiavi (hardcoded → `t()`) | 3h |
| Traduzione guide contestuali (~195 testi lunghi) | 6h |
| Pluralizzazione EN (verifica forme `_one`/`_other`) | 1h |
| QA: switch lingua, verificare troncamenti UI, formattazione numeri/date | 4h |
| **Totale EN** | **~18h (2.5gg)** |

### 5.2 Per FR (seconda lingua dopo EN)

| Attività | Ore |
|---|---|
| Traduzione ~260 chiavi (IT→FR assistita da LLM, revisione madrelingua) | 6h |
| Pluralizzazione FR (`_one`/`_other`/`_zero` per alcune forme) | 2h |
| Adattamento cultural-specific (formati data, unità di misura) | 2h |
| QA: verifica accenti, apostrofi, caratteri speciali, troncamenti | 3h |
| **Totale FR** | **~13h (2gg)** |

### 5.3 Costo Marginale Lingua N+1

Con l'infrastruttura react-i18next già attiva:
- **~10h** per lingue romanze/germaniche (stessa struttura grammaticale)
- **~15h** per lingue con plurali complessi (arabo: 6 forme plurali, russo: 4)
- **+20%** se richiesta revisione da traduttore professionista esterno

---

## 6. Key Naming Convention

### 6.1 Proposta: `namespace.section.element_variant`

**Principi:**
1. **Namespace come prefisso** — corrisponde al file JSON (`agents`, `skills`, etc.)
2. **Dot notation gerarchica** — dal generale allo specifico
3. **Snake_case per chiavi** — coerente con l'esistente
4. **No abbreviazioni ambigue** — `desc` invece di `dscr`, `err` non `e`
5. **Suffissi plurali standard** — `_one`, `_other` per i18next

### 6.2 Pattern per Tipo di Contenuto

| Tipo | Pattern | Esempio |
|---|---|---|
| Titolo pagina | `{ns}.title` | `agents.title` |
| Sottotitolo/descrizione | `{ns}.subtitle` | `agents.subtitle` → `"Configura agenti AI..."` |
| Pulsanti azione | `{ns}.{action}` | `agents.create`, `tools.delete` |
| Form label | `{ns}.form.{field}` | `agents.form.name`, `tools.form.code` |
| Form placeholder | `{ns}.form.{field}Placeholder` | `agents.form.namePlaceholder` |
| Validazione errore | `validation.{field}.{error}` | `validation.name.required` |
| Stato / badge | `{ns}.status.{state}` | `datasources.status.running` |
| Messaggi vuoto | `{ns}.empty.{context}` | `agents.empty.noAgents` |
| Toast / notifiche | `toast.{type}` | `toast.error`, `toast.success` |
| Errori generici | `errors.{category}` | `errors.network`, `errors.validation` |
| Guide contestuali | `guides.{view}.{field}` | `guides.agents-view.title` |

### 6.3 Esempio: Namespace `agents`

```json
{
  "title": "Gestore Agenti",
  "subtitle": "Configura agenti AI con qualsiasi provider — locale o cloud.",
  "create": "Nuovo Agente",
  "edit": "Modifica Agente",
  "noSystemPrompt": "Nessun prompt di sistema configurato.",
  "search": "Cerca...",
  "count_one": "{{count}} agente",
  "count_other": "{{count}} agenti",
  "empty": {
    "noAgents": "Nessun agente configurato"
  },
  "form": {
    "name": "Nome Agente",
    "namePlaceholder": "Es: Analista Finanze",
    "model": "Modello LLM",
    "modelPlaceholder": "Es: gpt-4o-mini, claude-3-5-sonnet",
    "apiKey": "API Key (override)",
    "apiKeyPlaceholder": "Inserisci solo per override globale",
    "baseUrl": "Base URL Provider",
    "baseUrlPlaceholder": "Es: https://api.openai.com/v1",
    "systemPrompt": "System Prompt",
    "systemPromptPlaceholder": "Definisci il ruolo e le restrizioni dell'agente...",
    "nameRequired": "Il nome è obbligatorio"
  },
  "update": "Aggiorna Agente"
}
```

### 6.4 Anti-Pattern da Evitare

| ❌ Evitare | ✅ Preferire | Motivazione |
|---|---|---|
| `agents_create_btn` | `agents.create` | Gerarchia vs flat, ridondanza `_btn` |
| `err.msg.1` | `errors.streamFailed` | Indici numerici = non semanticamente informativi |
| `datasourcesSubtitle` | `datasources.subtitle` | Dot notation = namespaces, camelCase lungo illeggibile |
| `common.save` (ambiguo) | `common.save` va bene, ma meglio `actions.save` se namespace dedicato | Evitare `common` come dumping ground |
| `agents.AGGIORNA_AGENTE` | `agents.update` | UPPER_CASE = urla; usa camelCase o PascalCase descrittivi |

### 6.5 Gestione Testi Lunghi (Guide, Descrizioni)

Per contenuti > 200 caratteri, usare chiavi strutturate con sottosezioni:

```json
{
  "guides": {
    "agents-view": {
      "title": "Gestione Agenti",
      "description": "Configura e orchestra i tuoi agenti AI...",
      "tips": {
        "systemPrompt": "Definisci un system prompt rigoroso...",
        "modelChoice": "Scegli il modello in base alla complessità...",
        "sharedAgents": "Sfrutta gli agenti condivisi...",
        "healthCheck": "Controlla regolarmente la sezione Health..."
      }
    }
  }
}
```

Questo permette di caricare/visualizzare i tips singolarmente senza parsare
array anonimi.

---

## 7. Component-by-Component String Inventory

Mappa completa file → stringhe → stato migrazione.

### agents (namespace)

| File | Chiave | Stato |
|---|---|---|
| `AgentsView.tsx` | `agents.title` | ✅ via `t()` |
| | `agents.subtitle` | ✅ via `t()` |
| | `agents.create` | ✅ via `t()` |
| | `agents.edit` | ✅ via `t()` |
| | `agents.search` | ✅ via `t()` |
| | `agents.noSystemPrompt` | ✅ via `t()` |
| `AgentForm.tsx` | `agents.create` (title) | ✅ via `t()` |
| | `agents.edit` (title) | ✅ via `t()` |
| | `agents.create` (button) | ✅ via `t()` |
| | `'Aggiorna Agente'` | ❌ **hardcoded** line 161 |
| | `agents.form.name` (placeholder) | ✅ via `t()` |
| | `agents.form.model` (placeholder) | ✅ via `t()` |
| | `agents.form.apiKey` (placeholder) | ✅ via `t()` |
| | `agents.form.baseUrl` (placeholder) | ✅ via `t()` |
| | `agents.form.systemPrompt` (placeholder) | ✅ via `t()` |
| `AgentFormSlideOver.tsx` | `agents.create` (title) | ✅ via `t()` |
| | `agents.edit` (title) | ✅ via `t()` |
| | `'Il nome è obbligatorio'` | ❌ **hardcoded** line 32 |

### skills (namespace)

| File | Chiave | Stato |
|---|---|---|
| `SkillsView.tsx` | `skills.title` | ✅ via `t()` |
| | `skills.subtitle` | ✅ via `t()` |
| | `skills.create` | ✅ via `t()` |
| | `skills.search` | ✅ via `t()` |
| `SkillForm.tsx` | `skills.create` (title button) | ✅ via `t()` |
| | `skills.edit` (title button) | ✅ via `t()` |
| | `'Salvataggio...'` | ❌ **hardcoded** line 148 |
| | `'Aggiorna Skill'` | ❌ **hardcoded** line 148 |
| | `skills.form.name` (placeholder) | ✅ via `t()` |
| | `skills.form.description` (placeholder) | ✅ via `t()` |
| `SkillFormSlideOver.tsx` | `skills.edit` (title) | ✅ via `t()` |
| | `skills.create` (title) | ✅ via `t()` |

### tools (namespace)

| File | Chiave | Stato |
|---|---|---|
| `ToolsView.tsx` | `tools.title` | ✅ via `t()` |
| | `tools.subtitle` | ✅ via `t()` |
| | `tools.create` | ✅ via `t()` |
| | `tools.search` | ✅ via `t()` |
| `ToolForm.tsx` | `tools.create` (title) | ✅ via `t()` |
| | `tools.edit` (title) | ✅ via `t()` |
| | `'Salvataggio...'` | ❌ **hardcoded** line 140 |
| | `tools.form.name` (placeholder) | ✅ via `t()` |
| | `tools.form.description` (placeholder) | ✅ via `t()` |
| `ToolFormSlideOver.tsx` | `tools.edit` (title) | ✅ via `t()` |
| | `tools.create` (title) | ✅ via `t()` |
| `ToolManagementView.tsx` | `tools.search` | ✅ via `t()` |
| | `'Sconosciuto'` (health) | ❌ **hardcoded** line 81 |
| | `'Ultimo check:'` | ❌ **hardcoded** line 111 |
| | `'Mai'` | ❌ **hardcoded** line 111 |
| `ToolIntelligenceView.tsx` | `'CodeFlow Analysis'` | ❌ **hardcoded** line 61 |
| | `'Usage Patterns'` | ❌ **hardcoded** line 96 |
| | `'Security Intelligence'` | ❌ **hardcoded** line 125 |
| | `'Top Users'` | ❌ **hardcoded** line 113 |
| | `'Execs'` | ❌ **hardcoded** line 87 |
| | `'Avg ms'` | ❌ **hardcoded** line 88 |
| | `'Risk:'` | ❌ **hardcoded** line 131 |
| | `'Loading intelligence data...'` | ❌ **hardcoded** line 45 |
| | `'No tool data available'` | ❌ **hardcoded** line 53 |
| | `'Cross-Context Recommendations'` | ❌ **hardcoded** line 156 |
| | `'Related:'` | ❌ **hardcoded** line 189 |

### datasources (namespace)

| File | Chiave | Stato |
|---|---|---|
| `DataSourcesView.tsx` | `datasources.title` | ✅ via `t()` |
| | `datasources.subtitle` | ✅ via `t()` |
| | `datasources.create` | ✅ via `t()` |
| | `datasources.status.running` | ✅ via `t()` |
| | `datasources.status.completed` | ✅ via `t()` |
| | `datasources.status.failed` | ✅ via `t()` |
| | `datasources.status.execute` | ✅ via `t()` |
| | `datasources.confirmDelete` | ✅ via `t()` |
| | `datasources.empty` | ✅ via `t()` |
| | `datasources.noPipeline` | ✅ via `t()` |
| | `datasources.logOutput` | ✅ via `t()` |
| | `'esecuzione'` (match) | ❌ **hardcoded** line 80 |
| | `'completato'` (match) | ❌ **hardcoded** line 80 |
| | `'fallito'` (match) | ❌ **hardcoded** line 80 |
| `DataSourceFormSlideOver.tsx` | `'Il nome è obbligatorio'` | ❌ **hardcoded** line 58 |
| | `'Il JSON di configurazione...'` | ❌ **hardcoded** line 63 |
| | `'La stringa di connessione...'` | ❌ **hardcoded** line 71 |
| | `'Indietro'` (step nav) | ❌ **hardcoded** line 274 |
| `DataSourceForm.tsx` | `'Nuova Sorgente Dati'` | ❌ **hardcoded** line 97 |

### ontology (nome suggerito: `ontology`)

| File | Chiave | Stato |
|---|---|---|
| `OntologyView.tsx` | `'Modellazione Business'` | ❌ **hardcoded** line 24 |
| | `"L'Ontologia è il filtro..."` | ❌ **hardcoded** line 25 |
| | `'Emergenza Automatica'` | ❌ **hardcoded** line 30 |
| | `'Pubblica Modello'` | ❌ **hardcoded** line 34 |
| | `'Editor Codice DSL Aleph'` | ❌ **hardcoded** line 44 |
| | `'Glossario Visivo'` | ❌ **hardcoded** line 60 |
| | `'Struttura rilevata...'` | ❌ **hardcoded** line 61 |
| | `'Tips'` | ❌ **hardcoded** line 117 |
| | Testo descrittivo Tips | ❌ **hardcoded** line 119 |

### guides (nuovo namespace)

| File | Chiave | Stato |
|---|---|---|
| `contextualGuides.ts` | Tutti gli 11 `title` + `description` + `tips` arrays | ❌ Tutto hardcoded IT (~195 testi) |
| `GuideTour.tsx` | `'Suggerimenti'` | ❌ **hardcoded** line 83 |
| | `'Correlati'` | ❌ **hardcoded** line 99 |
| | `'Precedente'` | ❌ **hardcoded** line 129 |
| | `'Chiudi'` / `'Prossimo'` | ❌ **hardcoded** line 151 |

### scenarioComparison (nuovo namespace)

| File | Chiave | Stato |
|---|---|---|
| `ScenarioComparisonView.tsx` | `'No Scenarios Found'` | ❌ **hardcoded** line 34 |
| | `'Run a prediction...'` | ❌ **hardcoded** line 36 |
| | `'Compare:'` | ❌ **hardcoded** line 65 |
| | `'Select 2 or 3...'` | ❌ **hardcoded** line 88 |
| | `'Conf:'` | ❌ **hardcoded** line 97 |
| | `'Confidence Score'` | ❌ **hardcoded** line 106 |
| | `'Trend Direction'` | ❌ **hardcoded** line 119 |
| | `'Key Signals'` | ❌ **hardcoded** line 132 |
| | `'Core Assumptions'` | ❌ **hardcoded** line 153 |
| | `'Scenario Detail'` | ❌ **hardcoded** line 166 |
| | `'Probability'` | ❌ **hardcoded** line 176 |

### dataHealth (nuovo namespace)

| File | Chiave | Stato |
|---|---|---|
| `DataHealthView.tsx` | `'Profilo Dati'` | ❌ **hardcoded** line 40 |
| | `'Unici'` | ❌ **hardcoded** line 45 |
| | `'Record'` | ❌ **hardcoded** line 49 |
| | `'Distribuzione Top 5'` | ❌ **hardcoded** line 57 |
| | `'[Vuoto]'` | ❌ **hardcoded** line 64 |
| | `'Seleziona un oggetto...'` | ❌ **hardcoded** line 87 |

### oracle (da completare)

| File | Chiave | Stato |
|---|---|---|
| `OracleView.tsx` | `oracle.title` | ✅ via `t()` |
| | `oracle.subtitle` | ✅ via `t()` |
| | `oracle.empty` | ✅ via `t()` |
| | `oracle.sentiment.*` | ✅ via `t()` |
| | `'Alta confidenza'` | ❌ **hardcoded** line 252 |
| | `'Confidenza media'` | ❌ **hardcoded** line 253 |
| | `'Bassa confidenza'` | ❌ **hardcoded** line 254 |

### common / shell (namespace `common`)

| File | Chiave | Stato |
|---|---|---|
| `SlideOverPanel.tsx` | `'Esci da schermo intero'` | ❌ **hardcoded** line 114 |
| | `'Schermo intero'` | ❌ **hardcoded** line 115 |
| `TerminalPrompt.tsx` | `'inserisci un comando...'` | ❌ **hardcoded** line 75 |
| | `'type freely...'` | ❌ **hardcoded** line 75 |
| `ChatExportMenu.tsx` | `'ESPORTA'` | ❌ **hardcoded** line 47 |
| | CSV header `'Role,Content...'` | ❌ **hardcoded** line 24 |
| `CopilotView.tsx` | `'Torna in fondo'` | ❌ **hardcoded** line 137 |
| `DataPanel.tsx` | `'DETAIL'` | ❌ **hardcoded** line 27 |
| | `'No data selected'` | ❌ **hardcoded** line 41 |
| `SetupWizard.tsx` | `'Errore nella creazione...'` | ❌ **hardcoded** line 58 |
| | `'Errore nella generazione...'` | ❌ **hardcoded** line 72 |
| | Set EN inline completo | ❌ **hardcoded** lines 35–46 |
| `ComponentsView.tsx` | Filtro / vuoto strings | ❌ **hardcoded** line 191 |
| `ExplorerView.tsx` | `'Cerca in...'` | ❌ **hardcoded** line 50 |
| | `'DuckDB Engine...'` | ❌ **hardcoded** line 78 |

---

## 8. Riepilogo Quantitativo

| Metrica | Valore |
|---|---|
| Chiavi totali in `locale.json` (solo IT) | 172 |
| Chiamate a `t()` esistenti | ~165 (in 37 file) |
| Stringhe hardcoded IT da migrare | ~50 |
| Stringhe hardcoded EN da migrare | ~40 |
| Nuove chiavi necessarie (stima) | ~90 |
| **Totale chiavi previste post-migrazione** | **~260** |
| Namespace proposti | 9 |
| Tempo setup react-i18next | 0.5gg |
| Tempo migrazione chiavi esistenti | 1.5gg |
| Tempo eliminazione hardcoded | 1gg |
| **Totale sforzo migrazione** | **~3gg** |
| Sforzo per lingua EN | ~2.5gg |
| Sforzo per lingua FR | ~2gg |

---

## 9. Rischi e Mitigazioni

| Rischio | Probabilità | Mitigazione |
|---|---|---|
| Bundle size aumenta (react-i18next + backend) | Alta | Tree-shaking di i18next; code-split dei namespaces; `i18next-http-backend` carica solo lingua attiva (~8KB gzippato) |
| Migrazione rompe test esistenti (mock `t()`) | Media | I test già mockano `t()` via `vi.mock('../i18n')` → aggiornare mock a `react-i18next`; stimato ~2h fix test |
| Guide contestuali (~195 testi) richiedono traduzione professionale | Media | Per EN/FR usare LLM + revisione interna; se qualità insufficiente, esternalizzare solo `guides.json` a traduttore (costo ~€300/lingua) |
| Stringhe hardcoded sfuggono alla migrazione | Media | Aggiungere ESLint rule `no-literal-string` (warn per ora) + grep CI check descritto in §4.5 |
| react-i18next breaking change in major update | Bassa | Lock versione in `package.json`; i18next ha API stabile da v19 (2021) |

---

## 10. Checkpoint: i18n-Ready Gate

Prima di dichiarare il frontend "i18n-ready", verificare:

- [ ] 100% stringhe UI passano attraverso `t()` (0 hardcoded)
- [ ] 3 lingue caricate: `public/locales/it/`, `public/locales/en/`, `public/locales/fr/`
- [ ] Language detector funzionante (switch lingua persiste in `localStorage`)
- [ ] Pluralizzazione testata per casi `n=0, 1, 2, 5, 21` in tutte e 3 le lingue
- [ ] Namespace lazy-loading verificato in Network tab (solo namespace attivo caricato)
- [ ] Testi lunghi (guide) non troncati in nessuna lingua (EN/FR ~30% più lunghi di IT)
- [ ] `date.toLocaleString()` rispetta locale attivo (non hardcoded `'it-IT'`)
- [ ] `number.toLocaleString()` rispetta locale attivo (stesso problema)
- [ ] Accessibilità: `lang` attribute su `<html>` cambia con lo switch lingua
- [ ] CI: grep check blocca nuove stringhe hardcoded (opzionale ma raccomandato)

---

*Documento generato da analisi statica del codebase Aleph-v2 frontend.*  
*Ultimo aggiornamento: 2 Maggio 2026*
