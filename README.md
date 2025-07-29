# blk — A Minimalist Systems Programming Language

`blk` is a statically typed, compiled systems programming language focused on simplicity, predictability, and a clean developer experience. Inspired by Jai, Zig, Odin, and C — but designed to be minimal, expressive, and powerful for tooling, low-level system utilities, and experimentation.

---

## ✨ Why blk?

- Minimal syntax: easy to read, easy to parse
- Expression-oriented core
- No classes or objects — just functions, types, and blocks
- Native targets (no runtime)
- Explicit memory control
- Uniform code blocks and low keyword count

---

## 🚀 Example

```blk
import "math"

User :: struct {
    name: string,
    age: int,

    greet: fn() {
        print("Hi, I'm " + name)
    }
}

fn main() {
    u := User{ name: "Ali", age: 22 }
    u.greet()

    status := Status::Offline("Network error")
    msg := match status {
        Online => "Online",
        Offline(reason) => "Offline: " + reason,
        Banned(code) => "Banned with code: " + code,
    }

    print(msg)
}
```

---

## ✅ Language Features

- **Static types**: `int`, `float`, `bool`, `string`, `[]type`, `array(type)`, `map(k, v)`, `fn(...) -> ...`
- **No `let/var` required**: use `:=` for all variable bindings
- **Top-level constants**: use `name :: value` syntax (Jai-style)
- **Structs with methods** using embedded functions
- **Enums** with optional values
- **Pattern matching** via `match` expressions
- **Struct and map literals** share `{}` syntax (based on context)
- **Minimal control flow**: `if`, `match`, `while`, `for`, `skip`
- **No `return` required** in last expression
- **Functions as first-class values**

---

## 🧱 Core Constructs

### Declarations

```blk
x := 42
name :: "blk"
add :: fn(a: int, b: int) int { a + b }
```

### Structs

```blk
Vec2 :: struct {
    x: float,
    y: float,

    length: fn() float {
        return math::sqrt(x * x + y * y)
    }
}

p := Vec2{ x: 3.0, y: 4.0 }
len := p.length()
```

### Enums

```blk
Color :: enum {
    Red,
    Green,
    Blue
}
```

---

## 🔁 Control Flow

### If expressions

```blk
msg := if x > 10 {
    "Large"
} else {
    "Small"
}
```

### Match expressions

```blk
kind := match value {
    1 => "one",
    2 => "two",
    _ => "many"
}
```

### While loops

```blk
i := 0
while i < 10 {
    print(i)
    i += 1
}
```

### For loops

```blk
# Iterate over an array
for idx, value in [1, 2, 3] {
    print(idx, value)
}

# Iterate over a map
for key, value in {"a": 1, "b": 2} {
    print(key, value)
}
```

### Skip

```blk
while x < 10 {
    x += 1
    if x % 2 == 0 {
        skip
    }
    print(x)
}
```

---

## 📦 Modules & Imports

```blk
import "math"
import "io"
```

- Always use `namespace::symbol` form.
- No aliasing (`as`) — keep module names explicit.
- Imports resolve to full top-level namespace.

---

## 🗃️ Data Types

### Arrays

```blk
nums := [1, 2, 3]
names := ["Alice", "Bob"]
```

### Fixed-size arrays

```blk
coords: [3]int = [1, 2, 3]
```

### Dynamic arrays

```blk
items: array(string) = ["a", "b", "c"]
```

**Note:** Inferred type from an array literal is `array(type)` (i.e dynamic array) where `type` is the element type.

### Maps

```blk
settings := {
    volume: 100,
    brightness: 80
}
```

---

## 🧠 Expression Orientation

Every block and control flow structure returns a value. Last expression in a block is implicitly returned.

```blk
double := fn(x: int) int {
    x * 2
}
```

---

## 🛠️ Development Roadmap

- [ ] Lexer and Tokenizer
- [ ] Parser and AST Generator
- [ ] Expression-based syntax model
- [ ] Semantic analysis and type system
- [ ] Pattern matcher and tag-dispatcher
- [ ] LLVM IR backend
- [ ] Standard library (math, io, array, etc.)
- [ ] Macros and compile-time evaluation
- [ ] Error reporting and diagnostics

---

## ⚙️ Tooling

### Compile

```bash
blk compile main.blk -o out
```

### Run

```bash
blk run main.blk
```

### Build (Go-based Compiler)

```bash
git clone https://github.com/yourname/blk
cd blk
go build -o blk cmd/main.go
./blk compile main.blk -o main
./main
```
