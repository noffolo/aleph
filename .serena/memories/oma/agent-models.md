# OMA Agent-Model Mapping

| Role | Model | Notes |
|------|-------|-------|
| Orchestrator | kimi-k2.6:cloud | 300 sub-agent swarm, 4K step coordination, multimodal, 24/7 persistent agents, native Ollama Cloud |
| Vision | gemma4:27b-cloud | Image analysis, UI review |
| Router | ministral-3:8b-cloud | Fast routing, triage, classification |
| Worker 1 | qwen3.5:35b-a3b:cloud | Active-parameter MoE, general tasks |
| Worker 2 | nemotron-3-super:cloud | Heavy computation, analysis |
| Coder Senior | GLM-5.1 | Code generation, refactoring, implementation |
| Oracle | GLM-5.1 | Deep analysis, architectural decisions |
| Revisore | GLM-5.1 | Code review, quality assurance |
| Planner | GLM-5.1 | Task decomposition, dependency mapping |