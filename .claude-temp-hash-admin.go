package main
import (
  "fmt"
  authpkg "cboard-go/internal/core/auth"
)
func main() {
  hashed, err := authpkg.HashPassword("<SERVER_ROOT_PASSWORD_REMOVED>")
  if err != nil { panic(err) }
  fmt.Print(hashed)
}
