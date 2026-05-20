# Contributing to ZPM

Thanks for your interest in contributing to ZPM! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful and constructive in all interactions. We're building a welcoming community.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally
3. **Create** a new branch for your feature/fix
4. **Make** your changes
5. **Test** your changes thoroughly
6. **Push** to your fork
7. **Submit** a Pull Request

## Development Setup

```bash
git clone https://github.com/shellhaki/zpm.git
cd zpm
zig build

# Run the test server
cd test-server
bun install
```

## Code Guidelines

### Zig Style

- Follow official Zig conventions
- Use snake_case for variables and functions
- Use CONSTANT_CASE for constants
- Keep functions focused and pure
- Prefer explicit over implicit

### Functional Programming

- Avoid mutable global state
- Pass data through function parameters
- Use error unions for error handling
- Compose small functions into larger ones

### Error Handling

Always use Zig's error handling:

```zig
fn mayFail(allocator: std.mem.Allocator) !void {
    const data = try allocator.alloc(u8, 100);
    defer allocator.free(data);
    // use data
}
```

### No Comments

- Code should be self-documenting
- Use clear, descriptive names
- Break complex logic into named functions
- Let the code tell the story

## Testing

### Manual Testing

```bash
# Build
zig build

# Test basic commands
cd zig-out/bin
./zpm start --name test "sleep 60"
./zpm list
./zpm stop test
./zpm purge test

# Test with Hono server
cd ../../test-server
bun run src/index.ts
# In another terminal:
curl http://localhost:3000/health
```

### Running Tests

```bash
# Build in debug mode for testing
zig build

# Execute the binaries
./zig-out/bin/zpmd &
./zig-out/bin/zpm list
```

## Pull Request Process

1. **Update** documentation if needed
2. **Test** your changes thoroughly
3. **Describe** what your PR does
4. **Reference** any related issues
5. **Keep** commits focused and logical

### PR Title Format

- `feat: add socket-based daemon communication`
- `fix: handle process spawn errors gracefully`
- `docs: update README with usage examples`
- `refactor: simplify registry loading logic`

### PR Description Template

```markdown
## Description
Briefly describe what this PR does.

## Changes
- Change 1
- Change 2
- Change 3

## Testing
How did you test this?

## Related Issues
Closes #123
```

## Areas for Contribution

### Easy (Good First Issues)

- Documentation improvements
- Adding error messages
- Code examples
- Test coverage

### Medium

- Additional commands (status, restart, etc.)
- Better error handling
- Performance optimizations
- Logging system

### Hard

- Socket-based daemon communication
- Multi-user support
- Process monitoring dashboard
- Cross-platform support

## Extending Package.json Support

The start handler can be extended to support other package managers and configuration files:

### Adding Support for pyproject.toml (Python)

```zig
fn getPythonScript(allocator: std.mem.Allocator, script_name: []const u8) ?[]const u8 {
    // Parse pyproject.toml
    // Extract [tool.scripts] or [project.scripts]
    // Return command
}
```

### Adding Support for Makefile

```zig
fn getMakeTarget(allocator: std.mem.Allocator, target_name: []const u8) ?[]const u8 {
    // Parse Makefile
    // Extract target command
    // Return command
}
```

**How to Contribute:**
1. Implement the parsing function
2. Add to start handler
3. Document in README
4. Add tests/examples
5. Submit PR!

## Reporting Bugs

### Bug Report Template

```markdown
## Description
Brief description of the bug.

## Steps to Reproduce
1. Run `zpm start --name test "command"`
2. Run `zpm list`
3. Observe unexpected behavior

## Expected Behavior
What should happen?

## Actual Behavior
What actually happened?

## Environment
- Zig version: 0.12.0
- OS: Linux
- Arch: x86_64
```

## Feature Requests

Describe the feature you'd like:

1. **Problem** - What problem does it solve?
2. **Solution** - How should it work?
3. **Examples** - Show usage examples
4. **Alternatives** - Are there other approaches?

## Project Structure

```
src/
├── main.zig       # Client CLI entry
├── daemon.zig     # Daemon entry & process management
├── registry.zig   # Process registry core
├── build.zig      # Build configuration
└── handlers/      # Command handlers
    ├── start.zig
    ├── stop.zig
    ├── list.zig
    └── purge.zig
```

## Key Concepts

### Registry

The process registry (`zpm.json`) is the single source of truth:

```zig
pub const Process = struct {
    name: []const u8,
    command: []const u8,
    status: Status,
    pid: u32,
};
```

### Process Lifecycle

```
START -> RUNNING -> STOP -> STOPPED -> PURGE -> [removed]
```

### Daemon Architecture

- Client connects via Unix socket
- Daemon handles command execution
- Processes fork and detach (setsid)
- Registry persisted to disk

## Communication

- **Issues** - For bugs and feature requests
- **Discussions** - For questions and ideas
- **Pull Requests** - For code contributions

## Recognition

Contributors will be:
- Added to CONTRIBUTORS.md
- Mentioned in release notes
- Credited in commit history

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

Open an issue or discussion. We're here to help!

---

Thank you for contributing to ZPM! 🎉
