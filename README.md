Hepi is a command-line tool for testing REST APIs using YAML-based configurations. It supports environment management, dynamic data generation, and response chaining, allowing you to build complex test scenarios where results from one request are used in subsequent ones.

## Installation

```bash
go install github.com/mitjafelicijan/hepi@latest
```

## CLI Usage

### Basic Usage

```bash
# Execute a single request
hepi -env staging -file test.yaml -req create_user

# Execute multiple requests
hepi -env staging -file test.yaml -req login,get_profile

# Use a specific environment
hepi -env staging -file test.yaml -req get_status
```

### Executing Groups

```bash
# Execute a group of requests defined in the YAML
hepi -env staging -file test.yaml -group auth_flow
```

### Overriding Variables

You can override variables defined in the `environments` section by passing them as system environment variables. This is useful for CI/CD or quick testing.

```bash
# Override the 'host' variable from the command line
host=https://httpbin.org go run main.go generators.go -env test -file test.yaml -group all
```

### Variable Precedence

When resolving `{{variable}}` placeholders, Hepi follows a strict lookup sequence. The first source to return a value wins.

1.  **System Environment**: Variables set in your shell or passed as command-line prefixes (e.g., `HOST=... go run ...`).
2.  **Local `.env` File**: Variables loaded from a `.env` file in the current directory. These provide defaults that can be overridden by the system environment.
3.  **YAML Environment**: Variables defined within the specific `environments` block selected via the `-env` flag.
4.  **Persistent Results**: Key-value pairs stored in `.hepi.json` from previous request executions (accessed via `{{request_name.path.to.key}}`).

#### Rationale

This hierarchy (System > .env > YAML > Results) is designed for **dynamic runtime overrides**:
*   **Non-destructive testing**: Override values from the CLI without modifying the static YAML configuration.
*   **Secret Management**: Keep sensitive credentials in the environment or `.env` files to avoid committing them to version control.
*   **CI/CD Integration**: Automated pipelines can inject configuration via environment variables which seamlessly take precedence.


### Options

*   `-env`: The environment to use.
*   `-file`: Path to the YAML configuration file.
*   `-req`: Comma-separated list of request names to execute.
*   `-group`: The name of a request group to execute.
*   `-headers`: Show response headers in the output.

## Core Concepts

### Environments

Environments allow you to define variables that change based on the target (e.g., local development vs. production). Each environment is a map of key-value pairs.

### Requests

Requests are the individual API calls you want to perform. Each request specifies its method, URL, headers, and body (`json`, `form`, or `files`).

### Groups

Groups are ordered lists of requests. Executing a group runs the requests in the specified sequence.

## Configuration Syntax

The configuration is defined in a YAML file (e.g., `test.yaml`).

### Substitution Syntax

Hepi uses two types of placeholders:

1.  **`{{variable}}`**: Used for substituting values from:
    *   **Environment Variables**: Values from a `.env` file (loaded automatically if present) or system environment variables.
    *   **Config Variables**: Variables defined in the `environments` section of the YAML.
    *   **Request Results**: Values captured from previous request responses (e.g., `{{login_req.token}}`). For arrays, use index notation (e.g., `{{setup_project.members.0.name}}`).
2.  **`[[generator]]`**: Used for generating dynamic data (e.g., `[[email]]`, `[[name]]`).
3.  **`[[oneof: a, b, c]]`**: Randomly selects one of the provided values.

### Result Chaining (Persistence)

