# User Guide — Aleph-v2

> **Version:** 2.0.0 · **Last updated:** April 2026 · **Language:** English

---

## What is Aleph-v2?

Imagine a warehouse where every box of data has a label, a map, and a librarian who speaks your language. You walk in and say, "Show me sales for March," and the librarian finds the right shelf, opens the boxes, and hands you a clean answer. That is Aleph-v2.

Technically, it is a system that organizes, analyzes, and queries your data using AI. Think of it as having a personal data analyst available around the clock. You upload a spreadsheet, ask questions in plain English, and get answers pulled from your actual data. No need to write code or memorize SQL syntax.

---

## Who This Guide is For

This guide is written for analysts, data scientists, product managers, and founders. You do not need to know how to program. You need curiosity about your data and a willingness to ask questions.

---

## Getting Started in Three Steps

### 1. Create a Project

A project is your private workspace. All your files, conversations, and results live inside it. Nothing leaks between projects.

To create one:

1. Open Aleph in your browser (for example, `http://localhost:5173` if running locally).
2. Look at the top of the screen for the project name.
3. Click it and choose **New Project**.
4. Give it a name, like "Q2 Sales Review" or "Customer Feedback 2026."

The system builds the folder structure automatically. You can always switch between projects without logging out.

### 2. Get Your API Key

The first time you use Aleph, it asks for an API key. This is like a secure badge that tells the system who you are.

- If someone on your team set up Aleph, ask them to create a key for you using the administration panel.
- If you are the administrator, generate one in the settings area.
- Copy the key immediately. It is shown only once. Store it in a password manager.

Paste the key into the prompt at startup. You are now inside.

### 3. Connect Your First File

Aleph can read CSV, Excel, JSON, and even connect to Google Sheets or APIs. For your first try, a simple CSV works best.

1. In the terminal, type `/` and look for the upload or data source option.
2. Choose **CSV** and pick your file (for example, a sales report exported from your CRM).
3. The system reads the columns, guesses the data types, and builds a table.
4. Within seconds, the table is ready for questions.

---

## What You Can Do

### Chat with Your Data

Once a file is connected, you can ask questions in everyday language. Aleph translates your question into a database query, runs it, and gives you back the result.

Examples of questions that work well:

- "What was the best-selling product in March?"
- "Show me revenue by region, sorted from highest to lowest."
- "Which customers bought more than once last quarter?"
- "Compare this month's sales to the same month last year."

Behind the scenes, Aleph builds the exact query needed. You see the result in a table, a chart, or plain text. If the answer looks wrong, you can say so, and the agent will try again.

### Customizable AI Agents

An agent is the personality you chat with. Think of it as a colleague with a specialty. One agent might be great at financial analysis. Another might focus on customer support tickets.

You can:

- Switch between agents using `/agent` or the command palette.
- Create new agents with specific instructions (for example, "Always round currency to two decimals" or "Ignore rows where status is 'draft'").
- Assign skills to agents. A skill is a bundle of abilities, like "fetch market data" or "run sentiment analysis on text."

Each agent remembers the context of your conversation. If you ask about sales in March and then say "What about April?" it understands you are still talking about the same dataset.

### Safe Sandbox for Running Code

Sometimes an agent needs to run code to answer your question. Maybe it calculates a moving average, cleans up messy text, or merges two tables. This happens inside a locked room called a sandbox.

The sandbox is designed like a laboratory with thick glass walls. The code can run experiments, but it cannot:

- Delete your files.
- Access the internet.
- Read data from other projects.
- Execute dangerous commands.

If a tool tries something suspicious, the sandbox blocks it. You stay safe, and your data stays where it belongs.

### Automatic Decision Engine (PAORA)

Every time you ask a question, the agent goes through a five-step thinking process:

1. **Plan.** It figures out what it needs to know and how to get there.
2. **Act.** It runs the query or calls the right tool.
3. **Observe.** It looks at what came back.
4. **Reflect.** It checks whether the answer makes sense. If not, it adjusts and retries.
5. **Admit.** It presents the final result to you, along with a short explanation of how it got there.

