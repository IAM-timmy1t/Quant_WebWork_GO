# QUANT_WebWork_GO API Specifications

## Overview

This document provides comprehensive specifications for the QUANT_WebWork_GO API interfaces. It covers both REST and GraphQL endpoints, authentication requirements, request/response formats, and error handling.

## API Versions

- Current version: `v1`
- Base URL: `/api/v1`

## Authentication

### JWT Authentication

All API endpoints (except those explicitly marked as public) require JWT authentication.

**Authentication Header Format:**

```http
Authorization: Bearer <jwt_token>
```

**JWT Token Structure:**

```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user_id",
    "iss": "quant_webwork_go",
    "iat": 1614556800,
    "exp": 1614643200,
    "role": "user|admin",
    "permissions": ["read", "write"]
  },
  "signature": "..."
}
```

## REST API Endpoints

### Health Check

**Endpoint:** `GET /health`  
**Authentication:** None (Public)  
**Description:** Provides health status of the system  
**Response:**

```json
{
  "status": "ok",
  "version": "1.0.0",
  "components": {
    "database": "ok",
    "cache": "ok",
    "bridge_system": "ok"
  },
  "timestamp": "2025-03-26T01:37:02Z"
}
```

### Bridge System

#### List Available Bridges

**Endpoint:** `GET /api/v1/bridges`  
**Authentication:** Required  
**Description:** Returns a list of available bridges  
**Response:**

```json
{
  "bridges": [
    {
      "id": "grpc-bridge-1",
      "type": "grpc",
      "status": "active",
      "protocols": ["grpc/http2"],
      "created_at": "2025-03-25T10:00:00Z"
    },
    {
      "id": "rest-bridge-1",
      "type": "rest",
      "status": "active",
      "protocols": ["http/1.1", "http/2"],
      "created_at": "2025-03-25T10:05:00Z"
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 10
}
```

#### Get Bridge Details

**Endpoint:** `GET /api/v1/bridges/{id}`  
**Authentication:** Required  
**Description:** Returns details about a specific bridge  
**Response:**

```json
{
  "id": "grpc-bridge-1",
  "type": "grpc",
  "status": "active",
  "protocols": ["grpc/http2"],
  "created_at": "2025-03-25T10:00:00Z",
  "metrics": {
    "requests_total": 1250,
    "error_rate": 0.01,
    "latency_p95_ms": 42
  },
  "config": {
    "max_connections": 100,
    "timeout_ms": 5000,
    "retry_policy": {
      "max_retries": 3,
      "backoff_ms": 100
    }
  }
}
```

#### Create Bridge

**Endpoint:** `POST /api/v1/bridges`  
**Authentication:** Required (Admin)  
**Description:** Creates a new bridge  
**Request:**

```json
{
  "type": "websocket",
  "config": {
    "max_connections": 200,
    "heartbeat_interval_ms": 30000,
    "protocols": ["ws", "wss"]
  }
}
```

**Response:**

```json
{
  "id": "websocket-bridge-1",
  "type": "websocket",
  "status": "initializing",
  "protocols": ["ws", "wss"],
  "created_at": "2025-03-26T01:38:15Z"
}
```

### Metrics

#### Get System Metrics

**Endpoint:** `GET /api/v1/metrics/system`  
**Authentication:** Required  
**Description:** Returns system-level metrics  

**Parameters:**

- `start_time` (optional): Start time for metrics range
- `end_time` (optional): End time for metrics range
- `interval` (optional): Interval for data points (e.g., 1m, 5m, 1h)

**Response:**

```json
{
  "timestamp": "2025-03-26T01:39:00Z",
  "interval": "5m",
  "metrics": {
    "cpu_usage": [
      {"timestamp": "2025-03-26T01:15:00Z", "value": 0.45},
      {"timestamp": "2025-03-26T01:20:00Z", "value": 0.52},
      {"timestamp": "2025-03-26T01:25:00Z", "value": 0.48}
    ],
    "memory_usage": [
      {"timestamp": "2025-03-26T01:15:00Z", "value": 512.5},
      {"timestamp": "2025-03-26T01:20:00Z", "value": 524.3},
      {"timestamp": "2025-03-26T01:25:00Z", "value": 518.7}
    ],
    "network_traffic": [
      {"timestamp": "2025-03-26T01:15:00Z", "in": 1024, "out": 896},
      {"timestamp": "2025-03-26T01:20:00Z", "in": 1156, "out": 921},
      {"timestamp": "2025-03-26T01:25:00Z", "in": 982, "out": 873}
    ]
  }
}
```

### User Management

#### Create User

**Endpoint:** `POST /api/v1/users`  
**Authentication:** Required (Admin)  
**Description:** Creates a new user  
**Request:**

```json
{
  "username": "new_user",
  "email": "user@example.com",
  "role": "user",
  "permissions": ["read", "write"]
}
```

**Response:**

```json
{
  "id": "user_12345",
  "username": "new_user",
  "email": "user@example.com",
  "role": "user",
  "permissions": ["read", "write"],
  "created_at": "2025-03-26T01:40:00Z"
}
```

### Security

#### IP Masking Status

**Endpoint:** `GET /api/v1/security/ipmasking/status`  
**Authentication:** Required (Admin)  
**Description:** Get current IP masking status  
**Response:**

