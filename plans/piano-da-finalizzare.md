# Piano Operativo: Aleph-v2 Finalization & Alignment

## 1. Obiettivo
Portare la codebase in `/Users/ff3300/Desktop/aleph-v2/` a uno stato "Production-Ready", allineandola all'architettura definita: backend Go (Connect RPC), sidecar Python (ML/Tools), frontend React (Type-safe), e infrastruttura Docker.

## 2. Roadmap di Esecuzione
1. **Configurazione Deployment:** Scrittura di `Dockerfile` e `docker-compose.yml`.
2. **Setup Backend:** Implementazione di `main.go`, `registry_handler.go`, `sandbox_handler.go`, `duckdb_registry.go`.
3. **Integrazione Python:** Popolamento di `aleph_tools/` e `nlp/main.py`.
4. **Finalizzazione Frontend:** Correzione configurazione Vite/TS e file client RPC.
5. **Documentazione:** Scrittura di `README.md` e API Docs.

## 3. Stato Post-Esecuzione
Tutti i file nella cartella di lavoro saranno allineati alla versione finale. La build sarà "Zero Errors".
