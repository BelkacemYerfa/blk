# blk — A Minimalist Systems Programming Language

`blk` is a statically typed, compiled programming language focused on simplicity, predictability, and a clean developer experience. Inspired by languages like Zig, Odin, and C — but designed to be approachable for learners and hobbyists interested in writing small tools, utilities, and experimenting with low-level systems programming.

---

## ✨ Why blk?

- Minimal syntax: easy to read, easy to parse
- Explicit types: no hidden behavior
- No classes or objects — just functions, types, and modules
- Single-pass compilation model
- Stack-based VM (planned) and direct native compilation targets

---

## 🚀 Example

```blk
import "math.blk"
import "utils.blk"

export fn main(): int {
    let result: int = utils::double(5)
    print(result)
    return 0
}

export fn add(a: int, b: int): int {
    return a + b
}
```

---

## ✅ Language Features

- Static types: `int`, `float`, `bool`, `string`, `[type]` (arrays), structs via `type`
- Functions with explicit return types and parameters
- Variables using `let` (immutable) and `var` (mutable)
- Custom types with `type`, including aliases and structs:

```blk
type ID = int

type Person {
    name: string
    age: int
}
```

- Array declaration using `[type]`:

```blk
let numbers: [int] = [1, 2, 3, 4]
let names: [string] = ["Alice", "Bob"]
```

- Import system using `import "file.blk"`. No aliasing via `as`; always use namespace prefix:

```blk
import "math.blk"
import "utils.blk"

let result: int = utils::double(10)
```

- Conditionals: `if`, `else if`, `else`
- Loops:

```blk
var i: int = 0
while i < 10 {
    print(i)
    i = i + 1
}
```

- Loop control keyword: `skip`:

```blk
var i: int = 0
while i < 10 {
    i = i + 1
    if i % 2 == 0 {
        skip
    }
    print(i)
}
```

- Explicit memory management (planned)
- No runtime garbage collection: fully manual or allocator-driven
- Macro system (planned): clean, hygienic, declarative and procedural macros planned for later versions

---

## 📚 Syntax Reference

### Variables

```blk
let x: int = 42
var y: float = 3.14
```

### Functions

```blk
fn double(value: int): int {
    return value * 2
}
```

### Custom Types

```blk
type Point {
    x: float
    y: float
}

fn length(p: Point): float {
    return math::sqrt(p.x * p.x + p.y * p.y)
}
```

### Arrays

```blk
let numbers: [int] = [10, 20, 30]
let first: int = numbers[0]
```

### Conditionals

```blk
if x > 10 {
    print("Greater")
} else {
    print("Smaller or equal")
}
```

### Loops

```blk
var i: int = 0
while i < 10 {
    print(i)
    i = i + 1
}
```

### Importing Modules

```blk
import "math.blk"
import "utils.blk"
```

- Always use `namespace::symbol` format. No **ALIASING** keyword.

### Exported Symbols

```blk
export fn add(a: int, b: int): int {
    return a + b
}
```

---

## 🛠️ Development Roadmap

- [x] Lexer and Tokenizer
- [ ] Parser and AST generator
- [ ] Semantic analysis and type checking
- [ ] Bytecode interpreter (stack-based VM)
- [ ] LLVM IR backend
- [ ] Official standard library (blk stdlib)
- [ ] Macro system

---

## ⚙️ Build & Usage

### Compiling blk programs

```bash
blk compile main.blk -o main.blkout
```

### Running blk programs

```bash
blk run main.blk
```

### Building the Compiler (Go example)

```bash
git clone https://github.com/yourname/blk
cd blk
go build -o blk cmd/main.go
./blk compile main.blk -o main
./main
```
