# Aleph-v2 API Documentation

## 1. RegistryService
Permette la gestione del ciclo di vita dei componenti (Tool, Skills, Sources).

### RegisterComponent
Registra un nuovo componente nel sistema.
- **Request:** `RegisterComponentRequest` (con `ComponentMetadata`)
- **Response:** `RegisterComponentResponse` (con `component_id`)

### ListComponents
Elenca i componenti filtrandoli per tipo, categoria o stato.
- **Request:** `ListComponentsRequest` (con `map<string, string> filter`)
- **Response:** `ListComponentsResponse`

---

## 2. SandboxService
Orchestra l'esecuzione sicura del codice in ambienti isolati.

### ExecuteTool
Esegue un tool specifico registrato nel registry.
- **Request:** `ExecuteToolRequest` (`tool_id`, `input_params` as struct)
- **Response:** `ExecuteToolResponse` (`ExecutionResult` con `stdout`, `exit_code`, ecc.)

### RunSkill
Esegue una skill che orchestra più tool.
- **Request:** `RunSkillRequest` (`skill_id`, `input_params`, `context`)
- **Response:** `RunSkillResponse`
