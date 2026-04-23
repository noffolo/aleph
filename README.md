# 🌌 Aleph-v2: Decision Intelligence System (beta)

*«Il dato è solo il battito cardiaco della realtà. Aleph è il sistema che gli conferisce un significato, trasformando il caos informativo in strategia azionabile.»*

## 📝 Descrizione
Aleph-v2 è un ecosistema di intelligenza strategica che trasforma flussi di dati grezzi in scenari predittivi chiari. È il tuo laboratorio decisionale per navigare l'incertezza.

## ✨ Caratteristiche
- **🧠 Predizioni Ensemble:** Fusione di modelli statistici, ML e LLM.
- **🛡️ Architettura Resiliente:** Sandbox isolati e circuit breaker distribuito.
- **🌐 Workspace Intelligente:** Viste adattive basate sull'intento.
- **🚀 Protocollo Genesis:** Evoluzione autonoma (proposta di tool/skill) sotto controllo umano.

## 🛠️ Stack Tecnologico
- **Backend:** Go (Connect RPC, DuckDB)
- **Intelligence:** Python (PyTorch, ONNX, Prophet, XGBoost, LangChain)
- **Frontend:** React, TypeScript, Vite, Tailwind CSS
- **Orchestrazione:** Docker, Docker Compose

## ⚡ Setup Rapido
1. `git clone <repo>`
2. Crea il tuo `.env` basato su `.env.example`.
3. `docker compose up --build -d`

## 🔬 Avvertenze (beta)
- Le predizioni sono fornite a scopo esplorativo e **non costituiscono consiglio decisionale**.
- I punteggi di confidenza e i valori di sentimento riflettono stime statistiche con margini di incertezza.
- I dati contrassegnati come _sintetici_ sono generati per default e non derivano da fonti osservazionali.
- Il sistema è in fase beta attiva: funzionalità, API e accuratezza possono cambiare.

## 🚀 Uso
Accedi all'interfaccia su `http://localhost:5173`.
Gli agenti AI interagiscono via RPC usando le chiavi API configurate.
