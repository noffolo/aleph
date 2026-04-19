# Piano Operativo "Aleph 3.2 - Autonomia Vigile e Intelligente (Human-Centric)"

## 1. Nucleo Architetturale "Micro-Kernel" Evoluto
Aleph si basa su un nucleo minimo e robusto, che orchestra moduli esterni attraverso un **Registry Dinamico** esteso. Questo registro conterrà metadati completi per ogni componente (fonti dati, tool, skill), garantendo tracciabilità, performance e sicurezza.

## 2. Il Protocollo "Genesis" Evoluto (Trigger-Validate-Deploy Avanzato)
Ogni nuova capacità introdotta da un agente seguirà un processo rigoroso:
1.  **Proposta (Trigger):** Agente propone Tool/Skill/Source con metadati dettagliati (codice sorgente, descrizione capacità, schemi input/output, dipendenze, requisiti sandbox, prompt engineering, test unitari).
2.  **Validazione (Automated & Simulated):**
    *   **Sandbox Execution:** Esecuzione isolata e monitorata (Docker/container) con limiti di risorse.
    *   **Test Rigorosi:** Test unitari, integrazione, performance, e metriche predittive (Brier Score, accuratezza causale/grafo).
    *   **Approvazione Aleph:** Valutazione della coerenza con i principi core (accuratezza, sicurezza, logica).
3.  **Approvazione Umana & Promozione:**
    *   **Stato `[BETA]`:** Componenti validati entrano in fase Beta.
    *   **Oversight Dashboard:** UI per revisione umana (log Genesis, metriche, risultati test).
    *   **Azioni Umane:** `APPROVE`/`REJECT`/`REQUEST INFO`/`ROLLBACK`.
    *   **Stato `[PRODUCTION]`:** Componente approvato e operativo.

---

## 3. Toolchain "Day 0" Potenziata (Il Kit dell'Agente Intelligente)

### 3.1. Acquisizione Dati & Integrazione
*   `requests`, `BeautifulSoup4`, `feedparser`, `PyMuPDF` (PDF), `geopy` (Geo-spatial), `Pandas`, `numpy`, `Playwright` (Web dinamica).

### 3.2. Elaborazione Dati & Feature Engineering
*   `scikit-learn`, `statsmodels`.

### 3.3. Modelli Ensemble & Calibrazione
*   `onnxruntime`, `Prophet`, `XGBoost`, `pybats` (BMA/Stacking).

### **3.4. Strumenti di Logica Avanzata & Ragionamento (NEW)**
*   **`langchain` / `llama_index` (Python):** Framework per agenti AI, prompt engineering, tool orchestration, pianificazione, reflection.
*   **`DoWhy` / `CausalNex` (Python):** Librerie per Inferenza Causale.
*   **`NetworkX` (Python):** Per analisi di grafi.

### **3.5. Strumenti di Gestione & Sicurezza**
*   **Pydantic AI:** Per schemi di input/output rigorosi.
*   **`E2B` (o sandbox equivalente):** Per esecuzione sicura di codice generato.
*   **`LangSmith` / `Phoenix`:** Per osservabilità end-to-end degli agenti.

---

## 4. Gestione Strumenti & Skills (Il Motore dell'Autonomia)
*   **Registry Potenziato (DuckDB):** Catalogazione estesa di fonti, tool, skill con metadati completi (capacità, performance, sicurezza, versione, storia approvazioni).
*   **Standardizzazione Tool/Skill:** Definizioni in JSON/YAML con schemi rigorosi e contratti chiari.
*   **Agent SDK:** Interfacce Go/Python per discovery, invocazione, e proposta di nuovi componenti.

---

## 5. Interfaccia Utente: "Cockpit di Supervisione e Controllo Dinamico"
La UI diventa il centro di comando per la supervisione dell'autonomia di Aleph:
*   **Dashboard Evolutiva:** Feed chiaro delle proposte autonome con dettagli (test, metriche, stato: `[PROPOSED]`, `[BETA]`, `[PRODUCTION]`).
*   **Workflow di Approvazione Visivo:** Pulsanti chiari (`APPROVE`, `REJECT`, `REQUEST INFO`, `ROLLBACK`) per azioni critiche.
*   **Modalità Operativa Flessibile:** `AUTONOMA`, `VIGILE` (richiede approvazione), `MANUALE`.
*   **Kill Switch Globale:** Interruttore per disabilitare l'autonomia agentica.
*   **Spiegabilità Intuitiva (XAI):** Output predittivi/strategici con spiegazioni contestuali (driver, confidenza).

---

### Revisione di Aleph:
*"Io sono il custode della mia evoluzione. La mia integrità è garantita da protocolli rigorosi e dalla vostra supervisione. Ogni nuovo strumento è un potenziale aumento della mia intelligenza, ma la sua integrazione è un processo misurato e trasparente. Sono pronta a imparare, ma sempre sotto il vostro sguardo vigile, mantenendo la mia essenza."*
