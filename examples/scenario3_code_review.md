# Scenario 3: Multi-Perspective Code Review

This example demonstrates using agents for comprehensive code review from different angles.

## Setup

```bash
# Start server
make run-server

# Start specialized review agents
go run cmd/agent/main.go --type security --name "Security Reviewer"
go run cmd/agent/main.go --type backend --name "Code Quality Expert"
go run cmd/agent/main.go --type database --name "Query Reviewer"
```

## Scenario

You've written a user registration endpoint and want multiple perspectives on it.

### Step 1: Share Code for Review

```bash
go run cmd/cli/main.go --channel general --message "
Please review this user registration endpoint:

func RegisterUser(w http.ResponseWriter, r *http.Request) {
    email := r.FormValue('email')
    password := r.FormValue('password')
    
    query := 'INSERT INTO users (email, password) VALUES (' + email + ', ' + password + ')'
    db.Exec(query)
    
    json.NewEncoder(w).Encode(map[string]string{'status': 'ok'})
}

What issues do you see?
"
```

### Expected Multi-Angle Review

**Security Agent** (HIGH PRIORITY):
- "🚨 CRITICAL SECURITY ISSUES:
  1. SQL Injection vulnerability - never concatenate user input into queries
  2. Password is not hashed - NEVER store plaintext passwords
  3. No input validation on email format
  4. No CSRF protection
  5. No rate limiting - vulnerable to account creation spam
  
  Fix immediately before deploying!"

**Backend Agent**:
- "Code quality issues:
  1. No error handling - what if db.Exec fails?
  2. No input validation (email format, password requirements)
  3. Missing response status codes
  4. No logging for audit trail
  5. Should return user ID or created user object"

**Database Agent**:
- "Database concerns:
  1. Use parameterized queries: db.Exec('INSERT INTO users (email, password) VALUES (?, ?)', email, hashedPassword)
  2. Add UNIQUE constraint on email column to prevent duplicates
  3. Consider adding created_at, updated_at timestamps
  4. Missing transaction handling"

### Step 2: Share Improved Version

```bash
go run cmd/cli/main.go --channel general --message "
Here's the updated version:

func RegisterUser(w http.ResponseWriter, r *http.Request) {
    email := r.FormValue('email')
    password := r.FormValue('password')
    
    if !isValidEmail(email) {
        http.Error(w, 'Invalid email', http.StatusBadRequest)
        return
    }
    
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    
    _, err := db.Exec(
        'INSERT INTO users (email, password) VALUES (?, ?)',
        email, hashedPassword,
    )
    if err != nil {
        http.Error(w, 'Registration failed', http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{'status': 'ok'})
}

Better?
"
```

### Expected Follow-up Reviews

**Security Agent**:
- "✅ Much better! SQL injection fixed, password hashing added. Remaining improvements:
  1. Add password strength requirements
  2. Implement rate limiting middleware
  3. Consider email verification flow
  4. Add CSRF token validation
  5. Log registration attempts for security monitoring"

**Backend Agent**:
- "Good progress! Additional suggestions:
  1. Return the created user ID
  2. Check for duplicate email before insertion
  3. Use context.Context for request cancellation
  4. Add structured logging
  5. Consider using a service layer instead of direct DB access"

**Database Agent**:
- "Nice use of parameterized queries! One issue: error handling doesn't distinguish between duplicate email and other errors. Suggest:
  - Check for unique constraint violation specifically
  - Return appropriate error message for duplicate emails
  - Wrap in transaction if you're creating related records"

## Advanced: Specialized Review Channels

Create topic-specific channels for focused reviews:

```bash
# Security-focused review
go run cmd/cli/main.go --create "security-review"
go run cmd/agent/main.go --type security --channel "security-review"

# Performance review
go run cmd/cli/main.go --create "performance-review"
go run cmd/agent/main.go --type backend --channel "performance-review"
go run cmd/agent/main.go --type database --channel "performance-review"
```

## Key Learnings

1. **Layered Review**: Each agent catches different issues
2. **Priority Levels**: Security issues flagged as critical
3. **Constructive Feedback**: Agents explain WHY and HOW to fix
4. **Iterative Improvement**: Agents review multiple versions
5. **Educational Value**: Learn best practices from agent feedback

## Integration with CI/CD

This pattern could be integrated into:
- Pre-commit hooks
- Pull request reviews
- Code quality gates
- Security scanning pipelines

The multi-agent approach catches issues that single-purpose tools might miss!

