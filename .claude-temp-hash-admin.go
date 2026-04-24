package main
import (
  "fmt"
  authpkg "cboard-go/internal/core/auth"
)
func main() {
  hashed, err := authpkg.HashPassword("Sikeming001@")
  if err != nil { panic(err) }
  fmt.Print(hashed)
}
