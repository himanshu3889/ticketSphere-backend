# 📚 TicketSphere – Backend

**TicketSphere** is a system designed to simulate how a production-scale ticket booking platform behaves under extreme load.

The goal of this project is not feature completeness, but architectural correctness under stress: handling flash traffic spikes during popular event releases, high contention scenarios where thousands of users compete for the same seats, and complex distributed transaction workflows while keeping performance predictable and latency low. 

It explores practical distributed systems patterns such as saga orchestration, optimistic and pessimistic concurrency control, Redis-based distributed locking with TTL, virtual waiting rooms, bulkhead resource isolation, event sourcing with CQRS, bloom filters, circuit breaker failure isolation, dead letter queues with exponential backoff retry.

It is designed as a playground for system design decisions that matter when scale breaks simple solutions.


go get ./...


---
*Created by [Himanshu](https://github.com/himanshu3889)*