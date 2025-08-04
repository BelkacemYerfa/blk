# blk — A Minimalist Dynamic Systems Language

`blk` is a dynamically typed, interpreted language focused on simplicity, expression-oriented design, and minimal syntax. Inspired by Jai, Zig, Odin, and C — but reimagined with flexible semantics and runtime evaluation at its core. Designed for quick scripting, tooling, and prototyping with low ceremony and high expressiveness.

---

## ✨ Why blk?

- Expression-oriented: every block returns a value
- Minimal syntax: easy to read and parse
- Dynamically typed, no explicit type declarations
- Interpreted: fast feedback, no build steps required
- Structs, enums, maps, arrays — all built-in
- Powerful block scoping and control flow
- Unified declaration model using `::` and `:=`

---

## 🚀 Example

```blk
import "math"

User :: struct {
    name,
    age,

    greet: fn() {
        print("Hi, I'm " + name)
    }
}

fn main() {
    u := User{ name: "Ali", age: 22 }
    u.greet()
    msg := if u.age > 18 {
        "Adult"
    } else {
        "Minor"
    }
    print(msg)
}
```

---

## ✅ Language Features

- **Dynamic values**: no static type annotations
- **All variables** declared with `:=`
- **Top-level constants** via `::`
- **Structs with inline methods**
- **Enums and tagged unions**
- **Pattern matching** via `match` expression
- **Expression-based blocks and control flow**
- **Unified literals**: maps and structs share `{}` syntax
- **No distinction between expressions and statements**

---

## 🧱 Core Constructs

### Declarations

```blk
x := 42
msg :: "Welcome to blk"
greet :: fn(name) {
    print("Hello " + name)
}
```

### Structs

```blk
Vec2 :: struct {
    x,
    y,
    len: fn() {
        sqrt(x * x + y * y)
    }
}

v := Vec2{
    x: 3,
    y: 4
}
print(v.len())
```

### Enums

```blk
Result :: enum {
    Ok,
    Error
}
```

---

## 🔁 Control Flow

### If expressions

```blk
name := if loggedIn {
    "User"
} else {
    "Guest"
}
```

### Match expressions

```blk
kind := match x {
    0 => "zero",
    1 => "one",
    _ => "other"
}
```

### While loops

```blk
i := 0
while i < 5 {
    print(i)
    i += 1
}
```

### For loops

```blk
for idx, val in [1, 2, 3] {
    print(idx, val)
}

for k, v in {a: 1, b: 2} {
    print(k, v)
}
```

### Skip

```blk
while true {
    if shouldSkip() {
        skip
    }
    doStuff()
}
```

---

## 📦 Modules & Imports

```blk
import "math"
import "utils"
```

No aliasing needed — always access via `utils::fn`.

---

## 🗃️ Data Types

### Arrays

```blk
nums := [1, 2, 3]
names := ["foo", "bar"]
```

### Maps

```blk
config := {
    host: "localhost",
    port: 8080
}
```

### Struct literals

```blk
person := Person{
    name: "Zed",
    age: 20
}
```

---

## 🧠 Expression-Based Semantics

Every code block is an expression. The last expression is the return value of the block — no `return` keyword required.

```blk
double := fn(x) {
    x * 2
}
```

---

## 🧪 Example Evaluation

```blk
result := fn(x, y) {
    if x > y {
        x
    } else {
        y
    }
}(10, 20)

print(result)  # 20
```

---

## 📐 Data Shape & Reflection

Types are tracked at runtime via introspection:

```blk
typeOf(x) == "int"
```

---

## 🛠️ Development Roadmap

- [x] Lexer and Tokenizer
- [x] Parser and AST
- [ ] Core Interpreter Engine
- [ ] Error System and Stack Traces
- [ ] REPL and Debugger
- [ ] Built-in Modules (math, io, time, etc.)

---

## ⚙️ Tooling

### Run

```bash
blk run main.blk
```

### Build

```bash
git clone https://github.com/yourname/blk
cd blk
go run cmd/main.go run examples/main.blk
```

REPL support planned in future versions.
