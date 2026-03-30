# mapper

Generic functional utilities for collection transformation and enum mapping.

## Functions

### `Slice[T, U any](items []T, fn func(T) U) []U`

Maps a slice by applying a function to each element. Returns a new slice of the same length.

```go
names := mapper.Slice(users, func(u User) string { return u.Name })
// []User{Name: "Alice", Name: "Bob"} -> []string{"Alice", "Bob"}
```

### `Enum[From comparable, To any](m map[From]To, from From, fallback To) To`

Looks up a value in a map, returning `fallback` if the key is not found.

```go
var grpcCode = map[errors.Code]codes.Code{
    errors.CodeNotFound:    codes.NotFound,
    errors.CodeUnauthorized: codes.Unauthenticated,
}

code := mapper.Enum(grpcCode, errors.CodeNotFound, codes.Internal)
// returns codes.NotFound
```

### `EnumWithOk[From comparable, To any](m map[From]To, from From) (To, bool)`

Looks up a value in a map, returning the value and whether the key was found.

```go
code, ok := mapper.EnumWithOk(grpcCode, errors.CodeNotFound)
// code = codes.NotFound, ok = true
```

## Usage

These utilities are used throughout the codebase for DTO↔entity transformations and enum/code lookups.

```go
// Transform a list of entities to DTOs
dtos := mapper.Slice(rooms, roomEntityToDTO)

// Map error codes to gRPC codes
grpcCode := mapper.Enum(errorCodeMap, errCode, codes.Internal)
```
