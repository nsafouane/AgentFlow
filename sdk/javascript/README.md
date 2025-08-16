# AgentFlow JavaScript SDK

JavaScript/TypeScript SDK for AgentFlow - Coming Soon

## Installation

```bash
npm install @agentflow/sdk
```

## Usage

```typescript
import { AgentFlowClient } from '@agentflow/sdk';

const client = new AgentFlowClient({
  endpoint: 'http://localhost:8080',
  apiKey: 'your-key'
});

const workflow = await client.createWorkflow({
  name: 'example',
  plan: plan
});
```

## Development

This SDK is currently a stub and will be implemented in future releases.