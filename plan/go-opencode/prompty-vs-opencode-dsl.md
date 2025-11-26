# Comparative Analysis: OpenCode Agentic Workflow DSL vs Microsoft Prompty

## Executive Summary

This document provides a comprehensive comparison between OpenCode's Agentic Workflow DSL and Microsoft's Prompty framework, analyzing their design philosophies, capabilities, and optimal use cases.

**Key Finding**: These are fundamentally different tools addressing different layers of the LLM application stack:
- **Prompty** = Single LLM invocation abstraction (prompt definition + model configuration)
- **OpenCode DSL** = Multi-step workflow orchestration (multi-agent coordination + flow control)

**Recommendation**: They are **complementary, not competing**. Prompty could potentially serve as the prompt definition layer within OpenCode's workflow steps.

---

## 1. Scope and Design Philosophy

### 1.1 OpenCode Agentic Workflow DSL

**Purpose**: Orchestrate complex multi-agent workflows with human-in-the-loop capabilities.

**Design Philosophy**:
- **Configuration-as-code**: Declarative YAML/Markdown defining workflow topology
- **Multi-agent coordination**: Define and orchestrate multiple specialized agents
- **Human-centric**: Built-in pause/approval steps for human oversight
- **Flow control**: Sequential, parallel, conditional, and loop execution patterns
- **LLM-based decisions**: Natural language conditions evaluated by LLM

**Primary Abstraction**: A *workflow* containing multiple *steps* executed by *agents*.

### 1.2 Microsoft Prompty

**Purpose**: Standardize LLM prompt definition across languages and frameworks.

**Design Philosophy**:
- **Unified asset class**: Single file format for prompt + configuration + metadata
- **Micro-orchestrator**: Focused on single LLM invocation excellence
- **Developer inner loop**: Rapid iteration with VS Code tooling
- **Framework agnostic**: Works with LangChain, Semantic Kernel, Prompt Flow
- **Cross-language**: Python, C#, JavaScript runtimes

**Primary Abstraction**: A *prompty file* containing a *templated prompt* with *model configuration*.

---

## 2. Feature Comparison Matrix

| Feature | OpenCode DSL | Prompty |
|---------|--------------|---------|
| **Core Focus** | Multi-step workflow orchestration | Single prompt definition |
| **File Format** | YAML frontmatter + Markdown | YAML frontmatter + Markdown/Jinja2 |
| **Multi-Agent Support** | Native (multiple agents per workflow) | No (single prompt per file) |
| **Sequential Steps** | `dependsOn` declarations | External (via frameworks) |
| **Parallel Execution** | `type: parallel` step | External (via frameworks) |
| **Conditional Branching** | `type: conditional` step | External (via frameworks) |
| **Loops** | `type: loop` step | External (via frameworks) |
| **Human Review/Pause** | `type: pause` step | Not supported |
| **LLM Condition Evaluation** | `conditionType: llm` | Not supported |
| **Template Engine** | `{{variable}}` interpolation | Jinja2 / Mustache |
| **Input/Output Schema** | Zod validation | JSON Schema |
| **Tool/Function Calling** | Per-agent tool permissions | Model parameters |
| **Model Configuration** | Per-agent model settings | Per-file model settings |
| **Streaming** | Limited | Full support |
| **Type Safety** | Runtime (Zod) | Runtime (per-language) |
| **Language Support** | TypeScript | Python, C#, JavaScript |
| **VS Code Extension** | No dedicated extension | Full IDE support |
| **Tracing/Observability** | Workflow events | OpenTelemetry integration |
| **Framework Integration** | OpenCode-native | LangChain, Semantic Kernel, Prompt Flow |

---

## 3. Syntax Comparison

### 3.1 File Structure

**OpenCode DSL** (workflow definition):
```yaml
---
id: code-review
name: Code Review Workflow
description: Multi-agent code review
version: 1.0.0
inputs:
  files:
    type: string
    description: Files to review
    required: true
orchestrator:
  mode: guided
  onError: pause
  maxRetries: 2
tags:
  - code-review
---

# Agent: analyzer

You are a code analyzer specializing in identifying issues...

```yaml
tools:
  read: true
  glob: true
  edit: false
model: anthropic/claude-sonnet-4-20250514
temperature: 0.3
```

# Agent: reviewer

You are a senior code reviewer...

# Steps

```yaml
- id: analyze
  type: agent
  agent: analyzer
  input: "Analyze {{files}}"
  output: analysis_result

