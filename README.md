# AP2 Assignment 2 — gRPC Migration

##  Overview

This project demonstrates the migration from REST-based communication to gRPC with server-side streaming between microservices.

The system consists of two services:
- Order Service (REST API)
- Payment Service (gRPC + REST)

---

##  Architecture

- Order Service sends payment requests via gRPC
- Payment Service processes payments and streams status updates
- Order Service listens to streaming updates and updates order status in database

---

##  Technologies

- Go (Golang)
- gRPC
- Protocol Buffers
- PostgreSQL
- Docker

---

##  gRPC Streaming

We implemented **server-side streaming**:

Payment Service sends multiple updates: