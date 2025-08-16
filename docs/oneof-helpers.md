# Oneof Helpers

> Eliminate boilerplate when working with protobuf oneof fields

The `protoc-gen-go-oneof-helper` plugin generates convenience constructor functions for protobuf messages containing oneof fields, dramatically reducing the verbosity and complexity of creating these objects.

## Table of Contents

- [Problem Statement](#problem-statement)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Generated Code Structure](#generated-code-structure)
- [Advanced Examples](#advanced-examples)
- [Best Practices](#best-practices)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Problem Statement

Protobuf oneof fields provide excellent type safety but create verbose, error-prone code when constructing messages manually.

### Without sebuf (verbose and error-prone):
```go
// Creating a user authentication request with email
request := &CreateUserRequest{
    AuthMethod: &CreateUserRequest_Email{
        Email: &CreateUserRequest_EmailAuth{
            Email:    "user@example.com",
            Password: "password123",
        },
    },
}

// Creating the same request with token auth
request := &CreateUserRequest{
    AuthMethod: &CreateUserRequest_Token{
        Token: &CreateUserRequest_TokenAuth{
            Token: "abc123token",
        },
    },
}
```

### With sebuf (clean and simple):
```go
// Email authentication
emailRequest := NewCreateUserRequestEmail("user@example.com", "password123")

// Token authentication  
tokenRequest := NewCreateUserRequestToken("abc123token")
```

## Installation

```bash
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
```

Verify installation:
```bash
protoc-gen-go-oneof-helper --version
```

## Quick Start

### 1. Define your protobuf with oneof fields

Create `auth.proto`:
```protobuf
syntax = "proto3";
package auth;
option go_package = "github.com/yourorg/yourapi/auth";

message LoginRequest {
  message EmailAuth {
    string email = 1;
    string password = 2;
  }
  
  message PhoneAuth {
    string phone = 1;
    string code = 2;
  }
  
  message SocialAuth {
    string provider = 1;  // "google", "facebook", etc.
    string token = 2;
  }
  
  oneof auth_method {
    EmailAuth email = 1;
    PhoneAuth phone = 2;
    SocialAuth social = 3;
  }
}
```

### 2. Generate helpers

```bash
protoc --go_out=. --go_opt=module=github.com/yourorg/yourapi \
       --go-oneof-helper_out=. \
       auth.proto
```

### 3. Use the generated helpers

```go
package main

import (
    "fmt"
    "github.com/yourorg/yourapi/auth"
)

func main() {
    // Create different authentication requests easily
    emailReq := auth.NewLoginRequestEmail("user@example.com", "secret123")
    phoneReq := auth.NewLoginRequestPhone("+1234567890", "123456")
    socialReq := auth.NewLoginRequestSocial("google", "oauth_token_here")
    
    processLogin(emailReq)
    processLogin(phoneReq)
    processLogin(socialReq)
}

func processLogin(req *auth.LoginRequest) {
    switch auth := req.AuthMethod.(type) {
    case *auth.LoginRequest_Email:
        fmt.Printf("Email login: %s\n", auth.Email.Email)
    case *auth.LoginRequest_Phone:
        fmt.Printf("Phone login: %s\n", auth.Phone.Phone)
    case *auth.LoginRequest_Social:
        fmt.Printf("Social login via %s\n", auth.Social.Provider)
    }
}
```

## Generated Code Structure

For the `LoginRequest` example above, the plugin generates:

```go
// NewLoginRequestEmail creates a new LoginRequest with Email set
func NewLoginRequestEmail(email string, password string) *LoginRequest {
    return &LoginRequest{
        AuthMethod: &LoginRequest_Email{
            Email: &LoginRequest_EmailAuth{
                Email:    email,
                Password: password,
            },
        },
    }
}

// NewLoginRequestPhone creates a new LoginRequest with Phone set  
func NewLoginRequestPhone(phone string, code string) *LoginRequest {
    return &LoginRequest{
        AuthMethod: &LoginRequest_Phone{
            Phone: &LoginRequest_PhoneAuth{
                Phone: phone,
                Code:  code,
            },
        },
    }
}

// NewLoginRequestSocial creates a new LoginRequest with Social set
func NewLoginRequestSocial(provider string, token string) *LoginRequest {
    return &LoginRequest{
        AuthMethod: &LoginRequest_Social{
            Social: &LoginRequest_SocialAuth{
                Provider: provider,
                Token:    token,
            },
        },
    }
}
```

### Function Naming Convention

Generated functions follow the pattern: `New{MessageName}{OneofFieldName}(parameters...)`

- `LoginRequest` + `Email` → `NewLoginRequestEmail`
- `PaymentMethod` + `CreditCard` → `NewPaymentMethodCreditCard`
- `DatabaseConfig` + `Postgres` → `NewDatabaseConfigPostgres`

## Advanced Examples

### Complex Nested Types

The plugin handles complex field types including repeated fields, maps, optional fields, and nested messages:

```protobuf
message ShoppingCart {
  message Item {
    string product_id = 1;
    int32 quantity = 2;
    map<string, string> metadata = 3;
    repeated string tags = 4;
  }
  
  message Discount {
    string code = 1;
    optional double percentage = 2;
    optional int64 amount_cents = 3;
  }
  
  oneof payment_method {
    CreditCard credit_card = 1;
    PayPal paypal = 2;
    GiftCard gift_card = 3;
  }
  
  repeated Item items = 4;
  optional Discount discount = 5;
}

message CreditCard {
  string number = 1;
  string cvv = 2;
  int32 exp_month = 3;
  int32 exp_year = 4;
}
```

Generated helper:
```go
// Complex types are handled automatically
cart := NewShoppingCartCreditCard("4111111111111111", "123", 12, 2025)

// The generated function signature matches the nested message fields
func NewShoppingCartCreditCard(number string, cvv string, expMonth int32, expYear int32) *ShoppingCart
```

### Enum Fields

```protobuf
enum Priority {
  LOW = 0;
  MEDIUM = 1;
  HIGH = 2;
  URGENT = 3;
}

message Task {
  message EmailNotification {
    string recipient = 1;
    Priority priority = 2;
  }
  
  message SlackNotification {
    string channel = 1;
    bool mention_everyone = 2;
  }
  
  oneof notification {
    EmailNotification email = 1;
    SlackNotification slack = 2;
  }
}
```

Usage:
```go
// Enums are handled naturally
emailTask := NewTaskEmail("admin@company.com", Priority_HIGH)
slackTask := NewTaskSlack("#alerts", true)
```

### Nested Messages and Recursive Structures

```protobuf
message Document {
  message Section {
    string title = 1;
    string content = 2;
    repeated Document nested_documents = 3;
  }
  
  message Attachment {
    string filename = 1;
    bytes data = 2;
    string mime_type = 3;
  }
  
  oneof content_type {
    Section section = 1;
    Attachment attachment = 2;
  }
}
```

The plugin correctly handles recursive and nested types:
```go
// Create a document with a section
doc := NewDocumentSection("Introduction", "Welcome to our API documentation", nil)

// Create a document with an attachment
attachment := NewDocumentAttachment("api.pdf", pdfData, "application/pdf")
```

## Best Practices

### 1. Organize Oneof Fields Logically

Group related alternatives in oneof fields:

```protobuf
// Good: Clear authentication methods
oneof auth_method {
  EmailAuth email = 1;
  PhoneAuth phone = 2;
  TokenAuth token = 3;
}

// Good: Payment options
oneof payment {
  CreditCard credit_card = 1;
  PayPal paypal = 2;
  BankTransfer bank_transfer = 3;
}
```

### 2. Use Descriptive Field Names

```protobuf
// Good: Clear field names
oneof notification_channel {
  EmailNotification email_notification = 1;
  SMSNotification sms_notification = 2;
  PushNotification push_notification = 3;
}

// Avoid: Generic names
oneof method {
  TypeA a = 1;
  TypeB b = 2;
}
```

### 3. Keep Nested Messages Simple

```protobuf
// Good: Simple, focused message
message EmailAuth {
  string email = 1;
  string password = 2;
}

// Consider refactoring: Too many fields
message ComplexAuth {
  string email = 1;
  string password = 2;
  string backup_email = 3;
  repeated string recovery_codes = 4;
  map<string, string> metadata = 5;
  // ... many more fields
}
```

### 4. Validation and Error Handling

The generated helpers create valid protobuf messages, but you should still validate business logic:

```go
func CreateUserWithEmail(email, password string) (*User, error) {
    // Use the helper to eliminate boilerplate
    request := auth.NewLoginRequestEmail(email, password)
    
    // Add your business validation
    if !isValidEmail(email) {
        return nil, fmt.Errorf("invalid email format: %s", email)
    }
    
    if len(password) < 8 {
        return nil, fmt.Errorf("password must be at least 8 characters")
    }
    
    return processAuthRequest(request)
}
```

## Configuration

The plugin works without configuration, but you can customize its behavior through protoc options:

### Basic Usage
```bash
protoc --go-oneof-helper_out=. your_file.proto
```

### Custom Output Directory
```bash
protoc --go-oneof-helper_out=./generated your_file.proto
```

### With Module Option
```bash
protoc --go_out=. --go_opt=module=github.com/yourorg/yourapi \
       --go-oneof-helper_out=. \
       your_file.proto
```

### Integration with Makefile

Add to your `Makefile`:
```makefile
.PHONY: generate
generate:
	protoc --go_out=. --go_opt=module=$(MODULE_NAME) \
	       --go-oneof-helper_out=. \
	       --proto_path=. \
	       $(PROTO_FILES)

.PHONY: clean-generated
clean-generated:
	find . -name "*_helpers.pb.go" -delete
```

## Troubleshooting

### Common Issues

#### 1. Plugin Not Found
```
protoc-gen-go-oneof-helper: program not found or is not executable
```

**Solution:**
```bash
# Ensure the plugin is in your PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Or reinstall
go install github.com/SebastienMelki/sebuf/cmd/protoc-gen-go-oneof-helper@latest
```

#### 2. No Helpers Generated
If no helper functions are generated, check:

- **Oneof fields exist**: The plugin only generates helpers for messages with oneof fields
- **Nested messages**: Only oneof fields containing message types (not scalars) get helpers
- **File generation**: Ensure the proto file is set to generate Go code

```protobuf
// This will generate helpers (message type in oneof)
oneof auth_method {
  EmailAuth email = 1;  // ✅ Message type
}

// This will NOT generate helpers (scalar type in oneof)  
oneof value {
  string text = 1;      // ❌ Scalar type
  int32 number = 2;     // ❌ Scalar type
}
```

#### 3. Import Path Issues
```
cannot find package "github.com/yourorg/yourapi/generated"
```

**Solution:** Ensure your module path matches your go.mod:
```bash
protoc --go_out=. --go_opt=module=$(cat go.mod | head -1 | cut -d' ' -f2) \
       --go-oneof-helper_out=. \
       your_file.proto
```

#### 4. Generated Files Not Updated
After changing proto definitions, regenerate:
```bash
# Clean old generated files
find . -name "*_helpers.pb.go" -delete

# Regenerate
make generate  # or your protoc command
```

### Getting Help

If you encounter issues:

1. **Try the simple demo** in the [examples guide](../examples/)
2. **Review the test cases** in `internal/oneofhelper/testdata/`
3. **File an issue** with your proto definition and generated code
4. **Join the discussion** in GitHub Discussions

## Integration with Other Tools

### With sebuf HTTP Generation

Combine oneof helpers with HTTP generation for clean API handlers:

```go
func (h *AuthHandler) Login(c *gin.Context) {
    var req struct {
        Email    string `json:"email,omitempty"`
        Password string `json:"password,omitempty"`
        Phone    string `json:"phone,omitempty"`
        Code     string `json:"code,omitempty"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    var loginReq *auth.LoginRequest
    if req.Email != "" {
        loginReq = auth.NewLoginRequestEmail(req.Email, req.Password)
    } else if req.Phone != "" {
        loginReq = auth.NewLoginRequestPhone(req.Phone, req.Code)
    } else {
        c.JSON(400, gin.H{"error": "missing authentication method"})
        return
    }
    
    // Process the clean, type-safe request
    user, err := h.authService.Login(loginReq)
    // ...
}
```

### With Testing

Generated helpers make testing much cleaner:

```go
func TestUserAuthentication(t *testing.T) {
    tests := []struct {
        name    string
        request *auth.LoginRequest
        wantErr bool
    }{
        {
            name:    "valid email auth",
            request: auth.NewLoginRequestEmail("test@example.com", "password123"),
            wantErr: false,
        },
        {
            name:    "valid phone auth", 
            request: auth.NewLoginRequestPhone("+1234567890", "123456"),
            wantErr: false,
        },
        {
            name:    "valid social auth",
            request: auth.NewLoginRequestSocial("google", "valid_token"),
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := authenticateUser(tt.request)
            if (err != nil) != tt.wantErr {
                t.Errorf("authenticateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

**Next:** Learn how to generate HTTP APIs from protobuf services with [HTTP Generation](./http-generation.md)

**See also:**
- [Getting Started Guide](./getting-started.md)
- [Simple Demo](./examples/)
- [Architecture Overview](./architecture.md)