- id: review
  type: agent
  agent: reviewer
  dependsOn: [analyze]
  input: "Review {{analysis_result}}"
  output: review_result

- id: human_review
  type: pause
  message: "Review before proceeding?"
  dependsOn: [review]
```
```

**Prompty** (single prompt definition):
```yaml
---
name: Code Analyzer
description: Analyze code for issues
metadata:
  authors: [team]
model:
  api: chat
  configuration:
    type: azure_openai
    azure_deployment: gpt-4
  parameters:
    temperature: 0.3
inputs:
  files:
    type: string
    description: Files to analyze
sample:
  files: "src/**/*.ts"
---

system:
You are a code analyzer specializing in identifying issues in source code.

When given files to analyze, you should:
1. Identify potential bugs and security issues
2. Check for performance concerns
3. Note best practice violations

# Files to Analyze
{% for file in files %}
- {{file}}
{% endfor %}

user:
Please analyze these files and provide a structured report.
```

### 3.2 Key Syntax Differences

| Aspect | OpenCode DSL | Prompty |
|--------|--------------|---------|
| **Agent Definition** | Markdown sections with YAML config | N/A (single prompt) |
| **Steps** | YAML array with types | N/A (single invocation) |
| **Variables** | `{{variable}}` | `{{variable}}` or `{% %}` (Jinja2) |
| **Conditionals** | Step-level conditional type | Jinja2 `{% if %}` (template-only) |
| **Loops** | Step-level loop type | Jinja2 `{% for %}` (template-only) |
| **Model Config** | Per-agent or orchestrator | Top-level `model:` section |
| **Tools** | Boolean map per agent | OpenAI function calling format |

---

## 4. What OpenCode DSL Has That Prompty Lacks

### 4.1 Multi-Step Workflow Orchestration

OpenCode can define complex execution flows:

```yaml
steps:
  - id: research
    type: agent
    agent: researcher

  - id: parallel_analysis
    type: parallel
    dependsOn: [research]
    steps: [security_check, performance_check, style_check]

  - id: decide
    type: conditional
    condition: "Are there critical issues?"
    conditionType: llm
    then: fix_issues
    else: approve

  - id: iterate
    type: loop
    until: "Is the code production ready?"
    conditionType: llm
    maxIterations: 3
    steps: [refine, validate]
```

Prompty has no native workflow capability - requires external frameworks.

### 4.2 Human-in-the-Loop Pauses

```yaml
- id: human_review
  type: pause
  message: "Review analysis before proceeding?"
  approvalVariable: approved
  options:
    allowEdit: true
    allowReject: true
    autoApproveAfter: 300000  # 5 min timeout
```

Prompty has no concept of execution pause/resume.

### 4.3 LLM-Based Condition Evaluation

```yaml
- id: quality_gate
  type: conditional
  condition: "Based on the review, is this code ready for production?"
  conditionType: llm
  then: deploy
  else: refine
```

Natural language conditions evaluated by LLM - unique to OpenCode.

### 4.4 Multi-Agent Coordination

OpenCode supports multiple agents with different capabilities:

```yaml
agents:
  analyzer:
    prompt: "You analyze code..."
    tools: { read: true, grep: true }
    temperature: 0.3

  fixer:
    prompt: "You fix issues..."
    tools: { read: true, edit: true }
    temperature: 0.2

  reviewer:
    prompt: "You review changes..."
    tools: { read: true }
    temperature: 0.5
```

Prompty is inherently single-prompt focused.

### 4.5 Orchestrator Configuration

```yaml
orchestrator:
  mode: guided       # auto | guided | manual
  onError: pause     # pause | retry | fail | skip
  maxRetries: 3
  defaultTimeout: 300000
  hooks:
    onStart: notify_start
    onComplete: notify_complete
```

Global workflow behavior configuration not available in Prompty.

---

## 5. What Prompty Has That OpenCode DSL Could Adopt

### 5.1 Sophisticated Template Engine

Prompty's Jinja2 support enables:

```jinja2
system:
{% if is_premium_user %}
You are a premium support agent with full access.
{% else %}
You are a standard support agent.
{% endif %}

# Context
{% for doc in documents %}
## {{doc.title}}
{{doc.content}}
---
{% endfor %}

{% macro format_error(error) %}
Error Code: {{error.code}}
Message: {{error.message}}
{% endmacro %}

{{ format_error(last_error) }}
```

