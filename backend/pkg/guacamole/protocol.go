package guacamole

import (
    "fmt"
    "io"
    "strconv"
    "strings"
)

// Instruction represents a Guacamole protocol instruction.
// Format: length.field1,length.field2,...,length.fieldN;
type Instruction struct {
    Op    string
    Args  []string
}

// WriteInstruction encodes an instruction and writes it to the writer.
func WriteInstruction(w io.Writer, op string, args ...string) error {
    parts := make([]string, 0, len(args)+1)
    parts = append(parts, encodeField(op))
    for _, arg := range args {
        parts = append(parts, encodeField(arg))
    }
    _, err := io.WriteString(w, strings.Join(parts, ",")+";")
    return err
}

func encodeField(s string) string {
    return fmt.Sprintf("%d.%s", len(s), s)
}

// ReadInstruction reads one instruction from the reader.
func ReadInstruction(r io.Reader) (*Instruction, error) {
    // Read until ';' delimiter
    var buf []byte
    one := make([]byte, 1)
    for {
        n, err := r.Read(one)
        if err != nil {
            return nil, err
        }
        if n == 0 {
            continue
        }
        if one[0] == ';' {
            break
        }
        buf = append(buf, one[0])
    }

    raw := string(buf)
    if raw == "" {
        return nil, fmt.Errorf("empty instruction")
    }

    // Parse fields: "5.abc,3.def;"
    fields := strings.Split(raw, ",")
    instr := &Instruction{}
    for i, field := range fields {
        dotIdx := strings.Index(field, ".")
        if dotIdx < 0 {
            continue
        }
        length, err := strconv.Atoi(field[:dotIdx])
        if err != nil {
            continue
        }
        value := field[dotIdx+1:]
        if len(value) > length {
            value = value[:length]
        }
        if i == 0 {
            instr.Op = value
        } else {
            instr.Args = append(instr.Args, value)
        }
    }
    return instr, nil
}
