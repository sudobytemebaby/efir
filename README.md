# 🛰 Efir

**What is Efir?**
It’s a simple, honest messenger. I’m building this because I’m tired of "everything apps" that try to be a store, a crypto wallet, and a social network all at once. Efir is just about talking to people without extra noise and stuff they usually don't expect to appear in a communiation tool.

**The Philosophy**
I am a firm believer in the **Unix philosophy**: a tool should do one thing and do it well. For me, Efir's only job is to let people communicate — via text or voice — as reliably as possible.

I also believe in **ownership**. Efir follows a self-hosting model: you can use my instance or spin up your own on your hardware with a single command. In a world of digital blocks and censorship, having your own communication node isn't just a feature — it's a right.

---

### Tech Stack

- **Go 1.24**: Backend services.
- **NATS JetStream**: Messaging and events.
- **Valkey**: Caching and rate limiting.
- **PostgreSQL**: Persistent storage.
- **gRPC**: Internal communication.
- **Grafana, Loki & Tempo**: Observability.

---

### Roadmap

- [x] **Module 0: Foundation** — Architecture, CI/CD, and infra are ready.
- [ ] **Module 1: MVP** — Auth, user profiles, and basic realtime chat.
- [ ] **Module 2: Scale & Security** — Sidecar PEP for traffic validation and horizontal scaling.
- [ ] **Module 3: Features** — Presence status, voice calls, media handling, and global search.

---

### Quick Start

If you have **Docker** and **Task** installed:

```bash
# Prepare the network and environment
task setup

# Spin up all services and infrastructure
task up
```
