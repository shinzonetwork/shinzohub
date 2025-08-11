# Shinzo ACP Testing

This directory contains comprehensive tests for the Access Control Policy (ACP) system in Shinzo.

## Overview

The testing system verifies that:
1. Users can only access resources they have permission for
2. Group memberships work correctly
3. Banned/blocked users are properly restricted
4. Data feed subscriptions work as expected

## Architecture

### Components

- **DID Generator**: Creates real DIDs using SourceHub and caches them
- **Test Helper**: Provides utilities for making requests to the registrar and DefraDB
- **Test Runner**: Orchestrates the test execution
- **Test Cases**: Based on `tests.yaml` from the ACP playground

### Test Users

The system creates DIDs for these test users:
- `randomUser` - Basic user with limited permissions
- `aHost` - Member of the "host" group
- `anIndexer` - Member of the "indexer" group  
- `subscriber` - Has query access to data feeds
- `creator` - Creator of data feeds
- `aBlockedIndexer` - Blocked indexer (should be denied access)
- `aBannedIndexer` - Banned indexer (should be denied access)
- And more...

## Usage

### Prerequisites

1. Start the Shinzo system:
   ```bash
   make bootstrap SOURCEHUB_PATH=/path/to/sourcehub INDEXER_PATH=/path/to/indexer
   ```

2. Wait for all services to be ready (check `.shinzohub/ready` file)

### Running Tests

#### Option 1: Full integration test workflow (Recommended)
```bash
make integration-test SOURCEHUB_PATH=/path/to/sourcehub
```

This will:
1. Bootstrap the entire system
2. Wait for all services to be ready
3. Generate real DIDs for test users
4. Run the ACP integration tests
5. Report results

#### Option 2: Run tests only (assumes services are running)
```bash
make test-acp
```

#### Option 3: Run tests with verbose output
```bash
make test-acp-v
```

#### Option 4: Use the test script directly
```bash
./scripts/test_acp.sh
```

This script will:
1. Check if services are running
2. Run the integration tests
3. Report results

#### Option 5: Run tests directly with go test
```bash
go test -v ./tests -run TestAccessControl
```

### Managing DIDs

**Note**: DIDs are now automatically generated during test setup using the SourceHub Go SDK. No manual DID management is required.

The system automatically:
- Generates unique DIDs for each test user
- Maps alias DIDs (e.g., `did:user:randomUser`) to real DIDs
- Uses real DIDs for all ACP operations
- Provides logging to show which alias resolves to which real DID

## Test Structure

### Test Function

The main test function `TestAccessControl` runs all test cases in sequence:

1. **Setup**: Generates real DIDs for test users using SourceHub SDK
2. **Service Check**: Waits for registrar and DefraDB to be ready
3. **Relationship Setup**: Establishes initial group memberships
4. **Test Execution**: Runs each test case and verifies results

### Test Cases

Each test case verifies:
- **User**: Which user is attempting the action (using alias DIDs like `did:user:randomUser`)
- **Resource**: Which resource they're trying to access
- **Action**: What action they're trying to perform (read, update, query, etc.)
- **Expected Result**: Whether the action should succeed or fail

### Example Test Case

```go
{
    Name:           "anIndexer_can_update_blocks",
    UserDID:        "did:user:anIndexer",  // Automatically resolved to real DID
    Resource:       "primitive:blocks", 
    Action:         "update",
    ExpectedResult: true,  // Should succeed
}
```

## How It Works

1. **DID Generation**: During test setup, real DIDs are generated using `did.ProduceDID()` from SourceHub SDK
2. **Alias Resolution**: Test cases use alias DIDs (e.g., `did:user:randomUser`) which are automatically resolved to real DIDs
3. **User Setup**: Users are added to appropriate groups via the registrar API using their real DIDs
4. **Access Testing**: Each test case attempts an action and verifies the result
5. **ACP Enforcement**: DefraDB enforces the ACP policies and returns success/failure

## Troubleshooting

### Common Issues

1. **Services not ready**: Ensure all services are running and healthy
2. **DID generation fails**: Check that SourceHub is running and accessible
3. **Tests fail**: Check that the policy is properly uploaded and groups are created

### Debugging

1. Check service logs:
   ```bash
   tail -f logs/registrar_logs.txt
   tail -f logs/sourcehub_logs.txt
   ```

2. Verify policy setup:
   ```bash
   # Check if policy exists
   sourcehubd query acp policy-ids --chain-id=sourcehub-dev
   
   # Check group memberships
   sourcehubd query acp relationships --chain-id=sourcehub-dev
   ```

3. Check if services are responding:
   ```bash
   # Check registrar
   curl -s http://localhost:8081/registrar/
   
   # Check DefraDB
   curl -s http://localhost:9181/graphql
   ```

### Service Management

- **Start services**: `make bootstrap SOURCEHUB_PATH=/path/to/sourcehub`
- **Stop services**: `make stop`
- **Check status**: Look for `.shinzohub/ready` file
- **View logs**: Check the `logs/` directory for service-specific log files

## Future Enhancements

1. **Real SourceHub Integration**: Replace placeholder DID creation with actual SourceHub API calls
2. **More Test Cases**: Add tests for delegation and complex permission scenarios
3. **Performance Testing**: Add benchmarks for ACP enforcement
4. **Negative Testing**: Add tests for edge cases and error conditions 