OpenCode uses simple `{{variable}}` interpolation only.

### 5.2 Cross-Language Runtimes

Prompty supports:
- **Python**: `pip install prompty[azure]`
- **C#**: `Prompty.Core` NuGet package
- **JavaScript**: `@prompty/runtime` npm package

OpenCode DSL is currently TypeScript-only.

### 5.3 VS Code Extension & Developer Experience

Prompty offers:
- Interactive prompt playground
- F5 run with verbose output
- Trace viewer for debugging
- Quick file templates
- Live variable rendering
- Code generation for frameworks

OpenCode has no dedicated IDE tooling.

### 5.4 Framework Integration

Prompty integrates with:

**LangChain**:
```python
from langchain_prompty import create_chat_prompt
prompt = create_chat_prompt("./analyzer.prompty")
chain = prompt | model | parser
```

**Semantic Kernel**:
```csharp
var function = kernel.CreateFunctionFromPromptyFile("analyzer.prompty");
await kernel.InvokeAsync(function, args);
```

**Prompt Flow**:
- Native visual workflow designer support
- Evaluation and tracing built-in

### 5.5 Streaming Support

Prompty has comprehensive streaming:
```yaml
model:
  parameters:
    stream: true
    stream_options:
      include_usage: true
```

With async iterator support in all runtimes.

### 5.6 Input/Output Schema Validation

```yaml
inputs:
  customer:
    type: object
    properties:
      name: { type: string }
      tier: { type: string, enum: [basic, premium] }
    required: [name, tier]

outputs:
  response:
    type: string
    description: The formatted response
```

More expressive than OpenCode's type-only validation.

### 5.7 Observability via OpenTelemetry

```python
from opentelemetry import trace
from prompty.tracer import Tracer

Tracer.add("OpenTelemetry", otel_span_generator)
```

Built-in tracing infrastructure with industry-standard integration.

---

## 6. Architectural Positioning

```
┌─────────────────────────────────────────────────────────────────┐
│                     Application Layer                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │           OpenCode Agentic Workflow DSL                  │    │
│  │  (Multi-step orchestration, agents, flow control)        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Prompty Layer                         │    │
│  │      (Single prompt definition, templating, config)      │    │
│  │   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │    │
│  │   │analyzer │  │reviewer │  │ fixer   │  │evaluator│    │    │
│  │   │.prompty │  │.prompty │  │.prompty │  │.prompty │    │    │
│  │   └─────────┘  └─────────┘  └─────────┘  └─────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Model Provider Layer                   │    │
│  │   (OpenAI, Azure OpenAI, Claude, Gemini, etc.)          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Position**:
- Prompty sits at the **prompt definition layer** (lowest abstraction)
- OpenCode DSL sits at the **workflow orchestration layer** (higher abstraction)
- They could be combined: OpenCode agents could use Prompty files for their prompts

---

## 7. Pros and Cons Summary

### 7.1 OpenCode Agentic Workflow DSL

| Pros | Cons |
|------|------|
| Native multi-agent orchestration | TypeScript-only runtime |
| Human-in-the-loop pause/approval | No dedicated IDE tooling |
| LLM-based condition evaluation | Simple template interpolation |
| Parallel and loop execution | Limited streaming support |
| Workflow state persistence | No cross-framework integration |
| Declarative YAML/Markdown format | Less mature ecosystem |
| Fine-grained tool permissions | No OpenTelemetry integration |

### 7.2 Microsoft Prompty

| Pros | Cons |
|------|------|
| Cross-language support (Python, C#, JS) | Single invocation focus only |
| Rich VS Code extension | No workflow orchestration |
| Powerful Jinja2 templating | No human-in-the-loop support |
| Framework integrations (LangChain, SK) | No multi-agent coordination |
| OpenTelemetry tracing | No LLM-based conditions |
| Streaming support | Requires external orchestration |
| Strong schema validation | No built-in parallelism |
| Large community adoption | No pause/resume capability |

---

## 8. Use Case Recommendations

### When to Use OpenCode Agentic Workflow DSL

1. **Multi-agent coordination**: Tasks requiring multiple specialized agents working together
2. **Complex workflows**: Sequential, parallel, and conditional execution patterns
3. **Human oversight**: Approval gates and review checkpoints
4. **Iterative refinement**: Loop-based processing with LLM-evaluated exit conditions
5. **Long-running tasks**: Workflows that may span multiple sessions
6. **Code-centric operations**: Development workflows with tool permissions

**Example Use Cases**:
- Multi-stage code review and fix workflows
- Document processing pipelines with human approval
- Research → Analysis → Implementation workflows
- Quality assurance with iterative refinement

### When to Use Microsoft Prompty

1. **Rapid prototyping**: Quick iteration on individual prompts
2. **Framework integration**: Building on LangChain, Semantic Kernel
3. **Cross-language projects**: Teams using multiple languages
4. **Prompt management**: Version control and standardization
5. **Single-shot tasks**: One-off LLM invocations
6. **IDE-centric development**: Leveraging VS Code tooling

**Example Use Cases**:
- Customer service response generation
- Content summarization
- Code explanation/documentation
- RAG query processing
- Evaluation and scoring prompts

---

## 9. Integration Opportunities

### 9.1 Prompty as Agent Prompt Source

OpenCode agents could load their prompts from Prompty files:

```yaml
# workflow.md
---
id: code-review
agents:
  analyzer:
    promptFile: ./prompts/analyzer.prompty  # Load from Prompty
    tools: { read: true, glob: true }

  reviewer:
    promptFile: ./prompts/reviewer.prompty
    tools: { read: true }