```json
{
  "enabled": true,
  "rotation_interval": "1h",
  "last_rotation": "2025-03-26T01:00:00Z",
  "next_rotation": "2025-03-26T02:00:00Z",
  "masked_ips_count": 247
}
```

#### Firewall Rules

**Endpoint:** `GET /api/v1/security/firewall/rules`  
**Authentication:** Required (Admin)  
**Description:** List active firewall rules  
**Response:**

```json
{
  "rules": [
    {
      "id": "rule_1",
      "type": "block",
      "source": "10.0.0.0/8",
      "destination": "0.0.0.0/0",
      "port": 0,
      "protocol": "any",
      "active": true
    },
    {
      "id": "rule_2",
      "type": "allow",
      "source": "0.0.0.0/0",
      "destination": "192.168.1.100",
      "port": 443,
      "protocol": "tcp",
      "active": true
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 10
}
```

## GraphQL API

The GraphQL API provides a more flexible interface for querying and mutating data.

**Endpoint:** `/api/v1/graphql`  
**Authentication:** Required (unless explicitly marked as public)

### Schema Overview

```graphql
type Query {
  bridges(status: String, type: String, limit: Int, offset: Int): [Bridge!]!
  bridge(id: ID!): Bridge
  metrics(type: String!, startTime: String, endTime: String, interval: String): MetricsData
  user(id: ID!): User
  ipMaskingStatus: IPMaskingStatus
  firewallRules(type: String, active: Boolean, limit: Int, offset: Int): [FirewallRule!]!
}

type Mutation {
  createBridge(input: CreateBridgeInput!): Bridge!
  updateBridge(id: ID!, input: UpdateBridgeInput!): Bridge!
  deleteBridge(id: ID!): Boolean!
  createUser(input: CreateUserInput!): User!
  updateIPMasking(input: UpdateIPMaskingInput!): IPMaskingStatus!
  addFirewallRule(input: AddFirewallRuleInput!): FirewallRule!
}

type Bridge {
  id: ID!
  type: String!
  status: String!
  protocols: [String!]!
  createdAt: String!
  metrics: BridgeMetrics
  config: BridgeConfig
}

type BridgeMetrics {
  requestsTotal: Int!
  errorRate: Float!
  latencyP95Ms: Int!
}

type BridgeConfig {
  maxConnections: Int!
  timeoutMs: Int!
  retryPolicy: RetryPolicy
}

type RetryPolicy {
  maxRetries: Int!
  backoffMs: Int!
}

type MetricsData {
  timestamp: String!
  interval: String!
  metrics: JSON!
}

type User {
  id: ID!
  username: String!
  email: String!
  role: String!
  permissions: [String!]!
  createdAt: String!
}

type IPMaskingStatus {
  enabled: Boolean!
  rotationInterval: String!
  lastRotation: String
  nextRotation: String
  maskedIpsCount: Int!
}

type FirewallRule {
  id: ID!
  type: String!
  source: String!
  destination: String!
  port: Int!
  protocol: String!
  active: Boolean!
}

input CreateBridgeInput {
  type: String!
  config: BridgeConfigInput!
}

input BridgeConfigInput {
  maxConnections: Int!
  timeoutMs: Int!
  retryPolicy: RetryPolicyInput
  protocols: [String!]!
}

input RetryPolicyInput {
  maxRetries: Int!
  backoffMs: Int!
}

input UpdateBridgeInput {
  status: String
  config: BridgeConfigInput
}

input CreateUserInput {
  username: String!
  email: String!
  role: String!
  permissions: [String!]!
}

input UpdateIPMaskingInput {
  enabled: Boolean!
  rotationInterval: String!
}

input AddFirewallRuleInput {
  type: String!
  source: String!
  destination: String!
  port: Int!
  protocol: String!
  active: Boolean!
}

scalar JSON
```

### Example GraphQL Queries

#### Query Bridges

```graphql
query {
  bridges(status: "active", limit: 5) {
    id
    type
    status
    protocols
    metrics {
      requestsTotal
      errorRate
    }
  }
}
```

#### Create Bridge Mutation

```graphql
mutation {
  createBridge(input: {
    type: "websocket",
    config: {
      maxConnections: 200,
      timeoutMs: 5000,
      retryPolicy: {
        maxRetries: 3,
        backoffMs: 100
      },
      protocols: ["ws", "wss"]
    }
  }) {
    id
    type
    status
    createdAt
  }
}
```

## Error Handling

All API endpoints follow a consistent error format:

### REST Error Format

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Bridge with ID 'invalid-id' not found",
    "details": {
      "resource_type": "bridge",
      "resource_id": "invalid-id"
    },
    "request_id": "req-12345-abcd-6789"
  }
}
```

### GraphQL Error Format

```json
{
  "errors": [
    {
      "message": "Bridge with ID 'invalid-id' not found",
      "path": ["bridge"],
      "extensions": {
        "code": "RESOURCE_NOT_FOUND",
        "details": {
          "resource_type": "bridge",
          "resource_id": "invalid-id"
        },
        "request_id": "req-12345-abcd-6789"
      }
    }
  ]
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `UNAUTHORIZED` | User is not authenticated |
| `FORBIDDEN` | User does not have permission |
| `RESOURCE_NOT_FOUND` | Requested resource does not exist |
| `VALIDATION_ERROR` | Request validation failed |
| `RATE_LIMITED` | Request rate limit exceeded |
| `INTERNAL_ERROR` | Internal server error occurred |
