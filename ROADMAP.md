# Project Roadmap

Vision and planned development for ZPM.

## Current Status: v0.1.0 (Foundation)

**Focus**: Core functionality and architecture

### ✅ Completed

- [x] Process spawning (fork + setsid)
- [x] Process registry (JSON-based)
- [x] CLI commands (start/stop/list/purge)
- [x] Basic daemon skeleton
- [x] Documentation
- [x] Test server (Hono + Bun)

### 🚧 In Progress

- [ ] Socket-based IPC daemon
- [ ] Process group management
- [ ] Better error handling

---

## v0.2.0: Daemon Communication (Q2-Q3 2026)

**Focus**: IPC and process monitoring

### Features

- [ ] **Unix Socket Server** in zpmd
  - Event loop for client connections
  - Parallel request handling
  - Connection pooling

- [ ] **Process Monitoring**
  - Track process status
  - Detect crashes
  - Log output capture

- [ ] **Improved Registry**
  - Atomic writes (prevent corruption)
  - Backup/restore
  - Migration tools

- [ ] **Enhanced CLI**
  - Status checking
  - Real-time monitoring
  - Process output viewing

### Breaking Changes

- Socket communication protocol (not compatible with v0.1)
- Registry format may change

### Migration Path

```bash
# Backup old registry
cp zpm.json zpm.json.v0.1.bak

# Upgrade
git pull && zig build

# Registry auto-migrates
./zpmd
```

---

## v0.3.0: Advanced Process Management (Q3-Q4 2026)

**Focus**: Reliability and scalability

### Features

- [ ] **Process Restart**
  - Automatic restart on crash
  - Max restart attempts
  - Exponential backoff

- [ ] **Process Groups**
  - Start/stop related processes
  - Dependency ordering
  - Coordinated lifecycle

- [ ] **Resource Limits**
  - Memory limits (ulimit)
  - CPU time limits
  - File descriptor limits

- [ ] **Logging**
  - Centralized log collection
  - Rotation and compression
  - Structured logging (JSON)

- [ ] **Health Checks**
  - HTTP/TCP checks
  - Custom scripts
  - Metrics collection

---

## v0.4.0: Web Interface & Multi-user (Q4 2026 - Q1 2027)

**Focus**: Management UI and collaboration

### Features

- [ ] **Web Dashboard**
  - Process status page
  - Real-time graphs
  - Log viewer
  - Command interface

- [ ] **REST API**
  - Full CRUD operations
  - Webhooks
  - Authentication

- [ ] **Multi-user Support**
  - User accounts
  - Permission system
  - Audit logs

- [ ] **Clustering**
  - Distributed process management
  - Failover
  - Load balancing

---

## v1.0.0: Production Ready (Q1-Q2 2027)

**Focus**: Stability, performance, and completeness

### Goals

- [ ] Cross-platform support (macOS, Windows, Linux)
- [ ] Performance benchmarks met
- [ ] 100% API stability
- [ ] Comprehensive test coverage (>80%)
- [ ] Production deployment guide
- [ ] Migration guide from PM2

### Features

- [ ] **Advanced Scheduling**
  - Cron-like scheduling
  - Environment-based execution
  - Conditional triggers

- [ ] **Plugin System**
  - Custom handlers
  - Extensions API
  - Community plugins

- [ ] **Metrics & Observability**
  - Prometheus export
  - Distributed tracing
  - Performance profiling

- [ ] **Enterprise Features**
  - LDAP/SASL integration
  - SSO support
  - Advanced RBAC

---

## Long Term (v2.0+)

### Vision Features

- **Kubernetes Integration**
  - CRD for process definitions
  - Service mesh support
  - Pod-like isolation

- **Cloud Native**
  - Container support
  - Serverless integration
  - Multi-cloud deployment

- **AI/ML Integration**
  - Anomaly detection
  - Auto-scaling
  - Predictive monitoring

- **Governance**
  - Compliance reporting
  - Audit trails
  - Cost tracking

---

## Community Contributions Welcome

### Requested Features

We're looking for community contributions in:

1. **Documentation**
   - Tutorial videos
   - Blog posts
   - Use case examples

2. **Integrations**
   - Jenkins plugin
   - Terraform provider
   - Ansible module

3. **Tools**
   - CLI enhancements
   - Shell completions
   - IDE plugins

4. **Platforms**
   - Windows support
   - Docker enhancements
   - systemd integration

### How to Contribute

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## Timeline

```
2026
├─ Q2-Q3: v0.2.0 (Daemon)
├─ Q3-Q4: v0.3.0 (Advanced)
└─ Q4: v0.4.0 (Web UI)

2027
├─ Q1-Q2: v1.0.0 (Stable)
└─ Q2+: v2.0+ (Enterprise)
```

---

## Decision Making

### Feature Prioritization

Features are prioritized based on:

1. **User Demand** - Community votes
2. **Use Cases** - Real-world applicability
3. **Complexity** - Implementation difficulty
4. **Impact** - How much it improves ZPM

### Breaking Changes

- Minimized when possible
- Versioned with clear migration path
- Backward compatibility maintained

### API Stability

- v1.0.0 will have stable API
- Pre-v1.0 changes are allowed
- Deprecation warnings for v0.x changes

---

## Success Metrics

### By v0.2.0

- [x] 1,000+ GitHub stars
- [ ] 100+ contributors
- [ ] Active community

### By v1.0.0

- [ ] 10,000+ stars
- [ ] Production deployments
- [ ] Documented case studies

### By v2.0.0

- [ ] 50,000+ stars
- [ ] Industry adoption
- [ ] Enterprise customers

---

## Feedback & Suggestions

Have ideas for the roadmap? 

1. **Open an Issue** - Describe your idea
2. **Start a Discussion** - Discuss with community
3. **Vote on Features** - Emoji react to proposals

---

## Related Projects

- **PM2** - Process manager for Node.js
- **Supervisor** - Process control system
- **systemd** - System service manager

---

**Last Updated**: May 20, 2026

**Current Version**: 0.1.0

**Want to shape the future?** Join us! 🚀
