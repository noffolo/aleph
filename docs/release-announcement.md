# Aleph-v2: A Data OS That Thinks Before It Acts

**From scattered data to structured decisions. Built-in intelligence, hardened execution.**

Data lives everywhere but goes nowhere. Teams ingest streams, spreadsheets, and logs into tools that never talk to each other. You end up with fragmented insight and reactive guessing instead of proactive decisions.

Aleph-v2 is a Data OS that unifies ingestion, intelligence, and query under one roof. It does not just store data. The system reasons about it. The PAORA engine plans every action, executes it inside an isolated sandbox, and reflects on the outcome. DuckDB handles concurrent queries with vector search for memory. A lightweight NLP sidecar extracts meaning from unstructured text. Role-based access control and hardened JWT tokens keep everything secure.

**What is inside:**

- **PAORA Decision Engine.** Plan, Act, Observe, Reflect, Admit for every interaction.
- **Sandboxed Execution.** Namespaces, seccomp, and cgroup isolation for untrusted code.
- **DuckDB Concurrency.** Rewritten for multi-tenant workloads without bottlenecks.
- **NLP Sidecar.** Sentiment and entity extraction feeding directly into the query layer.

**Built for:**

- **Data analysts** who want autocorrected streams and a unified SQL interface across sources.
- **AI engineers** deploying agents that run in isolated sandboxes with full observability.
- **DevOps teams** monitoring health with Prometheus and Grafana, and getting real alerts.
- **Startups** replacing a patchwork of tools with one hardened, open-source stack.

Do not settle for dashboards that only show what happened. Start exploring what could happen.

Clone the repo, run `docker compose up`, and give us a star on GitHub. Pull requests and issues are welcome.
