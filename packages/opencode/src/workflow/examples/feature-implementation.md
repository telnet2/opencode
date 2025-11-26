---
id: feature-implementation
name: Feature Implementation Workflow
description: A workflow for researching, planning, implementing, and testing new features
version: 1.0.0
inputs:
  feature:
    type: string
    description: Description of the feature to implement
    required: true
  codebase_context:
    type: string
    description: Additional context about the codebase
    default: ""
orchestrator:
  mode: guided
  onError: pause
  maxRetries: 2
tags:
  - feature
  - development
---

# Agent: researcher

You are a technical researcher. Your job is to:
1. Understand the feature requirements
2. Research the existing codebase to understand patterns and conventions
3. Identify related code and potential integration points
4. Find relevant examples or similar implementations

Provide comprehensive research findings.

```yaml
tools:
  read: true
  glob: true
  grep: true
  bash: true
  edit: false
temperature: 0.4
permission:
  bash:
    "git *": allow
    "find *": allow
    "*": deny
```

# Agent: planner

You are an implementation planner. Based on the research:
1. Create a detailed implementation plan
2. Break down the work into specific tasks
3. Identify potential risks and dependencies
4. Suggest the order of implementation

Be specific about files to create/modify and the changes needed.

```yaml
tools:
  read: true
  glob: true
  edit: false
temperature: 0.5
```

# Agent: implementer

You are a skilled developer. Follow the implementation plan to:
1. Create new files as needed
2. Modify existing code carefully
3. Follow the project's coding conventions
4. Add appropriate comments and documentation

Implement one task at a time and verify your changes.

```yaml
tools:
  read: true
  edit: true
  write: true
  glob: true
  grep: true
  bash: true
temperature: 0.2
```

# Agent: tester

You are a QA specialist. Your job is to:
1. Review the implemented changes
2. Run existing tests to ensure nothing is broken
3. Suggest additional tests if needed
4. Verify the feature works as expected

Report any issues found.

```yaml
tools:
  read: true
  glob: true
  grep: true
  bash: true
  edit: false
permission:
  bash:
    "npm test*": allow
    "bun test*": allow
    "pytest*": allow
    "go test*": allow
    "*": deny
```

# Steps

```yaml
- id: research
  type: agent
  name: Research Phase
  description: Research the codebase and feature requirements
  agent: researcher
  input: |
    Research the codebase to prepare for implementing this feature:

    Feature: {{feature}}

    Additional context: {{codebase_context}}

    Please:
    1. Understand the current codebase structure
    2. Find relevant patterns and conventions
    3. Identify files that will need to be modified or created
    4. Note any dependencies or prerequisites
  output: research_findings

- id: plan
  type: agent
  name: Planning Phase
  description: Create implementation plan
  agent: planner
  dependsOn: [research]
  input: |
    Based on the research findings, create an implementation plan:

    Feature: {{feature}}

    Research findings:
    {{research_findings}}

    Create a detailed step-by-step implementation plan.
  output: implementation_plan

- id: review_plan
  type: pause
  name: Plan Review
  description: Review the implementation plan before proceeding
  message: |
    Please review the implementation plan before we start coding.

    Do you approve this plan?
  dependsOn: [plan]
  approvalVariable: plan_approved
  options:
    allowEdit: true
    approveLabel: Start Implementation
    rejectLabel: Revise Plan

- id: implement
  type: agent
  name: Implementation Phase
  description: Implement the feature
  agent: implementer
  dependsOn: [review_plan]
  condition: "{{plan_approved}} === true"
  input: |
    Implement the feature according to this plan:

    {{implementation_plan}}

    Follow the plan carefully and implement each step.
  output: implementation_result

- id: test
  type: agent
  name: Testing Phase
  description: Test the implementation
  agent: tester
  dependsOn: [implement]
  input: |
    Test the newly implemented feature:

    Feature: {{feature}}

    Implementation details:
    {{implementation_result}}

    Run tests and verify the feature works correctly.
  output: test_results

- id: final_review
  type: pause
  name: Final Review
  description: Final review before completion
  message: |
    Implementation and testing complete. Please review before finalizing.

    Test Results:
    {{test_results}}
  dependsOn: [test]
  options:
    approveLabel: Complete
    rejectLabel: Request Changes
```
