# 🧠 Cortex Hands-On Tutorial: Give Your AI a Local Memory

> If you use AI to write code or analyze data, you've run into these problems. This tutorial skips the theory and goes straight to real scenarios — showing you exactly how Cortex solves them.

---

## 📖 Who This Tutorial Is For

| If you've ever felt this pain | This tutorial is for you |
|:-----------------------------|:------------------------|
| Asking your AI about project internals, getting "I can't access your local files" | ✅ |
| Copy-pasting code files to AI just to give it context | ✅ |
| Yesterday's architecture discussion — AI forgot everything today | ✅ |
| Project docs scattered everywhere, hard to find, AI can't find them either | ✅ |
| Asking the same question to AI multiple times because it doesn't remember | ✅ |
| Team members repeating the same mistakes because knowledge isn't shared | ✅ |

**In one sentence: Cortex gives your AI permanent memory and the ability to search all your local projects instantly.**

---

## Table of Contents

- [Scenario 1: AI Can't See Your Codebase](#scenario-1-ai-cant-see-your-codebase)
- [Scenario 2: AI Always Forgets Previous Discussions](#scenario-2-ai-always-forgets-previous-discussions)
- [Scenario 3: Project Documents Scattered Everywhere](#scenario-3-project-documents-scattered-everywhere)
- [Scenario 4: Repetitive Work — Same Questions Over and Over](#scenario-4-repetitive-work--same-questions-over-and-over)
- [Scenario 5: Team Collaboration — Knowledge Not Shared](#scenario-5-team-collaboration--knowledge-not-shared)
- [Scenario 6: Production — Turn Your Knowledge Base into a Service](#scenario-6-production--turn-your-knowledge-base-into-a-service)
- [Daily Workflow Summary](#daily-workflow-summary)
- [Appendix: Command Reference](#appendix-command-reference)

---

## Scenario 1: AI Can't See Your Codebase

### The Pain

You're in Cursor/Claude Code, and you ask:

> "How does the inventory replenishment flow work in our product module?"

AI responds:

> "I don't have access to your local project files. Please paste the relevant code..."

So you hunt through your codebase, find the right files, copy-paste them one by one. This happens a dozen times a day.

### The Solution

Index your project as an AI-searchable knowledge base with Cortex.

**Step 1: Get Cortex**

```bash
cortex --help
```

**Step 2: Index your project**

```bash
# Index your entire project
cortex index ~/projects/my-app

# Or index everything at once
cortex index ~/projects
```

**Step 3: Configure MCP for your AI client**

In your AI client's MCP settings, add:

| Field | Value |
|:------|:------|
| Name | `Cortex` |
| Command | `/usr/local/bin/cortex` |
| Args | `mcp` |
| Type | `stdio` |

**Step 4: Ask again**

> "How does the inventory replenishment flow work?"

This time, AI automatically calls `cortex_search`, finds the relevant code, and answers based on your actual project.

### The Result

```
Before: Hunt for files → Copy code → Paste to AI → Wait for answer
After: Just ask → AI auto-searches → Instant answer
Time saved: 30 min to 2 hours per day of "file搬运"
```

### Pro Tip

```bash
# Watch for file changes — auto-index on save
cortex watch ~/projects/my-app

# Now when you edit code, AI instantly sees the new content
# No manual re-indexing needed
```

---

## Scenario 2: AI Always Forgets Previous Discussions

### The Pain

Yesterday, you and AI spent an hour on architecture design:

```
You: Let's split the product module into three: Inventory, Pricing, Replenishment
AI: Great idea, that's a clean separation...
```

Today, new session:

```
You: Help me add a feature to the product module
AI: Sure, what does your product module look like?
You: (starting from scratch again...)
```

AI sessions are **stateless** — every new conversation is a clean slate. All those decisions, gone.

### The Solution

Use Cortex's memory system to persist important discussions.

**Method 1: Let AI auto-memorize**

Once MCP is configured, AI automatically calls `cortex_memory_write` when it detects important information. Just chat normally.

**Method 2: Manual memory writes**

```bash
# Save a technical decision
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{
    "content": "2026-05-13 Decision: Product module split into Inventory, Pricing, Replenishment — each independently deployable",
    "tags": ["architecture", "product-module", "decision"]
  }'

# Search memories later
curl -X GET "http://localhost:8080/v1/memory/search?q=product+module+split&top_k=5"
```

**Method 3: Turn project docs into memories**

```bash
# Index your docs directory — everything becomes searchable
cortex index ~/projects/my-app/docs
```

### The Result

```
Before: AI has amnesia every new session
After: AI "remembers" everything discussed — decisions, designs, API contracts
```

---

## Scenario 3: Project Documents Scattered Everywhere

### The Pain

Your project docs live in different places:

```
~/projects/my-app/README.md                — Project overview
~/projects/my-app/docs/architecture.md     — Architecture design
~/projects/my-app/docs/api.md              — API specs
~/Documents/learning/notes/deployment.md   — Deployment guide
~/Documents/project/faq.md                 — FAQs
```

Every time you need something:

> "Where did I write that? README? Some random doc?"

And AI can't find any of it — it doesn't know these files exist.

### The Solution

Index everything into Cortex and let AI do the searching.

```bash
# Index your whole workspace and document library
cortex index ~/projects
cortex index ~/Documents

# Check how many documents are indexed
cortex status
# Documents: 27142
```

Then just ask:

> "What's our deployment process? What environment variables do we need?"

AI searches across all your docs, finds the relevant content in `deployment.md`, and answers directly.

### The Result

```
Before: Guess folder → Guess filename → Open → Wrong → Try again
After: Just ask → AI finds it instantly across all documents
```

### Search Tips

```bash
# Narrow your search
cortex search "deployment environment variables" --top-k 5

# Raw RAG context (what AI uses to answer)
cortex context "deployment" --tokens 2000
```

---

## Scenario 4: Repetitive Work — Same Questions Over and Over

### The Pain

You constantly need to look up:

- "What's the storage capacity of the C5 cooler?"
- "What's the supplier data format?"
- "How is the replenishment threshold configured?"

Each time, you either dig through code/docs or ask AI (after pasting context). Same info, multiple times a week.

### The Solution

Pre-index everything so AI can answer instantly.

**Step 1: Index your project**

```bash
cortex index ~/projects/my-app
```

**Step 2: Use normally in your AI client**

With MCP configured, AI auto-searches. Just ask:

> "What's the C5 cooler capacity?"

AI does: `cortex_search("C5 cooler capacity")` → finds spec page → answers "180L".

**Step 3: Write a FAQ for common questions**

Create a `faq.md` for knowledge that doesn't have a home yet:

```markdown
# FAQ

## C5 Cooler
- Capacity: 180L
- Replenishment threshold: Trigger when stock < 10 units
- Supplier: Midea
```

Then index it:

```bash
cortex index ~/projects/my-app/docs/faq.md
```

### The Result

```
Before: Same info, 3 lookups per week, 5 minutes each
After: Ask AI, 3 seconds, 100% accuracy
Weekly saved: 25 minutes
```

---

## Scenario 5: Team Collaboration — Knowledge Not Shared

### The Pain

You're the expert on a particular module. Coworkers keep asking:

> "What's the API response format?"
> "What does this config option mean?"

You answer every time. Everyone pays the same "cognitive tax" repeatedly.

Worse — the bugs someone already fixed, someone else will hit again.

### The Solution

Build a shared knowledge base that everyone can search.

**Method 1: Shared directory**

Put docs on a network share, everyone indexes locally:

```bash
# Each team member runs:
cortex index \\server\project-docs
```

**Method 2: Centralized knowledge service**

```bash
# On a server:
cortex serve

# Team members query via API:
curl -X POST http://team-server:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query":"product API response format","top_k":5}'
```

**Method 3: Log your war stories**

```bash
# Hit a bug? Log the fix so AI warns others
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Gotcha: CSV import fails if encoding is not UTF-8 with BOM. Fix: add BOM detection at the start of import script.",
    "tags": ["gotcha", "csv", "import"]
  }'

# Next time someone hits the same issue, AI surfaces this memory
```

### The Result

```
Before: N people step into the same hole, each debugging from scratch
After: Someone hits a bug → logs it → AI auto-reminds → no one steps twice
```

---

## Scenario 6: Production — Turn Your Knowledge Base into a Service

### The Pain

Your local knowledge base is working great. But you want to:

- Access it from other computers
- Call it from automation scripts
- Monitor usage statistics

### The Solution

Run Cortex as a service.

**Start the REST API server:**

```bash
cortex serve

# Output:
# [INFO] starting REST API server  :8080
# [INFO] metrics server started     :9090
```

Now call it from anywhere:

```bash
# From any app
curl -X POST http://localhost:8080/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query":"replenishment flow","top_k":3}'

# From an automation script
python -c "
import requests
r = requests.post('http://localhost:8080/v1/search',
    json={'query': 'pending orders today', 'top_k': 10})
for result in r.json()['results']:
    print(result['chunk']['content'])
"
```

**Enable auto-backup:**

```yaml
# config.yaml
backup:
  auto_backup: true       # Daily auto backup
  max_backups: 30          # Keep 30 backups
```

**Monitor usage:**

```bash
# Prometheus metrics
curl http://localhost:9090/metrics | grep cortex
# cortex_search_total 1234     — Total searches
# cortex_index_total 5678      — Total files indexed
```

### The Result

```
Before: Knowledge base locked to one machine
After: Knowledge base becomes a service — any device, any script can call it
```

---

## Daily Workflow Summary

### 🟢 Day 1: Setup

```bash
# Index everything
cortex index ~/projects
cortex index ~/Documents

# Configure MCP in your AI client (one-time)
# Settings → MCP Servers → Add cortex

# Done. Start using it.
```

### 🟡 Daily

```bash
# Watch for changes while you work
cortex watch ~/projects

# Just code and ask questions normally
# AI auto-searches knowledge base, auto-memorizes important info
```

### 🔴 Weekly

```bash
# Deduplicate to keep knowledge base lean
cortex dedup

# Check knowledge base health
cortex status
```

### 🔵 When You Hit a Wall

```bash
# Log it to memory system (AI can do this for you too)
curl -X POST http://localhost:8080/v1/memory \
  -H "Content-Type: application/json" \
  -d '{"content":"Fix for bug X...","tags":["gotcha","module-name"]}'
```

---

## Appendix: Command Reference

### Indexing

| Command | What it does |
|:--------|:-------------|
| `cortex index <path>` | Index a directory |
| `cortex index --force <path>` | Force full re-index |
| `cortex watch <path>` | Watch directory, auto-index on change |
| `cortex index --workers 32` | Speed up (default 16) |

### Search

| Command | What it does |
|:--------|:-------------|
| `cortex search <query>` | Search knowledge base |
| `cortex search <query> --top-k 20` | Specify result count |
| `cortex search <query> --json` | JSON output |
| `cortex context <query>` | Get RAG context |

### Service

| Command | What it does |
|:--------|:-------------|
| `cortex mcp` | Start MCP server (for AI clients) |
| `cortex serve` | Start REST API (for programs) |
| `cortex serve` | Auto hot-reload config |

### Maintenance

| Command | What it does |
|:--------|:-------------|
| `cortex status` | View knowledge base status |
| `cortex usage` | View storage usage |
| `cortex dedup` | Deduplicate |
| `cortex setup` | Configuration wizard |
| `cortex version` | Version info |

---

> Based on Cortex v3.3
> GitHub: [https://github.com/lh123aa/cortex](https://github.com/lh123aa/cortex)
