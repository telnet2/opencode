# Workflow DSL Evaluation: OpenCode vs Eino Framework

## Executive Summary

This document compares the OpenCode Agentic Workflow DSL with CloudWeGo's [Eino framework](https://github.com/cloudwego/eino) to evaluate architectural similarities, differences, and potential integration opportunities.

**Key Finding**: Eino uses a **programmatic Go-based DSL** (code-as-configuration) while OpenCode uses a **declarative YAML/Markdown DSL** (configuration-as-code). These are complementary approaches that could be integrated.

---

## 1. CloudWeGo Eino Framework Overview

### 1.1 What is Eino?

[Eino](https://github.com/cloudwego/eino) ("I know") is ByteDance's open-source LLM application development framework for Go, inspired by LangChain and LlamaIndex. It provides:

- **Component abstractions**: ChatModel, Tool, Retriever, Embedder, etc.
- **Orchestration APIs**: Chain, Graph, Workflow
- **Stream processing**: Automatic concatenation, boxing, merging
- **Extensions** via [eino-ext](https://github.com/cloudwego/eino-ext): OpenAI, Claude, Gemini integrations

### 1.2 Eino's Orchestration APIs

| API | Description | Mode |
|-----|-------------|------|
| **Chain** | Linear sequential composition | Pregel (cyclic) |
| **Graph** | Flexible directed graphs (cyclic/acyclic) | Pregel or DAG |
| **Workflow** | Struct field-level data mapping | DAG only |

### 1.3 Eino Code Examples

**Chain (Sequential)**:
```go
chain, _ := NewChain[map[string]any, *Message]().
    AppendChatTemplate(prompt).
    AppendChatModel(model).
    Compile(ctx)
```

**Graph (Branching)**:
```go
graph := NewGraph[map[string]any, *schema.Message]()
graph.AddChatModelNode("node_model", chatModel)
graph.AddToolsNode("node_tools", toolsNode)
graph.AddBranch("node_model", branch)  // Conditional routing
graph.AddEdge("node_tools", END)
```

**Workflow (Field Mapping)**:
```go
wf := NewWorkflow[*schema.Message, *schema.Message]()
wf.AddChatModelNode("model", m).AddInput(START)
wf.AddLambdaNode("lambda1", lambda1).
    AddInput("model", MapFields("Content", "Input"))
```

---

## 2. Comparison: OpenCode vs Eino DSL

### 2.1 DSL Approach

| Aspect | OpenCode DSL | Eino DSL |
|--------|--------------|----------|
| **Style** | Declarative YAML/Markdown | Programmatic Go code |
| **Configuration** | External files (`.opencode/workflow/*.md`) | Inline code |
| **Type Safety** | Runtime validation (Zod) | Compile-time (Go generics) |
| **Human Readability** | High (YAML + natural language) | Moderate (Go code) |
| **Version Control** | File-based, easy to diff | Code changes |

### 2.2 Feature Comparison

| Feature | OpenCode | Eino |
|---------|----------|------|
| **Sequential Steps** | ✅ `dependsOn` | ✅ Chain API |
| **Parallel Execution** | ✅ `type: parallel` | ✅ `AppendParallel()` |
| **Conditional Branching** | ✅ `type: conditional` | ✅ `AddBranch()` |
| **Loops** | ✅ `type: loop` with while/until | ⚠️ Graph cycles (Pregel mode) |
| **Human Review Pauses** | ✅ `type: pause` | ❌ Not built-in |
| **LLM Condition Evaluation** | ✅ `conditionType: llm` | ❌ Manual implementation |
| **Field-Level Mapping** | ✅ `{{variable}}` interpolation | ✅ `MapFields()` |
| **Stream Processing** | ⚠️ Limited | ✅ Comprehensive |
| **Multi-Agent** | ✅ Workflow-defined agents | ✅ Host Multi-Agent |
| **Checkpointing** | ✅ Workflow instance state | ✅ `checkpoint.go` |

### 2.3 Agent Definition

**OpenCode** (Declarative):
```yaml
agents:
  analyzer:
    prompt: "You are a code analyzer..."
    tools:
      read: true
      glob: true
    temperature: 0.3
```

**Eino** (Programmatic):
```go
agent, _ := react.NewAgent(ctx, &react.AgentConfig{
    Model: chatModel,
    ToolsConfig: &react.ToolsNodeConfig{
        Tools: []tool.BaseTool{searchTool, calcTool},
    },
})
```

### 2.4 Workflow Steps

**OpenCode** (YAML):
```yaml
steps:
  - id: analyze
    type: agent
    agent: analyzer
    input: "Analyze {{files}}"
    output: analysis

  - id: decide
    type: conditional
    condition: "Does the analysis indicate critical issues?"
    conditionType: llm
    then: fix_issues
    else: approve
```

**Eino** (Go Code):
```go
graph.AddChatModelNode("analyze", analyzer)
graph.AddBranch("analyze", compose.NewStreamGraphBranch(
    func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (string, error) {
        // Custom branching logic
        return nextNode, nil
    },
    map[string]bool{"fix_issues": true, "approve": true},
))
```

---

## 3. What OpenCode Has That Eino Lacks

### 3.1 Declarative Human Review Pauses

OpenCode's `pause` step type is unique:
```yaml
- id: human_review
  type: pause
  message: "Review before proceeding"
  approvalVariable: approved
  options:
    allowEdit: true
    autoApproveAfter: 300000  # 5 min timeout
```

Eino has `interrupt.go` for runtime control but no declarative pause-for-approval pattern.

### 3.2 LLM-Based Condition Evaluation

OpenCode allows natural language conditions evaluated by LLM:
```yaml
- id: quality_check
  type: conditional
  condition: "Is the code production-ready based on: {{review}}?"
  conditionType: llm
  then: deploy
  else: refine
```

Eino requires custom implementation of such logic.

### 3.3 `llm_eval` Step Type

Dedicated step for LLM-based decisions:
```yaml
- id: categorize
  type: llm_eval
  prompt: "Categorize this issue: {{issue}}"
  outputFormat: choice
  choices: [bug, feature, enhancement]
  output: category
```

### 3.4 File-Based Workflow Definitions

Workflows stored as `.md` files with YAML frontmatter enable:
- Version control friendly
- Easy sharing between projects
- Non-programmer accessible
- Template-based reuse

---

## 4. What Eino Has That OpenCode Could Adopt

### 4.1 Compile-Time Type Safety

Eino's Go generics provide compile-time validation:
```go
// Type mismatch caught at compile time
graph := NewGraph[Input, Output]()
graph.AddNode("n1", nodeWithWrongType) // Compile error!
```

OpenCode could benefit from stricter schema validation.

### 4.2 Sophisticated Stream Processing

Eino's four streaming paradigms:
- **Invoke**: non-stream → non-stream
- **Stream**: non-stream → streaming output
- **Collect**: streaming input → non-stream
- **Transform**: streaming → streaming

OpenCode's workflow steps are primarily invoke-style.

### 4.3 Pregel Execution Model

Eino supports cyclic graphs with Pregel mode for iterative convergence, useful for:
- Multi-round refinement without explicit loop definitions
- Message-passing between agents
- Convergence-based termination

### 4.4 Field-Level Data Mapping

Eino's Workflow API offers precise struct field routing:
```go
wf.AddLambdaNode("process", fn).
    AddInput("upstream", MapFields("Response.Content", "Input.Text"))
```

OpenCode uses string interpolation (`{{var}}`), less type-safe.

---

## 5. Integration Opportunities

### 5.1 Eino as Execution Backend for OpenCode DSL

**Concept**: Parse OpenCode's YAML DSL and generate Eino Graph/Workflow code.

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│ workflow.yaml   │ ──▶ │ DSL Parser   │ ──▶ │ Eino Graph  │
│ (OpenCode DSL)  │     │ (Go/TS)      │     │ Execution   │
└─────────────────┘     └──────────────┘     └─────────────┘
```

**Benefits**:
- Declarative user experience (OpenCode)
- Type-safe execution (Eino)
- Stream processing capabilities (Eino)
- Human review pauses (OpenCode)

### 5.2 Go-Native OpenCode Implementation

If building a Go version of OpenCode:

1. **Use Eino's component abstractions**: ChatModel, Tool, Retriever
2. **Add declarative DSL layer**: YAML parser generating Eino graphs
3. **Implement pause/resume**: Add workflow checkpointing
4. **Add LLM evaluation**: Wrap condition evaluation in Eino nodes

### 5.3 Hybrid Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    OpenCode CLI/Server                   │
├─────────────────────────────────────────────────────────┤
│  Workflow DSL Parser (YAML/MD → Internal Representation) │
├─────────────────────────────────────────────────────────┤
│              Execution Engine Abstraction                │
├───────────────────────┬─────────────────────────────────┤
│  TypeScript Executor  │       Go Executor (Eino)        │
│  (Current OpenCode)   │       (High-performance)        │
└───────────────────────┴─────────────────────────────────┘
```

---

## 6. Recommendations

### 6.1 For OpenCode Enhancement

1. **Add stream processing support** to workflow steps (inspired by Eino)
2. **Consider Pregel-style cycles** for complex iterative patterns
3. **Strengthen type validation** at compile/parse time
4. **Add field-path mapping** syntax: `{{step.output.field}}`

### 6.2 For Go-OpenCode Development

1. **Leverage Eino's component layer** for model/tool integrations
2. **Build DSL parser** that generates Eino Graph structures
3. **Extend Eino** with:
   - Pause/resume workflow state
   - LLM condition evaluation nodes
   - YAML/Markdown workflow loading
4. **Contribute upstream** human-review patterns to Eino

### 6.3 Proposed Architecture for Go-OpenCode

```go
// workflow/dsl.go - Parse OpenCode YAML to Eino Graph
type WorkflowDSL struct {
    ID          string
    Steps       []StepDSL
    Agents      map[string]AgentDSL
    Orchestrator OrchestratorDSL
}

func (w *WorkflowDSL) ToEinoGraph(ctx context.Context) (*compose.Graph, error) {
    graph := compose.NewGraph[WorkflowInput, WorkflowOutput]()

    for _, step := range w.Steps {
        switch step.Type {
        case "agent":
            graph.AddChatModelNode(step.ID, w.resolveAgent(step.Agent))
        case "pause":
            graph.AddLambdaNode(step.ID, w.createPauseNode(step))
        case "conditional":
            graph.AddBranch(step.ID, w.createBranch(step))
        case "parallel":
            // Eino AppendParallel equivalent
        case "llm_eval":
            graph.AddChatModelNode(step.ID, w.createEvalNode(step))
        }
    }
    return graph, nil
}
```

---

## 7. Conclusion

| Aspect | Verdict |
|--------|---------|
| **DSL Similarity** | Different paradigms: Declarative (OpenCode) vs Programmatic (Eino) |
| **Feature Overlap** | ~70% overlap in core capabilities |
| **Unique to OpenCode** | Human review pauses, LLM conditions, declarative agents |
| **Unique to Eino** | Stream processing, compile-time safety, Pregel mode |
| **Integration Path** | OpenCode DSL → Eino execution backend is feasible |
| **Recommendation** | Use Eino components, add OpenCode's DSL layer on top |

**Bottom Line**: Eino and OpenCode's workflow DSL are **complementary, not competing**. Eino provides robust Go infrastructure for LLM orchestration, while OpenCode provides a user-friendly declarative DSL. A Go-native OpenCode could use Eino as its execution engine while preserving the declarative YAML/Markdown workflow definitions.

---

## 8. Code-Level Analysis (Source Review)

This section documents findings from reviewing the actual Eino source code (cloned locally).

### 8.1 Graph Implementation (`compose/graph.go`)

Key patterns discovered:

**Type-Safe Node Registration**:
```go
// Nodes are type-checked via Go generics
func (g *Graph[I, O]) AddChatModelNode(key string, chatModel model.BaseChatModel, opts ...GraphAddNodeOpt) error
func (g *Graph[I, O]) AddToolsNode(key string, node *ToolsNode, opts ...GraphAddNodeOpt) error
func (g *Graph[I, O]) AddLambdaNode(key string, lambda any, opts ...GraphAddNodeOpt) error
```

**Dual Execution Modes**:
- **Pregel mode** (`WithNodeTriggerMode(AnyPredecessor)`): Supports cycles, iterative execution
- **DAG mode**: Acyclic-only, topological ordering

**Branch Types**:
- `NewGraphBranch()` - synchronous routing decision
- `NewStreamGraphBranch()` - stream-aware routing
- `NewGraphMultiBranch()` - fan-out to multiple nodes

**State Handlers**:
```go
// Pre-handlers can read/modify state before node execution
compose.WithStatePreHandler(func(ctx context.Context, input I, state *S) (I, error) {
    // Modify input based on state
    return modifiedInput, nil
})
```

### 8.2 Chain Implementation (`compose/chain.go`)

The Chain API provides a fluent builder pattern:

```go
chain := NewChain[I, O]()
chain.AppendChatModel(model)
chain.AppendChatTemplate(template)
chain.AppendParallel(branch1, branch2, branch3)  // Parallel execution
chain.Compile(ctx)
```

**Key Insight**: Chain internally builds a Graph - it's syntactic sugar for linear workflows.

### 8.3 Workflow Implementation (`compose/workflow.go`)

Workflow adds field-level data mapping via `AddInput()`:

```go
wf := NewWorkflow[I, O]()
wf.AddChatModelNode("model", m).AddInput(START)
wf.AddLambdaNode("transform", fn).
    AddInput("model", MapFields("Content", "InputField"))  // Field mapping!
wf.End().AddInput("transform")
```

**Field Mapping**: `MapFields(srcField, dstField)` enables routing specific struct fields between nodes.

### 8.4 ReAct Agent Implementation (`flow/agent/react/react.go`)

The ReAct agent demonstrates complex graph orchestration:

```
          ┌──────────────────────────────────────┐
          │                                      │
          ▼                                      │
[START] → [ChatModel] ──branch──→ [ToolsNode] ──┘
                │                     │
                │                     ▼
                │              [direct_return] ──→ [END]
                │
                └─────────────────────────────────→ [END]
```

**Key Implementation Details**:

1. **State Management**:
```go
type state struct {
    Messages                 []*schema.Message
    ReturnDirectlyToolCallID string
}
```

2. **Tool Call Detection** (branching logic):
```go
modelPostBranchCondition := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (string, error) {
    if isToolCall, err := toolCallChecker(ctx, sr); err != nil {
        return "", err
    } else if isToolCall {
        return nodeKeyTools, nil  // Route to tools
    }
    return compose.END, nil  // Direct output
}
```

3. **Return Directly Pattern**:
```go
// Tools can signal early termination via SetReturnDirectly()
func SetReturnDirectly(ctx context.Context) error {
    return compose.ProcessState(ctx, func(ctx context.Context, s *state) error {
        s.ReturnDirectlyToolCallID = compose.GetToolCallID(ctx)
        return nil
    })
}
```

### 8.5 Multi-Agent Host Pattern (`flow/agent/multiagent/host/`)

The Host multi-agent pattern enables orchestration where a "host" agent delegates to specialist agents:

```go
type MultiAgentConfig struct {
    Host        Host          // Central coordinator
    Specialists []*Specialist // Domain experts
    Summarizer  *Summarizer   // Optional result aggregator
}

type Specialist struct {
    AgentMeta    // Name, IntendedUse
    ChatModel    model.BaseChatModel
    SystemPrompt string
    Invokable    compose.Invoke[...]  // Or custom lambda
    Streamable   compose.Stream[...]
}
```

**Graph Structure**:
```
[START] → [Host] ──branch──→ [Specialist1] ──┐
                │             [Specialist2] ──┼──→ [Collector] ──→ [Summarizer?] → [END]
                │             [Specialist3] ──┘
                │
                └─────────────────────────────────────────────────────────────────→ [END]
```

**Key Insight**: Specialists are exposed as "tools" to the Host agent, enabling tool-call routing:
```go
agentTools = append(agentTools, &schema.ToolInfo{
    Name: specialist.Name,
    Desc: specialist.IntendedUse,  // Host uses this to decide delegation
})
```

### 8.6 Eino-ext Components

**MCP Integration** (`components/tool/mcp/`):
- Wraps MCP servers as Eino tools
- Supports SSE and stdio transports
- Example: `mcpp.GetTools(ctx, &mcpp.Config{Cli: mcpClient})`

**Sequential Thinking Tool** (`components/tool/sequentialthinking/`):
- Implements chain-of-thought reasoning
- Step-by-step problem decomposition
- Similar concept to OpenCode's `llm_eval` step type

**Model Integrations**:
- OpenAI, Claude, Gemini, Ollama, Qwen, DeepSeek
- Each implements `model.BaseChatModel` interface

---

## 9. Architectural Mapping: OpenCode DSL → Eino

Based on code review, here's how OpenCode DSL concepts map to Eino implementations:

| OpenCode DSL | Eino Implementation |
|--------------|---------------------|
| `steps[].type: agent` | `graph.AddChatModelNode()` + ReAct agent |
| `steps[].type: parallel` | `chain.AppendParallel()` or `NewGraphMultiBranch()` |
| `steps[].type: conditional` | `graph.AddBranch()` with custom condition |
| `steps[].type: loop` | Graph cycle in Pregel mode + termination condition |
| `steps[].type: pause` | Custom lambda with checkpoint save (not built-in) |
| `steps[].type: transform` | `graph.AddLambdaNode()` |
| `steps[].type: llm_eval` | `graph.AddChatModelNode()` with structured output |
| `agents[].tools` | `compose.ToolsNodeConfig` |
| `orchestrator.mode` | Execution options (no direct equivalent) |
| `{{variable}}` interpolation | `compose.WithStatePreHandler()` + state access |

### 9.1 Proposed Go Implementation Pattern

```go
// Parse OpenCode YAML workflow
func ParseWorkflow(yamlContent []byte) (*WorkflowDSL, error) {
    var dsl WorkflowDSL
    yaml.Unmarshal(yamlContent, &dsl)
    return &dsl, dsl.Validate()
}

// Convert to Eino Graph
func (w *WorkflowDSL) ToEinoGraph(ctx context.Context) (*compose.Graph[WorkflowInput, WorkflowOutput], error) {
    graph := compose.NewGraph[WorkflowInput, WorkflowOutput](
        compose.WithGenLocalState(func(ctx context.Context) *WorkflowState {
            return &WorkflowState{Variables: w.Inputs}
        }),
    )

    // Add nodes for each step
    for _, step := range w.Steps {
        if err := w.addStepNode(graph, step); err != nil {
            return nil, err
        }
    }

    // Wire dependencies
    for _, step := range w.Steps {
        if err := w.wireStepDependencies(graph, step); err != nil {
            return nil, err
        }
    }

    return graph, nil
}

// Handle pause step type (OpenCode unique feature)
func (w *WorkflowDSL) addPauseNode(graph *compose.Graph, step StepDSL) error {
    pauseLambda := compose.InvokableLambda(func(ctx context.Context, input any) (any, error) {
        // Save checkpoint
        checkpoint := &Checkpoint{
            StepID:    step.ID,
            State:     extractState(ctx),
            Timestamp: time.Now(),
        }
        if err := w.checkpointer.Save(ctx, checkpoint); err != nil {
            return nil, err
        }

        // Signal pause to orchestrator
        return nil, ErrPauseRequested{
            Message: step.Message,
            StepID:  step.ID,
        }
    })
    return graph.AddLambdaNode(step.ID, pauseLambda)
}
```

---

## Sources

- [Eino GitHub Repository](https://github.com/cloudwego/eino)
- [Eino-ext Extensions](https://github.com/cloudwego/eino-ext)
- [Eino User Manual](https://www.cloudwego.io/docs/eino/)
- [Eino Orchestration Design](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/orchestration_design_principles/)
- [Eino Multi-Agent Hosting](https://www.cloudwego.io/docs/eino/core_modules/flow_integration_components/multi_agent_hosting/)
- [Eino ReAct Agent](https://www.cloudwego.io/docs/eino/core_modules/flow_integration_components/react_agent_manual/)
- Local source review: `vendor/eino/compose/*.go`, `vendor/eino/flow/agent/**/*.go`