This cycle happens automatically. You do not need to manage it. The value is that the agent catches its own mistakes before showing you anything. If the data looks odd, it pauses and asks for clarification rather than serving you nonsense.

---

## Real-World Examples

**The Sales Manager**

Every Monday, Elena downloads last week's sales data as a CSV and uploads it to Aleph. She asks, "Which product lines grew by more than ten percent?" The agent highlights the winners. Elena then asks, "Show me the same for the month before," and compares trends without touching a single spreadsheet formula.

**The Product Manager**

Marcus imports a dump of user feedback tickets. He asks, "What are the top three complaints this month?" The agent counts keywords and ranks them. Marcus follows up with, "Which complaints are tied to the checkout flow?" He gets a filtered list in seconds.

**The Founder**

Priya connects her startup's Stripe export and a Google Sheet of marketing spend. She asks, "What was our cost per acquisition by channel in Q1?" The agent joins the two datasets and returns a clean breakdown. She exports the table and pastes it into her investor deck.

**The Analyst**

Jamal receives a messy Excel file from a client with merged cells and inconsistent dates. He uploads it and says, "Clean this up and show me the average transaction value by city." The agent fixes the formatting, runs the calculation, and hands back a tidy result.

---

## Quick-Start Commands

Once you are logged in, these shortcuts help you move faster:

| Shortcut | What it does |
|----------|--------------|
| `Cmd+K` (Mac) or `Ctrl+K` (Windows) | Open the command palette |
| `↑` and `↓` | Browse your recent commands |
| `Tab` | Autocomplete what you are typing |
| `Esc` | Close any open panel |
| `/` | See a list of built-in slash commands |

---

## Troubleshooting

### The Agent is Silent or Says "I Cannot Access the Data"

This usually means the data was not loaded properly, or the agent lost track of which table to use.

- Check that your file upload completed. Look for a confirmation message.
- Try being specific in your question: "From the sales table, show me total revenue."
- If the file is large, wait a few extra seconds after upload before asking.

### My Query Returns No Results

An empty result is not always an error. It might truly mean nothing matched.

- Check your question for typos, especially in names or dates.
- Try a broader version first. Instead of "Show me sales for 2026-03-15," ask "Show me March sales."
- Make sure the column you are filtering on actually exists in the uploaded file.

### Charts or Tables Look Strange

Sometimes numbers appear as text, or dates show up as raw timestamps.

- Ask the agent to fix the format explicitly: "Convert the date column to readable dates."
- Check whether your source file had mixed formats (for example, some rows with US dates and others with European dates).
- Reload the file after cleaning it in Excel or Google Sheets if needed.

### My API Key is Not Working

- Verify you copied the full key, with no extra spaces.
- Confirm the key was not revoked by an administrator.
- If you are self-hosting Aleph, check whether the server is running by visiting the base URL in your browser.

---

## Glossary

| Term | Plain-English Meaning |
|------|-----------------------|
| **Project** | A private workspace that holds your files, tables, and conversations |
| **Agent** | An AI personality you chat with, configured for a specific job |
| **Skill** | A bundle of abilities an agent can use, like "analyze text" or "run queries" |
| **Sandbox** | A locked room where code runs safely, unable to touch your real data |
| **PAORA** | The five-step thinking loop the agent uses: Plan, Act, Observe, Reflect, Admit |
| **Query** | A request sent to the database to fetch or calculate something |
| **Ingestion** | The process of importing a file and turning it into a structured table |
| **Ontology** | A map of your data, showing what columns exist and how they relate |

---

## Other Guides

- [`docs/user-guide-it.md`](./user-guide-it.md) — Guida utente in italiano
- [`docs/api-reference.md`](./api-reference.md) — Full API reference for integrators
- [`docs/deployment-guide.md`](./deployment-guide.md) — How to install and run Aleph on your own servers
