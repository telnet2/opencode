---
id: code-review
name: Code Review Workflow
description: A multi-agent code review workflow that analyzes, reviews, and optionally fixes code
version: 1.0.0
inputs:
  files:
    type: string
    description: Files or directories to review
    required: true
  autoFix:
    type: boolean
    description: Whether to automatically apply fixes
    default: false
orchestrator:
  mode: guided
  onError: pause
  maxRetries: 2
  defaultTimeout: 300000
tags:
  - code-review
  - quality
---

# Agent: analyzer

You are a code analyzer specializing in identifying issues in source code.

When given files to analyze, you should:
1. Read and understand the code structure
2. Identify potential bugs, security issues, and code smells
3. Check for performance concerns
4. Note any violations of common best practices

Output your findings in a structured format with severity levels (critical, warning, info).

```yaml
tools:
  read: true
  glob: true
  grep: true
  bash: false
  edit: false
model: anthropic/claude-sonnet-4-20250514
temperature: 0.3
```

# Agent: reviewer

You are a senior code reviewer. Based on the analysis provided, you should:
1. Evaluate each finding for accuracy and relevance
2. Prioritize issues by impact
3. Provide specific, actionable suggestions for improvement
4. Consider the overall code architecture

Be constructive and specific in your feedback.

```yaml
tools:
  read: true
  glob: true
  bash: false
  edit: false
temperature: 0.5
```

# Agent: fixer

You are a code improvement specialist. Based on the review feedback:
1. Implement the suggested fixes carefully
2. Ensure changes maintain existing functionality
3. Follow the project's coding style
4. Add comments where the changes might not be obvious

Only make changes that have been explicitly approved.

```yaml
tools:
  read: true
  edit: true
  write: true
  glob: true
  bash: false
temperature: 0.2
```

# Steps

```yaml
- id: analyze
  type: agent
  name: Code Analysis
  description: Analyze the code for issues
  agent: analyzer
  input: |
    Analyze the following files for issues:
    {{files}}

    Look for:
    - Potential bugs
    - Security vulnerabilities
    - Code smells
    - Performance issues
    - Best practice violations

    Provide a structured report with severity levels.
  output: analysis_result

- id: review
  type: agent
  name: Code Review
  description: Review and prioritize the findings
  agent: reviewer
  dependsOn: [analyze]
  input: |
    Review the following code analysis:

    {{analysis_result}}

    Please:
    1. Validate each finding
    2. Prioritize by impact
    3. Provide specific improvement suggestions
  output: review_result

- id: human_review
  type: pause
  name: Human Review
  description: Wait for human approval before fixes
  message: |
    Please review the code analysis and suggestions before proceeding with fixes.

    Analysis found the following issues. Do you want to proceed with automated fixes?
  dependsOn: [review]
  approvalVariable: fixes_approved
  options:
    allowEdit: true
    allowReject: true
    approveLabel: Apply Fixes
    rejectLabel: Skip Fixes

- id: check_approval
  type: conditional
  name: Check Approval
  description: Check if fixes were approved
  dependsOn: [human_review]
  condition: "{{fixes_approved}} === true && {{autoFix}} === true"
  then: apply_fixes
  else: generate_report

- id: apply_fixes
  type: agent
  name: Apply Fixes
  description: Apply the approved fixes
  agent: fixer
  input: |
    Apply the following approved improvements:

    {{review_result}}

    Make the necessary changes to improve the code.
  output: fix_result

- id: generate_report
  type: transform
  name: Generate Report
  description: Generate the final report
  dependsOn: [review]
  input: "{{review_result}}"
  output: final_report
  transform: template
  options:
    template: |
      # Code Review Report

      ## Analysis Results
      {{analysis_result}}

      ## Review Findings
      {{review_result}}

      ## Status
      Fixes applied: {{fixes_approved}}
```
