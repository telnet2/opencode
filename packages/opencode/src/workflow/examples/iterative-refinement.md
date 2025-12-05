---
id: iterative-refinement
name: Iterative Code Refinement Workflow
description: Uses LLM evaluation to iteratively refine code until quality criteria are met
version: 1.0.0
inputs:
  code:
    type: string
    description: The code to refine
    required: true
  requirements:
    type: string
    description: Quality requirements for the code
    required: true
  maxIterations:
    type: number
    description: Maximum refinement iterations
    default: 3
orchestrator:
  mode: guided
  onError: pause
tags:
  - iterative
  - llm-eval
  - code-quality
---

# Agent: code_improver

You are a code improvement specialist. Given code and feedback, make targeted improvements.

Focus on:
1. Addressing specific issues mentioned in the feedback
2. Improving code quality incrementally
3. Maintaining existing functionality

Return the improved code.

```yaml
tools:
  read: false
  edit: false
temperature: 0.3
```

# Agent: code_reviewer

You are a code quality reviewer. Analyze code against requirements and provide specific feedback.

Evaluate:
1. Does the code meet the stated requirements?
2. Are there bugs or issues?
3. Is the code clean and maintainable?

Provide specific, actionable feedback.

```yaml
tools:
  read: false
temperature: 0.4
```

# Steps

```yaml
# Initial review
- id: initial_review
  type: agent
  name: Initial Code Review
  agent: code_reviewer
  input: |
    Review this code against the requirements:

    **Code:**
    ```
    {{code}}
    ```

    **Requirements:**
    {{requirements}}

    Provide specific feedback on what needs improvement.
  output: review_feedback

# LLM evaluation to check if code meets requirements
- id: check_quality
  type: llm_eval
  name: Quality Check
  prompt: |
    Based on this review feedback, does the code need more improvements?

    Review feedback:
    {{review_feedback}}

    Requirements:
    {{requirements}}

    Consider: Are there critical issues? Is the code production-ready?
  outputFormat: boolean
  output: needs_improvement
  temperature: 0.1
  dependsOn: [initial_review]

# Conditional branch based on LLM evaluation
- id: decide_action
  type: conditional
  name: Decide Next Action
  condition: "{{needs_improvement}} === true"
  conditionType: expression
  then: refinement_loop
  else: final_output
  dependsOn: [check_quality]

# Refinement loop with LLM-based exit condition
- id: refinement_loop
  type: loop
  name: Refinement Loop
  maxIterations: 3
  indexVariable: iteration
  steps:
    - improve_code
    - review_improvement
  until: |
    Based on the latest review, is the code now meeting all requirements and ready for production?

    Latest review: {{review_feedback}}
    Requirements: {{requirements}}
  conditionType: llm
  dependsOn: [decide_action]

# Improvement step (inside loop)
- id: improve_code
  type: agent
  name: Improve Code
  agent: code_improver
  input: |
    Improve this code based on the feedback (iteration {{iteration}}):

    **Current Code:**
    ```
    {{current_code}}
    ```

    **Feedback to address:**
    {{review_feedback}}

    **Original Requirements:**
    {{requirements}}
  output: current_code

# Review improvement (inside loop)
- id: review_improvement
  type: agent
  name: Review Improvement
  agent: code_reviewer
  input: |
    Review the improved code (iteration {{iteration}}):

    **Improved Code:**
    ```
    {{current_code}}
    ```

    **Original Requirements:**
    {{requirements}}

    Has the code improved? What issues remain?
  output: review_feedback
  dependsOn: [improve_code]

# Human review before finalizing
- id: human_review
  type: pause
  name: Final Human Review
  message: |
    The code has been refined through {{iteration}} iterations.

    **Final Code:**
    {{current_code}}

    **Final Review:**
    {{review_feedback}}

    Please approve to finalize or reject to continue refinement.
  dependsOn: [refinement_loop]
  options:
    allowEdit: true
    approveLabel: Accept Code
    rejectLabel: Continue Refining

# Final output
- id: final_output
  type: transform
  name: Generate Final Output
  dependsOn: [human_review]
  input: "{{current_code}}"
  output: final_code
  transform: template
  options:
    template: |
      # Refined Code

      After {{iteration}} iterations, here is the final code:

      ```
      {{current_code}}
      ```

      ## Review Summary
      {{review_feedback}}
```