When a request is executed, its response (if it's JSON) is stored in a local `.hepi.json` file. This allows subsequent requests to reference any field from the response using the `{{request_name.path.to.field}}` syntax.

## Data Generators (Fakers)

Hepi includes a wide range of generators for dynamic data. You can use these by wrapping the tag in double brackets, e.g., `[[email]]`.

| Tag | Description |
| :--- | :--- |
| `name` | Random full name |
| `first_name` | Random first name |
| `last_name` | Random last name |
| `email` | Random email address |
| `username` | Random username |
| `password` | Random password |
| `url` | Random URL |
| `phone` | Random phone number |
| `int` | Random integer (0 - 1,000,000) |
| `datetime` | Random time string |
| `date` | Random date string |
| `timestamp` | Random timestamp |
| `uuid_hyphenated`| Random hyphenated UUID |
| `jwt` | Random JWT token |
| `ipv4` | Random IPv4 address |
| `amount` | Random currency amount |
| `word` | Random word |
| `sentence` | Random sentence |
| `real_address` | Random real-world address |
| `cc_number` | Random credit card number |
| `cc_type` | Random credit card type |
| `domain_name` | Random domain name |
| `ipv6` | Random IPv6 address |
| `mac_address` | Random MAC address |
| `unix_time` | Random Unix timestamp |
| `currency` | Random currency code |

*Refer to `generators.go` for the latest implementation of these functions.*

## Persistence File

Hepi stores response data in `.hepi.json` in the current directory. This file is updated after every successful request that returns a JSON response. You can inspect this file or delete it to clear the "memory" of previous requests.

## Examples

### 1. Basic Request with Environments

This example shows how to define multiple environments and a simple GET request that uses the `host` variable.

```yaml
environments:
  local:
    host: http://localhost:8080
    api_key: "dev-key-123"
  staging:
    host: https://api.staging.example.com
    api_key: "staging-key-456"

requests:
  get_status:
    method: GET
    url: "{{host}}/v1/status"
    description: "Check API health status"
    headers:
      X-API-Key: "{{api_key}}"
      Accept: "application/json"
```

To execute this request:
```bash
hepi -env local -file test.yaml -req get_status
```

### 2. Request with Dynamic Data

Demonstrating the use of various generators to create a new resource with randomized data.

```yaml
environments:
  local:
    host: http://localhost:8080

requests:
  create_user:
    method: POST
    url: "{{host}}/v1/users"
    description: "Create a new user with random profile data"
    headers:
      Content-Type: "application/json"
    json:
      name: "[[name]]"
      email: "[[email]]"
      username: "[[username]]"
      password: "[[password]]"
      profile:
        bio: "[[sentence]]"
        age: "[[int]]"
        website: "[[url]]"
        phone: "[[phone]]"
```

To execute this request:
```bash
hepi -env local -file test.yaml -req create_user
```

### 3. Result Chaining (Persistence)

This scenario shows a full authentication flow where the token from the login response is reused in a subsequent request.

```yaml
environments:
  local:
    host: http://localhost:8080

requests:
  login:
    method: POST
    url: "{{host}}/v1/auth/login"
    description: "Authenticate and get a token"
    headers:
      Content-Type: "application/json"
    json:
      username: "admin"
      password: "secret-password"

  get_profile:
    method: GET
    url: "{{host}}/v1/profile"
    description: "Fetch user profile using the token from login"
    headers:
      Authorization: "Bearer {{login.token}}"
      Accept: "application/json"

groups:
  auth_flow:
    - login
    - get_profile
```

To execute this group:
```bash
hepi -env local -file test.yaml -group auth_flow
```

### 4. Form Data and Query Parameters

Example of a complex search request using both query parameters and URL-encoded form data.

```yaml
environments:
  local:
    host: http://localhost:8080

requests:
  search_items:
    method: POST
    url: "{{host}}/v1/search"
    description: "Search items with filters and pagination"
    params:
      q: "[[word]]"
      page: "1"
      limit: "20"
      sort: "[[oneof: asc, desc]]"
    form:
      category: "[[oneof: electronics, books, clothing]]"
      include_out_of_stock: "true"
      min_price: "[[int]]"
```

To execute this request:
```bash
hepi -env local -file test.yaml -req search_items
```

### 5. Nested JSON, Arrays, and Header Subscriptions

Showing how to handle complex data structures and reuse specific nested fields from previous results.

```yaml
environments:
  local:
    host: http://localhost:8080

requests:
  setup_project:
    method: POST
    url: "{{host}}/v1/projects"
    description: "Create a complex project structure"
    headers:
      Content-Type: "application/json"
    json:
      title: "Project [[word]]"
      settings:
        visibility: "[[oneof: public, private]]"
        notifications:
          email: true
          push: false
      tags: ["active", "[[word]]", "[[word]]"]
      members:
        - name: "[[name]]"
          role: "owner"
        - name: "[[name]]"
          role: "editor"

  verify_project:
    method: GET
    url: "{{host}}/v1/projects/{{setup_project.id}}"
    description: "Verify the project creation using the ID from the previous request"
    headers:
      X-Project-Owner: "{{setup_project.members.0.name}}"
      Accept: "application/json"
```

To execute these requests:
```bash
hepi -env local -file test.yaml -req setup_project,verify_project
```

### 6. File Uploads (Multipart)

Hepi supports uploading files using `multipart/form-data`. You can combine `form` fields and `files` in the same request.

```yaml
environments:
  local:
    host: http://localhost:8080

requests:
  upload_document:
    method: POST
    url: "{{host}}/v1/upload"
    description: "Upload a document with metadata"
    form:
      category: "financial"
      priority: "high"
    files:
      document: "path/to/report.pdf"
      thumbnail: "path/to/image.png"
```

To execute this request:
```bash
hepi -env local -file test.yaml -req upload_document
```

### 7. CRUD Operations (PUT, PATCH, DELETE)

Hepi supports all standard HTTP methods. This example shows how to update and delete resources.

```yaml
requests:
  update_user:
    method: PUT
    url: "{{host}}/v1/users/{{create_user.id}}"
    json:
      name: "[[name]]"
      active: true

  patch_settings:
    method: PATCH
    url: "{{host}}/v1/users/{{create_user.id}}/settings"
    form:
      theme: "dark"
      notifications: "enabled"

  delete_user:
    method: DELETE
    url: "{{host}}/v1/users/{{create_user.id}}"
    params:
      force: "true"
```

To execute these requests:
```bash
hepi -env local -file test.yaml -req update_user,patch_settings,delete_user
```