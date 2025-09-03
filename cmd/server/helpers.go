package main

// add is a tiny helper to give the main package a testable surface.
func add(a, b int) int { return a + b }

// sanitizePort returns a sensible default when empty.
func sanitizePort(p string) string {
    if p == "" {
        return "8080"
    }
    return p
}

// sumN sums 1..n (inclusive). Returns 0 for n<=0.
func sumN(n int) int {
    if n <= 0 { return 0 }
    s := 0
    for i := 1; i <= n; i++ { s += i }
    return s
}

// chooseCommitment returns the provided commitment, or the default value when empty.
func chooseCommitment(s string) string {
    if s == "" { return "finalized" }
    return s
}