---
```

**Benefits**:
- Leverage Prompty's templating (Jinja2)
- Use Prompty's VS Code tooling for prompt development
- Share prompts across projects
- Separate prompt definition from workflow orchestration

### 9.2 Unified File Format

Create a hybrid format supporting both:

```yaml
---
# Prompty-compatible header
name: Code Analyzer Agent
model:
  api: chat
  configuration:
    type: azure_openai
    azure_deployment: gpt-4
  parameters:
    temperature: 0.3
inputs:
  files: { type: string }

# OpenCode extensions
opencode:
  workflow:
    tools: { read: true, glob: true }
    permission: { edit: false }
---

system:
You are a code analyzer...

user:
Analyze: {{files}}
```

### 9.3 Feature Adoption

| Feature | Adopt From | Into |
|---------|------------|------|
| Jinja2 templating | Prompty | OpenCode |
| OpenTelemetry tracing | Prompty | OpenCode |
| VS Code extension | Prompty | OpenCode |
| Cross-language runtimes | Prompty | OpenCode |
| Human-in-the-loop | OpenCode | Prompty (via plugins) |
| LLM conditions | OpenCode | Prompty (via plugins) |
| Multi-agent | OpenCode | N/A (out of scope) |

---

## 10. Conclusion

### Summary of Key Differences

| Aspect | OpenCode DSL | Prompty |
|--------|--------------|---------|
| **Scope** | Workflow orchestration | Prompt definition |
| **Abstraction Level** | High (multi-step flows) | Low (single invocation) |
| **Primary Value** | Multi-agent coordination | Cross-platform standardization |
| **Unique Strength** | Human-in-the-loop | Framework ecosystem |
| **Best For** | Complex agentic workflows | Prompt management/sharing |

### Recommendation

**Use Together**:
- Prompty for **prompt definition and rapid iteration**
- OpenCode DSL for **workflow orchestration and multi-agent coordination**

**For Go-OpenCode Development**:
1. Consider adopting Prompty's templating approach (Jinja2)
2. Add OpenTelemetry integration inspired by Prompty
3. Explore loading Prompty files as agent prompt sources
4. Maintain unique workflow features (pause, LLM conditions, multi-agent)

**Bottom Line**: Prompty excels at the "what to say to the LLM" problem. OpenCode DSL excels at the "how to coordinate multiple LLM interactions" problem. A mature agentic platform could benefit from both.

---

## Sources

- [Microsoft Prompty Repository](https://github.com/microsoft/prompty)
- [Prompty Specification (Prompty.yaml)](https://github.com/microsoft/prompty/blob/main/Prompty.yaml)
- [OpenCode Workflow Schema](../packages/opencode/src/workflow/schema.ts)
- [OpenCode Workflow DSL vs Eino Evaluation](./workflow-dsl.md)
- Local code review: `prompty/runtime/prompty/**/*.py`, `opencode/src/workflow/**/*.ts`
