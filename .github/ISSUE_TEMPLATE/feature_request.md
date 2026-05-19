name: Feature Request
description: Suggest a new feature for ZPM
title: "[FEATURE] "
labels: ["enhancement"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for suggesting a feature! Please describe your idea.

  - type: textarea
    id: problem
    attributes:
      label: Problem Statement
      description: What problem does this solve?
      placeholder: |
        I'm trying to... but I can't because...
    validations:
      required: true

  - type: textarea
    id: solution
    attributes:
      label: Proposed Solution
      description: How should this work?
      placeholder: |
        I suggest adding a new command...
    validations:
      required: true

  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives Considered
      description: Other solutions you've considered
      placeholder: |
        - Option A: ...
        - Option B: ...

  - type: textarea
    id: examples
    attributes:
      label: Usage Examples
      description: How would users use this feature?
      placeholder: |
        ```bash
        zpm new-command --option value
        ```

  - type: textarea
    id: context
    attributes:
      label: Additional Context
      description: Any other information
