name: Bug Report
description: Report a bug or issue with ZPM
title: "[BUG] "
labels: ["bug"]
body:
  - type: markdown
    attributes:
      value: |
        Thanks for reporting a bug! Please fill out the form below.
  
  - type: textarea
    id: description
    attributes:
      label: Description
      description: Clear description of the bug
      placeholder: What happened?
    validations:
      required: true

  - type: textarea
    id: steps
    attributes:
      label: Steps to Reproduce
      description: How to reproduce the issue
      placeholder: |
        1. Run `zpm start --name test "command"`
        2. Do something
        3. Observe the bug
    validations:
      required: true

  - type: textarea
    id: expected
    attributes:
      label: Expected Behavior
      description: What should happen instead
    validations:
      required: true

  - type: textarea
    id: actual
    attributes:
      label: Actual Behavior
      description: What actually happened
    validations:
      required: true

  - type: textarea
    id: environment
    attributes:
      label: Environment
      description: System information
      placeholder: |
        - OS: Linux/macOS/Windows
        - Zig version: 0.12.0
        - ZPM version: 0.1.0
        - Arch: x86_64/aarch64
    validations:
      required: true

  - type: textarea
    id: logs
    attributes:
      label: Error Logs
      description: Any error messages or logs
      render: shell

  - type: checkboxes
    id: checklist
    attributes:
      label: Checklist
      options:
        - label: I searched existing issues
          required: true
        - label: I can reproduce the bug
          required: true
        - label: I'm using the latest version
          required: true
